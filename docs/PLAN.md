# Plan: tinywasm/mcp — HTTP Server Owner + Auth + Dynamic Providers

← Prerequisite for: [app PLAN](../../app/docs/PLAN.md) | [devtui PLAN](../../devtui/docs/PLAN.md)
Requires after: [user PLAN](../../user/docs/PLAN.md)

## References
- [ARCHITECTURE.md](ARCHITECTURE.md)
- `handler.go` — current thin handler (to be expanded)
- `server.go` — MCPServer tool registry
- `request_handler.go` — JSON-RPC message dispatch
- `handler_ide.go` — IDE config writer
- `provider.go` — Tool, ToolProvider, Loggable
- `interface.go` — interfaces

---

## Development Rules
- **SRP:** Every file must have a single, well-defined purpose.
- **Max 500 lines per file.**
- **No global state.** Use DI via interfaces.
- **Standard library only** in test assertions.
- **Test runner:** `gotest`. **Publish:** `gopush`.
- **Language:** Plans in English, chat in Spanish.
- **No code changes** until the user says "ejecuta" or "ok".

---

## Problem Summary

The previous refactor incorrectly moved HTTP server ownership to `tinywasm/app`
(`TinywasmHTTP`), creating a second HTTP server. This broke dynamic tool
registration (ProjectToolProxy never connected to MCPServer) and violated the
principle that `tinywasm/mcp` is the single multi-purpose HTTP+MCP server.

This plan re-establishes `mcp.Handler` as the sole HTTP server owner, adds a
clean `Authorizer` DI interface for RBAC, introduces a reusable `Client` type
for JSON-RPC calls, and introduces fixed vs. dynamic `ToolProvider` slots to
solve the daemon/project lifecycle mismatch.

---

## Architecture

```
HTTP :3030
├── POST /mcp  → auth middleware → custom method router
│                                  ├── tinywasm/action  (OnAction callback)
│                                  ├── tinywasm/state   (OnState callback)
│                                  └── MCPServer        (tools: fixed + dynamic)
└── GET  /logs → SSEHub.ServeHTTP
```

---

## New `handler.go` struct

```go
type Handler struct {
    config        Config
    sseHub        SSEHub                       // injected; nil = no /logs
    auth          Authorizer                   // default: noopAuthorizer (open)
    fixed         []ToolProvider               // permanent tools (daemon-level)
    dynamic       []ToolProvider               // project tools; updated atomically
    toolMeta      map[string]Tool              // name → Tool (for RBAC lookup)
    customMethods map[string]customMethodFunc  // JSON-RPC custom method registry
    mcpServer     *MCPServer                   // rebuilt on SetDynamicProviders
    server        *http.Server
    apiKey        string
    log           func(messages ...any)
    ideStatus     string
    mu            sync.RWMutex
}

// MethodFunc is the handler signature for consumer-defined JSON-RPC methods.
// params: raw JSON bytes of the "params" field. Return any serializable value or error.
type MethodFunc func(ctx context.Context, params []byte) (any, error)

// methodFunc is the internal alias (keeps json.RawMessage internal to handler routing).
type methodFunc func(ctx context.Context, params []byte) (any, error)
```

---

## New Public API

```go
// Constructor — sseHub injected here (was removed in bad refactor)
func NewHandler(config Config, sseHub SSEHub, fixedProviders []ToolProvider) *Handler

// Auth DI — call with user.Module or mcp.NewTokenAuthorizer to enable auth; default denies all
func (h *Handler) SetAuth(auth Authorizer)

// Dynamic providers — replaces project-level tools atomically
// Call with no args to clear (project stopped)
func (h *Handler) SetDynamicProviders(providers ...ToolProvider)

// RegisterMethod registers a consumer-defined JSON-RPC method.
// mcp.Handler is agnostic: it only routes by name, never inspects params or semantics.
// The consumer (e.g. tinywasm/app) defines names, param schemas, and logic entirely.
func (h *Handler) RegisterMethod(name string, fn MethodFunc)

// Lifecycle (restored)
func (h *Handler) Serve(exitChan chan bool) // blocks until exitChan closed
func (h *Handler) Stop()                    // graceful 5s shutdown

// IDE API key injection (written to IDE config headers by ConfigureIDEs)
func (h *Handler) SetAPIKey(key string)

// URL returns the base URL of the running HTTP server (e.g. "http://localhost:3030")
func (h *Handler) URL() string

// Unchanged from current
func (h *Handler) SetLog(f func(message ...any))
func (h *Handler) ConfigureIDEs()
func (h *Handler) Name() string
```

