package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
)

// Client implements the MCP client.
type Client struct {
	transport Interface

	initialized        bool
	notifications      []func(JSONRPCNotification)
	notifyMu           sync.RWMutex
	requestID          atomic.Int64
	clientCapabilities ClientCapabilities
	serverCapabilities ServerCapabilities
	protocolVersion    string
	samplingHandler    SamplingHandler
	rootsHandler       RootsHandler
	elicitationHandler ElicitationHandler
}

type ClientOption func(*Client)

// WithClientCapabilities sets the client capabilities for the client.
func WithClientCapabilities(capabilities ClientCapabilities) ClientOption {
	return func(c *Client) {
		c.clientCapabilities = capabilities
	}
}

// WithSamplingHandler sets the sampling handler for the client.
// When set, the client will declare sampling capability during initialization.
func WithSamplingHandler(handler SamplingHandler) ClientOption {
	return func(c *Client) {
		c.samplingHandler = handler
	}
}

// WithRootsHandler sets the roots handler for the client.
// WithRootsHandler returns a ClientOption that sets the client's RootsHandler.
// When provided, the client will declare the roots capability (ListChanged) during initialization.
func WithRootsHandler(handler RootsHandler) ClientOption {
	return func(c *Client) {
		c.rootsHandler = handler
	}
}

// WithElicitationHandler sets the elicitation handler for the client.
// When set, the client will declare elicitation capability during initialization.
func WithElicitationHandler(handler ElicitationHandler) ClientOption {
	return func(c *Client) {
		c.elicitationHandler = handler
	}
}

// WithSession assumes a MCP Session has already been initialized
func WithInitializedSession() ClientOption {
	return func(c *Client) {
		c.initialized = true
	}
}

// NewClient creates a new MCP client with the given transport.
// Usage:
//
//	stdio := NewStdio("./mcp_server", nil, "xxx")
//	client, err := NewClient(stdio)
//	if err != nil {
//	    log.Fatalf("Failed to create client: %v", err)
//	}
func NewClient(transport Interface, options ...ClientOption) *Client {
	client := &Client{
		transport: transport,
	}

	for _, opt := range options {
		opt(client)
	}

	return client
}

// Start initiates the connection to the server.
// Must be called before using the client.
func (c *Client) Start(ctx context.Context) error {
	if c.transport == nil {
		return fmt.Errorf("transport is nil")
	}

	// Start is idempotent - transports handle being called multiple times
	err := c.transport.Start(ctx)
	if err != nil {
		return err
	}

	c.transport.SetNotificationHandler(func(notification JSONRPCNotification) {
		c.notifyMu.RLock()
		defer c.notifyMu.RUnlock()
		for _, handler := range c.notifications {
			handler(notification)
		}
	})

	// Set up request handler for bidirectional communication (e.g., sampling)
	if bidirectional, ok := c.transport.(BidirectionalInterface); ok {
		bidirectional.SetRequestHandler(c.handleIncomingRequest)
	}

	return nil
}

// Close shuts down the client and closes the transport.
func (c *Client) Close() error {
	return c.transport.Close()
}

// OnNotification registers a handler function to be called when notifications are received.
// Multiple handlers can be registered and will be called in the order they were added.
func (c *Client) OnNotification(
	handler func(notification JSONRPCNotification),
) {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	c.notifications = append(c.notifications, handler)
}

// OnConnectionLost registers a handler function to be called when the connection is lost.
// This is useful for handling HTTP2 idle timeout disconnections that should not be treated as errors.
func (c *Client) OnConnectionLost(handler func(error)) {
	type connectionLostSetter interface {
		SetConnectionLostHandler(func(error))
	}
	if setter, ok := c.transport.(connectionLostSetter); ok {
		setter.SetConnectionLostHandler(handler)
	}
}

// sendRequest sends a JSON-RPC request to the server and waits for a response.
// Returns the raw JSON response message or an error if the request fails.
func (c *Client) sendRequest(
	ctx context.Context,
	method string,
	params any,
	header http.Header,
) (*json.RawMessage, error) {
	if !c.initialized && method != "initialize" {
		return nil, fmt.Errorf("client not initialized")
	}

	id := c.requestID.Add(1)

	request := JSONRPCRequest{
		JSONRPC: JSONRPC_VERSION,
		ID:      NewRequestId(id),
		Params:  params,
		Header:  header,
		Request: Request{
			Method: method,
		},
	}

	response, err := c.transport.SendRequest(ctx, request)
	if err != nil {
		return nil, NewError(err)
	}

	if response.Error != nil {
		return nil, &jsonRPCError{
			code:    response.Error.Code,
			message: response.Error.Message,
			data:    response.Error.Data,
		}
	}

	// Marshal the result back to JSON to return RawMessage
	bytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	raw := json.RawMessage(bytes)
	return &raw, nil
}

