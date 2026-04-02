# PLAN: tinywasm/mcp — Production Test Coverage

> Date: 2026-04-02
> Status: Ready for execution

---

## Context

The library has 17 tests covering basic happy paths and some error cases. For production readiness, critical paths lack coverage: TokenAuthorizer end-to-end, RBAC granularity, Execute error handling, protocol version validation, Bind action propagation, SSE on AddTool, and the Client package has zero tests.

## File Structure

Each test case lives in its own file, grouped by stage. All files share a common setup file with mocks and helpers.

```
tests/
├── setup_test.go           ← shared mocks, helpers, contains()
├── mcp_test.go             ← existing tests (keep as-is)
├── auth_test.go            ← Stage 1: auth + constantTimeEqual
├── rbac_test.go            ← Stage 2: RBAC + tool execution
├── protocol_test.go        ← Stage 3: protocol compliance
├── sse_test.go             ← Stage 4: SSE coverage
└── client_test.go          ← Stage 5: client
```

### setup_test.go — Shared Setup

Extracted from current `mcp_test.go` and extended:

```go
package mcp_test

import (
    "github.com/tinywasm/context"
    "github.com/tinywasm/fmt"
    "github.com/tinywasm/json"
    "github.com/tinywasm/mcp"
)

// --- Mocks ---

// mockAuth — simple auth that always succeeds with given id
type mockAuth struct {
    id string
}

func (m *mockAuth) Authorize(token string) (string, error) {
    return m.id, nil
}

func (m *mockAuth) Can(userID, resource string, action byte) bool {
    if userID == "forbidden-user" {
        return false
    }
    return true
}

// rbacAuth — records Can() args and allows control per resource/action
type rbacAuth struct {
    id             string
    lastResource   string
    lastAction     byte
    denyResource   string
    denyAction     byte
}

func (m *rbacAuth) Authorize(token string) (string, error) {
    return m.id, nil
}

func (m *rbacAuth) Can(userID, resource string, action byte) bool {
    m.lastResource = resource
    m.lastAction = action
    if resource == m.denyResource && action == m.denyAction {
        return false
    }
    return true
}

// emptyUserAuth — returns empty userID (tests empty rejection)
type emptyUserAuth struct{}

func (m *emptyUserAuth) Authorize(token string) (string, error) {
    return "", nil
}

func (m *emptyUserAuth) Can(userID, resource string, action byte) bool {
    return true
}

// mockSSE — records Publish calls
type mockSSE struct {
    lastData    []byte
    lastChannel string
    callCount   int
}

func (m *mockSSE) Publish(data []byte, channel string) {
    m.lastData = data
    m.lastChannel = channel
    m.callCount++
}

// --- Helpers ---

func contains(s, substr string) bool {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return true
        }
    }
    return false
}

func encodeResponse(resp mcp.JSONRPCMessage) string {
    var b []byte
    if f, ok := resp.(fmt.Fielder); ok {
        json.Encode(f, &b)
    }
    return string(b)
}

func newEchoTool() mcp.Tool {
    return mcp.Tool{
        Name:     "echo",
        Resource: "test",
        Action:   'r',
        Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
            return mcp.Text("ok"), nil
        },
    }
}
```

### Migration from mcp_test.go

Move `mockAuth`, `mockSSE`, `contains()` from `mcp_test.go` to `setup_test.go`. Remove duplicates from `mcp_test.go` — existing tests keep working since all files share `package mcp_test`.

## Execution Order

```
Stage 0 (setup) → Stage 1 (auth) → Stage 2 (RBAC+tools) → Stage 3 (protocol) → Stage 4 (SSE) → Stage 5 (client)
```

Stage 0 is a prerequisite. Stages 1-5 are independent and can run in parallel after Stage 0.

---

## Stage 0 — Setup Extraction

### Goal
Create `setup_test.go` with shared mocks and helpers. Clean `mcp_test.go` to remove duplicates.

### Steps
- [ ] Create `tests/setup_test.go` with mocks: `mockAuth`, `rbacAuth`, `emptyUserAuth`, `mockSSE`
- [ ] Add helpers: `contains()`, `encodeResponse()`, `newEchoTool()`
- [ ] Remove `mockAuth`, `mockSSE`, `contains()` from `mcp_test.go`
- [ ] Verify existing tests still pass

---

## Stage 1 — Auth Coverage

### File: `tests/auth_test.go`

### Goal
Cover TokenAuthorizer end-to-end, empty userID rejection, and constantTimeEqual edge cases.

### Tests

