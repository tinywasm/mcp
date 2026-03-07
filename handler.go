package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

const HandlerTypeLoggable = 4

// Config holds the MCP server configuration.
type Config struct {
	Port          string
	ServerName    string
	ServerVersion string
	AppName       string
	AppVersion    string
}

// MethodFunc is the handler signature for consumer-defined JSON-RPC methods.
type MethodFunc func(ctx context.Context, params []byte) (any, error)

// methodFunc is the internal alias.
type methodFunc func(ctx context.Context, params []byte) (any, error)

// Handler re-establishes HTTP server ownership.
type Handler struct {
	config        Config
	sseHub        SSEHub                       // injected; nil = no /logs
	auth          Authorizer                   // default: denyAllAuthorizer (secure by default)
	fixed         []ToolProvider               // permanent tools (daemon-level)
	dynamic       []ToolProvider               // project tools; updated atomically
	toolMeta      map[string]Tool              // name → Tool (for RBAC lookup)
	customMethods map[string]methodFunc        // JSON-RPC custom method registry
	mcpServer     *MCPServer                   // rebuilt on SetDynamicProviders
	server        *http.Server
	apiKey        string
	log           func(messages ...any)
	ideStatus     string
	mu            sync.RWMutex
}

// NewHandler creates a Handler with sseHub and fixed providers.
func NewHandler(config Config, sseHub SSEHub, fixedProviders []ToolProvider) *Handler {
	h := &Handler{
		config:        config,
		sseHub:        sseHub,
		auth:          denyAllAuthorizer{},
		fixed:         fixedProviders,
		toolMeta:      make(map[string]Tool),
		customMethods: make(map[string]methodFunc),
		log:           func(messages ...any) {},
	}
	h.rebuildMCPServer()
	return h
}

// SetAuth sets the authorizer.
func (h *Handler) SetAuth(auth Authorizer) {
	h.mu.Lock()
	h.auth = auth
	h.mu.Unlock()
}

// SetDynamicProviders replaces project-level tools atomically.
func (h *Handler) SetDynamicProviders(providers ...ToolProvider) {
	h.mu.Lock()
	h.dynamic = providers
	h.mu.Unlock()
	h.rebuildMCPServer()
}

// SetAPIKey sets the API key for the server.
func (h *Handler) SetAPIKey(key string) {
	h.mu.Lock()
	h.apiKey = key
	h.mu.Unlock()
}

// RegisterMethod registers a consumer-defined JSON-RPC method.
func (h *Handler) RegisterMethod(name string, fn MethodFunc) {
	h.mu.Lock()
	h.customMethods[name] = methodFunc(fn)
	h.mu.Unlock()
}

// URL returns the base URL of the running HTTP server.
func (h *Handler) URL() string {
	return "http://localhost:" + h.config.Port
}

// Name returns "MCP".
func (h *Handler) Name() string {
	return "MCP"
}

// SetLog sets the logger.
func (h *Handler) SetLog(f func(message ...any)) {
	h.log = f
}

// Serve starts the HTTP server and blocks until exitChan is closed.
func (h *Handler) Serve(exitChan chan bool) {
	mux := http.NewServeMux()
	mux.Handle("/mcp", h.HTTPHandler())
	if h.sseHub != nil {
		mux.Handle("/logs", h.sseHub)
	}

	h.server = &http.Server{
		Addr:    ":" + h.config.Port,
		Handler: mux,
	}

	go func() {
		h.log("MCP server listening on " + h.config.Port)
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.log("MCP server error:", err)
		}
	}()

	<-exitChan
	h.Stop()
}

// Stop gracefully shuts down the server.
func (h *Handler) Stop() {
	if h.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.server.Shutdown(ctx)
	}
}

func (h *Handler) rebuildMCPServer() {
	s := NewMCPServer(h.config.ServerName, h.config.ServerVersion,
		WithToolCapabilities(true),
		WithToolHandlerMiddleware(h.toolAuthMiddleware()),
	)
	newMeta := make(map[string]Tool)

	h.mu.RLock()
	all := append(append([]ToolProvider{}, h.fixed...), h.dynamic...)
	h.mu.RUnlock()

	for _, p := range all {
		if p == nil {
			continue
		}
		for _, tool := range p.GetMCPTools() {
			s.AddToolFromUser(tool, toolExecutorAdapter(tool.Execute))
			newMeta[tool.Name] = tool
		}
	}

	h.mu.Lock()
	h.mcpServer = s
	h.toolMeta = newMeta
	h.mu.Unlock()
}

func (h *Handler) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 1. Auth: inject identity into ctx
		h.mu.RLock()
		auth := h.auth
		h.mu.RUnlock()
		ctx := auth.InjectIdentity(r.Context(), r)

		// 2. Read + buffer body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", 400)
			return
		}

		// 3. Peek method name
		var peek struct {
			Method string `json:"method"`
		}
		json.Unmarshal(body, &peek)

		// 4. Custom method?
		h.mu.RLock()
		fn, isCustom := h.customMethods[peek.Method]
		h.mu.RUnlock()

		if isCustom {
			var req struct {
				ID     any             `json:"id"`
				Params json.RawMessage `json:"params"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				http.Error(w, "invalid request", 400)
				return
			}
			result, err := fn(ctx, req.Params)
			writeJSONRPCResponse(w, req.ID, result, err)
			return
		}

		// 5. Delegate to MCPServer
		r2 := r.WithContext(ctx)
		r2.Body = io.NopCloser(bytes.NewReader(body))
		h.mu.RLock()
		srv := h.mcpServer
		h.mu.RUnlock()

		NewStreamableHTTPServer(srv, WithEndpointPath("/mcp")).ServeHTTP(w, r2)
	})
}

func (h *Handler) toolAuthMiddleware() ToolHandlerMiddleware {
	return func(next ToolHandlerFunc) ToolHandlerFunc {
		return func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
			h.mu.RLock()
			meta, ok := h.toolMeta[req.Params.Name]
			auth := h.auth
			h.mu.RUnlock()

			if ok && meta.Resource != "" {
				if !auth.CanExecute(ctx, meta.Resource, meta.Action) {
					return NewToolResultError("access denied: insufficient permissions"), nil
				}
			}
			return next(ctx, req)
		}
	}
}

func writeJSONRPCResponse(w http.ResponseWriter, id any, result any, err error) {
	w.Header().Set("Content-Type", "application/json")
	var resp any
	if err != nil {
		resp = struct {
			JSONRPC string `json:"jsonrpc"`
			ID      any    `json:"id"`
			Error   any    `json:"error"`
		}{
			JSONRPC: "2.0",
			ID:      id,
			Error: struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{
				Code:    -32000,
				Message: err.Error(),
			},
		}
	} else {
		resp = struct {
			JSONRPC string `json:"jsonrpc"`
			ID      any    `json:"id"`
			Result  any    `json:"result"`
		}{
			JSONRPC: "2.0",
			ID:      id,
			Result:  result,
		}
	}
	json.NewEncoder(w).Encode(resp)
}
