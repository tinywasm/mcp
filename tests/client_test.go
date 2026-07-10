package mcp_test

import (
	"testing"

	"github.com/tinywasm/mcp"
)

func TestNewClient(t *testing.T) {
	c := mcp.NewClient("http://localhost:8080", "")
	if c == nil {
		t.Fatal("expected client")
	}
}