```
TestConstantTimeEqual_SameStrings
TestConstantTimeEqual_DifferentStrings
TestConstantTimeEqual_DifferentLengths
TestConstantTimeEqual_EmptyStrings
TestTokenAuthorizer_ValidToken_Passes
TestTokenAuthorizer_InvalidToken_Rejected
TestTokenAuthorizer_EmptyToken_Rejected
TestHandleMessage_TokenAuth_ValidToken_Returns_Result
TestHandleMessage_TokenAuth_InvalidToken_Returns_Unauthorized
TestHandleMessage_TokenAuth_NoToken_Returns_Unauthorized
TestHandleMessage_EmptyUserID_Rejected
TestOpenAuthorizer_ReturnsGuest
```

### Why
- `TokenAuthorizer` is the primary auth mechanism for production. Currently only tested via compile-time interface check, never exercised through `HandleMessage`.
- `constantTimeEqual` is a security-critical function with zero tests. Edge cases (empty strings, different lengths) must be verified.
- Empty userID rejection was implemented but never validated — a regression here silently breaks auth.

### Note
`constantTimeEqual` is unexported. Tests must exercise it indirectly via `NewTokenAuthorizer().Authorize()`. If direct testing is needed, consider exporting or adding an internal test file (`auth_internal_test.go` with `package mcp`).

### Steps
- [ ] Create `tests/auth_test.go`
- [ ] Add `TestTokenAuthorizer_*` (3 cases) — exercise via `mcp.NewTokenAuthorizer(key).Authorize(token)`
- [ ] Add `TestHandleMessage_TokenAuth_*` (3 cases) — set `CtxKeyAuthToken` in ctx, use `NewTokenAuthorizer`, verify response
- [ ] Add `TestHandleMessage_EmptyUserID_Rejected` — use `emptyUserAuth`, verify -32001
- [ ] Add `TestOpenAuthorizer_ReturnsGuest` — verify userID is "guest"
- [ ] Evaluate if `constantTimeEqual` needs internal test file for direct coverage

---

## Stage 2 — RBAC + Tool Execution

### File: `tests/rbac_test.go`

### Goal
Cover RBAC granularity (resource + action checked), Execute error path, tool not found, Bind action propagation, and concurrent access.

### Tests

```
TestHandleToolCall_Can_ChecksResource
TestHandleToolCall_Can_ChecksAction
TestHandleToolCall_ExecuteNeverCalledIfCanFalse
TestHandleToolCall_ToolNotFound
TestHandleToolCall_ExecuteReturnsError_IsErrorTrue
TestHandleToolCall_Bind_UsesToolAction
TestHandleToolCall_DuplicateToolName_Overwrites
TestConcurrent_AddToolAndCallTool
TestNewServer_ProviderInjectsTools
TestNewServer_ProviderWithInvalidTool_ReturnsError
```

### Why
- Current `mockAuth.Can()` only checks `userID == "forbidden-user"`, never inspects `resource` or `action`. A bug in how these values are passed to `Can()` would go undetected.
- When `tool.Execute` returns an error, the server wraps it as `IsError: true`. This path has zero coverage — a regression could leak stack traces or return wrong format.
- `Bind` was recently fixed to use `r.Action` instead of hardcoded `'c'`. No test validates this.
- `Server` uses `sync.RWMutex` for concurrent tool access. Never stress-tested.

### Steps
- [ ] Create `tests/rbac_test.go`
- [ ] Add `TestHandleToolCall_Can_ChecksResource` — use `rbacAuth`, register tool with resource "secrets", verify `rbacAuth.lastResource == "secrets"`
- [ ] Add `TestHandleToolCall_Can_ChecksAction` — register tool with action 'u', verify `rbacAuth.lastAction == 'u'`
- [ ] Add `TestHandleToolCall_ExecuteNeverCalledIfCanFalse` — Execute sets `called = true`, use `rbacAuth` with deny, verify `called == false`
- [ ] Add `TestHandleToolCall_ToolNotFound` — call nonexistent tool, verify INVALID_PARAMS error
- [ ] Add `TestHandleToolCall_ExecuteReturnsError_IsErrorTrue` — Execute returns `fmt.Err(...)`, verify response contains `is_error`
- [ ] Add `TestHandleToolCall_Bind_UsesToolAction` — tool with action 'u', Execute calls `req.Bind()`, verify Validate receives 'u'
- [ ] Add `TestHandleToolCall_DuplicateToolName_Overwrites` — AddTool "x" twice with different Execute, call, verify second handler runs
- [ ] Add `TestConcurrent_AddToolAndCallTool` — 10 goroutines adding tools + 10 goroutines calling HandleMessage, verify no race/panic
- [ ] Add `TestNewServer_ProviderInjectsTools` — provider with 2 tools, verify both callable via HandleMessage
- [ ] Add `TestNewServer_ProviderWithInvalidTool_ReturnsError` — provider returns tool with empty Name, verify NewServer returns error

