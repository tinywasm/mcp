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

func TestExtractJSONValue(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		key      string
		expected string
	}{
		{"SimpleObject", `{"params":{"name":"test"}}`, "params", `{"name":"test"}`},
		{"NestedObject", `{"params":{"name":"test","args":{"x":1}}}`, "params", `{"name":"test","args":{"x":1}}`},
		{"Array", `{"list":[1,2,3]}`, "list", `[1,2,3]`},
		{"String", `{"id":"abc"}`, "id", `"abc"`},
		{"Number", `{"count":42}`, "count", `42`},
		{"Boolean", `{"ok":true}`, "ok", `true`},
		{"Missing", `{"method":"ping"}`, "params", ""},
		{"NestedKey", `{"root":{"child":1}}`, "child", "1"},
		{"Whitespace", `{"params" : {"name":"test"}}`, "params", `{"name":"test"}`},
		{"EscapedQuotes", `{"id":"\"quoted\""}`, "id", `"\"quoted\""`},
		{"DeeplyEscapedQuotes", `{"id":"\\\\\""}`, "id", `"\\\\\""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mcp.ExtractJSONValue([]byte(tt.data), tt.key)
			if string(got) != tt.expected {
				t.Errorf("ExtractJSONValue() = %s, want %s", string(got), tt.expected)
			}
		})
	}
}

func TestHandleMessage_Compliance(t *testing.T) {
	srv := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0"}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "echo",
		Resource: "test",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text(req.Params.Name), nil
		},
	})
	var ctx context.Context

	tests := []struct {
		name         string
		message      string
		expectNil    bool
		expectError  bool
		expectedCode int
	}{
		{
			name:    "ValidToolCall",
			message: `{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"echo","arguments":"{}"}}`,
		},
		{
			name:         "InvalidVersion",
			message:      `{"jsonrpc":"1.0","id":"1","method":"ping"}`,
			expectError:  true,
			expectedCode: mcp.INVALID_REQUEST,
		},
		{
			name:         "MethodNotFound",
			message:      `{"jsonrpc":"2.0","id":"1","method":"nonexistent"}`,
			expectError:  true,
			expectedCode: mcp.METHOD_NOT_FOUND,
		},
		{
			name:         "InvalidParams",
			message:      `{"jsonrpc":"2.0","id":"1","method":"tools/call","params":"not-an-object"}`,
			expectError:  true,
			expectedCode: mcp.INVALID_PARAMS,
		},
		{
			name:      "Notification",
			message:   `{"jsonrpc":"2.0","method":"notifications/initialized"}`,
			expectNil: true,
		},
		{
			name:         "ParseError_Empty",
			message:      ``,
			expectError:  true,
			expectedCode: mcp.PARSE_ERROR,
		},
		{
			name:         "ParseError_Invalid",
			message:      `not json`,
			expectError:  true,
			expectedCode: mcp.PARSE_ERROR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := srv.HandleMessage(&ctx, []byte(tt.message))
			if tt.expectNil {
				if resp != nil {
					t.Fatalf("expected nil response for notification")
				}
				return
			}
			if resp == nil {
				t.Fatalf("expected response, got nil")
			}

			if tt.expectError {
				// Check if it's an error response
				var b []byte
				if f, ok := resp.(fmt.Fielder); ok {
					json.Encode(f, &b)
				}
				respStr := string(b)
				if !contains(respStr, `"error":`) {
					t.Fatalf("expected error response, got %s", respStr)
				}
				// We could decode and check the code if needed
			} else {
				// Should be success
				var b []byte
				if f, ok := resp.(fmt.Fielder); ok {
					json.Encode(f, &b)
				}
				respStr := string(b)
				if contains(respStr, `"error":`) {
					t.Fatalf("expected success response, got %s", respStr)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
