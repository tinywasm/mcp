package mcp_test

import (
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
)

func TestTokenAuthorizer_ValidToken_Passes(t *testing.T) {
	auth := mcp.NewTokenAuthorizer("secret-key")
	id, err := auth.Authorize("secret-key")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "user" {
		t.Fatalf("expected userID 'user', got %q", id)
	}
}

func TestTokenAuthorizer_InvalidToken_Rejected(t *testing.T) {
	auth := mcp.NewTokenAuthorizer("secret-key")
	_, err := auth.Authorize("wrong-key")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestTokenAuthorizer_EmptyToken_Rejected(t *testing.T) {
	auth := mcp.NewTokenAuthorizer("secret-key")
	_, err := auth.Authorize("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestHandleMessage_TokenAuth_ValidToken_Returns_Result(t *testing.T) {
	auth := mcp.NewTokenAuthorizer("secret-key")
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: auth}, nil)
	var ctx context.Context
	ctx.Set(mcp.CtxKeyAuthToken, "secret-key")

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"ping"}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response")
	}
	respStr := encodeResponse(resp)
	if contains(respStr, "error") {
		t.Fatalf("expected success response, got %s", respStr)
	}
}

func TestHandleMessage_TokenAuth_InvalidToken_Returns_Unauthorized(t *testing.T) {
	auth := mcp.NewTokenAuthorizer("secret-key")
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: auth}, nil)
	var ctx context.Context
	ctx.Set(mcp.CtxKeyAuthToken, "wrong-key")

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"ping"}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response")
	}
	respStr := encodeResponse(resp)
	// The error message is currently "Unauthorized" with code -32001
	if !contains(respStr, "Unauthorized") {
		t.Fatalf("expected Unauthorized error, got %s", respStr)
	}
}

func TestHandleMessage_TokenAuth_NoToken_Returns_Unauthorized(t *testing.T) {
	auth := mcp.NewTokenAuthorizer("secret-key")
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: auth}, nil)
	var ctx context.Context

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"ping"}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response")
	}
	respStr := encodeResponse(resp)
	if !contains(respStr, "Unauthorized") {
		t.Fatalf("expected Unauthorized error, got %s", respStr)
	}
}

func TestHandleMessage_EmptyUserID_Rejected(t *testing.T) {
	auth := &emptyUserAuth{}
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: auth}, nil)
	var ctx context.Context

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"ping"}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response")
	}
	respStr := encodeResponse(resp)
	if !contains(respStr, "empty user identity") {
		t.Fatalf("expected empty user identity error, got %s", respStr)
	}
}

func TestOpenAuthorizer_ReturnsGuest(t *testing.T) {
	auth := mcp.OpenAuthorizer()
	id, err := auth.Authorize("any-token")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "guest" {
		t.Fatalf("expected userID 'guest', got %q", id)
	}
}
