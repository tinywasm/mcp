package mcp

import (
	"context"
	"net/http"
)

// MCPClient is the interface for MCP client implementations.
type MCPClient interface {
	Initialize(ctx context.Context, request InitializeRequest) (*InitializeResult, error)
	Ping(ctx context.Context) error
	ListToolsByPage(ctx context.Context, request ListToolsRequest) (*ListToolsResult, error)
	ListTools(ctx context.Context, request ListToolsRequest) (*ListToolsResult, error)
	CallTool(ctx context.Context, request CallToolRequest) (*CallToolResult, error)
	Close() error
	OnNotification(handler func(notification JSONRPCNotification))
}

// SSEHub is the interface mcp.Handler uses for SSE transport.
// Implemented by tinywasm/sse.SSEServer.
// Single Publish method — message type is encoded in the JSON payload.
type SSEHub interface {
	http.Handler
	Publish(data []byte, channels ...string)
}