### REMOVED
```go
func (h *Handler) HTTPHandler() http.Handler        // DELETED — handler owns HTTP directly
func (h *Handler) OnAction(fn func(key, value string)) // DELETED — was coupling mcp to app domain
func (h *Handler) OnState(fn func() []byte)            // DELETED — use RegisterMethod instead
```

---

## File: `mcp_auth.go` (NEW)

```go
package mcp

import (
    "context"
    "net/http"
    "strings"
)

// Authorizer is the DI contract for authentication and authorization.
// tinywasm/user.Module implements this structurally — no import of mcp needed
// in user (Go implicit interface satisfaction).
//
// Design: context-based identity — mcp never knows the concrete user type.
// The implementor injects its identity under its own private context key.
type Authorizer interface {
    // InjectIdentity reads the request (Bearer token, cookie, etc.) and
    // injects the authenticated identity into ctx. Returns ctx unchanged
    // if auth fails — CanExecute will deny based on missing identity.
    InjectIdentity(ctx context.Context, r *http.Request) context.Context

    // CanExecute checks if the identity in ctx has permission to perform
    // action ('c','r','u','d') on resource. Returns true if auth passes
    // or if the tool has no Resource constraint (empty string).
    CanExecute(ctx context.Context, resource string, action byte) bool
}

// denyAllAuthorizer is the private default — always denies every request.
//
// Secure-by-default: forgetting to call SetAuth() causes all tool calls to be
// rejected immediately (visible failure), rather than silently leaving the
// endpoint open. The caller MUST explicitly choose an Authorizer.
type denyAllAuthorizer struct{}

func (denyAllAuthorizer) InjectIdentity(ctx context.Context, _ *http.Request) context.Context {
    return ctx
}
func (denyAllAuthorizer) CanExecute(_ context.Context, _ string, _ byte) bool { return false }

// OpenAuthorizer returns an Authorizer that grants full access to every request.
// Use this as an explicit, conscious opt-in for local / trusted environments
// where network-level security is sufficient and auth adds unnecessary friction.
// Calling SetAuth(mcp.OpenAuthorizer()) makes the decision visible in code review.
func OpenAuthorizer() Authorizer { return openAuthorizer{} }

type openAuthorizer struct{}

func (openAuthorizer) InjectIdentity(ctx context.Context, _ *http.Request) context.Context {
    return ctx
}
func (openAuthorizer) CanExecute(_ context.Context, _ string, _ byte) bool { return true }

// tokenKey is the context key for tokenAuthorizer identity.
type tokenKey struct{}

// tokenAuthorizer validates a static Bearer token.
// Use when a lightweight auth is needed without a full user.Module
// (e.g., local daemon securing its MCP endpoint for IDE clients).
type tokenAuthorizer struct{ token string }

// NewTokenAuthorizer returns an Authorizer that accepts only the given static token.
// The token is validated against the "Authorization: Bearer <token>" request header.
// Empty token is always rejected — use OpenAuthorizer() for no-auth mode.
func NewTokenAuthorizer(token string) Authorizer {
    return &tokenAuthorizer{token: token}
}

func (a *tokenAuthorizer) InjectIdentity(ctx context.Context, r *http.Request) context.Context {
    const prefix = "Bearer "
    auth := r.Header.Get("Authorization")
    if strings.HasPrefix(auth, prefix) && auth[len(prefix):] == a.token && a.token != "" {
        return context.WithValue(ctx, tokenKey{}, true)
    }
    return ctx // identity not injected → CanExecute will deny
}

func (a *tokenAuthorizer) CanExecute(ctx context.Context, _ string, _ byte) bool {
    v, _ := ctx.Value(tokenKey{}).(bool)
    return v
}
```

