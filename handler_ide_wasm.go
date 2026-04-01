//go:build wasm

package mcp

// ConfigureIDEs is a no-op in WASM environments.
func (s *Server) ConfigureIDEs() error { return nil }

// IDEStatus is a no-op in WASM environments.
func (s *Server) IDEStatus() string { return "" }