---

## Stage 3 — Protocol Compliance

### File: `tests/protocol_test.go`

### Goal
Cover protocol version validation, initialize response content, and edge cases in JSON parsing.

### Tests

```
TestInitialize_UnsupportedVersion_Rejected
TestInitialize_ValidVersion_ReturnsServerInfo
TestInitialize_GeneratesSessionID
TestInitialize_ExistingSessionID_Preserved
TestHandleMessage_NoID_NoMethod_ParseError
TestHandleMessage_NullBytes
TestExtractJSONString_MissingKey
TestExtractJSONString_NonStringValue
```

### Why
- Protocol version validation was added but has zero test coverage.
- `handleInitialize` generates a session ID via `unixid` but no test verifies it's actually set in context.
- Edge cases in `extractJSONString` (the internal parser) could cause panics on malformed input — production servers receive arbitrary bytes.

### Steps
- [ ] Create `tests/protocol_test.go`
- [ ] Add `TestInitialize_UnsupportedVersion_Rejected` — send params with version "1.0", verify INVALID_PARAMS error
- [ ] Add `TestInitialize_ValidVersion_ReturnsServerInfo` — verify response JSON contains server name and version
- [ ] Add `TestInitialize_GeneratesSessionID` — verify `ctx.Value(CtxKeySessionID) != ""` after initialize
- [ ] Add `TestInitialize_ExistingSessionID_Preserved` — pre-set session ID "abc", verify still "abc" after initialize
- [ ] Add `TestHandleMessage_NullBytes` — send `[]byte{0x00}`, verify error response, no panic
- [ ] Add `TestExtractJSONString_MissingKey` — via HandleMessage with missing fields
- [ ] Add `TestExtractJSONString_NonStringValue` — via HandleMessage with numeric id

---

## Stage 4 — SSE Coverage

### File: `tests/sse_test.go`

### Goal
Cover SSE notification on AddTool, SSE nil safety, and SSE not called for non-notification methods.

### Tests

```
TestAddTool_WithSSE_PublishesListChanged
TestAddTool_WithoutSSE_NoPanic
TestHandleMessage_WithoutSSE_NoPublish
TestSSE_PublishData_ContainsMethod
```

### Why
- `AddTool` publishes `notifications/tools/list_changed` via SSE but this is never tested.
- SSE nil check (`if s.SSE != nil`) is correct but never verified — a nil pointer dereference in production would crash the server.

### Steps
- [ ] Create `tests/sse_test.go`
- [ ] Add `TestAddTool_WithSSE_PublishesListChanged` — use `mockSSE`, add tool, verify `mockSSE.callCount == 1` and data contains "list_changed"
- [ ] Add `TestAddTool_WithoutSSE_NoPanic` — SSE nil, add tool, verify no panic and no error
- [ ] Add `TestHandleMessage_WithoutSSE_NoPublish` — send notification without SSE configured, verify no panic
- [ ] Add `TestSSE_PublishData_ContainsMethod` — verify published data is parseable and contains correct method string

---

## Stage 5 — Client

### File: `tests/client_test.go`

### Goal
Basic coverage for Client.Call, Client.Dispatch, and buildBody.

### Note
Client depends on `tinywasm/fetch` which may require a mock HTTP server or be untestable in unit tests. If `fetch` cannot be mocked, document this as an integration test requirement and skip for now.

### Tests (if testable)

```
TestClient_BuildBody_ValidParams
TestClient_BuildBody_NilParams
TestClient_BuildBody_InvalidParams
TestClient_NewClient_NormalizesURL
```

### Why
- Client has zero tests. `buildBody` is pure logic and testable regardless of fetch.
- URL normalization (TrimSuffix + "/mcp") could break with edge cases like trailing slashes or empty URLs.

### Steps
- [ ] Create `tests/client_test.go`
- [ ] Evaluate if `buildBody` can be tested by exporting or via `Client.Call` with mock
- [ ] If testable: add `TestClient_BuildBody_*` tests (3 cases)
- [ ] Add `TestClient_NewClient_NormalizesURL` — verify endpoint for various input URLs
- [ ] If not testable: document as `// TODO: requires integration test with HTTP mock`

---

## Summary

| Stage | File | Tests | Priority |
|-------|------|-------|----------|
| 0 — Setup | `setup_test.go` | — | Prerequisite |
| 1 — Auth | `auth_test.go` | 12 | Critical |
| 2 — RBAC + Tools | `rbac_test.go` | 10 | Critical |
| 3 — Protocol | `protocol_test.go` | 8 | High |
| 4 — SSE | `sse_test.go` | 4 | Medium |
| 5 — Client | `client_test.go` | 4 | Low |
| **Total** | **6 files** | **38** | |
