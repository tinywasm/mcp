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

	// ListResourcesByPage manually list resources by page.
	ListResourcesByPage(
		ctx context.Context,
		request ListResourcesRequest,
	) (*ListResourcesResult, error)

	// ListResources requests a list of available resources from the server
	ListResources(
		ctx context.Context,
		request ListResourcesRequest,
	) (*ListResourcesResult, error)

	// ListResourceTemplatesByPage manually list resource templates by page.
	ListResourceTemplatesByPage(
		ctx context.Context,
		request ListResourceTemplatesRequest,
	) (*ListResourceTemplatesResult,
		error)

	// ListResourceTemplates requests a list of available resource templates from the server
	ListResourceTemplates(
		ctx context.Context,
		request ListResourceTemplatesRequest,
	) (*ListResourceTemplatesResult,
		error)

	// ReadResource reads a specific resource from the server
	ReadResource(
		ctx context.Context,
		request ReadResourceRequest,
	) (*ReadResourceResult, error)

	// Subscribe requests notifications for changes to a specific resource
	Subscribe(ctx context.Context, request SubscribeRequest) error

	// Unsubscribe cancels notifications for a specific resource
	Unsubscribe(ctx context.Context, request UnsubscribeRequest) error

	// ListPromptsByPage manually list prompts by page.
	ListPromptsByPage(
		ctx context.Context,
		request ListPromptsRequest,
	) (*ListPromptsResult, error)

	// ListPrompts requests a list of available prompts from the server
	ListPrompts(
		ctx context.Context,
		request ListPromptsRequest,
	) (*ListPromptsResult, error)

	// GetPrompt retrieves a specific prompt from the server
	GetPrompt(
		ctx context.Context,
		request GetPromptRequest,
	) (*GetPromptResult, error)

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

	// SetLevel sets the logging level for the server
	SetLevel(ctx context.Context, request SetLevelRequest) error

	// Complete requests completion options for a given argument
	Complete(
		ctx context.Context,
		request CompleteRequest,
	) (*CompleteResult, error)

	// Close client connection and cleanup resources
	Close() error

	// OnNotification registers a handler for notifications
	OnNotification(handler func(notification JSONRPCNotification))
}
