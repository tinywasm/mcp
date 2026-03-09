package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/tinywasm/mcp"
)

type mockProvider struct {
	tools []mcp.Tool
}

func (p *mockProvider) GetMCPTools() []mcp.Tool {
	return p.tools
}

func TestHandler_Serve_StartsHTTP(t *testing.T) {
	config := mcp.Config{
		Port:          "3031", // Use a different port to avoid conflicts
		ServerName:    "Test Server",
		ServerVersion: "1.0.0",
	}
	h := mcp.NewHandler(config, nil, nil)
	h.SetAuth(mcp.OpenAuthorizer())

	exitChan := make(chan bool)
	go h.Serve(exitChan)

	// Wait for server to start
	var err error
	for i := 0; i < 20; i++ {
		_, err = http.Get("http://localhost:3031/mcp")
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err != nil {
		t.Errorf("Server failed to start: %v", err)
	}

	close(exitChan)
}

func TestHandler_OnAction_JSONRPCMethod(t *testing.T) {
	config := mcp.Config{
		Port:          "3032",
		ServerName:    "Test Server",
		ServerVersion: "1.0.0",
	}
	h := mcp.NewHandler(config, nil, nil)
	h.SetAuth(mcp.OpenAuthorizer())

	var mu sync.Mutex
	var receivedKey, receivedValue string
	done := make(chan struct{})

	h.RegisterMethod("tinywasm/action", func(ctx context.Context, params []byte) (any, error) {
		var p struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		json.Unmarshal(params, &p)
		mu.Lock()
		receivedKey = p.Key
		receivedValue = p.Value
		mu.Unlock()
		close(done)
		return nil, nil
	})

	server := httptest.NewServer(h.HTTPHandler())
	defer server.Close()

	client := mcp.NewClient(server.URL, "")
	client.Dispatch(context.Background(), "tinywasm/action", map[string]string{"key": "foo", "value": "bar"})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for action handler")
	}

	mu.Lock()
	key, val := receivedKey, receivedValue
	mu.Unlock()

	if key != "foo" || val != "bar" {
		t.Errorf("Expected foo/bar, got %s/%s", key, val)
	}
}

func TestHandler_SetDynamicProviders_ToolsVisible(t *testing.T) {
	config := mcp.Config{
		ServerName:    "Test Server",
		ServerVersion: "1.0.0",
	}
	h := mcp.NewHandler(config, nil, nil)
	h.SetAuth(mcp.OpenAuthorizer())

	p := &mockProvider{
		tools: []mcp.Tool{
			{Name: "dynamic_tool", Description: "Dynamic"},
		},
	}
	h.SetDynamicProviders(p)

	server := httptest.NewServer(h.HTTPHandler())
	defer server.Close()

	client := mcp.NewClient(server.URL, "")

	var result mcp.ListToolsResult
	done := make(chan bool)
	client.Call(context.Background(), "tools/list", nil, func(data []byte, err error) {
		if err != nil {
			t.Errorf("tools/list failed: %v", err)
		} else {
			json.Unmarshal(data, &result)
		}
		done <- true
	})
	<-done

	found := false
	for _, tool := range result.Tools {
		if tool.Name == "dynamic_tool" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Dynamic tool not found in tools/list")
	}
}

func TestHandler_ToolOutput_TextResult(t *testing.T) {
	config := mcp.Config{
		ServerName:    "Test Server",
		ServerVersion: "1.0.0",
	}
	p := &mockProvider{
		tools: []mcp.Tool{
			{
				Name:     "echo_tool",
				Execute:  func(args map[string]any) { /* tool will call logger */ },
			},
		},
	}
	h := mcp.NewHandler(config, nil, []mcp.ToolProvider{p})
	h.SetAuth(mcp.OpenAuthorizer())

	server := httptest.NewServer(h.HTTPHandler())
	defer server.Close()

	client := mcp.NewClient(server.URL, "")

	done := make(chan bool)
	client.Call(context.Background(), "tools/call", map[string]any{"name": "echo_tool"}, func(data []byte, err error) {
		var res mcp.CallToolResult
		json.Unmarshal(data, &res)
		if len(res.Content) == 0 {
			t.Errorf("Expected content in response")
		}
		_, ok := res.Content[0].(mcp.TextContent)
		if !ok {
			t.Errorf("Expected TextContent, got %T", res.Content[0])
		}
		done <- true
	})
	<-done
}

func TestHandler_ToolRBAC_Denied(t *testing.T) {
	config := mcp.Config{
		ServerName:    "Test Server",
		ServerVersion: "1.0.0",
	}
	p := &mockProvider{
		tools: []mcp.Tool{
			{
				Name:     "secure_tool",
				Resource: "secret",
				Action:   'r',
				Execute:  func(args map[string]any) {},
			},
		},
	}
	h := mcp.NewHandler(config, nil, []mcp.ToolProvider{p})
	// Default is denyAllAuthorizer

	server := httptest.NewServer(h.HTTPHandler())
	defer server.Close()

	client := mcp.NewClient(server.URL, "")

	done := make(chan bool)
	client.Call(context.Background(), "tools/call", map[string]any{"name": "secure_tool"}, func(data []byte, err error) {
		var res mcp.CallToolResult
		json.Unmarshal(data, &res)
		if !res.IsError {
			t.Errorf("Expected access denied error")
		}
		// res.Content[0] should be a text content
		if len(res.Content) == 0 {
			t.Errorf("Expected result content")
		}
		done <- true
	})
	<-done
}

func TestTokenAuthorizer_ValidToken_Injects(t *testing.T) {
	auth := mcp.NewTokenAuthorizer("secret-token")
	req, _ := http.NewRequest("POST", "/mcp", nil)
	req.Header.Set("Authorization", "Bearer secret-token")

	ctx := auth.InjectIdentity(context.Background(), req)
	if !auth.CanExecute(ctx, "any", 'r') {
		t.Errorf("Expected CanExecute to be true for valid token")
	}
}

func TestHandler_URL_ReturnsBaseURL(t *testing.T) {
	config := mcp.Config{Port: "3030"}
	h := mcp.NewHandler(config, nil, nil)
	expected := "http://localhost:3030"
	if h.URL() != expected {
		t.Errorf("Expected %s, got %s", expected, h.URL())
	}
}
