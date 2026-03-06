package mcp

import (
	"encoding/json"
	"net/http"
)

type StreamableHTTPSOption func(*StreamableHTTPServer)

// WithEndpointPath sets the HTTP path where the MCP endpoint is served.
func WithEndpointPath(path string) StreamableHTTPSOption {
	return func(s *StreamableHTTPServer) {
		s.endpointPath = path
	}
}

// WithStateLess configures whether the server is stateless.
func WithStateLess(stateless bool) StreamableHTTPSOption {
	return func(s *StreamableHTTPServer) {
		s.stateless = stateless
	}
}

// StreamableHTTPServer implements http.Handler for the MCP JSON-RPC protocol.
type StreamableHTTPServer struct {
	mcpServer    *MCPServer
	endpointPath string
	stateless    bool
}

// NewStreamableHTTPServer creates an http.Handler for the MCP JSON-RPC protocol.
func NewStreamableHTTPServer(s *MCPServer, opts ...StreamableHTTPSOption) *StreamableHTTPServer {
	server := &StreamableHTTPServer{
		mcpServer:    s,
		endpointPath: "/mcp",
		stateless:    false,
	}
	for _, opt := range opts {
		opt(server)
	}
	return server
}

// ServeHTTP implements http.Handler.
// Reads the JSON-RPC request body, calls MCPServer.HandleMessage, and writes the response.
func (s *StreamableHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	var message json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"error": map[string]any{
				"code":    -32700,
				"message": "Parse error",
			},
		})
		return
	}

	// Handle the message
	response := s.mcpServer.HandleMessage(r.Context(), message)

	// Write response
	w.Header().Set("Content-Type", "application/json")
	if response != nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