---

## File: `client.go` (NEW)

Lightweight, stateless JSON-RPC 2.0 caller. Shared by `tinywasm/app` and
`tinywasm/devtui` — eliminates duplicated HTTP+JSON-RPC logic in consumers.

NOT a replacement for `StreamableHTTP` (the full MCP session client). Use
`Client` for fire-and-forget action calls and simple queries (`tinywasm/action`,
`tinywasm/state`). Use `StreamableHTTP` for full MCP session protocol
(`initialize`, `tools/list`, `tools/call` with stateful sessions).

**Isomorphic design:** uses `tinywasm/fetch` and `tinywasm/json` so this file
compiles and runs correctly in both stdlib (daemon, devtui) and WASM (browser)
contexts. No `net/http` or `encoding/json` imports.

**Callback-based API:** `tinywasm/fetch` is inherently async (JS `fetch` in WASM,
goroutine in stdlib). Blocking on a channel in WASM is not possible — the API
uses callbacks to stay compatible with both runtimes.

```go
package mcp

import (
    "github.com/tinywasm/fetch"
    "github.com/tinywasm/json"
    . "github.com/tinywasm/fmt"
)

// rpcRequest is the JSON-RPC 2.0 request envelope.
// Struct avoids map allocations (WASM binary size concern).
type rpcRequest struct {
    JSONRPC string `json:"jsonrpc"`
    ID      int    `json:"id"`
    Method  string `json:"method"`
    Params  any    `json:"params"`
}

// Client is a lightweight stateless JSON-RPC 2.0 client for tinywasm/mcp endpoints.
// Thread-safe (no mutable state after construction).
// Compatible with stdlib and WASM (browser) environments.
type Client struct {
    endpoint string // always points to /mcp
    apiKey   string // optional Bearer token; empty = no auth header
}

// NewClient creates a Client targeting baseURL/mcp.
// baseURL: e.g. "http://localhost:3030" — the /mcp path is appended automatically.
// apiKey: Bearer token for secured endpoints; empty = open/local daemon.
func NewClient(baseURL, apiKey string) *Client {
    return &Client{
        endpoint: TrimSuffix(baseURL, "/") + "/mcp",
        apiKey:   apiKey,
    }
}

// Call sends a stateless JSON-RPC 2.0 POST and delivers the raw result bytes via callback.
// callback(nil, nil) when response has no result field.
// Uses tinywasm/fetch (async, WASM+stdlib compatible).
func (c *Client) Call(method string, params any, callback func([]byte, error)) {
    body := c.buildBody(method, params)
    if body == nil {
        if callback != nil {
            callback(nil, Err("mcp: failed to encode request"))
        }
        return
    }
    r := fetch.Post(c.endpoint).ContentTypeJSON().Body(body)
    if c.apiKey != "" {
        r = r.Header("Authorization", "Bearer "+c.apiKey)
    }
    r.Send(func(resp *fetch.Response, err error) {
        if err != nil {
            if callback != nil { callback(nil, err) }
            return
        }
        if callback == nil { return }
        // Decode envelope: {"jsonrpc":"2.0","id":1,"result":<any>}
        var envelope struct{ Result any `json:"result"` }
        if err := json.Decode(resp.Body(), &envelope); err != nil {
            callback(nil, err)
            return
        }
        if envelope.Result == nil {
            callback(nil, nil)
            return
        }
        // Re-encode result field to raw bytes for caller to decode into target type
        var resultBytes []byte
        if err := json.Encode(envelope.Result, &resultBytes); err != nil {
            callback(nil, err)
            return
        }
        callback(resultBytes, nil)
    })
}

// Dispatch sends a JSON-RPC 2.0 POST and ignores the response (fire-and-forget).
// Used for tinywasm/action calls where no return value is needed.
func (c *Client) Dispatch(method string, params any) {
    body := c.buildBody(method, params)
    if body == nil { return }
    r := fetch.Post(c.endpoint).ContentTypeJSON().Body(body)
    if c.apiKey != "" {
        r = r.Header("Authorization", "Bearer "+c.apiKey)
    }
    r.Send(func(*fetch.Response, error) {}) // ignore response
}

func (c *Client) buildBody(method string, params any) []byte {
    var body []byte
    if err := json.Encode(rpcRequest{
        JSONRPC: "2.0",
        ID:      1,
        Method:  method,
        Params:  params,
    }, &body); err != nil {
        return nil
    }
    return body
}
```

