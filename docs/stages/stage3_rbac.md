# Stage 3 — Mandatory RBAC (Unified Authorizer)

### Problem
`Tool.Resource` and `Tool.Action` are mandatory in the definition but are never verified in `handleToolCall`. The RBAC is declared but is declarative only.

### Design: A Single `Authorizer`, Two Responsibilities

**Reason for unification:** "Gatekeeper" as a separate name doesn't communicate much. `Authorizer` already implies "who can do what". Merging auth + RBAC into one interface eliminates ambiguity and forces the implementer to think about both dimensions.

```go
// Authorizer handles both authentication (who are you?) and authorization (what can you do?).
// Both methods are always called on every tool execution — no bypass.
type Authorizer interface {
    // Authorize validates the bearer token and returns the userID.
    // Return error to reject the request before any tool logic runs.
    Authorize(token string) (userID string, err error)

    // Can checks whether userID has permission to perform action on resource.
    // Called after Authorize succeeds, before tool.Execute runs.
    // Return false to reject with Unauthorized — the tool never runs.
    Can(userID, resource string, action byte) bool
}
```

### Implementations Included in mcp

```go
// NewTokenAuthorizer returns an Authorizer that accepts a single static API key.
// Can() always returns true — all authenticated callers can use all tools.
// Use for local dev environments or single-tenant setups.
func NewTokenAuthorizer(apiKey string) Authorizer

// OpenAuthorizer returns an Authorizer that accepts any token (including empty).
// Can() always returns true.
// ONLY for local/trusted environments — explicit opt-in, never a default.
func OpenAuthorizer() Authorizer
```

### Corrected `handleToolCall`

```go
func (s *Server) handleToolCall(ctx *context.Context, id string, params callToolParams) (*Result, *requestError) {
    tool, ok := s.tools[params.Name]
    if !ok {
        return nil, &requestError{id: id, code: INVALID_PARAMS, err: fmt.Err("mcp", "tool not found")}
    }

    userID := ctx.Value(CtxKeyUserID)  // set by HandleMessage after Authorize()

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

### tinywasm/user implements Authorizer

`tinywasm/user.Module` will implement `mcp.Authorizer` once the pending functions are completed:
```go
// In tinywasm/user — adapter that satisfies mcp.Authorizer
func (m *Module) MCPAuthorizer() mcp.Authorizer
```

Internally:
- `Authorize(token)`: `ValidateJWT(secret, token)` → userID (implementation pending in user)
- `Can(userID, resource, action)`: `m.HasPermission(userID, resource, action)` ✅ already exists

### Security Tests — Critical Edge Cases

```
TestHandleToolCall_Can_False_Rejected
    → valid user, incorrect resource → error -32001, Execute is never called

TestHandleToolCall_Can_True_Executes
    → valid user, correct resource and action → Execute is called

TestHandleToolCall_WrongAction_Rejected
    → user has 'r' on "catalog", tries 'u' → error -32001

TestHandleToolCall_WrongResource_Rejected
    → user has access to "catalog", tries "billing" → error -32001

TestHandleToolCall_CanNeverCalledIfAuthorizeFails
    → Authorize fails → Can is never invoked (verify with mock)

TestHandleToolCall_ExecuteNeverCalledIfCanFalse
    → Can returns false → Execute is never invoked (verify with mock)

TestAddTool_MissingResource_Rejected
    → Tool without Resource → AddTool returns error (already exists, keep it)

TestAddTool_MissingAction_Rejected
    → Tool without Action → AddTool returns error (already exists, keep it)

TestConcurrentToolCalls_DifferentUsers
    → two goroutines, users with different permissions, no race condition
```

### Steps
- [ ] Extend `Authorizer` with `Can(userID, resource string, action byte) bool`
- [ ] Update `NewTokenAuthorizer` — `Can` always returns `true`
- [ ] Update `OpenAuthorizer` — `Can` always returns `true`
- [ ] Update `handleToolCall` — call `s.auth.Can()` before `Execute`
- [ ] Add all tests listed above
- [ ] In `tinywasm/user`: add `MCPAuthorizer()` when `ValidateJWT` is implemented
