package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEHub is the DI interface for the SSE log transport.
// The consumer (app) creates the real implementation using tinywasm/sse
// and injects it via NewHandler.
// *sse.SSEServer from tinywasm/sse satisfies this interface automatically.
type SSEHub interface {
	http.Handler                          // serves the /logs endpoint
	Publish(data []byte, channel string)  // publishes structured log entries
}

const HandlerTypeLoggable = 4

type LogEntry struct {
	Id           string `json:"id"`
	Timestamp    string `json:"timestamp"`
	Content      string `json:"content"`
	Type         uint8  `json:"type"`
	TabTitle     string `json:"tab_title"`
	HandlerName  string `json:"handler_name"`
	HandlerColor string `json:"handler_color"`
	HandlerType  int    `json:"handler_type"`
}

type Config struct {
	Port          string
	ServerName    string
	ServerVersion string
	AppName       string
	AppVersion    string
}

type Handler struct {
	config        Config
	toolHandlers  []ToolProvider
	tui           tuiRefresher
	sseHub        SSEHub
	exitChan      chan bool
	log           func(messages ...any)
	ideStatus     string
	actionFunc    func(string, string)
	stateProvider func() []byte
	httpServer    *http.Server
	mu            sync.Mutex
	running       bool
}

// NewHandler creates a Handler with injected SSE transport.
func NewHandler(config Config, toolHandlers []ToolProvider, tui tuiRefresher, sseHub SSEHub, exitChan chan bool) *Handler {
	h := &Handler{
		config:       config,
		toolHandlers: toolHandlers,
		tui:          tui,
		sseHub:       sseHub,
		exitChan:     exitChan,
	}
	h.log = func(messages ...any) {
		h.PublishLog(fmt.Sprint(messages...))
	}
	return h
}

// Name returns the handler name for Loggable interface
func (h *Handler) Name() string {
	return "MCP"
}

// SetLog implements Loggable interface
func (h *Handler) SetLog(f func(message ...any)) {
	h.log = func(messages ...any) {
		if f != nil {
			f(messages...)
		}
		h.PublishLog(fmt.Sprint(messages...))
	}
}

// URL returns the address where the MCP server is serving.
func (h *Handler) URL() string {
	return "http://localhost:" + h.config.Port + "/mcp"
}

// OnUIAction sets the callback for generic UI actions triggered from TUI or IDE
func (h *Handler) OnUIAction(actionFunc func(key, value string)) {
	h.actionFunc = actionFunc
}

// RegisterStateProvider registers a function that returns current handler state as JSON bytes.
func (h *Handler) RegisterStateProvider(fn func() []byte) {
	h.mu.Lock()
	h.stateProvider = fn
	h.mu.Unlock()
}

func (h *Handler) handleStateGET(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	fn := h.stateProvider
	h.mu.Unlock()
	if fn == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(fn())
}

// Serve starts the Model Context Protocol server for LLM integration via HTTP
func (h *Handler) Serve() {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	// Create MCP server with tool capabilities
	s := NewMCPServer(
		h.config.ServerName,
		h.config.ServerVersion,
		WithToolCapabilities(true),
	)

	// Load tools from all registered handlers
	for _, handler := range h.toolHandlers {
		if handler == nil {
			continue
		}
		tools := handler.GetMCPToolsMetadata()
		for _, toolMeta := range tools {
			tool := BuildTool(toolMeta)
			s.AddTool(tool, toolExecutorAdapter(handler, toolMeta.Execute, h.tui))
		}
	}

	// Create MCP HTTP server
	mcpServer := NewStreamableHTTPServer(s,
		WithEndpointPath("/mcp"),
		WithStateLess(true),
	)

	// Set up router
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpServer)
	if h.sseHub != nil {
		mux.Handle("/logs", h.sseHub)
	}
	mux.HandleFunc("/action", h.handleActionPOST)
	mux.HandleFunc("/state", h.handleStateGET)
	mux.HandleFunc("/version", h.handleVersion)

	h.mu.Lock()
	h.httpServer = &http.Server{
		Addr:    ":" + h.config.Port,
		Handler: mux,
	}
	ideMsg := h.ideStatus
	h.mu.Unlock()

	// Consolidate startup messages into ONE log
	startupMsg := fmt.Sprintf("Started on :%s/mcp", h.config.Port)
	if ideMsg != "" {
		startupMsg = fmt.Sprintf("%s (%s)", startupMsg, ideMsg)
	}
	h.log(startupMsg)

	go func() {
		if err := h.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			h.log("MCP HTTP server stopped:", err)
		}
	}()

	// Wait for exit signal (value or close)
	<-h.exitChan

	// ALWAYS shutdown on exit
	h.Stop()
}

// Stop gracefully shuts down the MCP HTTP server
func (h *Handler) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running || h.httpServer == nil {
		return nil
	}

	h.log("Shutting down MCP server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.httpServer.Shutdown(ctx); err != nil {
		h.log("Error shutting down MCP server:", err)
	}

	h.running = false
	return nil
}

func (h *Handler) handleActionPOST(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "query param 'key' is required", http.StatusBadRequest)
		return
	}
	value := r.URL.Query().Get("value") // empty string if absent

	h.mu.Lock()
	actionCb := h.actionFunc
	h.mu.Unlock()

	if actionCb != nil {
		actionCb(key, value)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Action applied: %s", key)))
	} else {
		http.Error(w, "No action handler configured", http.StatusServiceUnavailable)
	}
}

// handleVersion returns the daemon's binary version as JSON for stale-daemon detection.
func (h *Handler) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"version":"` + h.config.AppVersion + `"}`))
}

// PublishTabLog publishes a structured log entry to the SSE stream with full routing metadata.
func (h *Handler) PublishTabLog(tabTitle, handlerName, handlerColor, msg string) {
	if h.sseHub == nil {
		return
	}
	entry := LogEntry{
		Id:           fmt.Sprintf("%d", time.Now().UnixNano()),
		Timestamp:    time.Now().Format("15:04:05"),
		Content:      msg,
		Type:         1,
		TabTitle:     tabTitle,
		HandlerName:  handlerName,
		HandlerColor: handlerColor,
		HandlerType:  HandlerTypeLoggable,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	h.sseHub.Publish(data, "logs")
}

// PublishLog publishes a log message to the BUILD section as a generic MCP log.
func (h *Handler) PublishLog(msg string) {
	h.PublishTabLog("BUILD", "MCP", "#f97316", msg)
}