---

## File: `provider.go` changes

Add optional RBAC fields to `Tool` (zero-value = no restriction, backward compatible):

```go
type Tool struct {
    Name        string
    Description string
    Parameters  []Parameter
    Execute     func(args map[string]any)

    // RBAC — optional. If Resource is empty, no access control applied.
    // Action: 'c' create, 'r' read, 'u' update, 'd' delete.
    Resource string
    Action   byte
}
```

---

## File: `interface.go` changes

Restore `SSEHub` interface (removed in bad refactor) with a **single publish method**.

**Why single `Publish`:**
After the JSON-RPC migration, `devtui` fetches handler state via `tinywasm/state`
JSON-RPC — not from SSE payload. The daemon no longer needs to stream full state JSON
over SSE. All it needs is a lightweight signal: *"state changed, re-fetch."*

That signal can travel through the same `Publish` channel using a reserved `type`
value in the JSON payload. `devtui` already parses every SSE message as JSON
(`tabContentDTO`) — adding a `TypeStateRefresh` constant costs nothing and eliminates
the need for a second publish method entirely.

**Signature fix required in `tinywasm/sse`:** `SSEServer.Publish` currently takes
`channel string` (single) while `sse.SSEPublisher` declares `channels ...string`
(variadic). Fix `SSEServer.Publish` to variadic before publishing mcp.

```go
// SSEHub is the interface mcp.Handler uses for SSE transport.
// Implemented by tinywasm/sse.SSEServer.
// Single Publish method — message type is encoded in the JSON payload.
type SSEHub interface {
    http.Handler
    Publish(data []byte, channels ...string)
}
```

`app.SSEPublisher` sends both log entries and state-refresh signals through this
single method (see `app/sse_publisher.go` changes below).

---

## Custom Method Routing in `handler.go`

`RegisterMethod` stores consumer-defined handlers. The HTTP handler peeks the
method name and routes to registered handlers before delegating to MCPServer.
`mcp.Handler` never inspects params or knows what any method does.

```go
// RegisterMethod is the public API — consumer defines name + logic entirely.
func (h *Handler) RegisterMethod(name string, fn MethodFunc) {
    h.mu.Lock()
    h.customMethods[name] = fn
    h.mu.Unlock()
}
```

Custom methods bypass RBAC — they are consumer-level hooks, not MCP tools.
Only MCP tools (via ToolProvider) go through `CanExecute`.

`mcpHTTPHandler` — the HTTP handler for POST /mcp:

```go
func (h *Handler) mcpHTTPHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        // 1. Auth: inject identity into ctx
        ctx := h.auth.InjectIdentity(r.Context(), r)

        // 2. Read + buffer body (needed for method peek + MCPServer delegation)
        body, err := io.ReadAll(r.Body)
        if err != nil { http.Error(w, "bad request", 400); return }

        // 3. Peek method name
        var peek struct{ Method string `json:"method"` }
        json.Unmarshal(body, &peek)

        // 4. Custom method?
        h.mu.RLock()
        fn, isCustom := h.customMethods[peek.Method]
        h.mu.RUnlock()
        if isCustom {
            var req struct {
                ID     any    `json:"id"`
                Params []byte `json:"params"` // raw bytes passed as-is to MethodFunc
            }
            json.Unmarshal(body, &req)
            result, err := fn(ctx, req.Params)
            writeJSONRPCResponse(w, req.ID, result, err)
            return
        }

        // 5. Delegate to MCPServer (tool calls, initialize, ping, tools/list)
        r2 := r.WithContext(ctx)
        r2.Body = io.NopCloser(bytes.NewReader(body))
        h.mu.RLock()
        srv := h.mcpServer
        h.mu.RUnlock()
        NewStreamableHTTPServer(srv, WithEndpointPath("/mcp")).ServeHTTP(w, r2)
    })
}
```

