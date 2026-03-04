package mcp_test

import (
	"context"
	"errors"
	"testing"

	"github.com/tinywasm/mcp"
)

type mockProvider struct {
	tools []mcp.ToolMetadata
}

func (m *mockProvider) GetMCPToolsMetadata() []mcp.ToolMetadata {
	return m.tools
}

func TestRegisterProvider(t *testing.T) {
	provider := &mockProvider{
		tools: []mcp.ToolMetadata{
			{
				Name:        "tool1",
				Description: "First tool",
				Execute: func(ctx context.Context, args map[string]any) (any, error) {
					return "result1", nil
				},
			},
			{
				Name:        "tool2",
				Description: "Second tool",
				Execute: func(ctx context.Context, args map[string]any) (any, error) {
					return "result2", nil
				},
			},
		},
	}

	server := mcp.NewMCPServer("test", "1.0")
	server.RegisterProvider(provider)

	tools := server.ListTools()
	if len(tools) != 2 {
		t.Fatalf("Expected 2 tools registered, got %d", len(tools))
	}

	if _, ok := tools["tool1"]; !ok {
		t.Errorf("Missing tool1")
	}
	if _, ok := tools["tool2"]; !ok {
		t.Errorf("Missing tool2")
	}
}

func TestRegisterProvider_Execute(t *testing.T) {
	provider := &mockProvider{
		tools: []mcp.ToolMetadata{
			{
				Name: "echo",
				Parameters: []mcp.ParameterMetadata{
					{Name: "msg", Type: "string"},
				},
				Execute: func(ctx context.Context, args map[string]any) (any, error) {
					return args["msg"], nil
				},
			},
		},
	}

	server := mcp.NewMCPServer("test", "1.0")
	server.RegisterProvider(provider)

	tools := server.ListTools()
	tool, ok := tools["echo"]
	if !ok {
		t.Fatalf("Tool not found")
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"msg": "hello context"}

	res, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("Result indicates error")
	}

	if res.StructuredContent != "hello context" {
		t.Errorf("Expected 'hello context', got '%v'", res.StructuredContent)
	}
}

func TestRegisterProvider_ExecuteError(t *testing.T) {
	provider := &mockProvider{
		tools: []mcp.ToolMetadata{
			{
				Name: "fail",
				Execute: func(ctx context.Context, args map[string]any) (any, error) {
					return nil, errors.New("expected failure")
				},
			},
		},
	}

	server := mcp.NewMCPServer("test", "1.0")
	server.RegisterProvider(provider)

	tools := server.ListTools()
	tool, ok := tools["fail"]
	if !ok {
		t.Fatalf("Tool not found")
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	res, err := tool.Handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Handler returned hard error: %v", err)
	}

	if !res.IsError {
		t.Errorf("Expected error result, got success")
	}

	if len(res.Content) == 0 {
		t.Fatalf("Expected error content")
	}

	text, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected text content")
	}

	if text.Text != "expected failure" {
		t.Errorf("Expected error message 'expected failure', got '%s'", text.Text)
	}
}
