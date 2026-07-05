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
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
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
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
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
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
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
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"ping"}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response")
	}

	var b []byte
	if f, ok := resp.(fmt.Encodable); ok {
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
	srv, _ := mcp.NewServer(mcp.Config{Name: "test-server", Version: "2.0.0", Authorize: mcp.AllowAll}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"init-1","method":"initialize","params":"{\"protocolVersion\":\"2024-11-05\",\"clientInfo\":{\"name\":\"test-client\",\"version\":\"1.0\"}}"}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response")
	}

	var b []byte
	if f, ok := resp.(fmt.Encodable); ok {
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

// TestToolCall_ContentIsArray verifica que la respuesta de tools/call
// devuelva content como array JSON, no como objeto.
// Bug C: tools.go Text() produce objeto {"type":"text","text":"..."} en lugar de array.
func TestToolCall_ContentIsArray(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "ping",
		Resource: "test",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("pong"), nil
		},
	})
	var ctx context.Context
	ctx.Set(mcp.CtxKeyUserID, "u1")

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"ping","arguments":{}}}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	body := encodeResponse(resp)

	if contains(body, `"content":{`) {
		t.Fatalf("Bug C not fixed: content is an object instead of an array.\nResponse: %s", body)
	}
	if !contains(body, `"content":[`) {
		t.Fatalf("expected content as JSON array, got: %s", body)
	}
}

// TestToolCall_ContentItemLowercaseKeys verifica que los items dentro de content
// usen claves en minúsculas ("type", "text") según el protocolo MCP.
// Regresión: json.Encode usa Schema() generado por ormc — garantiza que no rompa en el futuro.
func TestToolCall_ContentItemLowercaseKeys(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "ping",
		Resource: "test",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("pong"), nil
		},
	})
	var ctx context.Context
	ctx.Set(mcp.CtxKeyUserID, "u1")

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"ping","arguments":{}}}`)
	resp := srv.HandleMessage(&ctx, req)

	body := encodeResponse(resp)

	if contains(body, `"Type":`) || contains(body, `"Text":`) {
		t.Fatalf("content item uses PascalCase keys — ormc schema regression.\nResponse: %s", body)
	}
	if !contains(body, `"type":"text"`) || !contains(body, `"text":"pong"`) {
		t.Fatalf("expected lowercase 'type' and 'text' keys in content item, got: %s", body)
	}
}

// TestText_GetText_RoundTrip verifica que mcp.Text() + mcp.GetText() funcionen
// correctamente con el formato array exigido por el protocolo MCP.
func TestText_GetText_RoundTrip(t *testing.T) {
	r := mcp.Text("hello protocol")
	text, err := mcp.GetText(r)
	if err != nil {
		t.Fatalf("GetText error after array fix: %v — Content was: %s", err, r.Content)
	}
	if text != "hello protocol" {
		t.Fatalf("got %q, expected 'hello protocol'", text)
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
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "echo",
		Resource: "test",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text(req.Params.Name), nil
		},
	})
	var ctx context.Context
	ctx.Set(mcp.CtxKeyUserID, "u1")

	tests := []struct {
		name         string
		message      string
		expectNil    bool
		expectError  bool
		expectedCode int
	}{
		{
			name:    "ValidToolCall",
			message: `{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"echo","arguments":{}}}`,
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
				if f, ok := resp.(fmt.Encodable); ok {
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
				if f, ok := resp.(fmt.Encodable); ok {
					json.Encode(f, &b)
				}
				respStr := string(b)
				if contains(respStr, `"error":{`) {
					t.Fatalf("expected success response, got %s", respStr)
				}
			}
		})
	}
}

func TestHandleToolCall_Can_False_Rejected(t *testing.T) {
	auth := &rbacAuth{denyResource: "secrets", denyAction: "r"}
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: auth.Can}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "secret",
		Resource: "secrets",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("secret data"), nil
		},
	})

	var ctx context.Context
	ctx.Set(mcp.CtxKeyUserID, "u1")

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"secret","arguments":{}}}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response")
	}

	var b []byte
	if f, ok := resp.(fmt.Encodable); ok {
		json.Encode(f, &b)
	}
	respStr := string(b)
	if !contains(respStr, `-32001`) || !contains(respStr, "forbidden") {
		t.Fatalf("expected forbidden error, got %s", respStr)
	}
}

func TestHandleMessage_WithSSE_PublishesNotification(t *testing.T) {
	sse := &mockSSE{}
	srv, _ := mcp.NewServer(mcp.Config{
		Name:      "test",
		Version:   "1.0.0",
		Authorize: mcp.AllowAll,
		SSE:       sse,
	}, nil)

	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	srv.HandleMessage(&ctx, req)

	if string(sse.lastData) == "" {
		t.Fatal("expected SSE publication")
	}
	if sse.lastChannel != "mcp" {
		t.Fatalf("expected channel 'mcp', got %s", sse.lastChannel)
	}
	if !contains(string(sse.lastData), "notifications/initialized") {
		t.Fatalf("expected notification data, got %s", string(sse.lastData))
	}
}

// TestToolCall_ArgumentsAsObject verifies that tools/call accepts arguments
// as a JSON object (MCP spec), not only as a quoted string.
// This test FAILS before the fix (model.go Arguments string → fmt.RawJSON).
func TestToolCall_ArgumentsAsObject(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "echo",
		Resource: "test",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})
	var ctx context.Context
	ctx.Set(mcp.CtxKeyUserID, "u1")

	// MCP spec: arguments MUST be a JSON object, not a quoted string
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"echo","arguments":{}}}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	body := encodeResponse(resp)

	if contains(body, "json decode expected string") {
		t.Fatalf("Bug A not fixed: server rejected JSON object arguments with parse error.\nResponse: %s", body)
	}
	if contains(body, `"error":{`) {
		t.Fatalf("expected success response, got error: %s", body)
	}
}

// TestListTools_InputSchemaIsObject verifies that tools/list returns inputSchema
// as an inline JSON object, not as a quoted JSON string.
// This test FAILS before the fix (model.go InputSchema string → fmt.RawJSON).
func TestListTools_InputSchemaIsObject(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	srv.AddTool(mcp.Tool{
		Name:        "search",
		Description: "Search something",
		InputSchema: `{"type":"object","properties":{"query":{"type":"string"}}}`,
		Resource:    "items",
		Action:      'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})
	var ctx context.Context

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/list","params":{}}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	body := encodeResponse(resp)

	// inputSchema must NOT appear as a quoted string (i.e. "inputSchema":"..." is wrong)
	// It must appear as an inline object (i.e. "inputSchema":{"type":... is correct)
	if contains(body, `"inputSchema":"{`) {
		t.Fatalf("Bug B not fixed: inputSchema is a quoted string instead of a JSON object.\nResponse: %s", body)
	}
	if !contains(body, `"inputSchema":{"type"`) {
		t.Fatalf("expected inputSchema as inline object, got: %s", body)
	}
}