// Initialize negotiates with the server.
// Must be called after Start, and before any request methods.
func (c *Client) Initialize(
	ctx context.Context,
	request InitializeRequest,
) (*InitializeResult, error) {
	// Merge client capabilities with sampling capability if handler is configured
	capabilities := request.Params.Capabilities
	if c.samplingHandler != nil {
		capabilities.Sampling = &struct{}{}
	}
	if c.rootsHandler != nil {
		capabilities.Roots = &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{
			ListChanged: true,
		}
	}
	// Add elicitation capability if handler is configured
	if c.elicitationHandler != nil {
		capabilities.Elicitation = &ElicitationCapability{}
	}

	// Ensure we send a params object with all required fields
	params := struct {
		ProtocolVersion string                 `json:"protocolVersion"`
		ClientInfo      Implementation     `json:"clientInfo"`
		Capabilities    ClientCapabilities `json:"capabilities"`
	}{
		ProtocolVersion: request.Params.ProtocolVersion,
		ClientInfo:      request.Params.ClientInfo,
		Capabilities:    capabilities,
	}

	// By default, use client supported latest protocol version if version not specified
	if params.ProtocolVersion == "" {
		params.ProtocolVersion = LATEST_PROTOCOL_VERSION
	}

	response, err := c.sendRequest(ctx, "initialize", params, request.Header)
	if err != nil {
		return nil, err
	}

	var result InitializeResult
	if err := json.Unmarshal(*response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Validate protocol version
	if !slices.Contains(ValidProtocolVersions, result.ProtocolVersion) {
		return nil, UnsupportedProtocolVersionError{Version: result.ProtocolVersion}
	}

	// Store serverCapabilities and protocol version
	c.serverCapabilities = result.Capabilities
	c.protocolVersion = result.ProtocolVersion

	// Set protocol version on HTTP transports
	if httpConn, ok := c.transport.(HTTPConnection); ok {
		httpConn.SetProtocolVersion(result.ProtocolVersion)
	}

	// Send initialized notification
	notification := JSONRPCNotification{
		JSONRPC: JSONRPC_VERSION,
		Notification: Notification{
			Method: "notifications/initialized",
		},
	}

	err = c.transport.SendNotification(ctx, notification)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to send initialized notification: %w",
			err,
		)
	}

	c.initialized = true
	return &result, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.sendRequest(ctx, "ping", nil, nil)
	return err
}

// ListResourcesByPage manually list resources by page.
func (c *Client) ListResourcesByPage(
	ctx context.Context,
	request ListResourcesRequest,
) (*ListResourcesResult, error) {
	result, err := listByPage[ListResourcesResult](ctx, c, request.PaginatedRequest, request.Header, "resources/list")
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListResources(
	ctx context.Context,
	request ListResourcesRequest,
) (*ListResourcesResult, error) {
	result, err := c.ListResourcesByPage(ctx, request)
	if err != nil {
		return nil, err
	}
	for result.NextCursor != "" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			request.Params.Cursor = result.NextCursor
			newPageRes, err := c.ListResourcesByPage(ctx, request)
			if err != nil {
				return nil, err
			}
			result.Resources = append(result.Resources, newPageRes.Resources...)
			result.NextCursor = newPageRes.NextCursor
		}
	}
	return result, nil
}

func (c *Client) ListResourceTemplatesByPage(
	ctx context.Context,
	request ListResourceTemplatesRequest,
) (*ListResourceTemplatesResult, error) {
	result, err := listByPage[ListResourceTemplatesResult](ctx, c, request.PaginatedRequest, request.Header, "resources/templates/list")
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListResourceTemplates(
	ctx context.Context,
	request ListResourceTemplatesRequest,
) (*ListResourceTemplatesResult, error) {
	result, err := c.ListResourceTemplatesByPage(ctx, request)
	if err != nil {
		return nil, err
	}
	for result.NextCursor != "" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			request.Params.Cursor = result.NextCursor
			newPageRes, err := c.ListResourceTemplatesByPage(ctx, request)
			if err != nil {
				return nil, err
			}
			result.ResourceTemplates = append(result.ResourceTemplates, newPageRes.ResourceTemplates...)
			result.NextCursor = newPageRes.NextCursor
		}
	}
	return result, nil
}

func (c *Client) ReadResource(
	ctx context.Context,
	request ReadResourceRequest,
) (*ReadResourceResult, error) {
	response, err := c.sendRequest(ctx, "resources/read", request.Params, request.Header)
	if err != nil {
		return nil, err
	}

	return ParseReadResourceResult(response)
}

