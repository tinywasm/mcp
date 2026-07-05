package mcp_test

import (
	"testing"

	"github.com/tinywasm/mcp"
	"github.com/tinywasm/router"
)

func TestServerImplementsAPIModule(t *testing.T) {
	var _ router.APIModule = (*mcp.Server)(nil)
}

type mockRouter struct {
	router.Router
	path    string
	handler router.HandlerFunc
}

func (m *mockRouter) Post(path string, h router.HandlerFunc) router.Route {
	m.path = path
	m.handler = h
	return nil
}

type mockContext struct {
	router.Context
	body        []byte
	headers     map[string]string
	status      int
	writtenData []byte
	userID      string
}

func (m *mockContext) Body() []byte { return m.body }
func (m *mockContext) GetHeader(key string) string {
	if m.headers == nil {
		return ""
	}
	return m.headers[key]
}
func (m *mockContext) SetHeader(key, value string) {
	if m.headers == nil {
		m.headers = make(map[string]string)
	}
	m.headers[key] = value
}
func (m *mockContext) WriteStatus(code int) { m.status = code }
func (m *mockContext) Write(data []byte) (int, error) {
	m.writtenData = append(m.writtenData, data...)
	return len(data), nil
}
func (m *mockContext) UserID() string { return m.userID }

func TestMountAPI(t *testing.T) {
	server, _ := mcp.NewServer(mcp.Config{
		Name:      "test-server",
		Version:   "1.0.0",
		Authorize: mcp.AllowAll,
	}, nil)

	mr := &mockRouter{}
	server.MountAPI(mr)

	if mr.path != mcp.MCPPath {
		t.Errorf("expected path %s, got %s", mcp.MCPPath, mr.path)
	}

	if mr.handler == nil {
		t.Fatal("handler not registered")
	}

	// Test a simple initialize request
	ctx := &mockContext{
		body: []byte(`{"jsonrpc":"2.0","id":"1","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}`),
	}

	mr.handler(ctx)

	if ctx.headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ctx.headers["Content-Type"])
	}

	if !contains(string(ctx.writtenData), `"result"`) {
		t.Errorf("expected result in response, got %s", string(ctx.writtenData))
	}
	if !contains(string(ctx.writtenData), `"protocolVersion"`) {
		t.Errorf("expected protocolVersion in response, got %s", string(ctx.writtenData))
	}
}

func TestModelName(t *testing.T) {
	server, _ := mcp.NewServer(mcp.Config{
		Name:      "my-mcp-server",
		Authorize: mcp.AllowAll,
	}, nil)

	if server.ModelName() != "my-mcp-server" {
		t.Errorf("expected ModelName my-mcp-server, got %s", server.ModelName())
	}
}
