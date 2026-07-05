package mcp_test

import (
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
	"github.com/tinywasm/mcp"
)

// --- Mocks ---

// rbacAuth — records Authorize args and allows control per resource/action
type rbacAuth struct {
	lastResource string
	lastAction   string
	denyResource string
	denyAction   string
}

func (m *rbacAuth) Can(userID, resource, action string) bool {
	m.lastResource = resource
	m.lastAction = action
	if resource == m.denyResource && action == m.denyAction {
		return false
	}
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
	if f, ok := resp.(fmt.Encodable); ok {
		json.Encode(f, &b)
	}
	return string(b)
}
