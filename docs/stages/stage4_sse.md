# Stage 4 — SSE Transport (Streamable HTTP)

### Problem
`tinywasm/sse` must be injectable for Streamable HTTP support. The consumer (`tinywasm/app`) owns the HTTP server and routing — `mcp` only needs to know how to publish responses via SSE when streaming is enabled.

### SSETransport Interface

`*sse.SSEServer` has:
- `Publish(data []byte, channel string)` ✅

`mcp` defines:
```go
// SSEPublisher allows HandleMessage to publish streaming responses.
// Satisfied directly by *sse.SSEServer — no adapter needed.
// Build tag !wasm: SSE only runs server-side.
type SSEPublisher interface {
    Publish(data []byte, channel string)
}
```

### Config
```go
type Config struct {
    Name    string
    Version string
    Auth    Authorizer    // required — nil causes NewServer to return error
    SSE     SSEPublisher  // optional — if nil, consumer handles response delivery
}
```

> **Note:** `Port` and `APIKey` were removed in Stage 1 (extracted to app). `mcp` does not own HTTP routing — `HTTPHandler()` and `RegisterRoutes()` belong to `tinywasm/app`, not here. See "No HTTP ownership" decision in PLAN.md.

### How the consumer uses it

```go
// In tinywasm/app — the consumer owns HTTP routing
mux.Handle("POST /mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // extract token, read body
    resp := mcpServer.HandleMessage(ctx, body)
    // write JSON-RPC response to w
    // if SSE needed, app calls sseServer.Publish(...)
}))
mux.Handle("GET /events", sseServer) // SSE endpoint
```

### Tests

```
TestHandleMessage_WithSSE_PublishesNotification
    → server with SSEPublisher → notifications/tools_list_changed triggers Publish

TestHandleMessage_WithoutSSE_NoPublish
    → server without SSEPublisher → no panic, notifications are no-ops

TestConfig_SSENil_Accepted
    → NewServer with SSE nil → no error (SSE is optional)
```

### Steps
- [ ] Add `SSEPublisher` interface to `server_sse.go` (build `!wasm`)
- [ ] Add `SSE SSEPublisher` to `Config`
- [ ] Use `s.sse.Publish(...)` in notification handlers when SSE is present
- [ ] Add `go.mod` dependency `tinywasm/sse` (verify version compatibility)
- [ ] Verify that `*sse.SSEServer` satisfies `SSEPublisher` without adapter
- [ ] Add all tests listed above
