package mcp

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"sync"
)

// ServerOption is a function that configures an MCPServer.
type ServerOption func(*MCPServer)

// ToolHandlerFunc handles tool calls with given arguments.
type ToolHandlerFunc func(ctx context.Context, request CallToolRequest) (*CallToolResult, error)

// ToolHandlerMiddleware is a middleware function that wraps a ToolHandlerFunc.
type ToolHandlerMiddleware func(ToolHandlerFunc) ToolHandlerFunc

// ToolFilterFunc is a function that filters tools based on context, typically using session information.
type ToolFilterFunc func(ctx context.Context, tools []ProtocolTool) []ProtocolTool

// NotificationHandlerFunc handles incoming notifications.
type NotificationHandlerFunc func(ctx context.Context, notification JSONRPCNotification)

// ServerTool combines a ProtocolTool with its ToolHandlerFunc.
type ServerTool struct {
	Tool    ProtocolTool
	Handler ToolHandlerFunc
}

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

type serverCapabilities struct {
	tools *toolCapabilities
}

type toolCapabilities struct {
	listChanged bool
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

type serverKey struct{}

func createErrorResponse(id any, code int, message string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: JSONRPC_VERSION,
		ID:      NewRequestId(id),
		Error:   &JSONRPCErrorDetails{Code: code, Message: message},
	}
}

func createResponse(id any, result any) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: JSONRPC_VERSION,
		ID:      NewRequestId(id),
		Result:  result,
	}
}

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
		Error:   JSONRPCErrorDetails{Code: e.code, Message: e.err.Error()},
	}
}

func (e *requestError) Unwrap() error {
	return e.err
}

// WithPaginationLimit sets the pagination limit for the server.
func WithPaginationLimit(limit int) ServerOption {
	return func(s *MCPServer) {
		s.paginationLimit = &limit
	}
}

// WithToolHandlerMiddleware allows adding a middleware for the tool handler call chain.
func WithToolHandlerMiddleware(mw ToolHandlerMiddleware) ServerOption {
	return func(s *MCPServer) {
		s.toolMiddlewareMu.Lock()
		s.toolHandlerMiddlewares = append(s.toolHandlerMiddlewares, mw)
		s.toolMiddlewareMu.Unlock()
	}
}

// WithToolFilter adds a filter function that will be applied to tools before they are returned.
func WithToolFilter(filter ToolFilterFunc) ServerOption {
	return func(s *MCPServer) {
		s.toolFiltersMu.Lock()
		s.toolFilters = append(s.toolFilters, filter)
		s.toolFiltersMu.Unlock()
	}
}

// WithInstructions sets the server instructions returned in the initialize response.
func WithInstructions(instructions string) ServerOption {
	return func(s *MCPServer) {
		s.instructions = instructions
	}
}

// WithToolCapabilities configures tool-related server capabilities.
func WithToolCapabilities(listChanged bool) ServerOption {
	return func(s *MCPServer) {
		s.capabilities.tools = &toolCapabilities{
			listChanged: listChanged,
		}
	}
}

// NewMCPServer creates a new MCP server instance.
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

// AddTools registers multiple tools at once.
func (s *MCPServer) AddTools(tools ...ServerTool) {
	s.toolsMu.Lock()
	for _, entry := range tools {
		s.tools[entry.Tool.Name] = entry
	}
	s.toolsMu.Unlock()
}

// AddTool registers a new tool and its handler (using ProtocolTool directly).
func (s *MCPServer) AddTool(tool ProtocolTool, handler ToolHandlerFunc) {
	s.AddTools(ServerTool{Tool: tool, Handler: handler})
}

// AddToolFromUser registers a user-facing Tool, converting to ProtocolTool internally.
func (s *MCPServer) AddToolFromUser(userTool Tool, handler ToolHandlerFunc) {
	protoTool := ProtocolTool{
		Name:        userTool.Name,
		Description: userTool.Description,
		InputSchema: parametersToInputSchema(userTool.Parameters),
	}
	s.AddTool(protoTool, handler)
}

