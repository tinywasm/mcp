package mcp

import (
	"github.com/tinywasm/model"
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
	authorize    model.Authorizer
	log          func(messages ...any)
	SSE          SSEPublisher
}

type Config struct {
	Name      string
	Version   string
	Authorize model.Authorizer // nil = every guarded tool is denied
	SSE       SSEPublisher
}

func NewServer(config Config, providers []ToolProvider) (*Server, error) {
	if config.Authorize == nil {
		return nil, fmt.Err("mcp", "Authorize is required")
	}
	s := &Server{
		name:      config.Name,
		version:   config.Version,
		authorize: config.Authorize,
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

func negotiateVersion(clientVersion string) string {
	for _, v := range SupportedProtocolVersions {
		if v == clientVersion {
			return v
		}
	}
	return LATEST_PROTOCOL_VERSION
}

func (s *Server) AddTool(tool Tool) error {
	if tool.Name == "" || tool.Action == 0 || tool.Execute == nil {
		return fmt.Err("mcp", "invalid tool: Name, Action and Execute are required")
	}

	// The access declaration and the resource must agree, and disagreeing is fatal at
	// startup — never a runtime surprise.
	switch tool.Access {
	case model.AccessGuarded:
		// A guarded tool with no resource used to authorize against "", which simply denied
		// every call: the tool looked protected and was in fact unreachable, silently.
		if tool.Resource == "" {
			return fmt.Err("mcp", "tool", tool.Name, "is guarded but declares no Resource — it would deny every call")
		}
	default:
		// A resource on a tool nobody checks reads as protection and gives none.
		if tool.Resource != "" {
			return fmt.Err("mcp", "tool", tool.Name, "declares Resource", string(tool.Resource),
				"but its Access does not check it — remove one or the other")
		}
	}
	s.mu.Lock()
	s.tools[tool.Name] = tool
	s.mu.Unlock()

	if s.SSE != nil {
		notification := JSONRPCNotification{
			Jsonrpc: JSONRPC_VERSION,
			Method:  "notifications/tools/list_changed",
		}
		var data []byte
		json.Encode(&notification, &data)
		s.SSE.Publish(data, "mcp")
	}

	return nil
}

func (s *Server) handleInitialize(ctx *context.Context, id RequestId, params initializeParams) (*initializeResult, *requestError) {
	agreedVersion := negotiateVersion(params.ProtocolVersion)
	res := &initializeResult{
		ProtocolVersion: agreedVersion,
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

func (s *Server) handlePing(ctx *context.Context, id RequestId) (*EmptyResult, *requestError) {
	return &EmptyResult{}, nil
}

func (s *Server) handleListTools(ctx *context.Context, id RequestId) (*listToolsResult, *requestError) {
	s.mu.RLock()
	var toolsJSON string
	toolsJSON = "["
	first := true
	for _, t := range s.tools {
		if !first {
			toolsJSON += ","
		}
		schema := inputSchemaOf(t.Args)
		var entryJSON string
		entry := toolEntry{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		}
		json.Encode(&entry, &entryJSON)
		toolsJSON += entryJSON
		first = false
	}
	toolsJSON += "]"
	s.mu.RUnlock()
	return &listToolsResult{Tools: toolsJSON}, nil
}

func (s *Server) handleToolCall(ctx *context.Context, id RequestId, params CallToolParams) (*Result, *requestError) {
	s.mu.RLock()
	tool, ok := s.tools[params.Name]
	s.mu.RUnlock()
	if !ok {
		return nil, &requestError{id: id, code: INVALID_PARAMS, err: fmt.Err("mcp", "tool not found")}
	}

	userID := ctx.Value(CtxKeyUserID)
	switch tool.Access {
	case model.AccessPublic:
		// no identity needed: declared on purpose

	case model.AccessAuthenticated:
		// identity is the check; the caller acts on themselves, so there is no resource
		if userID == "" {
			return nil, &requestError{id: id, code: -32001, err: fmt.Err("forbidden: authentication required")}
		}

	default: // model.AccessGuarded — the zero value: identity AND permission
		if userID == "" {
			return nil, &requestError{id: id, code: -32001, err: fmt.Err("forbidden: authentication required")}
		}
		// model.Allowed denies when Authorize is nil: the absence of an answer is not permission.
		if !model.Allowed(s.authorize, userID, tool.Resource, tool.Action) {
			return nil, &requestError{id: id, code: -32001, err: fmt.Err("forbidden")}
		}
	}

	req := Request{Params: params, Action: tool.actionByte()}
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
	id   RequestId
	code int
	err  error
}

func (e *requestError) Error() string { return e.err.Error() }
func (e *requestError) ToJSONRPCError() JSONRPCMessage {
	return newErrorResponse(e.id, e.code, e.err.Error(), nil)
}

func createErrorResponse(id RequestId, code int, message string) JSONRPCMessage {
	return newErrorResponse(id, code, message, nil)
}

func createResponse(id RequestId, result model.Encodable) JSONRPCMessage {
	return newResultResponse(id, result)
}
