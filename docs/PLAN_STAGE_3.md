# Stage 3 — Core File Rewrite

← Back to [PLAN.md](PLAN.md)

> **Strategy:** Do NOT use shell scripts or sed to edit existing files. For each file below, either
> write the **complete replacement content** or follow the **exact deletion list** provided.
> Execute files in the order listed. Run `go build ./...` after each section to verify.

---

## Prerequisites

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

---

## Section A — Complete File Replacements

Write each file below with exactly the content shown. Overwrite the existing file entirely.

---

### A1. `errors.go`

```go
package mcp

import "errors"

var (
	ErrUnsupported                = errors.New("not supported")
	ErrToolNotFound               = errors.New("tool not found")
	ErrNotificationNotInitialized = errors.New("notification channel not initialized")
	ErrNotificationChannelBlocked = errors.New("notification channel queue is full - client may not be processing notifications fast enough")
)
```

---

### A2. `session.go`

Session lifecycle (create, store, destroy) is managed externally by the consumer
(`tinywasm/user`). This file is the **protocol contract only** — the interface the transport
layer and `HandleMessage` use to send notifications to the current connection.

```go
package mcp

import "context"

// ClientSession is the protocol contract for an active client connection.
// It is implemented by the consumer (e.g. tinywasm/user) and injected via context.
type ClientSession interface {
	// SessionID is a unique identifier for this connection.
	SessionID() string
	// NotificationChannel is the channel the server writes outbound notifications to.
	NotificationChannel() chan<- JSONRPCNotification
	// Initialized reports whether the MCP handshake is complete.
	Initialized() bool
	// Initialize marks the session as ready to receive notifications.
	Initialize()
}

// SessionWithStreamableHTTPConfig is an optional extension for streamable HTTP transport.
type SessionWithStreamableHTTPConfig interface {
	ClientSession
	UpgradeToSSEWhenReceiveNotification()
}

type clientSessionKey struct{}

// ContextWithSession stores a ClientSession in a context.
// Called by the consumer before invoking HandleMessage.
func ContextWithSession(ctx context.Context, session ClientSession) context.Context {
	return context.WithValue(ctx, clientSessionKey{}, session)
}

// SessionFromContext retrieves the ClientSession from a context.
func SessionFromContext(ctx context.Context) ClientSession {
	session, _ := ctx.Value(clientSessionKey{}).(ClientSession)
	return session
}

// SendNotification sends a notification to the session currently in ctx.
// Returns ErrNotificationNotInitialized if no initialized session is present.
func SendNotification(ctx context.Context, method string, params map[string]any) error {
	session := SessionFromContext(ctx)
	if session == nil || !session.Initialized() {
		return ErrNotificationNotInitialized
	}
	if sh, ok := session.(SessionWithStreamableHTTPConfig); ok {
		sh.UpgradeToSSEWhenReceiveNotification()
	}
	select {
	case session.NotificationChannel() <- JSONRPCNotification{
		JSONRPC: JSONRPC_VERSION,
		Notification: Notification{
			Method: method,
			Params: NotificationParams{AdditionalFields: params},
		},
	}:
		return nil
	default:
		return ErrNotificationChannelBlocked
	}
}
```

---

### A3. `interface.go`

```go
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
```

---

### A4. `client.go`

