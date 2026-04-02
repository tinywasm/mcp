package mcp_test

import (
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
)

func TestAddTool_WithSSE_PublishesListChanged(t *testing.T) {
	sse := &mockSSE{}
	srv, _ := mcp.NewServer(mcp.Config{
		Name:    "test",
		Version: "1.0.0",
		Auth:    mcp.OpenAuthorizer(),
		SSE:     sse,
	}, nil)

	err := srv.AddTool(mcp.Tool{
		Name:     "new-tool",
		Resource: "res",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("AddTool error: %v", err)
	}

	if sse.callCount != 1 {
		t.Fatalf("expected 1 SSE publish, got %d", sse.callCount)
	}
	if !contains(string(sse.lastData), "notifications/tools/list_changed") {
		t.Fatalf("expected list_changed notification, got %s", string(sse.lastData))
	}
}

func TestAddTool_WithoutSSE_NoPanic(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{
		Name:    "test",
		Version: "1.0.0",
		Auth:    mcp.OpenAuthorizer(),
		SSE:     nil,
	}, nil)

	err := srv.AddTool(mcp.Tool{
		Name:     "new-tool",
		Resource: "res",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})
	if err != nil {
		t.Fatalf("AddTool error: %v", err)
	}
}

func TestHandleMessage_WithoutSSE_NoPublish(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{
		Name:    "test",
		Version: "1.0.0",
		Auth:    mcp.OpenAuthorizer(),
		SSE:     nil,
	}, nil)

	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp != nil {
		t.Fatal("expected nil response for notification")
	}
}

func TestSSE_PublishData_ContainsMethod(t *testing.T) {
	sse := &mockSSE{}
	srv, _ := mcp.NewServer(mcp.Config{
		Name:    "test",
		Version: "1.0.0",
		Auth:    mcp.OpenAuthorizer(),
		SSE:     sse,
	}, nil)

	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","method":"my/custom/notification"}`)
	srv.HandleMessage(&ctx, req)

	if sse.callCount != 1 {
		t.Fatalf("expected 1 SSE publish, got %d", sse.callCount)
	}
	if !contains(string(sse.lastData), "my/custom/notification") {
		t.Fatalf("expected custom notification, got %s", string(sse.lastData))
	}
}
