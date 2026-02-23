# Post-Reset Restoration Plan (tinywasm/mcp)

This document provides precise instructions and original code snippets to restore the consolidated `mcp` library following the repository reset. 

**Reference File**: Use [ORIGINAL_SERVER_REFERENCE.go.txt](ORIGINAL_SERVER_REFERENCE.go.txt) for the full source of truth regarding original logic, helper functions, and server-side utilities.

## 1. Core JSON-RPC Types (types.go)

We have flattened the `JSONRPCRequest` and `JSONRPCNotification` structures. **Jules MUST ensure these definitions are exactly as follows**, as many test failures stem from structural mismatches.

```go
// types.go

// JSONRPCRequest represents a request that expects a response.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      RequestId   `json:"id"`
	Method  string      `json:"method"` // Flattened
	Params  any         `json:"params,omitempty"`
	Header  http.Header `json:"-"`
}

// JSONRPCNotification represents a notification which does not expect a response.
type JSONRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"` // Flattened
	Params  any    `json:"params,omitempty"`
}

// Add this method to JSONRPCErrorDetails to fix client.go:173
func (e *JSONRPCErrorDetails) AsError() error {
	return fmt.Errorf("JSON-RPC error: %d: %s", e.Code, e.Message)
}
```

## 2. Restore Streamable HTTP Server (streamable_http.go)

The reset deleted the server-side implementation which `handler.go` depends on. Restore the logic for `NewStreamableHTTPServer` and its options.

```go
// streamable_http_server.go (or similar name in mcp root)

type StreamableHTTPServer struct {
	server *MCPServer
	path   string
}

func NewStreamableHTTPServer(s *MCPServer, opts ...StreamableHTTPServerOption) *StreamableHTTPServer {
	srv := &StreamableHTTPServer{
		server: s,
		path:   "/mcp",
	}
	for _, opt := range opts {
		opt(srv)
	}
	return srv
}

func (s *StreamableHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Implement the SSE/POST logic here
}

// Options
type StreamableHTTPServerOption func(*StreamableHTTPServer)

func WithEndpointPath(path string) StreamableHTTPServerOption {
	return func(s *StreamableHTTPServer) { s.path = path }
}

### Implementation Steps:
1. Copy the relevant `ServeHTTP` and SSE logic from **ORIGINAL_SERVER_REFERENCE.go.txt** to `streamable_http.go`.
2. Adapt the types to use the flattened `mcp` namespace (e.g., `mcp.JSONRPCRequest` instead of `mcp.Request`).
3. Ensure `NewStreamableHTTPServer` is exported and matches the signature expected by `handler.go`.
```

## 3. Fix Client Implementation (client.go)

Fix the recursive `Start` method and restore missing interface satisfyers.

```go
// client.go

func (c *Client) Start(ctx context.Context) error {
	if c.transport == nil {
		return fmt.Errorf("transport is nil")
	}
	// CRITICAL: Call transport.Start, NOT c.Start
	return c.transport.Start(ctx)
}

func (c *Client) SendRequest(ctx context.Context, method string, params any) (*JSONRPCResponse, error) {
	id := NewRequestId(c.requestID.Add(1))
	request := JSONRPCRequest{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Method:  method, // Clean flattened usage
		Params:  params,
	}
	return c.transport.SendRequest(ctx, request)
}
```

## 4. Massive Test Update

All files in `tests/` and `session.go` must be updated to handle the flattened structures.

### A. Fix Struct Literals
**Problem**: `unknown field Request in struct literal`
- **Before**: `mcp.JSONRPCRequest{ Request: mcp.Request{ Method: "foo" }, ... }`
- **After**: `mcp.JSONRPCRequest{ Method: "foo", ... }`

### B. Fix Notification Params access
**Problem**: `notification.Params.AdditionalFields undefined`
- **Context**: `Params` is now of type `any`. If you previously accessed `notification.Params.AdditionalFields["key"]`, you must now use a type assertion.
- **Before**: `params := notification.Params.AdditionalFields`
- **After**: `params := notification.Params.(map[string]any)`

### C. Search & Replace conceptual guide:
- Find: `Request: mcp.Request{` -> Remove and promote inner fields.
- Find: `Notification: mcp.Notification{` -> Remove and promote inner fields.
- Find: `.Params.AdditionalFields` -> Change to `.(map[string]any)` or check if `.Params` can be used directly as a map.

## 5. Flattening Utility Functions (utils.go & tools.go)

Many helper functions still use the old nested structure. They must be updated to use the flattened `Method` and `Params` fields directly.

```go
// utils.go

func NewProgressNotification(...) ProgressNotification {
	return ProgressNotification{
		JSONRPC: JSONRPC_VERSION,
		Method:  "notifications/progress", // Flattened
		Params: struct { ... }{ ... },
	}
}

func NewLoggingMessageNotification(...) LoggingMessageNotification {
	return LoggingMessageNotification{
		JSONRPC: JSONRPC_VERSION,
		Method:  "notifications/message", // Flattened
		Params:  ...
	}
}
```

## 6. Implementing Server-to-Client Requests (session.go / server.go)

The `MCPServer` needs implementation for methods that trigger client capabilities.

```go
// server.go or session.go

func (s *MCPServer) RequestSampling(ctx context.Context, request CreateMessageRequest) (*CreateMessageResult, error) {
	session := ClientSessionFromContext(ctx)
	if s, ok := session.(SessionWithSampling); ok {
		return s.RequestSampling(ctx, request)
	}
	return nil, fmt.Errorf("session does not support sampling")
}

func (s *MCPServer) ListRoots(ctx context.Context, request ListRootsRequest) (*ListRootsResult, error) {
	session := ClientSessionFromContext(ctx)
	if s, ok := session.(SessionWithRoots); ok {
		return s.ListRoots(ctx, request)
	}
	return nil, fmt.Errorf("session does not support roots")
}
```

## 7. Advanced Transport Sessions (transport_sse.go & transport_stdio.go)

The concrete session types (e.g., `sseSession`, `stdioSession`) must implement the sending logic for these requests, including **Request ID tracking** and **Response channels**, as seen in `ORIGINAL_SERVER_REFERENCE.go.txt`.

### Key implementation detail for transport sessions:
- Use a `sync.Map` or map with mutex to track `pendingRequests`.
- When `RequestSampling` is called, generate a new ID, create a channel, and wait.
- When the client sends a response, find the channel by ID and deliver the result.

## 8. Summary of Indefinitions

The following symbols are currently undefined and MUST be restored/linked:
- `NewStreamableHTTPServer` (in `handler.go`)
- `WithEndpointPath` (in `handler.go`)
- `UnsupportedProtocolVersionError` (Add to `errors.go`)

```go
// errors.go
type UnsupportedProtocolVersionError struct {
	Version string
}
func (e *UnsupportedProtocolVersionError) Error() string {
	return fmt.Sprintf("unsupported protocol version: %s", e.Version)
}
```
