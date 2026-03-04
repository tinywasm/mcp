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
}

type ClientOption func(*Client)

// WithClientCapabilities sets the client capabilities for the client.
func WithClientCapabilities(capabilities ClientCapabilities) ClientOption {
	return func(c *Client) {
		c.clientCapabilities = capabilities
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
	capabilities := request.Params.Capabilities

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

// handleIncomingRequest processes incoming requests from the server.
// This is the main entry point for server-to-client requests like sampling and elicitation.
func (c *Client) handleIncomingRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	switch request.Method {
	case string(MethodPing):
		return c.handlePingRequestTransport(ctx, request)
	default:
		return nil, fmt.Errorf("unsupported request method: %s", request.Method)
	}
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