```go
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

// WithInitializedSession assumes a MCP Session has already been initialized.
func WithInitializedSession() ClientOption {
	return func(c *Client) {
		c.initialized = true
	}
}

// NewClient creates a new MCP client with the given transport.
func NewClient(transport Interface, options ...ClientOption) *Client {
	client := &Client{transport: transport}
	for _, opt := range options {
		opt(client)
	}
	return client
}

// Start initiates the connection to the server.
func (c *Client) Start(ctx context.Context) error {
	if c.transport == nil {
		return fmt.Errorf("transport is nil")
	}
	if err := c.transport.Start(ctx); err != nil {
		return err
	}
	c.transport.SetNotificationHandler(func(notification JSONRPCNotification) {
		c.notifyMu.RLock()
		defer c.notifyMu.RUnlock()
		for _, handler := range c.notifications {
			handler(notification)
		}
	})
	if bidirectional, ok := c.transport.(BidirectionalInterface); ok {
		bidirectional.SetRequestHandler(c.handleIncomingRequest)
	}
	return nil
}

// Close shuts down the client and closes the transport.
func (c *Client) Close() error {
	return c.transport.Close()
}

// OnNotification registers a handler for notifications.
func (c *Client) OnNotification(handler func(notification JSONRPCNotification)) {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	c.notifications = append(c.notifications, handler)
}

// OnConnectionLost registers a handler for when the connection is lost.
func (c *Client) OnConnectionLost(handler func(error)) {
	type connectionLostSetter interface {
		SetConnectionLostHandler(func(error))
	}
	if setter, ok := c.transport.(connectionLostSetter); ok {
		setter.SetConnectionLostHandler(handler)
	}
}

func (c *Client) sendRequest(ctx context.Context, method string, params any, header http.Header) (*json.RawMessage, error) {
	if !c.initialized && method != "initialize" {
		return nil, fmt.Errorf("client not initialized")
	}
	id := c.requestID.Add(1)
	request := JSONRPCRequest{
		JSONRPC: JSONRPC_VERSION,
		ID:      NewRequestId(id),
		Params:  params,
		Header:  header,
		Request: Request{Method: method},
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
	bytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	raw := json.RawMessage(bytes)
	return &raw, nil
}

// Initialize negotiates with the server. Must be called after Start.
func (c *Client) Initialize(ctx context.Context, request InitializeRequest) (*InitializeResult, error) {
	params := struct {
		ProtocolVersion string             `json:"protocolVersion"`
		ClientInfo      Implementation     `json:"clientInfo"`
		Capabilities    ClientCapabilities `json:"capabilities"`
	}{
		ProtocolVersion: request.Params.ProtocolVersion,
		ClientInfo:      request.Params.ClientInfo,
		Capabilities:    request.Params.Capabilities,
	}
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
	if !slices.Contains(ValidProtocolVersions, result.ProtocolVersion) {
		return nil, UnsupportedProtocolVersionError{Version: result.ProtocolVersion}
	}
	c.serverCapabilities = result.Capabilities
	c.protocolVersion = result.ProtocolVersion
	if httpConn, ok := c.transport.(HTTPConnection); ok {
		httpConn.SetProtocolVersion(result.ProtocolVersion)
	}
	err = c.transport.SendNotification(ctx, JSONRPCNotification{
		JSONRPC: JSONRPC_VERSION,
		Notification: Notification{Method: "notifications/initialized"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send initialized notification: %w", err)
	}
	c.initialized = true
	return &result, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.sendRequest(ctx, "ping", nil, nil)
	return err
}

// ListToolsByPage manually lists tools by page.
func (c *Client) ListToolsByPage(ctx context.Context, request ListToolsRequest) (*ListToolsResult, error) {
	return listByPage[ListToolsResult](ctx, c, request.PaginatedRequest, request.Header, "tools/list")
}

// ListTools requests all tools, paginating automatically.
func (c *Client) ListTools(ctx context.Context, request ListToolsRequest) (*ListToolsResult, error) {
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

// CallTool invokes a specific tool on the server.
func (c *Client) CallTool(ctx context.Context, request CallToolRequest) (*CallToolResult, error) {
	response, err := c.sendRequest(ctx, "tools/call", request.Params, request.Header)
	if err != nil {
		return nil, err
	}
	return ParseCallToolResult(response)
}

func listByPage[T any](ctx context.Context, client *Client, request PaginatedRequest, header http.Header, method string) (*T, error) {
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

// GetTransport gives access to the underlying transport.
func (c *Client) GetTransport() Interface { return c.transport }

// GetServerCapabilities returns the server capabilities.
func (c *Client) GetServerCapabilities() ServerCapabilities { return c.serverCapabilities }

// GetClientCapabilities returns the client capabilities.
func (c *Client) GetClientCapabilities() ClientCapabilities { return c.clientCapabilities }

// GetSessionId returns the session ID of the client.
func (c *Client) GetSessionId() string {
	if c.transport == nil {
		return ""
	}
	return c.transport.GetSessionId()
}

// IsInitialized returns true if the client has been initialized.
func (c *Client) IsInitialized() bool { return c.initialized }

// handleIncomingRequest processes server-to-client requests.
func (c *Client) handleIncomingRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	switch request.Method {
	case string(MethodPing):
		return c.handlePingRequestTransport(ctx, request)
	default:
		return nil, fmt.Errorf("unsupported request method: %s", request.Method)
	}
}

func (c *Client) handlePingRequestTransport(_ context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	b, _ := json.Marshal(&EmptyResult{})
	response := NewJSONRPCResultResponse(request.ID, b)
	return &response, nil
}

// UnsupportedProtocolVersionError is returned when the server proposes an unsupported version.
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
```

