package mcp

import (
	"sync"

	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
	"github.com/tinywasm/unixid"
)

type Server struct {
	mu           sync.RWMutex
	name         string
	version      string
	instructions string
	tools        map[string]Tool
	providers    []ToolProvider
	auth         Authorizer
	log          func(messages ...any)
	SSE          SSEPublisher
}

type Config struct {
	Name    string
	Version string
	Auth    Authorizer
	SSE     SSEPublisher
}

func NewServer(config Config, providers []ToolProvider) (*Server, error) {
	if config.Auth == nil {
		return nil, fmt.Err("mcp", "Auth is required — use mcp.OpenAuthorizer() for open access")
	}
	s := &Server{
		name:      config.Name,
		version:   config.Version,
		auth:      config.Auth,
		SSE:       config.SSE,
		tools:     make(map[string]Tool),
		providers: providers,
		log:       func(messages ...any) {},
	}
	for _, p := range providers {
		if p != nil {
			for _, tool := range p.Tools() {
				if err := s.AddTool(tool); err != nil {
					return nil, err
				}
			}
		}
	}
	return s, nil
}

func (s *Server) AddTool(tool Tool) error {
	if tool.Name == "" || tool.Resource == "" || tool.Action == 0 || tool.Execute == nil {
		return fmt.Err("mcp", "invalid tool")
	}
	s.mu.Lock()
	s.tools[tool.Name] = tool
	s.mu.Unlock()

	if s.SSE != nil {
		notification := JSONRPCNotification{
			JSONRPC: JSONRPC_VERSION,
			Method:  "notifications/tools/list_changed",
		}
		var data []byte
		json.Encode(&notification, &data)
		s.SSE.Publish(data, "mcp")
	}

	return nil
}

func (s *Server) handleInitialize(ctx *context.Context, id string, params initializeParams) (*initializeResult, *requestError) {
	if params.ProtocolVersion != LATEST_PROTOCOL_VERSION {
		return nil, &requestError{id: id, code: INVALID_PARAMS, err: fmt.Err("mcp", "unsupported protocol version: "+params.ProtocolVersion)}
	}
	res := &initializeResult{
		ProtocolVersion: LATEST_PROTOCOL_VERSION,
		ServerInfo: implementationInfo{
			Name:    s.name,
			Version: s.version,
		},
	}
	res.Capabilities = `{"tools":{"listChanged":true}}`
	if ctx.Value(CtxKeySessionID) == "" {
		uid, _ := unixid.NewUnixID()
		ctx.Set(CtxKeySessionID, uid.GetNewID())
	}
	return res, nil
}

func (s *Server) handlePing(ctx *context.Context, id string) (*EmptyResult, *requestError) {
	return &EmptyResult{}, nil
}

func (s *Server) handleListTools(ctx *context.Context, id string) (*listToolsResult, *requestError) {
	s.mu.RLock()
	var toolsJSON string
	toolsJSON = "["
	first := true
	for _, t := range s.tools {
		if !first {
			toolsJSON += ","
		}
		var entryJSON string
		entry := toolEntry{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
		json.Encode(&entry, &entryJSON)
		toolsJSON += entryJSON
		first = false
	}
	toolsJSON += "]"
	s.mu.RUnlock()
	return &listToolsResult{Tools: toolsJSON}, nil
}

func (s *Server) handleToolCall(ctx *context.Context, id string, params CallToolParams) (*Result, *requestError) {
	s.mu.RLock()
	tool, ok := s.tools[params.Name]
	s.mu.RUnlock()
	if !ok {
		return nil, &requestError{id: id, code: INVALID_PARAMS, err: fmt.Err("mcp", "tool not found")}
	}

	userID := ctx.Value(CtxKeyUserID)
	if !s.auth.Can(userID, tool.Resource, tool.Action) {
		return nil, &requestError{id: id, code: -32001, err: fmt.Err("forbidden")}
	}

	req := Request{Params: params, Action: tool.Action}
	result, err := tool.Execute(ctx, req)
	if err != nil {
		return &Result{IsError: true, Content: Text(err.Error()).Content}, nil
	}
	return result, nil
}

func (s *Server) handleNotification(ctx *context.Context, notification JSONRPCNotification) {
	if s.SSE != nil {
		var data []byte
		json.Encode(&notification, &data)
		s.SSE.Publish(data, "mcp")
	}
}

type requestError struct {
	id   string
	code int
	err  error
}

func (e *requestError) Error() string { return e.err.Error() }
func (e *requestError) ToJSONRPCError() JSONRPCMessage {
	return newErrorResponse(e.id, e.code, e.err.Error(), nil)
}

func createErrorResponse(id string, code int, message string) JSONRPCMessage {
	return newErrorResponse(id, code, message, nil)
}

func createResponse(id string, result fmt.Fielder) JSONRPCMessage {
	return newResultResponse(id, result)
}