---

## Tool Auth Middleware (registered in `rebuildMCPServer`)

```go
func (h *Handler) toolAuthMiddleware() ToolHandlerMiddleware {
    return func(next ToolHandlerFunc) ToolHandlerFunc {
        return func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
            h.mu.RLock()
            meta, ok := h.toolMeta[req.Params.Name]
            h.mu.RUnlock()
            if ok && meta.Resource != "" {
                if !h.auth.CanExecute(ctx, meta.Resource, meta.Action) {
                    return NewToolResultError("access denied: insufficient permissions"), nil
                }
            }
            return next(ctx, req)
        }
    }
}
```

---

## Dynamic Provider Rebuild

```go
func (h *Handler) SetDynamicProviders(providers ...ToolProvider) {
    h.mu.Lock()
    h.dynamic = providers
    h.mu.Unlock()
    h.rebuildMCPServer()
}

func (h *Handler) rebuildMCPServer() {
    s := NewMCPServer(h.config.ServerName, h.config.ServerVersion,
        WithToolCapabilities(true),
        WithToolHandlerMiddleware(h.toolAuthMiddleware()),
    )
    newMeta := make(map[string]Tool)
    h.mu.RLock()
    all := append(append([]ToolProvider{}, h.fixed...), h.dynamic...)
    h.mu.RUnlock()

    for _, p := range all {
        if p == nil { continue }
        for _, tool := range p.GetMCPTools() {
            s.AddToolFromUser(tool, toolExecutorAdapter(tool.Execute))
            newMeta[tool.Name] = tool
        }
    }
    h.mu.Lock()
    h.mcpServer = s
    h.toolMeta = newMeta
    h.mu.Unlock()
}
```

---

## IDE Config: API Key injection (`handler_ide.go`)

Add `SupportsHeaders bool` to `IDEInfo`:

```go
type IDEInfo struct {
    // ... existing fields unchanged ...
    SupportsHeaders bool // true = inject Authorization header
}
```

Updated IDE entries:
```go
{ID: "vsc",         SupportsHeaders: true, ...}  // confirmed
{ID: "antigravity", SupportsHeaders: true, ...}  // confirmed
{ID: "claude-code", SupportsHeaders: true, ...}  // confirmed
```

`writeMCPConfig` injects header if conditions met:
```go
// After building serverEntry:
if h.apiKey != "" && ide.SupportsHeaders {
    serverEntry["headers"] = map[string]string{
        "Authorization": "Bearer " + h.apiKey,
    }
}
```

`SetAPIKey` must be called before `ConfigureIDEs()`.

---

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `mcp_auth.go` | **CREATE** | `Authorizer` interface + `noopAuthorizer` + `tokenAuthorizer` + `NewTokenAuthorizer` |
| `client.go` | **CREATE** | `Client` struct + `NewClient(baseURL, apiKey)` + `Call(method, params)` |
| `handler.go` | **MODIFY** | Restore `Serve`/`Stop`/`SSEHub`; add `SetAuth`, `SetDynamicProviders`, `SetAPIKey`, `OnAction`, `OnState`, `URL`, `rebuildMCPServer`, `mcpHTTPHandler`, `toolAuthMiddleware` |
| `provider.go` | **MODIFY** | Add `Resource string`, `Action byte` to `Tool` |
| `interface.go` | **MODIFY** | Restore `SSEHub` (add `PublishEvent`) |
| `handler_ide.go` | **MODIFY** | Add `SupportsHeaders bool`; set all 3 IDEs to `true`; inject headers in `writeMCPConfig` |
| `handler_executor.go` | NO CHANGE | Already correct |
| `server.go` | NO CHANGE | MCPServer stays as pure tool registry |
| `request_handler.go` | NO CHANGE | Unknown methods return `METHOD_NOT_FOUND` (custom methods intercepted at Handler level) |

---

## Execution Steps