---

### A5. `request_handler.go`

Remove the `// Code generated` header comment. Write this exact content:

```go
package mcp

import (
	"context"
	"encoding/json"
	"net/http"
)

// HandleMessage processes an incoming JSON-RPC message and returns a response.
func (s *MCPServer) HandleMessage(ctx context.Context, message json.RawMessage) JSONRPCMessage {
	ctx = context.WithValue(ctx, serverKey{}, s)

	var baseMessage struct {
		JSONRPC string    `json:"jsonrpc"`
		Method  MCPMethod `json:"method"`
		ID      any       `json:"id,omitempty"`
		Result  any       `json:"result,omitempty"`
	}
	if err := json.Unmarshal(message, &baseMessage); err != nil {
		return createErrorResponse(nil, PARSE_ERROR, "Failed to parse message")
	}
	if baseMessage.JSONRPC != JSONRPC_VERSION {
		return createErrorResponse(baseMessage.ID, INVALID_REQUEST, "Invalid JSON-RPC version")
	}
	if baseMessage.ID == nil {
		var notification JSONRPCNotification
		if err := json.Unmarshal(message, &notification); err != nil {
			return createErrorResponse(nil, PARSE_ERROR, "Failed to parse notification")
		}
		s.handleNotification(ctx, notification)
		return nil
	}
	if baseMessage.Result != nil {
		return nil
	}

	h := ctx.Value(requestHeader)
	headers, ok := h.(http.Header)
	if headers == nil || !ok {
		headers = make(http.Header)
	}

	var reqErr *requestError
	switch baseMessage.Method {
	case MethodInitialize:
		var request InitializeRequest
		var result *InitializeResult
		if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			reqErr = &requestError{id: baseMessage.ID, code: INVALID_REQUEST, err: &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method}}
		} else {
			request.Header = headers
			result, reqErr = s.handleInitialize(ctx, baseMessage.ID, request)
		}
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)

	case MethodPing:
		var request PingRequest
		var result *EmptyResult
		if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			reqErr = &requestError{id: baseMessage.ID, code: INVALID_REQUEST, err: &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method}}
		} else {
			request.Header = headers
			result, reqErr = s.handlePing(ctx, baseMessage.ID, request)
		}
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)

	case MethodToolsList:
		var request ListToolsRequest
		var result *ListToolsResult
		if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			reqErr = &requestError{id: baseMessage.ID, code: INVALID_REQUEST, err: &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method}}
		} else {
			request.Header = headers
			result, reqErr = s.handleListTools(ctx, baseMessage.ID, request)
		}
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)

	case MethodToolsCall:
		var request CallToolRequest
		var result *CallToolResult
		if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			reqErr = &requestError{id: baseMessage.ID, code: INVALID_REQUEST, err: &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method}}
		} else {
			request.Header = headers
			result, reqErr = s.handleToolCall(ctx, baseMessage.ID, request)
		}
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)

	default:
		return createErrorResponse(baseMessage.ID, METHOD_NOT_FOUND, "Method not found: "+string(baseMessage.Method))
	}
}
```

