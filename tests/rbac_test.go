package mcp_test

import (
	"testing"
	"sync"

	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/mcp"
)

func TestHandleToolCall_Can_ChecksResource(t *testing.T) {
	auth := &rbacAuth{id: "user"}
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: auth}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "tool1",
		Resource: "secrets",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"tool1","arguments":"{}"}}`)
	srv.HandleMessage(&ctx, req)

	if auth.lastResource != "secrets" {
		t.Fatalf("expected lastResource 'secrets', got %q", auth.lastResource)
	}
}

func TestHandleToolCall_Can_ChecksAction(t *testing.T) {
	auth := &rbacAuth{id: "user"}
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: auth}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "tool1",
		Resource: "res",
		Action:   'u',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"tool1","arguments":"{}"}}`)
	srv.HandleMessage(&ctx, req)

	if auth.lastAction != 'u' {
		t.Fatalf("expected lastAction 'u', got %c", auth.lastAction)
	}
}

func TestHandleToolCall_ExecuteNeverCalledIfCanFalse(t *testing.T) {
	called := false
	auth := &rbacAuth{id: "user", denyResource: "secrets", denyAction: 'r'}
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: auth}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "tool1",
		Resource: "secrets",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			called = true
			return mcp.Text("ok"), nil
		},
	})
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"tool1","arguments":"{}"}}`)
	srv.HandleMessage(&ctx, req)

	if called {
		t.Fatal("Execute should not have been called")
	}
}

func TestHandleToolCall_ToolNotFound(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"nonexistent","arguments":"{}"}}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, "INVALID_PARAMS") && !contains(respStr, "-32602") {
		t.Fatalf("expected INVALID_PARAMS error, got %s", respStr)
	}
}

func TestHandleToolCall_ExecuteReturnsError_IsErrorTrue(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "fail",
		Resource: "res",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return nil, fmt.Err("mcp", "test", "failed")
		},
	})
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"fail","arguments":"{}"}}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, "isError") {
		t.Fatalf("expected response with isError, got %s", respStr)
	}
}

type validateFielder struct {
	Foo string
	lastAction byte
}

func (v *validateFielder) SafeFields() []fmt.Field {
	return []fmt.Field{
		{Name: "Foo", Type: fmt.FieldText},
	}
}

func (v *validateFielder) Pointers() []any {
	return []any{&v.Foo}
}

func (v *validateFielder) Schema() []fmt.Field {
	return v.SafeFields()
}

func (v *validateFielder) Validate(action byte) error {
	v.lastAction = action
	return nil
}

func TestHandleToolCall_Bind_UsesToolAction(t *testing.T) {
	var target validateFielder
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "tool1",
		Resource: "res",
		Action:   'u',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			req.Bind(&target)
			return mcp.Text("ok"), nil
		},
	})
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"tool1","arguments":"{\"Foo\":\"bar\"}"}}`)
	srv.HandleMessage(&ctx, req)

	if target.lastAction != 'u' {
		t.Fatalf("expected Bind to use action 'u', got %c", target.lastAction)
	}
}

func TestHandleToolCall_DuplicateToolName_Overwrites(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "tool1",
		Resource: "res",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("first"), nil
		},
	})
	srv.AddTool(mcp.Tool{
		Name:     "tool1",
		Resource: "res",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("second"), nil
		},
	})
	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"tool1","arguments":"{}"}}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, "second") {
		t.Fatalf("expected 'second' to overwrite 'first', got %s", respStr)
	}
}

func TestConcurrent_AddToolAndCallTool(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	var wg sync.WaitGroup
	count := 20

	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(n int) {
			defer wg.Done()
			srv.AddTool(mcp.Tool{
				Name:     "t" + string(rune('0'+n)),
				Resource: "res",
				Action:   'r',
				Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
					return mcp.Text("ok"), nil
				},
			})
		}(i)
	}

	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(n int) {
			defer wg.Done()
			var ctx context.Context
			req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"t` + string(rune('0'+n)) + `","arguments":"{}"}}`)
			srv.HandleMessage(&ctx, req)
		}(i)
	}
	wg.Wait()
}

type mockProvider struct {
	tools []mcp.Tool
	err   error
}

func (p *mockProvider) Tools() []mcp.Tool {
	return p.tools
}

func TestNewServer_ProviderInjectsTools(t *testing.T) {
	p := &mockProvider{
		tools: []mcp.Tool{
			{Name: "t1", Resource: "r", Action: 'r', Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) { return mcp.Text("1"), nil }},
			{Name: "t2", Resource: "r", Action: 'r', Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) { return mcp.Text("2"), nil }},
		},
	}
	srv, err := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, []mcp.ToolProvider{p})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var ctx context.Context
	req1 := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"t1","arguments":"{}"}}`)
	resp1 := srv.HandleMessage(&ctx, req1)
	if !contains(encodeResponse(resp1), "1") {
		t.Fatal("t1 not registered")
	}
}

func TestNewServer_ProviderWithInvalidTool_ReturnsError(t *testing.T) {
	p := &mockProvider{
		tools: []mcp.Tool{
			{Name: "", Resource: "r", Action: 'r'}, // Invalid
		},
	}
	_, err := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, []mcp.ToolProvider{p})
	if err == nil {
		t.Fatal("expected error for invalid tool from provider")
	}
}
