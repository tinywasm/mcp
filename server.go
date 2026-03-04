package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"sync"
	"net/http"
)

// ServerOption is a function that configures an MCPServer.
type ServerOption func(*MCPServer)

// ToolHandlerFunc handles tool calls with given arguments.
type ToolHandlerFunc func(ctx context.Context, request CallToolRequest) (*CallToolResult, error)

// ToolHandlerMiddleware is a middleware function that wraps a ToolHandlerFunc.
type ToolHandlerMiddleware func(ToolHandlerFunc) ToolHandlerFunc

// ToolFilterFunc is a function that filters tools based on context, typically using session information.
type ToolFilterFunc func(ctx context.Context, tools []Tool) []Tool

// ServerTool combines a Tool with its ToolHandlerFunc.
type ServerTool struct {
	Tool    Tool
	Handler ToolHandlerFunc
}

// serverKey is the context key for storing the server instance
type serverKey struct{}

// ServerFromContext retrieves the MCPServer instance from a context
func ServerFromContext(ctx context.Context) *MCPServer {
	if srv, ok := ctx.Value(serverKey{}).(*MCPServer); ok {
		return srv
	}
	return nil
}

// UnparsableMessageError is attached to the RequestError when json.Unmarshal
// fails on the request.
type UnparsableMessageError struct {
	message json.RawMessage
	method  MCPMethod
	err     error
}

func (e *UnparsableMessageError) Error() string {
	return fmt.Sprintf("unparsable %s request: %s", e.method, e.err)
}

func (e *UnparsableMessageError) Unwrap() error {
	return e.err
}

func (e *UnparsableMessageError) GetMessage() json.RawMessage {
	return e.message
}

func (e *UnparsableMessageError) GetMethod() MCPMethod {
	return e.method
}

// requestError is an error that can be converted to a JSON-RPC error.
// Implements Unwrap() to allow inspecting the error chain.
type requestError struct {
	id   any
	code int
	err  error
}

func (e *requestError) Error() string {
	return fmt.Sprintf("request error: %s", e.err)
}

func (e *requestError) ToJSONRPCError() JSONRPCError {
	return JSONRPCError{
		JSONRPC: JSONRPC_VERSION,
		ID:      NewRequestId(e.id),
		Error:   NewJSONRPCErrorDetails(e.code, e.err.Error(), nil),
	}
}

func (e *requestError) Unwrap() error {
	return e.err
}

// NotificationHandlerFunc handles incoming notifications.
type NotificationHandlerFunc func(ctx context.Context, notification JSONRPCNotification)

// MCPServer implements a Model Context Protocol server that can handle tool requests.
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
	sessions               sync.Map
}

// WithPaginationLimit sets the pagination limit for the 
func WithPaginationLimit(limit int) ServerOption {
	return func(s *MCPServer) {
		s.paginationLimit = &limit
	}
}

// serverCapabilities defines the supported features of the MCP server
type serverCapabilities struct {
	tools *toolCapabilities
}

// toolCapabilities defines the supported tool-related features
type toolCapabilities struct {
	listChanged bool
}

// WithToolHandlerMiddleware allows adding a middleware for the
// tool handler call chain.
func WithToolHandlerMiddleware(
	toolHandlerMiddleware ToolHandlerMiddleware,
) ServerOption {
	return func(s *MCPServer) {
		s.toolMiddlewareMu.Lock()
		s.toolHandlerMiddlewares = append(s.toolHandlerMiddlewares, toolHandlerMiddleware)
		s.toolMiddlewareMu.Unlock()
	}
}

// WithToolFilter adds a filter function that will be applied to tools before they are returned in list_tools
func WithToolFilter(
	toolFilter ToolFilterFunc,
) ServerOption {
	return func(s *MCPServer) {
		s.toolFiltersMu.Lock()
		s.toolFilters = append(s.toolFilters, toolFilter)
		s.toolFiltersMu.Unlock()
	}
}

