package mcp_test

import (
	"reflect"
	"testing"

	"github.com/tinywasm/mcp"
)

func TestBuildMCPTool(t *testing.T) {
	meta := mcp.ToolMetadata{
		Name:        "test-tool",
		Description: "A tool for testing",
		Parameters: []mcp.ParameterMetadata{
			{
				Name:        "str_param",
				Type:        "string",
				Description: "A string parameter",
				Required:    true,
				EnumValues:  []string{"a", "b"},
				Default:     "a",
			},
			{
				Name:        "num_param",
				Type:        "number",
				Description: "A number parameter",
				Default:     float64(42),
			},
			{
				Name:     "bool_param",
				Type:     "boolean",
				Required: true,
			},
		},
	}

	tool := mcp.BuildMCPTool(meta)

	if tool.Name != "test-tool" {
		t.Errorf("Expected name 'test-tool', got '%s'", tool.Name)
	}

	if tool.Description != "A tool for testing" {
		t.Errorf("Expected description 'A tool for testing', got '%s'", tool.Description)
	}

	props := tool.InputSchema.Properties
	if len(props) != 3 {
		t.Fatalf("Expected 3 properties, got %d", len(props))
	}

	// Verify required array
	required := tool.InputSchema.Required
	if len(required) != 2 {
		t.Fatalf("Expected 2 required properties, got %d", len(required))
	}

	// Verify string parameter
	strProp, ok := props["str_param"].(map[string]any)
	if !ok {
		t.Fatalf("str_param not found or wrong type")
	}
	if strProp["type"] != "string" {
		t.Errorf("str_param type wrong: %v", strProp["type"])
	}
	if strProp["description"] != "A string parameter" {
		t.Errorf("str_param description wrong: %v", strProp["description"])
	}
	if strProp["default"] != "a" {
		t.Errorf("str_param default wrong: %v", strProp["default"])
	}
	if !reflect.DeepEqual(strProp["enum"], []string{"a", "b"}) {
		t.Errorf("str_param enum wrong: %v", strProp["enum"])
	}

	// Verify number parameter
	numProp, ok := props["num_param"].(map[string]any)
	if !ok {
		t.Fatalf("num_param not found or wrong type")
	}
	if numProp["type"] != "number" {
		t.Errorf("num_param type wrong: %v", numProp["type"])
	}
	if numProp["default"] != float64(42) {
		t.Errorf("num_param default wrong: %v", numProp["default"])
	}

	// Verify boolean parameter
	boolProp, ok := props["bool_param"].(map[string]any)
	if !ok {
		t.Fatalf("bool_param not found or wrong type")
	}
	if boolProp["type"] != "boolean" {
		t.Errorf("bool_param type wrong: %v", boolProp["type"])
	}
}
