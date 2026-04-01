# Stage 2 — Startup Security

### Problem
`NewServer` accepts `Config.Auth == nil` silently. Any request passes without authentication.

### Change
```go
// NewServer returns error if config.Auth is nil — security must be explicit.
// For local/trusted environments use mcp.OpenAuthorizer() as an explicit opt-in.
func NewServer(config Config, providers []ToolProvider) (*Server, error) {
    if config.Auth == nil {
        return nil, fmt.Err("mcp", "Auth is required — use mcp.OpenAuthorizer() for open access")
    }
    // ...
}
```

### Impact on Consumers
- `tinywasm/app` uses `mcpHandler.SetAuth(...)` post-construction — must migrate to `Config.Auth`
- Every existing consumer must add `Auth` to their `Config`

### Security Tests — Critical Edge Cases

```
TestNewServer_NilAuth_ReturnsError
    → NewServer with Auth nil should return error, not server

TestNewServer_OpenAuthorizer_Starts
    → NewServer with OpenAuthorizer() should start without error

TestHandleMessage_NoAuthHeader_Rejected
    → request without Authorization header with TokenAuthorizer → error -32001

TestHandleMessage_InvalidToken_Rejected
    → incorrect token → error -32001

TestHandleMessage_EmptyToken_Rejected
    → Authorization: Bearer  (empty) → error -32001

TestHandleMessage_WrongScheme_Rejected
    → Authorization: Basic xxx (not Bearer) → error -32001

TestHandleMessage_TokenWithSpaces_Rejected
    → token with spaces or control characters → error -32001

TestHandleMessage_ValidToken_Passes
    → correct token → executes method

TestHandleMessage_OpenAuthorizer_NoHeader_Passes
    → OpenAuthorizer without header → passes (explicit opt-in)
```

### Steps
- [ ] Change `NewServer` to `NewServer(config, providers) (*Server, error)`
- [ ] Add security tests listed above
- [ ] Add `NewTokenAuthorizer(apiKey string) Authorizer` to `mcp_auth.go`
- [ ] Add `OpenAuthorizer() Authorizer` to `mcp_auth.go`

> **Note:** `SetAuth` post-construction was removed from the plan. It contradicts the "Auth required at startup" guarantee — if you can `SetAuth(nil)` after construction, the invariant is broken. Auth is immutable after `NewServer`.
