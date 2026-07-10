package mcp_test

import (
	"github.com/tinywasm/json"
	"github.com/tinywasm/model"
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
	if f, ok := resp.(model.Encodable); ok {
		json.Encode(f, &b)
	}
	return string(b)
}

// mockArgsWithBoolAndString — simple model.Fielder for testing
type mockArgsWithBoolAndString struct {
	flag bool
	text string
}

func (m *mockArgsWithBoolAndString) Schema() []model.Field {
	return []model.Field{
		{Name: "flag", Type: model.Bool()},
		{Name: "text", Type: model.Text(), NotNull: true},
	}
}

func (m *mockArgsWithBoolAndString) Encode(target *string) error {
	*target = `{"flag":false,"text":""}`
	return nil
}

func (m *mockArgsWithBoolAndString) Decode(data []byte) error {
	return nil
}

func (m *mockArgsWithBoolAndString) Pointers() []any {
	return []any{&m.flag, &m.text}
}
