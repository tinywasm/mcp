# Restoring MCP Library - Implementation Plan

## Goal Description
The `tinywasm/mcp` library is missing several critical fixes that were detailed in [FIXED_PROMP.md](FIXED_PROMP.md) but ignored in recent commits. We need to implement these missing changes to restore full functionality and eliminate compilation workarounds.

## Proposed Changes

### `mcp` Library Core (types & tools)
#### [MODIFY] types.go
- Flatten `JSONRPCRequest` and `JSONRPCNotification` by removing the embedded `Request`/`Notification` structs and promoting `Method`.
- Add `AsError() error` to `JSONRPCErrorDetails`.

#### [MODIFY] utils.go & tools.go
- Update helper functions like `NewProgressNotification` and `NewLoggingMessageNotification` to use the flattened notification structure.

#### [MODIFY] session.go & tests
- Fix struct literals across `session.go` and all files in `tests/` to remove the `Request: mcp.Request{}` wrapping and directly assign `Method`.
- Fix `.Params.AdditionalFields` accesses to use map type assertions `.(map[string]any)` since `Params` is now `any`.
- Move `e2e/sampling_http_test.go` to the `tests/` directory.

### Transports & Server
#### [NEW/MODIFY] streamable_http.go
- Implement `StreamableHTTPServer` and `NewStreamableHTTPServer`.
- Implement `ServeHTTP` logic to handle SSE/POST requests as expected by [handler.go](handler.go).

#### [MODIFY] handler.go
- Uncomment `NewStreamableHTTPServer` and its router registration `mux.Handle("/mcp", mcpServer)`.

#### [MODIFY] server.go / session.go
- Implement `RequestSampling` and `ListRoots` methods on `MCPServer` that delegate to the context's session.

#### [MODIFY] transport_sse.go & transport_stdio.go
- Implement bidirectional request tracking (`pendingRequests` map) to handle Server-to-Client requests like sampling.

## Verification Plan
### Automated Tests
- Run `gotest` in `tinywasm/mcp` directory directly. This will automatically execute all unit and integration tests.
- Verify that `tests/session_resource_templates_test.go`, `tests/prompt_mixed_content_test.go`, and other tests pass without structural errors.