---

## Section B — Surgical Deletions from `utils.go`

Delete each of the following **complete function bodies** from `utils.go`. Search by function signature and delete from the `func` line through its closing `}`.

**Functions to delete:**
- `func AsTextResourceContents`
- `func AsBlobResourceContents`
- `func AsEmbeddedResource`
- `func NewLoggingMessageNotification`
- `func NewPromptMessage`
- `func NewResourceLink`
- `func NewEmbeddedResource`
- `func NewToolResultResource`
- `func NewListResourcesResult`
- `func NewListResourceTemplatesResult`
- `func NewReadResourceResult`
- `func NewListPromptsResult`
- `func NewGetPromptResult`
- `func ParseAnnotations`
- `func ParseGetPromptResult`
- `func ParseReadResourceResult`
- `func ParseResourceContents`

**Inside `func ParseContent`** — delete only the two case blocks (keep `case ContentTypeText`, `case ContentTypeImage`, `case ContentTypeAudio`):
```go
// DELETE these two cases:
case ContentTypeLink:
    // ... entire block
case ContentTypeResource:
    // ... entire block
```

**At the top of `utils.go`** — the `var ( _ ClientRequest = ... )` assertion blocks. Delete these specific lines (leave the rest):
```go
// DELETE these lines from the var blocks:
_ ClientRequest = (*GetPromptRequest)(nil)
_ ClientRequest = (*ListPromptsRequest)(nil)
_ ClientRequest = (*ListResourcesRequest)(nil)
_ ClientRequest = (*ReadResourceRequest)(nil)
_ ClientRequest = (*SubscribeRequest)(nil)
_ ClientRequest = (*UnsubscribeRequest)(nil)
_ ClientRequest = (*CompleteRequest)(nil)
_ ClientRequest = (*SetLevelRequest)(nil)
_ ClientNotification = (*RootsListChangedNotification)(nil)
_ ClientResult = (*CreateMessageResult)(nil)
_ ClientResult = (*ListRootsResult)(nil)
_ ServerRequest = (*CreateMessageRequest)(nil)
_ ServerRequest = (*ListRootsRequest)(nil)
_ ServerNotification = (*LoggingMessageNotification)(nil)
_ ServerNotification = (*ResourceUpdatedNotification)(nil)
_ ServerNotification = (*ResourceListChangedNotification)(nil)
_ ServerNotification = (*PromptListChangedNotification)(nil)
_ ServerResult = (*CompleteResult)(nil)
_ ServerResult = (*GetPromptResult)(nil)
_ ServerResult = (*ListPromptsResult)(nil)
_ ServerResult = (*ListResourcesResult)(nil)
_ ServerResult = (*ReadResourceResult)(nil)
```

Also **remove the `import "github.com/tinywasm/mcp/internal/cast"` line** from `utils.go` imports, and replace all `cast.ToBool(v)`, `cast.ToInt64(v)`, etc. usages with inline implementations.

> **Alternative for cast replacement:** Keep `internal/cast` package — it is NOT in the delete list of Phase 2. Just keep it and keep the import. Only remove it if `go mod tidy` says it is external/not tinywasm-owned.

---

## Section C — Structural Rewrite of `server.go`

### C1. Replace the `MCPServer` struct

`MCPServer` is **stateless regarding sessions**. It does not store sessions.
Session lifecycle is managed by the consumer; the current session is read from context
inside `HandleMessage` using `SessionFromContext(ctx)`.

Find the `type MCPServer struct { ... }` block and replace it entirely with:

