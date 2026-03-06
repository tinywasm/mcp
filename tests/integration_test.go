package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tinywasm/mcp"
)

type MemoryHTTPTransport struct {
	httpServer http.Handler
	sessionId  string
}

func (m *MemoryHTTPTransport) Start(ctx context.Context) error {
	return nil
}

func (m *MemoryHTTPTransport) Close() error {
	return nil
}

func (m *MemoryHTTPTransport) SendRequest(ctx context.Context, request mcp.JSONRPCRequest) (*mcp.JSONRPCResponse, error) {
	reqBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	if m.sessionId != "" {
		req.Header.Set("X-Session-ID", m.sessionId)
	}

	w := httptest.NewRecorder()
	m.httpServer.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", w.Code)
	}

	var res mcp.JSONRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		return nil, err
	}

	if newSessionId := w.Header().Get("X-Session-ID"); newSessionId != "" {
		m.sessionId = newSessionId
	}

	return &res, nil
}

func (m *MemoryHTTPTransport) SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	return nil
}

func (m *MemoryHTTPTransport) GetSessionId() string {
	return m.sessionId
}

func (m *MemoryHTTPTransport) SetNotificationHandler(handler func(mcp.JSONRPCNotification)) {
	// Not needed for this integration test
}

func TestIntegration_Lifecycle(t *testing.T) {
	// 1. Setup Server
	server := mcp.NewMCPServer("integration-server", "1.0", mcp.WithToolCapabilities(true))

	server.AddTool(mcp.ProtocolTool{
		Name:        "echo",
		Description: "Echoes the input",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]any{
				"message": map[string]any{"type": "string"},
			},
			Required: []string{"message"},
		},
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		msg := req.GetString("message", "")
		return mcp.NewToolResultText("echoed: " + msg), nil
	})

	httpServer := mcp.NewStreamableHTTPServer(server, mcp.WithEndpointPath("/mcp"), mcp.WithStateLess(true))

	// 2. Setup Transport & Client
	transport := &MemoryHTTPTransport{httpServer: httpServer}
	client := mcp.NewClient(transport)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 3. Start Lifecycle
	err := client.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	// 4. Initialize
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "integration-client", Version: "1.0"}

	initRes, err := client.Initialize(ctx, initReq)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if initRes.ProtocolVersion != mcp.LATEST_PROTOCOL_VERSION {
		t.Errorf("Expected protocol version %s, got %s", mcp.LATEST_PROTOCOL_VERSION, initRes.ProtocolVersion)
	}
	if initRes.ServerInfo.Name != "integration-server" {
		t.Errorf("Expected server name 'integration-server', got '%s'", initRes.ServerInfo.Name)
	}

	// 5. Ping
	err = client.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// 6. List Tools
	listRes, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(listRes.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(listRes.Tools))
	}
	if listRes.Tools[0].Name != "echo" {
		t.Errorf("Expected tool 'echo', got '%s'", listRes.Tools[0].Name)
	}

	// 7. Call Tool
	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = "echo"
	callReq.Params.Arguments = map[string]any{"message": "hello world"}

	callRes, err := client.CallTool(ctx, callReq)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if callRes.IsError {
		t.Fatalf("Expected success, got error")
	}
	if len(callRes.Content) != 1 {
		t.Fatalf("Expected 1 content, got %d", len(callRes.Content))
	}

	textContent := callRes.Content[0].(mcp.TextContent)
	if textContent.Text != "echoed: hello world" {
		t.Errorf("Expected 'echoed: hello world', got '%s'", textContent.Text)
	}

	// 8. Close
	err = client.Close()
	if err != nil {
		t.Fatalf("Failed to close client: %v", err)
	}
}
