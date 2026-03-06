package mcp

import (
	"net/http"
	"sync"
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

// Handler is a pure MCP JSON-RPC 2.0 handler.
// It only knows how to serve the /mcp endpoint.
// The caller (app) mounts it in an http.ServeMux and owns the HTTP server.
type Handler struct {
	config       Config
	toolHandlers []ToolProvider
	log          func(messages ...any)
	ideStatus    string
	mu           sync.Mutex
}

// NewHandler creates a Handler. No sseHub, no exitChan, no tuiRefresher.
func NewHandler(config Config, toolHandlers []ToolProvider) *Handler {
	return &Handler{
		config:       config,
		toolHandlers: toolHandlers,
		log:          func(messages ...any) {},
	}
}

// Name returns "MCP" — satisfies the Loggable interface used by devtui/app.
func (h *Handler) Name() string {
	return "MCP"
}

// SetLog sets the logger (used by ConfigureIDEs and internal errors).
func (h *Handler) SetLog(f func(message ...any)) {
	h.log = f
}

// URL returns the MCP endpoint address.
func (h *Handler) URL() string {
	return "http://localhost:" + h.config.Port + "/mcp"
}

// HTTPHandler returns the http.Handler for the /mcp endpoint only.
// Mount it as: mux.Handle("/mcp", mcpHandler.HTTPHandler())
func (h *Handler) HTTPHandler() http.Handler {
	s := NewMCPServer(h.config.ServerName, h.config.ServerVersion, WithToolCapabilities(true))
	for _, provider := range h.toolHandlers {
		if provider == nil {
			continue
		}
		for _, tool := range provider.GetMCPTools() {
			s.AddToolFromUser(tool, toolExecutorAdapter(tool.Execute))
		}
	}
	return NewStreamableHTTPServer(s, WithEndpointPath("/mcp"), WithStateLess(true))
}
