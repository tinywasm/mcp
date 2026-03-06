package mcp

import "context"

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
