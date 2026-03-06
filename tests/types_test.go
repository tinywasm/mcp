package mcp_test

import (
	"encoding/json"
	"testing"

	"github.com/tinywasm/mcp"
)

func TestRequestId_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		id       mcp.RequestId
		expected string
	}{
		{"string", mcp.NewRequestId("test-id"), `"test-id"`},
		{"int64", mcp.NewRequestId(int64(42)), `42`},
		{"float64", mcp.NewRequestId(float64(3.14)), `3.14`},
		{"nil", mcp.NewRequestId(nil), `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.id)
			if err != nil {
				t.Fatalf("Failed to marshal RequestId: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(data))
			}
		})
	}
}

func TestRequestId_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected mcp.RequestId
	}{
		{"string", `"test-id"`, mcp.NewRequestId("test-id")},
		{"int64", `42`, mcp.NewRequestId(int64(42))},
		{"float64", `3.14`, mcp.NewRequestId(float64(3.14))},
		{"nil", `null`, mcp.NewRequestId(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id mcp.RequestId
			err := json.Unmarshal([]byte(tt.data), &id)
			if err != nil {
				t.Fatalf("Failed to unmarshal RequestId: %v", err)
			}
			if id.Value() != tt.expected.Value() {
				t.Errorf("Expected %v, got %v", tt.expected.Value(), id.Value())
			}
		})
	}
}

func TestMeta_AdditionalFields(t *testing.T) {
	// Test Unmarshal
	data := []byte(`{"progressToken": 123, "customField": "value", "otherField": true}`)
	var meta mcp.Meta
	err := json.Unmarshal(data, &meta)
	if err != nil {
		t.Fatalf("Failed to unmarshal Meta: %v", err)
	}

	if meta.ProgressToken != float64(123) {
		t.Errorf("Expected progressToken 123, got %v", meta.ProgressToken)
	}

	if meta.AdditionalFields["customField"] != "value" {
		t.Errorf("Expected customField 'value', got %v", meta.AdditionalFields["customField"])
	}

	if meta.AdditionalFields["otherField"] != true {
		t.Errorf("Expected otherField true, got %v", meta.AdditionalFields["otherField"])
	}

	// Test Marshal
	marshaledData, err := json.Marshal(&meta)
	if err != nil {
		t.Fatalf("Failed to marshal Meta: %v", err)
	}

	var raw map[string]any
	json.Unmarshal(marshaledData, &raw)

	if raw["progressToken"] != float64(123) {
		t.Errorf("Expected progressToken 123 in marshaled data, got %v", raw["progressToken"])
	}

	if raw["customField"] != "value" {
		t.Errorf("Expected customField 'value' in marshaled data, got %v", raw["customField"])
	}

	if raw["otherField"] != true {
		t.Errorf("Expected otherField true in marshaled data, got %v", raw["otherField"])
	}
}

func TestNotificationParams_MarshalJSON(t *testing.T) {
	params := mcp.NotificationParams{
		Meta: map[string]any{
			"progressToken": 456,
		},
		AdditionalFields: map[string]any{
			"key1": "val1",
			"key2": 2,
		},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal NotificationParams: %v", err)
	}

	var raw map[string]any
	err = json.Unmarshal(data, &raw)
	if err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	meta, ok := raw["_meta"].(map[string]any)
	if !ok {
		t.Fatalf("Expected _meta to be a map")
	}

	if meta["progressToken"] != float64(456) {
		t.Errorf("Expected _meta.progressToken 456, got %v", meta["progressToken"])
	}

	if raw["key1"] != "val1" {
		t.Errorf("Expected key1 'val1', got %v", raw["key1"])
	}

	if raw["key2"] != float64(2) {
		t.Errorf("Expected key2 2, got %v", raw["key2"])
	}
}

func TestNotificationParams_UnmarshalJSON(t *testing.T) {
	data := []byte(`{"_meta": {"progressToken": 789}, "custom1": "hello", "custom2": false}`)

	var params mcp.NotificationParams
	err := json.Unmarshal(data, &params)
	if err != nil {
		t.Fatalf("Failed to unmarshal NotificationParams: %v", err)
	}

	if params.Meta["progressToken"] != float64(789) {
		t.Errorf("Expected _meta.progressToken 789, got %v", params.Meta["progressToken"])
	}

	if params.AdditionalFields["custom1"] != "hello" {
		t.Errorf("Expected AdditionalFields.custom1 'hello', got %v", params.AdditionalFields["custom1"])
	}

	if params.AdditionalFields["custom2"] != false {
		t.Errorf("Expected AdditionalFields.custom2 false, got %v", params.AdditionalFields["custom2"])
	}
}
