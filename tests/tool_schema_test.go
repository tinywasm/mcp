package mcp_test

import (
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/model"
)

// mockAllFieldTypes covers FieldInt, FieldFloat, FieldBool, FieldIntSlice, FieldStruct, FieldStructSlice, FieldText, FieldRaw, FieldBlob
type mockAllFieldTypes struct {
	textVal      string
	intVal       int64
	floatVal     float64
	boolVal      bool
	intSliceVal  []int64
	rawVal       string
	blobVal      string
}

func (m *mockAllFieldTypes) Schema() []model.Field {
	return []model.Field{
		{Name: "text_field", Type: model.Text(), NotNull: true},
		{Name: "int_field", Type: model.Int()},
		{Name: "float_field", Type: model.Float()},
		{Name: "bool_field", Type: model.Bool()},
		{Name: "int_slice_field", Type: model.IntSlice()},
		{Name: "raw_field", Type: model.Raw()},
		{Name: "blob_field", Type: model.Blob()},
	}
}

func (m *mockAllFieldTypes) Encode(target *string) error {
	*target = `{}`
	return nil
}

func (m *mockAllFieldTypes) Decode(data []byte) error {
	return nil
}

func (m *mockAllFieldTypes) Pointers() []any {
	return []any{&m.textVal, &m.intVal, &m.floatVal, &m.boolVal, &m.intSliceVal, &m.rawVal, &m.blobVal}
}

func TestToolSchema_AllFieldTypes_GeneratesCorrectSchemas(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	args := &mockAllFieldTypes{}
	srv.AddTool(mcp.Tool{
		Name:        "tool_all_types",
		Description: "Tool with all field types",
		Args:        args,
		Resource:    "res",
		Action:      model.Read,
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})

	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/list"}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)

	// Check all types
	tests := []struct {
		name     string
		expected string
	}{
		{"FieldText", `"text_field":{"type":"string"}`},
		{"FieldInt", `"int_field":{"type":"integer"}`},
		{"FieldFloat", `"float_field":{"type":"number"}`},
		{"FieldBool", `"bool_field":{"type":"boolean"}`},
		{"FieldIntSlice", `"int_slice_field":{"type":"array","items":{"type":"integer"}}`},
		{"FieldRaw (string)", `"raw_field":{"type":"string"}`},
		{"FieldBlob (string)", `"blob_field":{"type":"string"}`},
		{"NotNull required", `"required":["text_field"]`},
		{"Has properties", `"properties":{`},
	}

	for _, tt := range tests {
		if !contains(respStr, tt.expected) {
			t.Errorf("expected %s (%s) in schema, got %s", tt.name, tt.expected, respStr)
		}
	}
}

func TestToolSchema_WithArgs_ContainsProperties(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	args := &mockArgsWithBoolAndString{}
	srv.AddTool(mcp.Tool{
		Name:        "tool_with_args",
		Description: "A tool with args",
		Args:        args,
		Resource:    "res",
		Action:      model.Read,
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})

	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/list"}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, `"type":"object"`) {
		t.Fatalf("expected 'type':'object' in schema, got %s", respStr)
	}
	if !contains(respStr, `"properties"`) {
		t.Fatalf("expected 'properties' in schema, got %s", respStr)
	}
	if !contains(respStr, `"flag":{"type":"boolean"}`) {
		t.Fatalf("expected flag:boolean in schema, got %s", respStr)
	}
	if !contains(respStr, `"text":{"type":"string"}`) {
		t.Fatalf("expected text:string in schema, got %s", respStr)
	}
	if !contains(respStr, `"required":["text"]`) {
		t.Fatalf("expected required:[text] in schema, got %s", respStr)
	}
}

func TestToolSchema_WithoutArgs_EmptyObject(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	srv.AddTool(mcp.Tool{
		Name:        "tool_without_args",
		Description: "A tool without args",
		Args:        nil,
		Resource:    "res",
		Action:      model.Read,
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})

	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/list"}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if !contains(respStr, `{"type":"object","properties":{}}`) {
		t.Fatalf("expected empty object schema, got %s", respStr)
	}
}

func TestToolSchema_NeverNull(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Authorize: mcp.AllowAll}, nil)
	srv.AddTool(mcp.Tool{
		Name:        "tool_no_args",
		Description: "Tool without args",
		Args:        nil,
		Resource:    "res",
		Action:      model.Read,
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("ok"), nil
		},
	})

	var ctx context.Context
	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/list"}`)
	resp := srv.HandleMessage(&ctx, req)

	respStr := encodeResponse(resp)
	if contains(respStr, `"inputSchema":null`) {
		t.Fatalf("inputSchema must never be null, got %s", respStr)
	}
}
