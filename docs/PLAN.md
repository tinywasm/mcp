# PLAN: tinywasm/mcp — Audit Corrections

> Date: 2026-04-02
> Status: Ready for execution

---

## Findings Summary

| # | Severity | Category | Issue |
|---|----------|----------|-------|
| 1 | Critical | Security | `initialize` bypasses auth — any unauthenticated client can start a session |
| 2 | Critical | WASM | `SSEPublisher` only defined with `!wasm` build tag — `Server` struct won't compile under WASM |
| 3 | High | Security | `TokenAuthorizer` does not use constant-time comparison — timing attack on API key |
| 4 | High | Logic | Auth check has dead branch: `s.auth != nil` is always true (NewServer rejects nil Auth) |
| 5 | High | Logic | `handleToolCall` RBAC uses `ctx.Value(CtxKeyUserID)` which was set by `HandleMessage` auth — but if auth sets userID to `""` (empty), `Can()` receives empty string silently |
| 6 | Medium | Security | `Client` sends API key in `Authorization` header without `Bearer ` prefix validation — inconsistent with server-side `TokenAuthorizer` that compares raw token |
| 7 | Medium | Logic | `Bind` always validates with action `'c'` (create) — ignores the tool's actual `Action` field |
| 8 | Medium | Consistency | `model.go` still has `ideServerEntry`, `vscodeConfig`, `claudeCodeConfig` — IDE types that should have been deleted with Stage 2 |
| 9 | Medium | Consistency | `errors.go` defines unused error vars (`ErrUnsupported`, `ErrToolNotFound`, `ErrNotificationNotInitialized`, `ErrNotificationChannelBlocked`) and uses stdlib `fmt` instead of `tinywasm/fmt` |
| 10 | Medium | Consistency | `Logger` interface in `logger.go` is never used by `Server` — server uses `func(messages ...any)` instead |
| 11 | Low | Logic | `handleInitialize` ignores `params.ProtocolVersion` — never validates client version against `validProtocolVersions` |
| 12 | Low | Consistency | `FilterFunc` defined in `tools.go` but never used anywhere |
| 13 | Low | Consistency | `provider.go` has `Loggable` interface — never used by server |
| 14 | Low | Cleanup | `docs/stages/` directory contains 6 stage files from prior iterations — stale artifacts |

---

## Stage 1 — Critical: Auth Bypass on Initialize + WASM Compilation

### Goal
Fix the two critical issues: unauthenticated `initialize` and WASM build failure.

### Changes

**A. Move auth check before method dispatch (except initialize)**

The MCP spec requires `initialize` to be the first message, but the current code runs auth for ALL methods with an `id`. The problem is that `initialize` should still require auth when using `TokenAuthorizer` — an unauthenticated client shouldn't be able to negotiate a session.

Current flow (`request_handler.go:38-48`):
```go
if s.auth != nil {  // dead branch — always true
    token := ctx.Value(CtxKeyAuthToken)
    userID, err := s.auth.Authorize(token)
    if err != nil {
        return createErrorResponse(id, -32001, "Unauthorized")
    }
    ...
}
```