func (c *Client) Subscribe(
	ctx context.Context,
	request SubscribeRequest,
) error {
	_, err := c.sendRequest(ctx, "resources/subscribe", request.Params, request.Header)
	return err
}

func (c *Client) Unsubscribe(
	ctx context.Context,
	request UnsubscribeRequest,
) error {
	_, err := c.sendRequest(ctx, "resources/unsubscribe", request.Params, request.Header)
	return err
}

func (c *Client) ListPromptsByPage(
	ctx context.Context,
	request ListPromptsRequest,
) (*ListPromptsResult, error) {
	result, err := listByPage[ListPromptsResult](ctx, c, request.PaginatedRequest, request.Header, "prompts/list")
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListPrompts(
	ctx context.Context,
	request ListPromptsRequest,
) (*ListPromptsResult, error) {
	result, err := c.ListPromptsByPage(ctx, request)
	if err != nil {
		return nil, err
	}
	for result.NextCursor != "" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			request.Params.Cursor = result.NextCursor
			newPageRes, err := c.ListPromptsByPage(ctx, request)
			if err != nil {
				return nil, err
			}
			result.Prompts = append(result.Prompts, newPageRes.Prompts...)
			result.NextCursor = newPageRes.NextCursor
		}
	}
	return result, nil
}

func (c *Client) GetPrompt(
	ctx context.Context,
	request GetPromptRequest,
) (*GetPromptResult, error) {
	response, err := c.sendRequest(ctx, "prompts/get", request.Params, request.Header)
	if err != nil {
		return nil, err
	}

	return ParseGetPromptResult(response)
}

func (c *Client) ListToolsByPage(
	ctx context.Context,
	request ListToolsRequest,
) (*ListToolsResult, error) {
	result, err := listByPage[ListToolsResult](ctx, c, request.PaginatedRequest, request.Header, "tools/list")
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) ListTools(
	ctx context.Context,
	request ListToolsRequest,
) (*ListToolsResult, error) {
	result, err := c.ListToolsByPage(ctx, request)
	if err != nil {
		return nil, err
	}
	for result.NextCursor != "" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			request.Params.Cursor = result.NextCursor
			newPageRes, err := c.ListToolsByPage(ctx, request)
			if err != nil {
				return nil, err
			}
			result.Tools = append(result.Tools, newPageRes.Tools...)
			result.NextCursor = newPageRes.NextCursor
		}
	}
	return result, nil
}

func (c *Client) CallTool(
	ctx context.Context,
	request CallToolRequest,
) (*CallToolResult, error) {
	response, err := c.sendRequest(ctx, "tools/call", request.Params, request.Header)
	if err != nil {
		return nil, err
	}

	return ParseCallToolResult(response)
}

func (c *Client) SetLevel(
	ctx context.Context,
	request SetLevelRequest,
) error {
	_, err := c.sendRequest(ctx, "logging/setLevel", request.Params, request.Header)
	return err
}