```go
// MCPServer implements a Model Context Protocol server.
// It is stateless: session lifecycle is managed by the consumer via context.
type MCPServer struct {
	toolsMu                sync.RWMutex
	toolMiddlewareMu       sync.RWMutex
	notificationHandlersMu sync.RWMutex
	capabilitiesMu         sync.RWMutex
	toolFiltersMu          sync.RWMutex

	name                   string
	version                string
	instructions           string
	tools                  map[string]ServerTool
	toolHandlerMiddlewares []ToolHandlerMiddleware
	toolFilters            []ToolFilterFunc
	notificationHandlers   map[string]NotificationHandlerFunc
	capabilities           serverCapabilities
	paginationLimit        *int
}
```

### C2. Replace the `serverCapabilities` struct and related types

Find and replace the `serverCapabilities`, `resourceCapabilities`, `promptCapabilities`, `toolCapabilities`, `taskCapabilities` structs with:

```go
type serverCapabilities struct {
	tools *toolCapabilities
}

type toolCapabilities struct {
	listChanged bool
}
```

### C3. Delete these top-level declarations from `server.go`

Search for each by name and delete the complete declaration (struct, type, or function):

**Types/structs to delete:**
- `type resourceEntry struct`
- `type resourceTemplateEntry struct`
- `type taskEntry struct`
- `type ResourceHandlerFunc func`
- `type ResourceTemplateHandlerFunc func`
- `type PromptHandlerFunc func`
- `type TaskToolHandlerFunc func`
- `type ResourceHandlerMiddleware func`
- `type ServerTaskTool struct`
- `type ServerPrompt struct`
- `type ServerResource struct`
- `type ServerResourceTemplate struct`

**ServerOption functions to delete:**
- `func WithResourceCapabilities`
- `func WithPromptCompletionProvider`
- `func WithResourceCompletionProvider`
- `func WithResourceHandlerMiddleware`
- `func WithTaskCapabilities`
- `func WithMaxConcurrentTasks`
- `func WithLoggingCapabilities`
- `func WithSamplingCapabilities`
- `func WithElicitationCapabilities`
- `func WithRootsCapabilities`
- `func WithCompletionCapabilities`
- `func WithPromptCapabilities`

**Methods to delete** (search for `func (s *MCPServer) MethodName`):
- `func (s *MCPServer) AddResource`
- `func (s *MCPServer) AddResources`
- `func (s *MCPServer) DeleteResource`
- `func (s *MCPServer) DeleteResources`
- `func (s *MCPServer) AddResourceTemplate`
- `func (s *MCPServer) AddResourceTemplates`
- `func (s *MCPServer) DeleteResourceTemplate`
- `func (s *MCPServer) DeleteResourceTemplates`
- `func (s *MCPServer) AddPrompt`
- `func (s *MCPServer) AddPrompts`
- `func (s *MCPServer) DeletePrompt`
- `func (s *MCPServer) DeletePrompts`
- `func (s *MCPServer) AddTaskTool`
- `func (s *MCPServer) SetHooks`
- `func (s *MCPServer) SetTaskHooks`
- `func (s *MCPServer) handleListResources`
- `func (s *MCPServer) handleReadResource`
- `func (s *MCPServer) handleListResourceTemplates`
- `func (s *MCPServer) handleResourceSubscribe`
- `func (s *MCPServer) handleResourceUnsubscribe`
- `func (s *MCPServer) handleListPrompts`
- `func (s *MCPServer) handleGetPrompt`
- `func (s *MCPServer) handleSetLevel`
- `func (s *MCPServer) handleComplete`
- `func (s *MCPServer) handleSamplingCreateMessage`
- `func (s *MCPServer) handleElicitation`
- `func (s *MCPServer) handleListRoots`
- `func (s *MCPServer) handleTasksList`
- `func (s *MCPServer) RegisterSession`
- `func (s *MCPServer) UnregisterSession`
- `func (s *MCPServer) WithContext`
- `func (s *MCPServer) AddSessionTool`
- `func (s *MCPServer) AddSessionTools`
- `func (s *MCPServer) DeleteSessionTools`
- `func (s *MCPServer) AddSessionResource`
- `func (s *MCPServer) AddSessionResources`
- `func (s *MCPServer) DeleteSessionResources`
- `func (s *MCPServer) AddSessionResourceTemplate`
- `func (s *MCPServer) AddSessionResourceTemplates`
- `func (s *MCPServer) DeleteSessionResourceTemplates`
- `func (s *MCPServer) sendNotificationToAllClients`
- `func (s *MCPServer) sendNotificationToSpecificClient`
- `func (s *MCPServer) SendNotificationToAllClients`
- `func (s *MCPServer) SendNotificationToClient`
- `func (s *MCPServer) SendNotificationToSpecificClient`
- `func (s *MCPServer) SendLogMessageToClient`
- `func (s *MCPServer) SendLogMessageToSpecificClient`
- `func (s *MCPServer) buildLogNotification`
- `func (s *MCPServer) handleTasksList`
- `func (s *MCPServer) handleTasksCreate`
- `func (s *MCPServer) handleTasksResult`
- `func (s *MCPServer) handleTasksCancel`
- Any method named `SendLogMessageToClient`
- Any method named `SendLogMessageToSpecificClient`
- Any method named `buildLogNotification`