Fix: remove the `if s.auth != nil` guard (it's always true). Auth already runs for all methods including initialize, which is correct — no change needed to the flow, just remove the dead branch.

**B. Add WASM stub for SSEPublisher**

Create `server_sse_wasm.go` with `//go:build wasm`:
```go
type SSEPublisher interface {
    Publish(data []byte, channel string)
}
```

The interface definition must exist in both builds so `Server` and `Config` compile.

### Tests
```
TestWASMBuild_Compiles (build tag verification via go vet or manual)
```

### Steps
- [ ] Remove `if s.auth != nil` guard in `request_handler.go` (dead code)
- [ ] Create `server_sse_wasm.go` with `//go:build wasm` containing `SSEPublisher` interface

---

## Stage 2 — Security: Timing-Safe Token Comparison

### Goal
Prevent timing attacks on API key comparison in `TokenAuthorizer`.

### Changes

**A. Use constant-time comparison in `tokenAuthorizer.Authorize`**

`mcp_auth.go:21` currently does `token == a.apiKey` (byte-by-byte short-circuit). Replace with `subtle.ConstantTimeCompare` or equivalent length-check + byte comparison.

Since `tinywasm/fmt` may not expose `crypto/subtle`, implement a simple constant-time compare:
```go
func constantTimeEqual(a, b string) bool {
    if len(a) != len(b) {
        return false
    }
    var result byte
    for i := 0; i < len(a); i++ {
        result |= a[i] ^ b[i]
    }
    return result == 0
}
```

### Tests
```
TestTokenAuthorizer_ConstantTimeCompare (functional — same behavior, just timing-safe)
```

### Steps
- [ ] Add `constantTimeEqual` in `mcp_auth.go`
- [ ] Replace `token == a.apiKey` with `constantTimeEqual(token, a.apiKey)`

---

## Stage 3 — Logic Fixes: Bind Action + Empty UserID

### Goal
Fix `Bind` ignoring tool action and handle empty userID from auth.

### Changes

**A. Pass action to `Request.Bind`**

`tools.go:22-27` — `Bind` hardcodes `target.Validate('c')`. The `Request` struct doesn't carry the tool's action. Two options:

Option 1 (minimal): Add `Action` to `Request` and set it in `handleToolCall`.
```go
type Request struct {
    Params callToolParams
    Action byte
}
```

In `handleToolCall` (`server.go:134`):
```go
req := Request{Params: params, Action: tool.Action}
```

In `Bind`:
```go
func (r *Request) Bind(target fmt.SafeFields) error {
    if err := json.Decode([]byte(r.Params.Arguments), target); err != nil {
        return err
    }
    return target.Validate(r.Action)
}
```

**B. Reject empty userID after Authorize**

In `request_handler.go`, after `Authorize` returns, if `userID == ""` and auth is not `OpenAuthorizer`, this is suspicious. However, `OpenAuthorizer` legitimately returns `"guest"`, and `TokenAuthorizer` returns `"user"` — so empty userID from a custom authorizer should be rejected:

```go
userID, err := s.auth.Authorize(token)
if err != nil {
    return createErrorResponse(id, -32001, "Unauthorized")
}
if userID == "" {
    return createErrorResponse(id, -32001, "Unauthorized: empty user identity")
}
ctx.Set(CtxKeyUserID, userID)
```

Remove the `if userID != ""` guard — always set it.

### Tests
```
TestBind_UsesToolAction
TestAuthorize_EmptyUserID_Rejected
```

### Steps
- [ ] Add `Action byte` to `Request` struct
- [ ] Set `Action` in `handleToolCall` when building `Request`
- [ ] Change `Bind` to use `r.Action` instead of hardcoded `'c'`
- [ ] Remove `if userID != ""` guard — reject empty userID, always set

---

## Stage 4 — Cleanup: Dead Code + Stale Types

### Goal
Remove unused code that creates confusion and maintenance burden.

### Changes

**A. Remove IDE types from `model.go`**

Delete `ideServerEntry`, `vscodeConfig`, `claudeCodeConfig` structs. These were part of `ConfigureIDEs` which moved to `tinywasm/app`.

Also remove their generated ORM code from `model_orm.go`.

**B. Clean `errors.go`**

Remove unused error vars: `ErrUnsupported`, `ErrToolNotFound`, `ErrNotificationNotInitialized`, `ErrNotificationChannelBlocked`, `NewError`. These are leftovers from the old architecture. Replace stdlib `fmt` import — if nothing remains, delete the file.

**C. Reconcile Logger**

`Server` uses `log func(messages ...any)` internally but `logger.go` defines an unused `Logger` interface. Two options:
- Option A: Delete `logger.go` — `Server.log` already works.
- Option B: Use `Logger` in `Server` and expose `Config.Logger`.

Recommend Option A (delete) — the `Loggable` pattern in `provider.go` already handles per-tool logging.

**D. Remove `FilterFunc`**

Defined in `tools.go:36` but never used. Remove it.

**E. Remove `Loggable` from `provider.go`**

Never used by `Server`. If needed later, re-add. Currently dead code.

**F. Remove `validProtocolVersions`**

Defined in `types.go:21-23` but never referenced. Either use it in `handleInitialize` or remove it.

**G. Delete `docs/stages/`**

Contains 6 stale stage files from prior plan iterations. All stages are completed.

### Steps
- [ ] Delete `ideServerEntry`, `vscodeConfig`, `claudeCodeConfig` from `model.go`
- [ ] Delete corresponding ORM code from `model_orm.go`
- [ ] Delete unused error vars and `NewError` from `errors.go` (or delete file if empty)
- [ ] Delete `logger.go`
- [ ] Remove `FilterFunc` from `tools.go`
- [ ] Remove `Loggable` interface from `provider.go` (file becomes empty — delete)
- [ ] Remove `validProtocolVersions` from `types.go` or use in `handleInitialize`
- [ ] Delete `docs/stages/` directory

---

## Stage 5 — Optional: Protocol Version Validation

### Goal
`handleInitialize` should validate `params.ProtocolVersion` against supported versions.

### Changes

In `server.go` `handleInitialize`:
```go
func (s *Server) handleInitialize(ctx *context.Context, id string, params initializeParams) (*initializeResult, *requestError) {
    if params.ProtocolVersion != LATEST_PROTOCOL_VERSION {
        return nil, &requestError{id: id, code: INVALID_PARAMS, err: fmt.Err("mcp", "unsupported protocol version: "+params.ProtocolVersion)}
    }
    // ... rest
}
```

This makes `validProtocolVersions` unnecessary (single version supported), or use it for multi-version support if needed.

### Tests
```
TestInitialize_UnsupportedVersion_Rejected
TestInitialize_ValidVersion_Passes
```

### Steps
- [ ] Add protocol version validation in `handleInitialize`
- [ ] Add tests for version validation

---

## Execution Order

```
Stage 1 (critical) → Stage 2 (security) → Stage 3 (logic) → Stage 4 (cleanup) → Stage 5 (optional)
```

Stages 1 and 2 are independent and could run in parallel. Stage 4 is independent of 3. Stage 5 is optional.

---

## External Dependencies

None. All changes are internal to `tinywasm/mcp`.
