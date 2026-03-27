package mcp_test

import (
	"strings"
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
	"github.com/tinywasm/mcp"
)

func TestAddTool_Valid(t *testing.T) {
	srv := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0"}, nil)
	err := srv.AddTool(mcp.Tool{
		Name:     "search",
		Resource: "items",
		Action:   'r',
		Run: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestHandleMessage_Ping(t *testing.T) {
	srv := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0"}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"ping"}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response")
	}
	var b []byte
	if f, ok := resp.(fmt.Fielder); ok {
		json.Encode(f, &b)
	}
	if string(b) == "" {
		t.Fatal("expected non-empty response body")
	}
	if !strings.Contains(string(b), "result") {
		t.Fatalf("expected result in response, got %s", string(b))
	}
}

func TestHandleMessage_Initialize(t *testing.T) {
	srv := mcp.NewServer(mcp.Config{Name: "test-server", Version: "2.0.0"}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"init-1","method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test-client","version":"1.0"}}}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response")
	}
	var b []byte
	if f, ok := resp.(fmt.Fielder); ok {
		json.Encode(f, &b)
	}

	s := string(b)
	if !strings.Contains(s, "protocol_version") || !strings.Contains(s, "test-server") {
		t.Fatalf("unexpected initialize response: %s", s)
	}
}

func TestContextWithSessionID(t *testing.T) {
	var ctx context.Context
	mcp.ContextWithSessionID(&ctx, "1718000000000000000")
	id := mcp.SessionIDFromContext(&ctx)
	if id != "1718000000000000000" {
		t.Fatalf("got %q", id)
	}
}

func TestTextRoundTrip(t *testing.T) {
	r := mcp.Text("hello world")
	text, err := mcp.GetText(r)
	if err != nil || text != "hello world" {
		t.Fatalf("got %q %v", text, err)
	}
}

type mockAuth struct{ id string }

func (m *mockAuth) Authorize(token string) (string, error) {
	if token == "valid" {
		return m.id, nil
	}
	return "", fmt.Err("mcp", "auth", "unauthorized")
}

func TestAuthorizer_Satisfies(t *testing.T) {
	var _ mcp.Authorizer = &mockAuth{} // compile-time check
}
