// Package client provides MCP (Model Context Protocol) client implementations.
package mcp

import (
	"context"
)

// MCPClient represents an MCP client interface
type MCPClient interface {
	// Initialize sends the initial connection request to the server
	Initialize(
		ctx context.Context,
		request InitializeRequest,
	) (*InitializeResult, error)

	// Ping checks if the server is alive
	Ping(ctx context.Context) error

	// ListToolsByPage manually list tools by page.
	ListToolsByPage(
		ctx context.Context,
		request ListToolsRequest,
	) (*ListToolsResult, error)

	// ListTools requests a list of available tools from the server
	ListTools(
		ctx context.Context,
		request ListToolsRequest,
	) (*ListToolsResult, error)

	// CallTool invokes a specific tool on the server
	CallTool(
		ctx context.Context,
		request CallToolRequest,
	) (*CallToolResult, error)

	// Close client connection and cleanup resources
	Close() error

	// OnNotification registers a handler for notifications
	OnNotification(handler func(notification JSONRPCNotification))
}
