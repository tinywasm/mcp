# PLAN: tinywasm/mcp — Pending Corrections

> Date: 2026-04-01
> Status: Ready for execution — Stage 2 first

---

## Context

Stages 1 and 1.5 were completed in a prior iteration. This plan covers the remaining work: security hardening, RBAC enforcement, SSE transport, and documentation rewrite.

## Current State (verified 2026-04-01)

| Item | Status |
|------|--------|
| `NewServer` returns `*Server` (no error) | Needs change |
| `Config` has `Port`, `APIKey` | Must remove |
| `Server` has `apiKey`, `port`, `ideStatus` | Must remove |
| `handler_ide.go` + `handler_ide_wasm.go` exist | Must delete |
| `Authorizer` only has `Authorize()` | Missing `Can()` |
| `handleToolCall` skips RBAC | Must enforce |
| No `SSEPublisher` interface | Must add |
| `ARCHITECTURE.md` references deleted types | Must rewrite |
| `README.md` references `HTTPHandler()`, `NewBearerAuth()` | Must fix |

---

## Confirmed Design Decisions (inherited)

| Decision | Resolution |
|----------|-----------|
| Auth nil | `NewServer` returns error if `Config.Auth == nil` |
| Authorizer | Single interface with `Authorize()` + `Can()` |
| HTTP Transport | Streamable HTTP via `tinywasm/sse` injected by consumer |
| SSE injection | `Config.SSE SSEPublisher` — consumer passes `*sse.SSEServer` |
| No HTTP ownership | `mcp` never owns HTTP routing. Only exposes `HandleMessage()` |
| ConfigureIDEs | Moves to `tinywasm/app` — delete from mcp |

---

## Execution Order

```
Stage 2 → Stage 3 → Stage 4 → Stage 5
```

Stages 2 and 3 are blockers for Stage 4 because SSE publishing depends on `Authorizer` having `Can()`.

---

## Stage 2 — Startup Security

### Goal
`NewServer` must reject nil Auth. Remove HTTP-layer fields (`Port`, `APIKey`) that belong to `tinywasm/app`. Delete `ConfigureIDEs` from mcp.

### Changes

**A. `NewServer` returns `(*Server, error)`**

```go
func NewServer(config Config, providers []ToolProvider) (*Server, error) {
    if config.Auth == nil {
        return nil, fmt.Err("mcp", "Auth is required — use mcp.OpenAuthorizer() for open access")
    }
    // ... existing logic ...
    return s, nil
}
```

**B. Clean Config and Server structs**

Remove from `Config`: `Port`, `APIKey`
Remove from `Server`: `apiKey`, `port`, `ideStatus`

Final Config:
```go
type Config struct {
    Name    string
    Version string
    Auth    Authorizer
}
```

**C. Delete IDE files**

Delete `handler_ide.go` and `handler_ide_wasm.go`. `ConfigureIDEs` moves to `tinywasm/app`.

**D. Add built-in Authorizer implementations in mcp_auth.go**

```go
func NewTokenAuthorizer(apiKey string) Authorizer
func OpenAuthorizer() Authorizer
```

Both implement `Can()` returning `true` always (Stage 3 adds the full interface).

### Tests

```
TestNewServer_NilAuth_ReturnsError
TestNewServer_OpenAuthorizer_Starts
TestNewServer_TokenAuthorizer_Starts
TestHandleMessage_NoAuthToken_Rejected      (TokenAuthorizer)
TestHandleMessage_InvalidToken_Rejected     (TokenAuthorizer)
TestHandleMessage_ValidToken_Passes         (TokenAuthorizer)
TestHandleMessage_OpenAuthorizer_NoToken_Passes
```

### Steps
- [ ] Change `NewServer` signature to return `(*Server, error)`
- [ ] Add nil Auth validation in `NewServer`
- [ ] Remove `Port`, `APIKey` from `Config`
- [ ] Remove `port`, `apiKey`, `ideStatus` from `Server`
- [ ] Delete `handler_ide.go`
- [ ] Delete `handler_ide_wasm.go`
- [ ] Add `NewTokenAuthorizer(apiKey)` and `OpenAuthorizer()` in mcp_auth.go
- [ ] Update all tests to handle `(*Server, error)` return and pass Auth
- [ ] Add security tests listed above

### Impact on Consumers
- `tinywasm/app` must handle `(*Server, error)` return
- `tinywasm/app` must own `Port`, `APIKey`, and `ConfigureIDEs` logic
- All consumers must provide `Auth` in Config

---

## Stage 3 — Mandatory RBAC (Unified Authorizer)

### Goal
Extend `Authorizer` with `Can()`. Enforce RBAC check in `handleToolCall` before `Execute`.

### Changes

**A. Extend Authorizer interface**

```go
type Authorizer interface {
    Authorize(token string) (userID string, err error)
    Can(userID, resource string, action byte) bool
}
```

**B. Update built-in implementations**

`NewTokenAuthorizer` and `OpenAuthorizer` — `Can()` always returns `true`.

**C. Enforce in handleToolCall**