// WithRecovery adds a middleware that recovers from panics in tool handlers.
func WithRecovery() ServerOption {
	return WithToolHandlerMiddleware(func(next ToolHandlerFunc) ToolHandlerFunc {
		return func(ctx context.Context, request CallToolRequest) (result *CallToolResult, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf(
						"panic recovered in %s tool handler: %v",
						request.Params.Name,
						r,
					)
				}
			}()
			return next(ctx, request)
		}
	})
}

// WithToolCapabilities configures tool-related server capabilities
func WithToolCapabilities(listChanged bool) ServerOption {
	return func(s *MCPServer) {
		// Always create a non-nil capability object
		s.capabilities.tools = &toolCapabilities{
			listChanged: listChanged,
		}
	}
}

// WithInstructions sets the server instructions for the client returned in the initialize response
func WithInstructions(instructions string) ServerOption {
	return func(s *MCPServer) {
		s.instructions = instructions
	}
}

// NewMCPServer creates a new MCP server instance with the given name, version and options
func NewMCPServer(
	name, version string,
	opts ...ServerOption,
) *MCPServer {
	s := &MCPServer{
		tools:                  make(map[string]ServerTool),
		toolHandlerMiddlewares: make([]ToolHandlerMiddleware, 0),
		name:                   name,
		version:                version,
		notificationHandlers:   make(map[string]NotificationHandlerFunc),
		capabilities: serverCapabilities{
			tools: nil,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// AddTool registers a new tool and its handler
func (s *MCPServer) AddTool(tool Tool, handler ToolHandlerFunc) {
	s.AddTools(ServerTool{Tool: tool, Handler: handler})
}

// Register tool capabilities due to a tool being added.  Default to
// listChanged: true, but don't change the value if we've already explicitly
// registered tools.listChanged false.
func (s *MCPServer) implicitlyRegisterToolCapabilities() {
	s.implicitlyRegisterCapabilities(
		func() bool { return s.capabilities.tools != nil },
		func() { s.capabilities.tools = &toolCapabilities{listChanged: true} },
	)
}

func (s *MCPServer) implicitlyRegisterCapabilities(check func() bool, register func()) {
	s.capabilitiesMu.RLock()
	if check() {
		s.capabilitiesMu.RUnlock()
		return
	}
	s.capabilitiesMu.RUnlock()

	s.capabilitiesMu.Lock()
	if !check() {
		register()
	}
	s.capabilitiesMu.Unlock()
}

// AddTools registers multiple tools at once
func (s *MCPServer) AddTools(tools ...ServerTool) {
	s.implicitlyRegisterToolCapabilities()

	s.toolsMu.Lock()
	for _, entry := range tools {
		name := entry.Tool.Name
		s.tools[name] = entry
	}
	s.toolsMu.Unlock()

	// When the list of available tools changes, servers that declared the listChanged capability SHOULD send a notification.
	if s.capabilities.tools.listChanged {
		// Send notification to all initialized sessions
		s.SendNotificationToAllClients(MethodNotificationToolsListChanged, nil)
	}
}

// SetTools replaces all existing tools with the provided list
func (s *MCPServer) SetTools(tools ...ServerTool) {
	s.toolsMu.Lock()
	s.tools = make(map[string]ServerTool, len(tools))
	s.toolsMu.Unlock()
	s.AddTools(tools...)
}

// GetTool retrieves the specified tool
func (s *MCPServer) GetTool(toolName string) *ServerTool {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()
	if tool, ok := s.tools[toolName]; ok {
		return &tool
	}
	return nil
}

func (s *MCPServer) ListTools() map[string]*ServerTool {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()
	if len(s.tools) == 0 {
		return nil
	}
	// Create a copy to prevent external modification
	toolsCopy := make(map[string]*ServerTool, len(s.tools))
	for name, tool := range s.tools {
		toolsCopy[name] = &tool
	}
	return toolsCopy
}

// DeleteTools removes tools from the server
func (s *MCPServer) DeleteTools(names ...string) {
	s.toolsMu.Lock()
	var exists bool
	for _, name := range names {
		if _, ok := s.tools[name]; ok {
			delete(s.tools, name)
			exists = true
		}
	}
	s.toolsMu.Unlock()

	// When the list of available tools changes, servers that declared the listChanged capability SHOULD send a notification.
	if exists && s.capabilities.tools != nil && s.capabilities.tools.listChanged {
		// Send notification to all initialized sessions
		s.SendNotificationToAllClients(MethodNotificationToolsListChanged, nil)
	}
}

// AddNotificationHandler registers a new handler for incoming notifications
func (s *MCPServer) AddNotificationHandler(
	method string,
	handler NotificationHandlerFunc,
) {
	s.notificationHandlersMu.Lock()
	defer s.notificationHandlersMu.Unlock()
	s.notificationHandlers[method] = handler
}

func (s *MCPServer) handleInitialize(
	ctx context.Context,
	_ any,
	request InitializeRequest,
) (*InitializeResult, *requestError) {
	capabilities := ServerCapabilities{}

	// Only add tool capabilities if they're configured
	if s.capabilities.tools != nil {
		capabilities.Tools = &struct {
			ListChanged bool `json:"listChanged,omitempty"`
		}{
			ListChanged: s.capabilities.tools.listChanged,
		}
	}

	result := InitializeResult{
		ProtocolVersion: s.protocolVersion(request.Params.ProtocolVersion),
		ServerInfo: Implementation{
			Name:    s.name,
			Version: s.version,
		},
		Capabilities: capabilities,
		Instructions: s.instructions,
	}

	if session := ClientSessionFromContext(ctx); session != nil {
		session.Initialize()

		// Store client info if the session supports it
		if sessionWithClientInfo, ok := session.(SessionWithClientInfo); ok {
			sessionWithClientInfo.SetClientInfo(request.Params.ClientInfo)
			sessionWithClientInfo.SetClientCapabilities(request.Params.Capabilities)
		}
	}

	return &result, nil
}

func (s *MCPServer) protocolVersion(clientVersion string) string {
	// For backwards compatibility, if the server does not receive an MCP-Protocol-Version header,
	// and has no other way to identify the version - for example, by relying on the protocol version negotiated
	// during initialization - the server SHOULD assume protocol version 2025-03-26
	// https://modelcontextprotocol.io/specification/2025-06-18/basic/transports#protocol-version-header
	if len(clientVersion) == 0 {
		clientVersion = "2025-03-26"
	}

	if slices.Contains(ValidProtocolVersions, clientVersion) {
		return clientVersion
	}

	return LATEST_PROTOCOL_VERSION
}

func (s *MCPServer) handlePing(
	_ context.Context,
	_ any,
	_ PingRequest,
) (*EmptyResult, *requestError) {
	return &EmptyResult{}, nil
}

func listByPagination[T Named](
	_ context.Context,
	s *MCPServer,
	cursor Cursor,
	allElements []T,
) ([]T, Cursor, error) {
	startPos := 0
	if cursor != "" {
		c, err := base64.StdEncoding.DecodeString(string(cursor))
		if err != nil {
			return nil, "", err
		}
		cString := string(c)
		startPos = sort.Search(len(allElements), func(i int) bool {
			return allElements[i].GetName() > cString
		})
	}
	endPos := len(allElements)
	if s.paginationLimit != nil {
		if len(allElements) > startPos+*s.paginationLimit {
			endPos = startPos + *s.paginationLimit
		}
	}
	elementsToReturn := allElements[startPos:endPos]
	// set the next cursor
	nextCursor := func() Cursor {
		if s.paginationLimit != nil && len(elementsToReturn) >= *s.paginationLimit {
			nc := elementsToReturn[len(elementsToReturn)-1].GetName()
			toString := base64.StdEncoding.EncodeToString([]byte(nc))
			return Cursor(toString)
		}
		return ""
	}()
	return elementsToReturn, nextCursor, nil
}

func (s *MCPServer) handleListTools(
	ctx context.Context,
	id any,
	request ListToolsRequest,
) (*ListToolsResult, *requestError) {
	// Get the base tools from the server
	s.toolsMu.RLock()
	tools := make([]Tool, 0, len(s.tools))

	// Get all tool names for consistent ordering
	toolNames := make([]string, 0, len(s.tools))
	for name := range s.tools {
		toolNames = append(toolNames, name)
	}

	// Sort the tool names for consistent ordering
	sort.Strings(toolNames)

	// Add tools in sorted order
	for _, name := range toolNames {
		if tool, ok := s.tools[name]; ok {
			tools = append(tools, tool.Tool)
		}
	}
	s.toolsMu.RUnlock()

	// Check if there are session-specific tools
	session := ClientSessionFromContext(ctx)
	if session != nil {
		if sessionWithTools, ok := session.(SessionWithTools); ok {
			if sessionTools := sessionWithTools.GetSessionTools(); sessionTools != nil {
				// Override or add session-specific tools
				// We need to create a map first to merge the tools properly
				toolMap := make(map[string]Tool)

				// Add global tools first
				for _, tool := range tools {
					toolMap[tool.Name] = tool
				}

				// Then override with session-specific tools
				for name, serverTool := range sessionTools {
					toolMap[name] = serverTool.Tool
				}

				// Convert back to slice
				tools = make([]Tool, 0, len(toolMap))
				for _, tool := range toolMap {
					tools = append(tools, tool)
				}

				// Sort again to maintain consistent ordering
				sort.Slice(tools, func(i, j int) bool {
					return tools[i].Name < tools[j].Name
				})
			}
		}
	}

	// Apply tool filters if any are defined
	s.toolFiltersMu.RLock()
	if len(s.toolFilters) > 0 {
		for _, filter := range s.toolFilters {
			tools = filter(ctx, tools)
		}
	}
	s.toolFiltersMu.RUnlock()

	// Apply pagination
	toolsToReturn, nextCursor, err := listByPagination(
		ctx,
		s,
		request.Params.Cursor,
		tools,
	)
	if err != nil {
		return nil, &requestError{
			id:   id,
			code: INVALID_PARAMS,
			err:  err,
		}
	}

	result := ListToolsResult{
		Tools: toolsToReturn,
		PaginatedResult: PaginatedResult{
			NextCursor: nextCursor,
		},
	}
	return &result, nil
}

func (s *MCPServer) handleToolCall(
	ctx context.Context,
	id any,
	request CallToolRequest,
) (*CallToolResult, *requestError) {
	// First check session-specific tools
	var tool ServerTool
	var ok bool

	session := ClientSessionFromContext(ctx)
	if session != nil {
		if sessionWithTools, typeAssertOk := session.(SessionWithTools); typeAssertOk {
			if sessionTools := sessionWithTools.GetSessionTools(); sessionTools != nil {
				var sessionOk bool
				tool, sessionOk = sessionTools[request.Params.Name]
				if sessionOk {
					ok = true
				}
			}
		}
	}

	// If not found in session tools, check global tools
	if !ok {
		s.toolsMu.RLock()
		tool, ok = s.tools[request.Params.Name]
		s.toolsMu.RUnlock()
	}

	if !ok {
		return nil, &requestError{
			id:   id,
			code: INVALID_PARAMS,
			err:  fmt.Errorf("tool '%s' not found: %w", request.Params.Name, ErrToolNotFound),
		}
	}

	finalHandler := tool.Handler

	s.toolMiddlewareMu.RLock()
	mw := s.toolHandlerMiddlewares

	// Apply middlewares in reverse order
	for i := len(mw) - 1; i >= 0; i-- {
		finalHandler = mw[i](finalHandler)
	}
	s.toolMiddlewareMu.RUnlock()

	result, err := finalHandler(ctx, request)
	if err != nil {
		return nil, &requestError{
			id:   id,
			code: INTERNAL_ERROR,
			err:  err,
		}
	}

	return result, nil
}

func (s *MCPServer) handleNotification(
	ctx context.Context,
	notification JSONRPCNotification,
) JSONRPCMessage {
	s.notificationHandlersMu.RLock()
	handler, ok := s.notificationHandlers[notification.Method]
	s.notificationHandlersMu.RUnlock()

	if ok {
		handler(ctx, notification)
	}
	return nil
}

func createResponse(id any, result any) JSONRPCMessage {
	return NewJSONRPCResultResponse(NewRequestId(id), result)
}

func createErrorResponse(
	id any,
	code int,
	message string,
) JSONRPCMessage {
	return JSONRPCError{
		JSONRPC: JSONRPC_VERSION,
		ID:      NewRequestId(id),
		Error:   NewJSONRPCErrorDetails(code, message, nil),
	}
}

// HandleMessage processes an incoming JSON-RPC message and returns an appropriate response
func (s *MCPServer) HandleMessage(
	ctx context.Context,
	message json.RawMessage,
) JSONRPCMessage {
	// Add server to context
	ctx = context.WithValue(ctx, serverKey{}, s)
	var err *requestError

	var baseMessage struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  MCPMethod `json:"method"`
		ID      any           `json:"id,omitempty"`
		Result  any           `json:"result,omitempty"`
	}

	if err := json.Unmarshal(message, &baseMessage); err != nil {
		return createErrorResponse(
			nil,
			PARSE_ERROR,
			"Failed to parse message",
		)
	}

	// Check for valid JSONRPC version
	if baseMessage.JSONRPC != JSONRPC_VERSION {
		return createErrorResponse(
			baseMessage.ID,
			INVALID_REQUEST,
			"Invalid JSON-RPC version",
		)
	}

	if baseMessage.ID == nil {
		var notification JSONRPCNotification
		if err := json.Unmarshal(message, &notification); err != nil {
			return createErrorResponse(
				nil,
				PARSE_ERROR,
				"Failed to parse notification",
			)
		}
		s.handleNotification(ctx, notification)
		return nil // Return nil for notifications
	}

	if baseMessage.Result != nil {
		// this is a response to a request sent by the server
		return nil
	}

	// Get request header from ctx
	h := ctx.Value(requestHeader)
	headers, ok := h.(http.Header)

	if headers == nil || !ok {
		headers = make(http.Header)
	}

	switch baseMessage.Method {
	case MethodInitialize:
		var request InitializeRequest
		var result *InitializeResult
		if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			err = &requestError{
				id:   baseMessage.ID,
				code: INVALID_REQUEST,
				err:  &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method},
			}
		} else {
			request.Header = headers
			result, err = s.handleInitialize(ctx, baseMessage.ID, request)
		}
		if err != nil {
			return err.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)
	case MethodPing:
		var request PingRequest
		var result *EmptyResult
		if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			err = &requestError{
				id:   baseMessage.ID,
				code: INVALID_REQUEST,
				err:  &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method},
			}
		} else {
			request.Header = headers
			result, err = s.handlePing(ctx, baseMessage.ID, request)
		}
		if err != nil {
			return err.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)
	case MethodToolsList:
		var request ListToolsRequest
		var result *ListToolsResult
		if s.capabilities.tools == nil {
			err = &requestError{
				id:   baseMessage.ID,
				code: METHOD_NOT_FOUND,
				err:  fmt.Errorf("tools %w", ErrUnsupported),
			}
		} else if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			err = &requestError{
				id:   baseMessage.ID,
				code: INVALID_REQUEST,
				err:  &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method},
			}
		} else {
			request.Header = headers
			result, err = s.handleListTools(ctx, baseMessage.ID, request)
		}
		if err != nil {
			return err.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)
	case MethodToolsCall:
		var request CallToolRequest
		var result any
		if s.capabilities.tools == nil {
			err = &requestError{
				id:   baseMessage.ID,
				code: METHOD_NOT_FOUND,
				err:  fmt.Errorf("tools %w", ErrUnsupported),
			}
		} else if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			err = &requestError{
				id:   baseMessage.ID,
				code: INVALID_REQUEST,
				err:  &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method},
			}
		} else {
			request.Header = headers
			result, err = s.handleToolCall(ctx, baseMessage.ID, request)
		}
		if err != nil {
			return err.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, result)
	default:
		return createErrorResponse(
			baseMessage.ID,
			METHOD_NOT_FOUND,
			fmt.Sprintf("Method %s not found", baseMessage.Method),
		)
	}
}

// getSessionID extracts the session ID from the context.
func getSessionID(ctx context.Context) string {
	if session := ClientSessionFromContext(ctx); session != nil {
		return session.SessionID()
	}
	return ""
}
