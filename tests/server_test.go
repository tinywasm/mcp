package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tinywasm/mcp"
)

func TestServer_Initialize(t *testing.T) {
	server := mcp.NewMCPServer("test-init", "2.0", mcp.WithToolCapabilities(true), mcp.WithInstructions("test instructions"))

	// We can't call handleInitialize directly from mcp_test.
	// We will test it using an HTTP mock request on the StreamableHTTPServer.
	httpServer := mcp.NewStreamableHTTPServer(server, mcp.WithEndpointPath("/mcp"), mcp.WithStateLess(true))

	initReq := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId("init-1"),
		Request: mcp.Request{Method: string(mcp.MethodInitialize)},
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo:      mcp.Implementation{Name: "test-client", Version: "1.0"},
		},
	}

	reqBytes, _ := json.Marshal(initReq)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	httpServer.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res mcp.JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	resBytes, _ := json.Marshal(res.Result)
	var initRes mcp.InitializeResult
	if err := json.Unmarshal(resBytes, &initRes); err != nil {
		t.Fatalf("Failed to decode init result: %v", err)
	}

	if initRes.ProtocolVersion != mcp.LATEST_PROTOCOL_VERSION {
		t.Errorf("Expected protocol version %s, got %s", mcp.LATEST_PROTOCOL_VERSION, initRes.ProtocolVersion)
	}
	if initRes.ServerInfo.Name != "test-init" {
		t.Errorf("Expected server name 'test-init', got '%s'", initRes.ServerInfo.Name)
	}
	if initRes.ServerInfo.Version != "2.0" {
		t.Errorf("Expected server version '2.0', got '%s'", initRes.ServerInfo.Version)
	}
	if initRes.Capabilities.Tools == nil || !initRes.Capabilities.Tools.ListChanged {
		t.Errorf("Expected tool capabilities to be true")
	}
	if initRes.Instructions != "test instructions" {
		t.Errorf("Expected instructions 'test instructions', got '%s'", initRes.Instructions)
	}
}

func TestServer_Ping(t *testing.T) {
	server := mcp.NewMCPServer("test-ping", "1.0")
	httpServer := mcp.NewStreamableHTTPServer(server, mcp.WithEndpointPath("/mcp"), mcp.WithStateLess(true))

	pingReq := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(123),
		Request: mcp.Request{Method: string(mcp.MethodPing)},
	}

	reqBytes, _ := json.Marshal(pingReq)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	httpServer.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected HTTP 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var res mcp.JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if res.Error != nil {
		t.Fatalf("Expected success, got error: %v", res.Error)
	}

	// Result should be empty map or EmptyResult
	resBytes, _ := json.Marshal(res.Result)
	if string(resBytes) != "{}" {
		t.Errorf("Expected empty object {}, got %s", string(resBytes))
	}
}

func TestAddTool(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0")
	tool := mcp.Tool{
		Name:        "test-tool",
		Description: "A test tool",
	}
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("success"), nil
	}
	server.AddToolFromUser(tool, handler)

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
	tool := mcp.Tool{
		Name:        "echo",
		Description: "Echoes input",
		Parameters: []mcp.Parameter{
			{
				Name:        "message",
				Description: "Input message",
				Required:    true,
				Type:        "string",
			},
		},
	}
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		msg, ok := args["message"].(string)
		if !ok {
			return mcp.NewToolResultError("missing message"), nil
		}
		return mcp.NewToolResultText(msg), nil
	}

	server.AddToolFromUser(tool, handler)

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
	tool := mcp.Tool{
		Name:        "fail-tool",
		Description: "fails",
	}
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("expected failure"), nil
	}

	server.AddToolFromUser(tool, handler)

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

