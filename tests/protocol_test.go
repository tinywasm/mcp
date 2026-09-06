package mcp_test

import (
	"testing"

	"webtyp.com/context"
	"webtyp.com/mcp"
)

func TestInitialize_UnsupportedVersion_Negotiated(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"1.0","clientInfo":{"name":"test","version":"1"}}}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, "2025-11-25") {
		t.Fatalf("expected protocol version 2025-11-25 in response, got %s", respStr)
	}
	if contains(respStr, "error") {
		t.Fatalf("expected no error for unsupported version, got %s", respStr)
	}
}

func TestInitialize_ValidVersion_ReturnsServerInfo(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "my-server", Version: "2.3.4", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"test","version":"1"}}}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, "my-server") || !contains(respStr, "2.3.4") {
		t.Fatalf("expected server info in response, got %s", respStr)
	}
	if !contains(respStr, "2025-11-25") {
		t.Fatalf("expected protocol version 2025-11-25 in response, got %s", respStr)
	}
}

func TestInitialize_GeneratesSessionID(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"test","version":"1"}}}`)
	srv.HandleMessage(&ctx, req)

	sessionID := ctx.Value(mcp.CtxKeySessionID)
	if sessionID == "" {
		t.Fatal("expected session ID to be generated and set in context")
	}
}

func TestInitialize_ExistingSessionID_Preserved(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	ctx.Set(mcp.CtxKeySessionID, "existing-session")
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2025-11-25","clientInfo":{"name":"test","version":"1"}}}`)
	srv.HandleMessage(&ctx, req)

	sessionID := ctx.Value(mcp.CtxKeySessionID)
	if sessionID != "existing-session" {
		t.Fatalf("expected session ID 'existing-session' to be preserved, got %q", sessionID)
	}
}

func TestInitialize_OlderSupportedVersion_Accepted(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1"}}}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, "2024-11-05") {
		t.Fatalf("expected protocol version 2024-11-05 in response, got %s", respStr)
	}
	if contains(respStr, `"error":{`) {
		t.Fatalf("expected no error for supported version, got %s", respStr)
	}
}

func TestInitialize_NewerUnknownVersion_DowngradesGracefully(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2099-12-31","clientInfo":{"name":"test","version":"1"}}}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, "2025-11-25") {
		t.Fatalf("expected server to respond with latest version 2025-11-25, got %s", respStr)
	}
	if contains(respStr, `"error":{`) {
		t.Fatalf("expected no error for newer unknown version, got %s", respStr)
	}
}

func TestHandleMessage_NullBytes(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
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
	if !contains(respStr, `"id":null`) {
		t.Fatalf("expected id:null in parse error, got %s", respStr)
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

func TestHandleMessage_NumericId_EchoedAsNumber(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":123,"method":"ping"}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, `"id":123`) {
		t.Fatalf("expected id:123 as number, got %s", respStr)
	}
}

func TestHandleMessage_StringId_EchoedAsString(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"abc","method":"ping"}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, `"id":"abc"`) {
		t.Fatalf("expected id:\"abc\" as string, got %s", respStr)
	}
}

func TestSuccessResponseHasNoErrorKey(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"ping"}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if contains(respStr, "\"error\"") {
		t.Fatalf("success response must not contain \"error\" key: %s", respStr)
	}
	if !contains(respStr, "\"result\"") {
		t.Fatalf("success response must contain \"result\" key: %s", respStr)
	}
}

func TestErrorResponseHasNoResultKey(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	// Invalid request to trigger error
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"unknown_method"}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if contains(respStr, "\"result\"") {
		t.Fatalf("error response must not contain \"result\" key: %s", respStr)
	}
	if !contains(respStr, "\"error\"") {
		t.Fatalf("error response must contain \"error\" key: %s", respStr)
	}
}
