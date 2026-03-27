//go:build !wasm

package mcp

import (
	"net/http"
)

type StreamableHTTPSOption func(*StreamableHTTPServer)

func WithEndpointPath(path string) StreamableHTTPSOption {
	return func(s *StreamableHTTPServer) {
		s.endpointPath = path
	}
}

func WithStateLess(stateless bool) StreamableHTTPSOption {
	return func(s *StreamableHTTPServer) {
		s.stateless = stateless
	}
}

type StreamableHTTPServer struct {
	server       *Server
	endpointPath string
	stateless    bool
}

func NewStreamableHTTPServer(s *Server, opts ...StreamableHTTPSOption) *StreamableHTTPServer {
	server := &StreamableHTTPServer{
		server:       s,
		endpointPath: "/mcp",
		stateless:    false,
	}
	for _, opt := range opts {
		opt(server)
	}
	return server
}

func (s *StreamableHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