### Step 1 — Fix `tinywasm/sse` (prerequisite)
`SSEServer.Publish(data []byte, channel string)` → `Publish(data []byte, channels ...string)`
to match `SSEPublisher` interface. Publish `tinywasm/sse` before proceeding.

### Step 2 — Create `mcp_auth.go`
`Authorizer` interface + `denyAllAuthorizer` + `openAuthorizer` + `OpenAuthorizer()` +
`tokenAuthorizer` + `NewTokenAuthorizer`.

### Step 3 — Create `client.go`
`Client` struct + `NewClient` + `Call`.

### Step 4 — Modify `interface.go`
Restore `SSEHub` with corrected variadic `Publish` + `PublishEvent`.

### Step 5 — Modify `provider.go`
Add `Resource string` + `Action byte` to `Tool` (zero-value, backward compatible).

### Step 6 — Modify `handler.go`
- New struct fields: `sseHub`, `auth`, `fixed`, `dynamic`, `toolMeta`, `customMethods`, `mcpServer`, `server`, `apiKey`
- New signature: `NewHandler(config Config, sseHub SSEHub, fixedProviders []ToolProvider)`
- Add `URL() string` returning base URL (protocol + host + port, no path)
- Remove `HTTPHandler()` entirely
- Add: `SetAuth`, `SetDynamicProviders`, `SetAPIKey`, `OnAction`, `OnState`, `Serve`, `Stop`, `rebuildMCPServer`, `mcpHTTPHandler`, `toolAuthMiddleware`, `registerMethod`
- Call `rebuildMCPServer()` at end of `NewHandler`

### Step 7 — Modify `handler_ide.go`
`SupportsHeaders` on all 3 IDEs + header injection.

### Step 8 — Run tests and publish
```bash
gotest
gopush 'feat: mcp owns HTTP server, Authorizer DI, tokenAuthorizer, reusable Client, dynamic providers, RBAC, API key IDE config'
```

---

## Test Strategy

| Test | Validates |
|------|-----------|
| `TestHandler_Serve_StartsHTTP` | `Serve` brings up server on configured port |
| `TestHandler_OnAction_JSONRPCMethod` | `tinywasm/action` called → fn receives key/value |
| `TestHandler_OnState_JSONRPCMethod` | `tinywasm/state` returns fn() JSON |
| `TestHandler_SetDynamicProviders_ToolsVisible` | Tools from dynamic providers in `tools/list` |
| `TestHandler_SetDynamicProviders_PreservesFixed` | Fixed tools survive `SetDynamicProviders` |
| `TestHandler_SetDynamicProviders_ClearsDynamic` | `SetDynamicProviders()` (no args) clears dynamic tools |
| `TestHandler_ToolRBAC_Denied` | `Resource` set + `CanExecute=false` → `access denied` |
| `TestHandler_ToolRBAC_Allowed` | `Resource` set + `CanExecute=true` → tool executes |
| `TestHandler_DenyAllDefault_BlocksAll` | Default (no SetAuth) → all tool calls rejected |
| `TestHandler_OpenAuthorizer_AllowsAll` | `SetAuth(mcp.OpenAuthorizer())` → all tools pass |
| `TestHandler_SetAPIKey_ConfigureIDEs_WritesHeaders` | API key written to IDE config for all 3 IDEs |
| `TestTool_ZeroValueResource_NoRBAC` | Empty `Resource` skips RBAC entirely |
| `TestTokenAuthorizer_ValidToken_Injects` | Correct Bearer token → CanExecute true |
| `TestTokenAuthorizer_WrongToken_Denies` | Wrong Bearer token → CanExecute false |
| `TestTokenAuthorizer_EmptyToken_Denies` | Empty token config → always denies |
| `TestClient_Call_SendsCorrectBody` | Body: `jsonrpc:"2.0"`, method, params |
| `TestClient_Call_WithAPIKey_SetsAuthHeader` | `Authorization: Bearer <key>` present |
| `TestClient_Call_NoAPIKey_NoAuthHeader` | No auth header when apiKey empty |
| `TestHandler_URL_ReturnsBaseURL` | `URL()` returns scheme+host+port without path |
| Existing `tests/server_test.go` etc. | Must pass without changes |