func (c *Client) Complete(
	ctx context.Context,
	request CompleteRequest,
) (*CompleteResult, error) {
	response, err := c.sendRequest(ctx, "completion/complete", request.Params, request.Header)
	if err != nil {
		return nil, err
	}

	var result CompleteResult
	if err := json.Unmarshal(*response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// RootListChanges sends a roots list-changed notification to the server.
func (c *Client) RootListChanges(
	ctx context.Context,
) error {
	// Send root list changes notification
	notification := JSONRPCNotification{
		JSONRPC: JSONRPC_VERSION,
		Notification: Notification{
			Method: MethodNotificationRootsListChanged,
		},
	}

	err := c.transport.SendNotification(ctx, notification)
	if err != nil {
		return fmt.Errorf(
			"failed to send root list change notification: %w",
			err,
		)
	}
	return nil
}

// handleIncomingRequest processes incoming requests from the server.
// This is the main entry point for server-to-client requests like sampling and elicitation.
func (c *Client) handleIncomingRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	switch request.Method {
	case string(MethodSamplingCreateMessage):
		return c.handleSamplingRequestTransport(ctx, request)
	case string(MethodElicitationCreate):
		return c.handleElicitationRequestTransport(ctx, request)
	case string(MethodPing):
		return c.handlePingRequestTransport(ctx, request)
	case string(MethodListRoots):
		return c.handleListRootsRequestTransport(ctx, request)
	default:
		return nil, fmt.Errorf("unsupported request method: %s", request.Method)
	}
}

// handleSamplingRequestTransport handles sampling requests at the transport level.
func (c *Client) handleSamplingRequestTransport(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	if c.samplingHandler == nil {
		return nil, fmt.Errorf("no sampling handler configured")
	}

	// Parse the request parameters
	var params CreateMessageParams
	if request.Params != nil {
		paramsBytes, err := json.Marshal(request.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		if err := json.Unmarshal(paramsBytes, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
	}

	// Fix content parsing - HTTP transport unmarshals TextContent as map[string]any
	// Use the helper function to properly handle content from different transports
	for i := range params.Messages {
		if contentMap, ok := params.Messages[i].Content.(map[string]any); ok {
			// Parse the content map into a proper Content type
			content, err := ParseContent(contentMap)
			if err != nil {
				return nil, fmt.Errorf("failed to parse content for message %d: %w", i, err)
			}
			params.Messages[i].Content = content
		}
	}

	// Create the MCP request
	mcpRequest := CreateMessageRequest{
		Request: Request{
			Method: string(MethodSamplingCreateMessage),
		},
		CreateMessageParams: params,
	}

	// Call the sampling handler
	result, err := c.samplingHandler.CreateMessage(ctx, mcpRequest)
	if err != nil {
		return nil, err
	}

	// Marshal the result
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Create the transport response
	response := NewJSONRPCResultResponse(request.ID, json.RawMessage(resultBytes))

	return &response, nil
}

// handleListRootsRequestTransport handles list roots requests at the transport level.
func (c *Client) handleListRootsRequestTransport(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	if c.rootsHandler == nil {
		return nil, fmt.Errorf("no roots handler configured")
	}

	// Create the MCP request
	mcpRequest := ListRootsRequest{
		Request: Request{
			Method: string(MethodListRoots),
		},
	}

	// Call the list roots handler
	result, err := c.rootsHandler.ListRoots(ctx, mcpRequest)
	if err != nil {
		return nil, err
	}

	// Marshal the result
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Create the transport response
	response := NewJSONRPCResultResponse(request.ID, json.RawMessage(resultBytes))

	return &response, nil
}

// handleElicitationRequestTransport handles elicitation requests at the transport level.
func (c *Client) handleElicitationRequestTransport(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	if c.elicitationHandler == nil {
		return nil, fmt.Errorf("no elicitation handler configured")
	}

	// Parse the request parameters
	var params ElicitationParams
	if request.Params != nil {
		paramsBytes, err := json.Marshal(request.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		if err := json.Unmarshal(paramsBytes, &params); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params: %w", err)
		}
	}

	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid elicitation params: %w", err)
	}

	// Create the MCP request
	mcpRequest := ElicitationRequest{
		Request: Request{
			Method: string(MethodElicitationCreate),
		},
		Params: params,
	}

	// Call the elicitation handler
	result, err := c.elicitationHandler.Elicit(ctx, mcpRequest)
	if err != nil {
		return nil, err
	}

	// Marshal the result
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	// Create the transport response
	response := NewJSONRPCResultResponse(request.ID, resultBytes)

	return &response, nil
}

func (c *Client) handlePingRequestTransport(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	b, _ := json.Marshal(&EmptyResult{})
	response := NewJSONRPCResultResponse(request.ID, b)
	return &response, nil
}

func listByPage[T any](
	ctx context.Context,
	client *Client,
	request PaginatedRequest,
	header http.Header,
	method string,
) (*T, error) {
	response, err := client.sendRequest(ctx, method, request.Params, header)
	if err != nil {
		return nil, err
	}
	var result T
	if err := json.Unmarshal(*response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

// Helper methods

// GetTransport gives access to the underlying transport layer.
// Cast it to the specific transport type and obtain the other helper methods.
func (c *Client) GetTransport() Interface {
	return c.transport
}

// GetServerCapabilities returns the server capabilities.
func (c *Client) GetServerCapabilities() ServerCapabilities {
	return c.serverCapabilities
}

// GetClientCapabilities returns the client capabilities.
func (c *Client) GetClientCapabilities() ClientCapabilities {
	return c.clientCapabilities
}

// GetSessionId returns the session ID of the client.
// If the transport does not support sessions, it returns an empty string.
func (c *Client) GetSessionId() string {
	if c.transport == nil {
		return ""
	}
	return c.transport.GetSessionId()
}

// IsInitialized returns true if the client has been initialized.
func (c *Client) IsInitialized() bool {
	return c.initialized
}

// UnsupportedProtocolVersionError is returned when the server suggests a protocol version that is not supported by the client.
type UnsupportedProtocolVersionError struct {
	Version string
}

func (e UnsupportedProtocolVersionError) Error() string {
	return fmt.Sprintf("unsupported protocol version: %s", e.Version)
}

type jsonRPCError struct {
	code    int
	message string
	data    any
}

func (e *jsonRPCError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.code, e.message)
}