// GetTool retrieves the specified tool.
func (s *MCPServer) GetTool(toolName string) *ServerTool {
	s.toolsMu.RLock()
	defer s.toolsMu.RUnlock()
	if tool, ok := s.tools[toolName]; ok {
		return &tool
	}
	return nil
}

// ListTools returns all registered tools.
func (s *MCPServer) ListTools(ctx context.Context) []ProtocolTool {
	s.toolsMu.RLock()
	tools := make([]ProtocolTool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool.Tool)
	}
	s.toolsMu.RUnlock()

	// Apply filters
	s.toolFiltersMu.RLock()
	for _, filter := range s.toolFilters {
		tools = filter(ctx, tools)
	}
	s.toolFiltersMu.RUnlock()

	// Sort
	slices.SortFunc(tools, func(a, b ProtocolTool) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return tools
}

func (s *MCPServer) handleInitialize(ctx context.Context, id any, request InitializeRequest) (*InitializeResult, *requestError) {
	capabilities := ServerCapabilities{}
	if s.capabilities.tools != nil {
		capabilities.Tools = &ToolsCapability{
			ListChanged: s.capabilities.tools.listChanged,
		}
	}

	result := InitializeResult{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ServerInfo: Implementation{
			Name:    s.name,
			Version: s.version,
		},
		Capabilities: capabilities,
		Instructions: s.instructions,
	}

	if session := SessionFromContext(ctx); session != nil {
		session.Initialize()
	}

	return &result, nil
}

func (s *MCPServer) handlePing(_ context.Context, _ any, _ PingRequest) (*EmptyResult, *requestError) {
	return &EmptyResult{}, nil
}

func (s *MCPServer) handleListTools(ctx context.Context, id any, request ListToolsRequest) (*ListToolsResult, *requestError) {
	tools := s.ListTools(ctx)

	// Apply pagination
	toolsToReturn, nextCursor, err := listByPagination(s, request.Params.Cursor, tools)
	if err != nil {
		return nil, &requestError{id: id, code: INVALID_PARAMS, err: err}
	}

	return &ListToolsResult{
		Tools: toolsToReturn,
		PaginatedResult: PaginatedResult{
			NextCursor: nextCursor,
		},
	}, nil
}

func (s *MCPServer) handleToolCall(ctx context.Context, id any, request CallToolRequest) (*CallToolResult, *requestError) {
	s.toolsMu.RLock()
	tool, ok := s.tools[request.Params.Name]
	s.toolsMu.RUnlock()

	if !ok {
		return nil, &requestError{
			id:   id,
			code: INVALID_PARAMS,
			err:  fmt.Errorf("tool '%s' not found", request.Params.Name),
		}
	}

	finalHandler := tool.Handler
	s.toolMiddlewareMu.RLock()
	mw := s.toolHandlerMiddlewares
	for i := len(mw) - 1; i >= 0; i-- {
		finalHandler = mw[i](finalHandler)
	}
	s.toolMiddlewareMu.RUnlock()

	result, err := finalHandler(ctx, request)
	if err != nil {
		return nil, &requestError{id: id, code: INTERNAL_ERROR, err: err}
	}

	return result, nil
}

func (s *MCPServer) handleNotification(ctx context.Context, notification JSONRPCNotification) {
	s.notificationHandlersMu.RLock()
	handler, ok := s.notificationHandlers[notification.Method]
	s.notificationHandlersMu.RUnlock()
	if ok {
		handler(ctx, notification)
	}
}

func listByPagination[T Named](s *MCPServer, cursor Cursor, allElements []T) ([]T, Cursor, error) {
	limit := 100
	if s.paginationLimit != nil {
		limit = *s.paginationLimit
	}
	
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
	
	endPos := startPos + limit
	if endPos > len(allElements) {
		endPos = len(allElements)
	}

	var nextCursor Cursor
	if endPos < len(allElements) {
		nextCursor = Cursor(base64.StdEncoding.EncodeToString([]byte(allElements[endPos-1].GetName())))
	}

	return allElements[startPos:endPos], nextCursor, nil
}