### C4. Update `NewMCPServer` constructor

Find `func NewMCPServer(...)` and update the struct initialization to only set fields that still exist:

```go
func NewMCPServer(name, version string, opts ...ServerOption) *MCPServer {
	s := &MCPServer{
		name:                 name,
		version:              version,
		tools:                make(map[string]ServerTool),
		notificationHandlers: make(map[string]NotificationHandlerFunc),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
```

---

## Section D — Deletion List for `tools.go`

### D1. Remove `internal/jsonschema` import and its two usages

1. Delete the import line: `"github.com/tinywasm/mcp/internal/jsonschema"`
2. Find the two functions that use `jsonschema.Reflector{...}`. These are tool builder functions (names likely contain `NewToolFromFunc` or similar). Delete them entirely.

### D2. Remove task-related declarations

- From `type CallToolParams struct`: delete the `Task *TaskParams` field.
- Delete `type TaskSupport` type and its constants (`TaskSupportForbidden`, `TaskSupportOptional`, `TaskSupportRequired`).
- Delete `type ToolExecution struct`.
- From `type Tool struct`: delete the `Execution *ToolExecution` field (if present).

---

## Section E — Deletion List for `types.go`

Delete each of the following top-level declarations. Search by exact name, find the full block, delete it.

### E1. MCPMethod constants to delete

Delete from the `const (...)` block (or individual `const` declarations):
- `MethodResourcesList`
- `MethodResourcesTemplatesList`
- `MethodResourcesRead`
- `MethodResourcesSubscribe`
- `MethodResourcesUnsubscribe`
- `MethodPromptsList`
- `MethodPromptsGet`
- `MethodSamplingCreateMessage`
- `MethodLoggingSetLevel`
- `MethodSetLogLevel`
- `MethodCompletionComplete`
- `MethodListRoots`
- `MethodElicitationCreate`
- `MethodNotificationResourcesUpdated`
- `MethodNotificationResourcesListChanged`
- `MethodNotificationPromptsListChanged`
- `MethodNotificationRootsListChanged`

### E2. Types to delete

**Resource types:**
- `type Resource struct`
- `type ResourceTemplate struct`
- `type ResourceTemplateURITemplate` (and its methods)
- `type ResourceContents interface`
- `type TextResourceContents struct`
- `type BlobResourceContents struct`
- `type ResourceLink struct`
- `type EmbeddedResource struct`
- `type ReadResourceRequest struct`
- `type ReadResourceResult struct`
- `type ListResourcesRequest struct`
- `type ListResourcesResult struct`
- `type ListResourceTemplatesRequest struct`
- `type ListResourceTemplatesResult struct`
- `type SubscribeRequest struct`
- `type UnsubscribeRequest struct`
- `type ResourceSubscription struct`
- `type ResourceUpdatedNotification struct`
- `type ResourceListChangedNotification struct`