func TestServer_ListTools_Pagination(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0", mcp.WithPaginationLimit(2))

	// Add 5 tools
	for i := 0; i < 5; i++ {
		toolName := string(rune('a' + i)) // a, b, c, d, e
		server.AddTool(mcp.ProtocolTool{Name: toolName, Description: toolName}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return mcp.NewToolResultText("ok"), nil
		})
	}

	// Because ListTools logic handles HTTP transport details like request IDs,
	// we will use StreamableHTTPServer.
	httpServer := mcp.NewStreamableHTTPServer(server, mcp.WithEndpointPath("/mcp"), mcp.WithStateLess(true))

	// Page 1
	listReq1 := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(1),
		Request: mcp.Request{Method: string(mcp.MethodToolsList)},
	}

	reqBytes, _ := json.Marshal(listReq1)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	httpServer.ServeHTTP(w, req)

	var res1 mcp.JSONRPCResponse
	json.Unmarshal(w.Body.Bytes(), &res1)

	resBytes1, _ := json.Marshal(res1.Result)
	var listRes1 mcp.ListToolsResult
	json.Unmarshal(resBytes1, &listRes1)

	if len(listRes1.Tools) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(listRes1.Tools))
	}
	if listRes1.Tools[0].Name != "a" || listRes1.Tools[1].Name != "b" {
		t.Errorf("Expected tools 'a', 'b', got %s, %s", listRes1.Tools[0].Name, listRes1.Tools[1].Name)
	}
	if listRes1.NextCursor == "" {
		t.Fatalf("Expected next cursor")
	}

	// Page 2
	listReq2 := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(2),
		Request: mcp.Request{Method: string(mcp.MethodToolsList)},
	}
	listReq2.Params = mcp.PaginatedParams{Cursor: listRes1.NextCursor}

	reqBytes2, _ := json.Marshal(listReq2)
	req2 := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqBytes2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	httpServer.ServeHTTP(w2, req2)

	var res2 mcp.JSONRPCResponse
	json.Unmarshal(w2.Body.Bytes(), &res2)

	resBytes2, _ := json.Marshal(res2.Result)
	var listRes2 mcp.ListToolsResult
	json.Unmarshal(resBytes2, &listRes2)

	if len(listRes2.Tools) != 2 {
		t.Fatalf("Expected 2 tools on page 2, got %d", len(listRes2.Tools))
	}
	if listRes2.Tools[0].Name != "c" || listRes2.Tools[1].Name != "d" {
		t.Errorf("Expected tools 'c', 'd', got %s, %s", listRes2.Tools[0].Name, listRes2.Tools[1].Name)
	}
	if listRes2.NextCursor == "" {
		t.Fatalf("Expected next cursor")
	}

	// Page 3
	listReq3 := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(3),
		Request: mcp.Request{Method: string(mcp.MethodToolsList)},
	}
	listReq3.Params = mcp.PaginatedParams{Cursor: listRes2.NextCursor}

	reqBytes3, _ := json.Marshal(listReq3)
	req3 := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqBytes3))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()

	httpServer.ServeHTTP(w3, req3)

	var res3 mcp.JSONRPCResponse
	json.Unmarshal(w3.Body.Bytes(), &res3)

	resBytes3, _ := json.Marshal(res3.Result)
	var listRes3 mcp.ListToolsResult
	json.Unmarshal(resBytes3, &listRes3)

	if len(listRes3.Tools) != 1 {
		t.Fatalf("Expected 1 tool on page 3, got %d", len(listRes3.Tools))
	}
	if listRes3.Tools[0].Name != "e" {
		t.Errorf("Expected tool 'e', got %s", listRes3.Tools[0].Name)
	}
	if listRes3.NextCursor != "" {
		t.Fatalf("Expected empty next cursor, got %s", listRes3.NextCursor)
	}
}

func TestServer_ToolMiddleware(t *testing.T) {
	callCount := 0
	middleware := func(next mcp.ToolHandlerFunc) mcp.ToolHandlerFunc {
		return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			callCount++
			return next(ctx, request)
		}
	}

	server := mcp.NewMCPServer("test", "1.0", mcp.WithToolHandlerMiddleware(middleware))

	server.AddTool(mcp.ProtocolTool{Name: "test-mw"}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	})

	httpServer := mcp.NewStreamableHTTPServer(server, mcp.WithEndpointPath("/mcp"), mcp.WithStateLess(true))

	callReq := mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(1),
		Request: mcp.Request{Method: string(mcp.MethodToolsCall)},
		Params: mcp.CallToolParams{
			Name: "test-mw",
		},
	}

	reqBytes, _ := json.Marshal(callReq)
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	httpServer.ServeHTTP(w, req)

	if callCount != 1 {
		t.Errorf("Expected middleware to be called once, got %d", callCount)
	}
}

func TestServer_ToolFilter(t *testing.T) {
	filter := func(ctx context.Context, tools []mcp.ProtocolTool) []mcp.ProtocolTool {
		var filtered []mcp.ProtocolTool
		for _, tool := range tools {
			if tool.Name != "hidden" {
				filtered = append(filtered, tool)
			}
		}
		return filtered
	}

	server := mcp.NewMCPServer("test", "1.0", mcp.WithToolFilter(filter))

	server.AddTool(mcp.ProtocolTool{Name: "visible"}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	})
	server.AddTool(mcp.ProtocolTool{Name: "hidden"}, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	})

	tools := server.ListTools(context.Background())

	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool after filter, got %d", len(tools))
	}
	if tools[0].Name != "visible" {
		t.Errorf("Expected 'visible' tool, got '%s'", tools[0].Name)
	}
}
