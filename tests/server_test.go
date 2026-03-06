package mcp_test

import (
	"context"
	"testing"
	"time"

	"github.com/tinywasm/mcp"
)

func TestAddTool(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0")
	tool := mcp.NewTool("test-tool", "A test tool", mcp.ToolInputSchema{Type: "object"})
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("success"), nil
	}
	server.AddTool(tool, handler)

	tools := server.ListTools(context.Background())
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	if tools[0].Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got '%s'", tools[0].Name)
	}
}

func TestCallTool(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0")
	tool := mcp.NewTool("echo", "Echoes input", mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"message": map[string]any{"type": "string"},
		},
		Required: []string{"message"},
	})
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		msg, ok := args["message"].(string)
		if !ok {
			return mcp.NewToolResultError("missing message"), nil
		}
		return mcp.NewToolResultText(msg), nil
	}

	server.AddTool(tool, handler)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "echo"
	req.Params.Arguments = map[string]any{"message": "hello world"}

	toolEntry := server.GetTool("echo")
	if toolEntry == nil {
		t.Fatalf("Tool not found")
	}

	result, err := toolEntry.Handler(ctx, req)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if result.IsError {
		t.Errorf("Expected success, got error result")
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("Expected TextContent")
	}

	if textContent.Text != "hello world" {
		t.Errorf("Expected 'hello world', got '%s'", textContent.Text)
	}
}

func TestCallTool_Error(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0")
	tool := mcp.NewTool("fail-tool", "fails", mcp.ToolInputSchema{Type: "object"})
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("expected failure"), nil
	}

	server.AddTool(tool, handler)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := mcp.CallToolRequest{}
	req.Params.Name = "fail-tool"

	toolEntry := server.GetTool("fail-tool")
	if toolEntry == nil {
		t.Fatalf("Tool not found")
	}

	result, err := toolEntry.Handler(ctx, req)
	if err != nil {
		t.Fatalf("Handler shouldn't return Go error here: %v", err)
	}
	if !result.IsError {
		t.Fatalf("Expected error result")
	}
}
