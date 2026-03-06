# MCP Library Testing Plan - Stage 2 (Deep Coverage)

This plan expands the test suite to cover complex components and edge cases that were missed in Stage 1, aiming to significantly increase code coverage.

## Prerequisites

- `gotest` must be installed.
- Stage 1 tests must pass.

## Development Rules

- **SRP & DI:** Strict adherence to dependency injection for testing HTTP and SSE.
- **Framework-less:** Use only `net/http/httptest`.
- **Dual Testing:** Ensure WASM-compatible code is tested.

## Proposed Changes

### [Component] Utilities & Helpers
Exhaustive testing of polymorphic helpers.

#### [NEW] [utils_test.go](tests/utils_test.go)
- Test `AsTextContent`, `AsImageContent`, `AsAudioContent` with valid and invalid inputs.
- Test `NewToolResultJSON` with complex nested structs.
- Test `GetTextFromContent` with all supported types (string, TextContent, map).
- Test `ExtractString` with various map structures.

### [Component] HTTP Transport
Verify the real HTTP and session logic.

#### [NEW] [transport_http_test.go](tests/transport_http_test.go)
- Run `httptest.NewServer` to mock an MCP server.
- Test `StreamableHTTP.SendRequest` and session ID extraction from headers.
- Test `StreamableHTTP.SendNotification`.
- Verify error handling for non-200 responses.

### [Component] Integrated Handler
Test the high-level TUI/IDE bridge.

#### [NEW] [handler_test.go](tests/handler_test.go)
- Test `Handler.Serve` (using a dynamic port and stopping it).
- Test `/state` and `/version` endpoints.
- Test `/action` endpoint with custom callbacks.
- Test `PublishLog` and its interaction with a mock `SSEHub`.

### [Component] Server & Client Edge Cases
Push for 100% logic branch coverage.

#### [MODIFY] [server_test.go](tests/server_test.go)
- Add tests for `listByPagination` edge cases (empty list, invalid cursor, end of list).
- Add tests for `handleToolCall` error mapping (method not found, internal error).

#### [MODIFY] [client_test.go](tests/client_test.go)
- Test `handleIncomingRequest` (server-to-client Pings).
- Test notification dispatching to multiple subscribers.

## Verification Plan

### Automated Tests
```bash
gotest
```

### Manual Verification
- Targets: > 70% total coverage.
- Inspect `gotest` output for all packages.
