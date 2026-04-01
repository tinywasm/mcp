# Stage 2 — Startup Security

### Problem
`NewServer` accepts `Config.Auth == nil` silently. Any request passes without authentication.
Additionally, `Config` still contains `Port` and `APIKey` which belong to `tinywasm/app`, not to the protocol layer.

### Changes

#### A. `NewServer` returns error if Auth is nil

```go
func NewServer(config Config, providers []ToolProvider) (*Server, error) {
    if config.Auth == nil {
        return nil, fmt.Err("mcp", "Auth is required — use mcp.OpenAuthorizer() for open access")
    }
    // ...
    return s, nil
}
```

#### B. Remove `Port` and `APIKey` from Config and Server

These fields belong to `tinywasm/app` (HTTP layer), not to the protocol core.

```go
// BEFORE
type Config struct {
    Name    string
    Version string
    Auth    Authorizer
    Port    string   // REMOVE
    APIKey  string   // REMOVE
}

// AFTER
type Config struct {
    Name    string
    Version string
    Auth    Authorizer
}
```

Also remove from `Server` struct: `apiKey`, `port`, `ideStatus` fields.

#### C. Delete handler_ide.go and handler_ide_wasm.go

These files implement `ConfigureIDEs` which depends on `Port` and `APIKey`. This functionality was already decided to move to `tinywasm/app` (see Stage 1 cleanup notes). Delete these files entirely.

### Impact on Consumers
- `tinywasm/app` must handle `(*Server, error)` return from `NewServer`
- `tinywasm/app` must provide `Port` and `APIKey` in its own config, not in `mcp.Config`
- All existing consumers must add `Auth` to their `Config`

### Security Tests

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

TestHandleMessage_ValidToken_Passes
    → correct token → executes method

TestHandleMessage_OpenAuthorizer_NoHeader_Passes
    → OpenAuthorizer without header → passes (explicit opt-in)
```

### Steps
- [ ] Change `NewServer` signature to `NewServer(config, providers) (*Server, error)`
- [ ] Remove `Port`, `APIKey` from `Config` struct
- [ ] Remove `port`, `apiKey`, `ideStatus` from `Server` struct
- [ ] Delete handler_ide.go and handler_ide_wasm.go if they exist (ConfigureIDEs moves to app)
- [ ] Add `NewTokenAuthorizer(apiKey string) Authorizer` to mcp_auth.go
- [ ] Add `OpenAuthorizer() Authorizer` to mcp_auth.go
- [ ] Add all security tests listed above
- [ ] Verify existing tests compile (update test helpers to pass Auth)

> **Note:** No `SetAuth` post-construction. Auth is immutable after `NewServer` — allowing mutation would break the "Auth required at startup" invariant.
