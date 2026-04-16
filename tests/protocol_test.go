package mcp_test

import (
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
)

func TestInitialize_UnsupportedVersion_Rejected(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"1.0","clientInfo":{"name":"test","version":"1"}}}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, "unsupported protocol version") {
		t.Fatalf("expected unsupported protocol version error, got %s", respStr)
	}
}

func TestInitialize_ValidVersion_ReturnsServerInfo(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "my-server", Version: "2.3.4", Auth: mcp.OpenAuthorizer()}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1"}}}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, "my-server") || !contains(respStr, "2.3.4") {
		t.Fatalf("expected server info in response, got %s", respStr)
	}
	if !contains(respStr, "2024-11-05") {
		t.Fatalf("expected protocol version in response, got %s", respStr)
	}
}

func TestInitialize_GeneratesSessionID(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1"}}}`)
	srv.HandleMessage(&ctx, req)

	sessionID := ctx.Value(mcp.CtxKeySessionID)
	if sessionID == "" {
		t.Fatal("expected session ID to be generated and set in context")
	}
}

func TestInitialize_ExistingSessionID_Preserved(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	var ctx context.Context
	ctx.Set(mcp.CtxKeySessionID, "existing-session")
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1"}}}`)
	srv.HandleMessage(&ctx, req)

	sessionID := ctx.Value(mcp.CtxKeySessionID)
	if sessionID != "existing-session" {
		t.Fatalf("expected session ID 'existing-session' to be preserved, got %q", sessionID)
	}
}

func TestHandleMessage_NullBytes(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	var ctx context.Context
	req := []byte{0x00}
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected error response for null bytes")
	}
	respStr := encodeResponse(resp)
	if !contains(respStr, "PARSE_ERROR") && !contains(respStr, "-32700") {
		t.Fatalf("expected PARSE_ERROR, got %s", respStr)
	}
}

func TestExtractJSONValue_MissingKey(t *testing.T) {
	data := []byte(`{"foo":"bar"}`)
	got := mcp.ExtractJSONValue(data, "missing")
	if got != nil {
		t.Fatalf("expected nil for missing key, got %q", string(got))
	}
}

func TestExtractJSONValue_NonStringValue(t *testing.T) {
	data := []byte(`{"id":42}`)
	got := mcp.ExtractJSONValue(data, "id")
	if string(got) != "42" {
		t.Fatalf("expected '42', got %q", string(got))
	}
}
