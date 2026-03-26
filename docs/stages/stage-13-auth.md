# Stage 13 — auth.go (Authorizer interface)

## 13.1 — Define Authorizer interface only

`tinywasm/mcp` must NOT implement authentication — that belongs to `tinywasm/user`.
Define only the interface that callers (tinywasm/user) implement:

```go
// auth.go — no build tag (WASM-safe, interface only)
package mcp

// Authorizer validates a bearer token and returns the user ID.
// Implement this interface in tinywasm/user (AuthModeBearer).
type Authorizer interface {
    Authorize(token string) (userID string, err error)
}
```

## 13.2 — Wire into Handler

```go
type Config struct {
    // ... existing fields ...
    Auth Authorizer // optional; nil = no auth required
}
```

In `request_handler.go`, extract token from context and call `Auth.Authorize`:

```go
if h.config.Auth != nil {
    token := ctx.Value(CtxKeyAuthToken)
    userID, err := h.config.Auth.Authorize(token)
    if err != nil {
        return sendUnauthorized(ctx)
    }
    ctx.Set(CtxKeyUserID, userID)
}
```

## 13.3 — tinywasm/user compatibility

`tinywasm/user.AuthModeBearer` must satisfy `mcp.Authorizer`.
No code changes needed in `tinywasm/mcp` — the interface is the contract.

If `tinywasm/user` does not yet expose a method matching:
```go
Authorize(token string) (userID string, err error)
```
create a plan in `tinywasm/user` to add a thin adapter.

## 13.4 — Remove any existing auth implementation

Delete any bearer-token parsing, JWT decoding, or secret-key storage
that currently lives in `tinywasm/mcp`. MCP only calls the interface.