**Prompt types:**
- `type Prompt struct`
- `type PromptArgument struct`
- `type PromptMessage struct`
- `type GetPromptRequest struct`
- `type GetPromptResult struct`
- `type ListPromptsRequest struct`
- `type ListPromptsResult struct`
- `type PromptListChangedNotification struct`

**Sampling types:**
- `type SamplingHandler interface`
- `type CreateMessageRequest struct`
- `type CreateMessageResult struct`
- `type CreateMessageParams struct`
- `type SamplingMessage struct`
- `type ModelPreferences struct`
- `type ModelHint struct`

**Elicitation types:**
- `type ElicitationHandler interface`
- `type ElicitationRequest struct`
- `type ElicitationResult struct`
- `type ElicitationParams struct`
- `type ElicitationCapability struct`
- Any other `type Elicitation*` types

**Roots types:**
- `type RootsHandler interface`
- `type ListRootsRequest struct`
- `type ListRootsResult struct`
- `type Root struct`
- `type RootsListChangedNotification struct`

**Logging types:**
- `type LoggingLevel` (type + constants + methods)
- `type LoggingMessageNotification struct`
- `type SetLevelRequest struct`

**Completion types:**
- `type CompletionArgument struct`
- `type CompleteRequest struct`
- `type CompleteResult struct`
- `type PromptCompletionProvider interface`
- `type ResourceCompletionProvider interface`

**Task types:**
- All `type Task*` types
- All `type CreateTask*` types

### E3. Simplify `ServerCapabilities` struct

Find `type ServerCapabilities struct` and replace with:

```go
type ServerCapabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}
```

> **Note:** Check how `ToolsCapability` is defined. If the existing field is already `Tools *struct{ ListChanged bool }` or similar, keep the exact existing field type — just remove the other fields (Resources, Prompts, Logging, Sampling, Elicitation, Roots, Tasks, Completions).

### E4. Simplify `ClientCapabilities` struct

Find `type ClientCapabilities struct` and remove these fields:
- `Sampling *struct{} ...`
- `Roots *struct{ ListChanged bool } ...`
- `Elicitation *ElicitationCapability ...`

Keep only `Experimental` and any tool-related fields.

---

## Section E2 — Delete Eliminated Transport Files

Delete these files entirely (SSE and stdio are removed; HTTP streaming stays but is cleaned below):

```bash
rm transport_sse.go
rm transport_stdio.go
rm stdio.go
rm inprocess.go
rm inprocess_session.go
rm transport_inprocess.go
```

> `transport_streamable_http.go` and `http.go` **stay** but need OAuth stripped (Section E3).

---

## Section E3 — Strip OAuth from `transport_streamable_http.go`

Remove the following from `transport_streamable_http.go`:

- Field `oauthHandler *OAuthHandler` from the StreamableHTTP client struct
- Function `WithHTTPOAuth(config OAuthConfig) StreamableHTTPCOption` entirely
- `var ErrOAuthAuthorizationRequired = errors.New(...)` sentinel error
- `type OAuthAuthorizationRequiredError struct` and its `Error()` + `Unwrap()` methods
- All `if c.oauthHandler != nil { ... }` blocks
- The `oauthHandler.SetBaseURL(...)` block inside the constructor
- Methods `GetOAuthHandler() *OAuthHandler` and `IsOAuthEnabled() bool`

---

## Section E4 — Strip OAuth from `http.go`

Remove from `http.go`:

- Any `OAuthHandler` field or import reference
- Any `oauth` middleware or handler wiring
- Any `WithOAuth*` option functions

Keep the core `NewStreamableHTTPServer` / `HTTPHandler()` logic intact.

---

## Section F — Build, Test, Verify

```bash
# 1. Delete files from Phase 1 that are not yet deleted (verify)
# 2. Tidy
go mod tidy

# 3. Build — fix any remaining compile errors caused by leftover type references
go build ./...

# 4. Run tests
gotest

# 5. Push
gopush 'refactor: mcp protocol-only, HTTP transport, no sessions/resources/prompts'
```

---

← Back to [PLAN.md](PLAN.md)
