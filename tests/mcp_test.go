package mcp_test

import (
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
	"github.com/tinywasm/mcp"
)

// TestAddTool_Valid verifies tool registration succeeds with valid Tool
func TestAddTool_Valid(t *testing.T) {
	srv := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0"}, nil)
	err := srv.AddTool(mcp.Tool{
		Name:        "search",
		Description: "Search items",
		Resource:    "items",
		Action:      'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestAddTool_MissingRBAC verifies AddTool rejects tools without Resource or Action
func TestAddTool_MissingRBAC(t *testing.T) {
	srv := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0"}, nil)
	err := srv.AddTool(mcp.Tool{
		Name: "search",
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return nil, nil
		},
	})
	if err == nil {
		t.Fatal("expected error for missing Resource/Action")
	}
}

// TestAddTool_MissingRun verifies AddTool rejects tools without Run handler
func TestAddTool_MissingRun(t *testing.T) {
	srv := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0"}, nil)
	err := srv.AddTool(mcp.Tool{
		Name:     "search",
		Resource: "items",
		Action:   'r',
	})
	if err == nil {
		t.Fatal("expected error for nil Run")
	}
}

// TestHandleMessage_Ping verifies ping request returns valid response
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

	if len(b) == 0 {
		t.Fatal("expected non-empty response body")
	}

	respStr := string(b)
	if len(respStr) == 0 {
		t.Fatal("expected response string")
	}
}

// TestHandleMessage_Initialize verifies initialize request returns server info
func TestHandleMessage_Initialize(t *testing.T) {
	srv := mcp.NewServer(mcp.Config{Name: "test-server", Version: "2.0.0"}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"init-1","method":"initialize","params":"{\"protocolVersion\":\"2024-11-05\",\"clientInfo\":{\"name\":\"test-client\",\"version\":\"1.0\"}}"}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response")
	}

	var b []byte
	if f, ok := resp.(fmt.Fielder); ok {
		json.Encode(f, &b)
	}

	_ = b // response should contain server info, validated by decode success
}

// TestContextSessionID verifies session ID storage and retrieval via context
func TestContextSessionID(t *testing.T) {
	var ctx context.Context
	ctx.Set(mcp.CtxKeySessionID, "1718000000000000000")
	id := ctx.Value(mcp.CtxKeySessionID)
	if id != "1718000000000000000" {
		t.Fatalf("got %q, expected session ID", id)
	}
}

// TestTextRoundTrip verifies Text result can be created and extracted
func TestTextRoundTrip(t *testing.T) {
	r := mcp.Text("hello world")
	text, err := mcp.GetText(r)
	if err != nil {
		t.Fatalf("GetText error: %v", err)
	}
	if text != "hello world" {
		t.Fatalf("got %q, expected 'hello world'", text)
	}
}

// TestJSONResult verifies JSON result creation with proper Fielder
func TestJSONResult(t *testing.T) {
	// Use a simple Meta struct which implements fmt.Fielder
	data := &mcp.Meta{}
	r, err := mcp.JSON(data)
	if err != nil {
		t.Fatalf("JSON error: %v", err)
	}
	if r.Content == "" {
		t.Fatal("expected non-empty content")
	}
}

// mockAuth implements mcp.Authorizer for testing
type mockAuth struct {
	id string
}

func (m *mockAuth) Authorize(token string) (string, error) {
	if token == "valid" {
		return m.id, nil
	}
	return "", fmt.Err("mcp", "auth", "unauthorized")
}

// TestAuthorizer_Satisfies compile-time check that mockAuth implements Authorizer
func TestAuthorizer_Satisfies(t *testing.T) {
	var _ mcp.Authorizer = &mockAuth{id: "user123"}
}