```go
func (s *Server) handleToolCall(ctx *context.Context, id string, params callToolParams) (*Result, *requestError) {
    tool, ok := s.tools[params.Name]
    if !ok {
        return nil, &requestError{id: id, code: INVALID_PARAMS, err: fmt.Err("mcp", "tool not found")}
    }

    userID := ctx.Value(CtxKeyUserID)
    if !s.auth.Can(userID, tool.Resource, tool.Action) {
        return nil, &requestError{id: id, code: -32001, err: fmt.Err("mcp", "forbidden")}
    }

    req := Request{Params: params}
    result, err := tool.Execute(ctx, req)
    if err != nil {
        return &Result{IsError: true, Content: Text(err.Error()).Content}, nil
    }
    return result, nil
}
```

### Tests

```
TestHandleToolCall_Can_False_Rejected
TestHandleToolCall_Can_True_Executes
TestHandleToolCall_WrongAction_Rejected
TestHandleToolCall_WrongResource_Rejected
TestHandleToolCall_CanNeverCalledIfAuthorizeFails
TestHandleToolCall_ExecuteNeverCalledIfCanFalse
TestConcurrentToolCalls_DifferentUsers
```

### Steps
- [ ] Add `Can(userID, resource string, action byte) bool` to `Authorizer` interface
- [ ] Update `NewTokenAuthorizer` — `Can` always returns `true`
- [ ] Update `OpenAuthorizer` — `Can` always returns `true`
- [ ] Add RBAC check in `handleToolCall` before `Execute`
- [ ] Add all RBAC tests listed above
- [ ] Update mockAuth in tests to implement `Can()`

---

## Stage 4 — SSE Transport (Streamable HTTP)

### Goal
Inject `tinywasm/sse` via interface for streaming notifications. Consumer owns HTTP routing.

### Changes

**A. SSEPublisher interface** (build `!wasm`)

```go
type SSEPublisher interface {
    Publish(data []byte, channel string)
}
```

**B. Add to Config**

```go
type Config struct {
    Name    string
    Version string
    Auth    Authorizer
    SSE     SSEPublisher  // optional — nil means no streaming
}
```

**C. Use in notification handlers**

When SSE is present, `s.sse.Publish(...)` on tool list changes and other notifications.

### Tests

```
TestHandleMessage_WithSSE_PublishesNotification
TestHandleMessage_WithoutSSE_NoPublish
TestConfig_SSENil_Accepted
```

### Steps
- [ ] Add `SSEPublisher` interface in a `server_sse.go` file (build `!wasm`)
- [ ] Add `SSE SSEPublisher` field to `Config` and `Server`
- [ ] Use `s.sse.Publish(...)` in notification handlers when SSE is present
- [ ] Verify `*sse.SSEServer` satisfies `SSEPublisher` without adapter
- [ ] Add all SSE tests listed above

---

## Stage 5 — Documentation

### Goal
ARCHITECTURE.md and README.md must reflect the actual code after stages 2-4.

### ARCHITECTURE.md — Full Rewrite

Remove references to: `MCPServer`, `ProtocolTool`, `SSEHub`, `Handler`, `session.go`, `handler.go`, `handler_executor.go`, `transport_streamable_http.go`, `tool_builders.go`, `request_handler.go` (as router — it's now inline in server).

Document current files:
- server.go — `Server`, `Config`, `NewServer`, `HandleMessage`, tool handlers
- request_handler.go — `HandleMessage` dispatch, `ExtractJSONValue`, JSON extraction
- mcp_auth.go — `Authorizer`, `NewTokenAuthorizer`, `OpenAuthorizer`
- provider.go — `Tool`, `ToolProvider`, `Request`, `Result`
- tools.go — `toolEntry`, `InputSchema`, wire format
- types.go — JSON-RPC 2.0 types, error codes
- model.go / model_orm.go — data models
- client.go — MCP client
- constants.go — protocol constants
- errors.go — error helpers
- utils.go — result helpers (`Text`, `JSON`, `GetText`)
- logger.go — logging interface

Add diagrams:
- Auth + RBAC flow
- Streamable HTTP flow with SSE

### README.md — Updates

- Remove `srv.HTTPHandler()` (doesn't exist — consumer owns HTTP)
- Remove `user.NewBearerAuth(secret)` — replace with `NewTokenAuthorizer` / `OpenAuthorizer` / `userModule.MCPAuthorizer()`
- Add `ormc` installation section
- Add SSE section
- Fix API reference table

### Steps
- [x] Rewrite ARCHITECTURE.md with current file structure
- [x] Add auth + RBAC flow diagram
- [x] Add SSE flow diagram
- [x] Update README: auth section with actual implementations
- [x] Update README: remove `HTTPHandler()` reference
- [x] Update README: add `ormc` install section
- [x] Update README: add SSE section
- [x] Update README: fix API reference table

---

## External Dependencies

| Package | Stage | Reason |
|---------|-------|--------|
| `github.com/tinywasm/sse` | Stage 4 | `SSEPublisher` interface satisfaction |

Note: dependency only in `!wasm` builds. Protocol core remains dependency-free.
