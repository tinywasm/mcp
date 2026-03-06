package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/tinywasm/mcp"
)

type MockTransport struct {
	mcp.Interface

	startFunc            func(context.Context) error
	closeFunc            func() error
	sendRequestFunc      func(context.Context, mcp.JSONRPCRequest) (*mcp.JSONRPCResponse, error)
	sendNotificationFunc func(context.Context, mcp.JSONRPCNotification) error
	getSessionIdFunc     func() string

	notificationHandler func(mcp.JSONRPCNotification)
}

func (m *MockTransport) Start(ctx context.Context) error {
	if m.startFunc != nil {
		return m.startFunc(ctx)
	}
	return nil
}

func (m *MockTransport) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *MockTransport) SendRequest(ctx context.Context, request mcp.JSONRPCRequest) (*mcp.JSONRPCResponse, error) {
	if m.sendRequestFunc != nil {
		return m.sendRequestFunc(ctx, request)
	}
	return nil, fmt.Errorf("sendRequest not implemented")
}

func (m *MockTransport) SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	if m.sendNotificationFunc != nil {
		return m.sendNotificationFunc(ctx, notification)
	}
	return nil
}

func (m *MockTransport) GetSessionId() string {
	if m.getSessionIdFunc != nil {
		return m.getSessionIdFunc()
	}
	return "mock-session"
}

func (m *MockTransport) SetNotificationHandler(handler func(mcp.JSONRPCNotification)) {
	m.notificationHandler = handler
}

func TestClient_Initialize(t *testing.T) {
	mock := &MockTransport{
		sendRequestFunc: func(ctx context.Context, req mcp.JSONRPCRequest) (*mcp.JSONRPCResponse, error) {
			if req.Method != "initialize" {
				return nil, fmt.Errorf("unexpected method: %s", req.Method)
			}
			res := mcp.InitializeResult{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				Capabilities: mcp.ServerCapabilities{
					Tools: &mcp.ToolsCapability{ListChanged: true},
				},
				ServerInfo: mcp.Implementation{Name: "mock-server", Version: "1.0"},
			}
			return &mcp.JSONRPCResponse{
				JSONRPC: mcp.JSONRPC_VERSION,
				ID:      req.ID,
				Result:  res,
			}, nil
		},
		sendNotificationFunc: func(ctx context.Context, notif mcp.JSONRPCNotification) error {
			if notif.Method != "notifications/initialized" {
				return fmt.Errorf("unexpected notification: %s", notif.Method)
			}
			return nil
		},
	}

	client := mcp.NewClient(mock)
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	req := mcp.InitializeRequest{}
	req.Params.ClientInfo = mcp.Implementation{Name: "test-client", Version: "1.0"}

	res, err := client.Initialize(context.Background(), req)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if res.ServerInfo.Name != "mock-server" {
		t.Errorf("Expected server name 'mock-server', got '%s'", res.ServerInfo.Name)
	}

	if !client.IsInitialized() {
		t.Errorf("Expected client to be initialized")
	}
}

func TestClient_ListTools(t *testing.T) {
	page := 0
	mock := &MockTransport{
		sendRequestFunc: func(ctx context.Context, req mcp.JSONRPCRequest) (*mcp.JSONRPCResponse, error) {
			if req.Method != "tools/list" {
				return nil, fmt.Errorf("unexpected method: %s", req.Method)
			}

			var res mcp.ListToolsResult
			if page == 0 {
				res = mcp.ListToolsResult{
					Tools: []mcp.ProtocolTool{
						mcp.NewProtocolTool("tool1"),
						mcp.NewProtocolTool("tool2"),
					},
					PaginatedResult: mcp.PaginatedResult{NextCursor: "page2"},
				}
				page++
			} else {
				res = mcp.ListToolsResult{
					Tools: []mcp.ProtocolTool{
						mcp.NewProtocolTool("tool3"),
					},
					PaginatedResult: mcp.PaginatedResult{NextCursor: ""},
				}
			}

			return &mcp.JSONRPCResponse{
				JSONRPC: mcp.JSONRPC_VERSION,
				ID:      req.ID,
				Result:  res,
			}, nil
		},
	}

	client := mcp.NewClient(mock, mcp.WithInitializedSession())
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	res, err := client.ListTools(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(res.Tools) != 3 {
		t.Fatalf("Expected 3 tools across pages, got %d", len(res.Tools))
	}

	if res.Tools[0].Name != "tool1" || res.Tools[2].Name != "tool3" {
		t.Errorf("Unexpected tools: %+v", res.Tools)
	}
}

func TestClient_CallTool(t *testing.T) {
	mock := &MockTransport{
		sendRequestFunc: func(ctx context.Context, req mcp.JSONRPCRequest) (*mcp.JSONRPCResponse, error) {
			if req.Method != "tools/call" {
				return nil, fmt.Errorf("unexpected method: %s", req.Method)
			}

			paramsBytes, _ := json.Marshal(req.Params)
			var params mcp.CallToolParams
			json.Unmarshal(paramsBytes, &params)

			if params.Name == "fail" {
				return &mcp.JSONRPCResponse{
					JSONRPC: mcp.JSONRPC_VERSION,
					ID:      req.ID,
					Result: mcp.CallToolResult{
						IsError: true,
						Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "failed"}},
					},
				}, nil
			}

			return &mcp.JSONRPCResponse{
				JSONRPC: mcp.JSONRPC_VERSION,
				ID:      req.ID,
				Result: mcp.CallToolResult{
					IsError: false,
					Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "success"}},
				},
			}, nil
		},
	}

	client := mcp.NewClient(mock, mcp.WithInitializedSession())
	if err := client.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Test success
	req := mcp.CallToolRequest{}
	req.Params.Name = "success"
	res, err := client.CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		t.Errorf("Expected success, got error")
	}

	textContent := res.Content[0].(mcp.TextContent)
	if textContent.Text != "success" {
		t.Errorf("Expected 'success', got '%s'", textContent.Text)
	}

	// Test error
	req.Params.Name = "fail"
	res, err = client.CallTool(context.Background(), req)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !res.IsError {
		t.Errorf("Expected error result")
	}

	textContent = res.Content[0].(mcp.TextContent)
	if textContent.Text != "failed" {
		t.Errorf("Expected 'failed', got '%s'", textContent.Text)
	}
}
