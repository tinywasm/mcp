package mcp_test

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
	"github.com/tinywasm/mcp"
)

// --- Mocks ---

// mockAuth — simple auth that always succeeds with given id
type mockAuth struct {
	id string
}

func (m *mockAuth) Authorize(token string) (string, error) {
	return m.id, nil
}

func (m *mockAuth) Can(userID, resource string, action byte) bool {
	if userID == "forbidden-user" {
		return false
	}
	return true
}

// rbacAuth — records Can() args and allows control per resource/action
type rbacAuth struct {
	id           string
	lastResource string
	lastAction   byte
	denyResource string
	denyAction   byte
}

func (m *rbacAuth) Authorize(token string) (string, error) {
	return m.id, nil
}

func (m *rbacAuth) Can(userID, resource string, action byte) bool {
	m.lastResource = resource
	m.lastAction = action
	if resource == m.denyResource && action == m.denyAction {
		return false
	}
	return true
}

// emptyUserAuth — returns empty userID (tests empty rejection)
type emptyUserAuth struct{}

func (m *emptyUserAuth) Authorize(token string) (string, error) {
	return "", nil
}

func (m *emptyUserAuth) Can(userID, resource string, action byte) bool {
	return true
}

// mockSSE — records Publish calls
type mockSSE struct {
	lastData    []byte
	lastChannel string
	callCount   int
}

func (m *mockSSE) Publish(data []byte, channel string) {
	m.lastData = data
	m.lastChannel = channel
	m.callCount++
}

// --- Helpers ---

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func encodeResponse(resp mcp.JSONRPCMessage) string {
	var b []byte
	if f, ok := resp.(fmt.Fielder); ok {
		json.Encode(f, &b)
	}
	return string(b)
}

func newEchoTool() mcp.Tool {
	return mcp.Tool{
		Name:     "echo",
		Resource: "test",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	}
}
