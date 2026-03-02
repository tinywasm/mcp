package mcp

import (
	"context"
	"errors"
	"testing"
)

type mockProvider struct{}

func (m *mockProvider) GetMCPToolsMetadata() []ToolMetadata {
	return []ToolMetadata{
		{
			Name:        "success_tool",
			Description: "A tool that succeeds",
			Parameters: []ParameterMetadata{
				{
					Name:        "param1",
					Description: "A parameter",
					Type:        "string",
					Required:    true,
				},
			},
			Execute: func(ctx context.Context, args map[string]any) (any, error) {
				return "success with " + args["param1"].(string), nil
			},
		},
		{
			Name:        "error_tool",
			Description: "A tool that fails",
			Execute: func(ctx context.Context, args map[string]any) (any, error) {
				return nil, errors.New("an error occurred")
			},
		},
		{
			Name:        "nil_exec_tool",
			Description: "A tool with no executor",
			Execute:     nil,
		},
	}
}

func TestRegisterProvider(t *testing.T) {
	srv := NewMCPServer("test_server", "1.0.0")
	provider := &mockProvider{}

	srv.RegisterProvider(provider)

	// Verify all tools are registered
	tools := srv.ListTools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools to be registered, got %d", len(tools))
	}

	for _, name := range []string{"success_tool", "error_tool", "nil_exec_tool"} {
		if tools[name] == nil {
			t.Errorf("tool %s not found in registered tools", name)
		}
	}

	// Test success_tool
	req := CallToolRequest{
		Params: CallToolParams{
			Name: "success_tool",
			Arguments: map[string]any{
				"param1": "test_value",
			},
		},
	}
	ctx := context.Background()
	result, err := srv.GetTool("success_tool").Handler(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error result")
	}

	// Test error_tool
	reqError := CallToolRequest{
		Params: CallToolParams{
			Name: "error_tool",
		},
	}
	resultError, err := srv.GetTool("error_tool").Handler(ctx, reqError)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resultError.IsError {
		t.Fatalf("expected tool error result")
	}

	// Test nil_exec_tool
	reqNilExec := CallToolRequest{
		Params: CallToolParams{
			Name: "nil_exec_tool",
		},
	}
	resultNilExec, err := srv.GetTool("nil_exec_tool").Handler(ctx, reqNilExec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resultNilExec.IsError {
		t.Fatalf("expected tool error result for nil executor")
	}
}
