package mcp

import (
	"encoding/json"
	"fmt"
)

// Tool represents the definition for a tool the client can call.
type Tool struct {
	// Meta is a metadata object that is reserved by MCP for storing additional information.
	Meta *Meta `json:"_meta,omitempty"`
	// The name of the tool.
	Name string `json:"name"`
	// A human-readable description of the tool.
	Description string `json:"description,omitempty"`
	// A JSON Schema object defining the expected parameters for the tool.
	InputSchema ToolInputSchema `json:"inputSchema"`
	// Alternative to InputSchema - allows arbitrary JSON Schema to be provided
	RawInputSchema json.RawMessage `json:"-"` // Hide this from JSON marshaling
	// A JSON Schema object defining the expected output returned by the tool .
	OutputSchema ToolOutputSchema `json:"outputSchema,omitempty"`
	// Optional JSON Schema defining expected output structure
	RawOutputSchema json.RawMessage `json:"-"` // Hide this from JSON marshaling
	// Optional properties describing tool behavior
	Annotations ToolAnnotation `json:"annotations"`
	// Support for deferred loading
	DeferLoading bool `json:"defer_loading,omitempty"`
	// Icons provides visual identifiers for the tool
	Icons []Icon `json:"icons,omitempty"`
}

// GetName returns the name of the tool.
func (t Tool) GetName() string {
	return t.Name
}

// MarshalJSON implements the json.Marshaler interface for Tool.
// It handles marshaling either InputSchema or RawInputSchema based on which is set.
func (t Tool) MarshalJSON() ([]byte, error) {
	// Create a map to build the JSON structure
	m := make(map[string]any, 5)

	// Add the name and description
	m["name"] = t.Name
	if t.Description != "" {
		m["description"] = t.Description
	}

	// Determine which input schema to use
	if t.RawInputSchema != nil {
		if t.InputSchema.Type != "" {
			return nil, fmt.Errorf("tool %s has both InputSchema and RawInputSchema set: %w", t.Name, errToolSchemaConflict)
		}
		m["inputSchema"] = t.RawInputSchema
	} else {
		// Use the structured InputSchema
		m["inputSchema"] = t.InputSchema
	}

	// Add output schema if present
	if t.RawOutputSchema != nil {
		if t.OutputSchema.Type != "" {
			return nil, fmt.Errorf("tool %s has both OutputSchema and RawOutputSchema set: %w", t.Name, errToolSchemaConflict)
		}
		m["outputSchema"] = t.RawOutputSchema
	} else if t.OutputSchema.Type != "" { // If no output schema is specified, do not return anything
		m["outputSchema"] = t.OutputSchema
	}

	m["annotations"] = t.Annotations

	if t.DeferLoading {
		m["defer_loading"] = t.DeferLoading
	}

	// Marshal Meta if present
	if t.Meta != nil {
		m["_meta"] = t.Meta
	}

	if t.Icons != nil {
		m["icons"] = t.Icons
	}

	return json.Marshal(m)
}

// ToolArgumentsSchema represents a JSON Schema for tool arguments.
type ToolArgumentsSchema struct {
	Defs                 map[string]any `json:"$defs,omitempty"`
	Type                 string         `json:"type"`
	Properties           map[string]any `json:"properties,omitempty"`
	Required             []string       `json:"required,omitempty"`
	AdditionalProperties any            `json:"additionalProperties,omitempty"`
}

type ToolInputSchema ToolArgumentsSchema // For retro-compatibility
type ToolOutputSchema ToolArgumentsSchema

// MarshalJSON implements the json.Marshaler interface for ToolInputSchema.
func (tis ToolInputSchema) MarshalJSON() ([]byte, error) {
	return toolArgumentsSchemaMarshalJSON(ToolArgumentsSchema(tis))
}

// MarshalJSON implements the json.Marshaler interface for ToolOutputSchema.
func (tis ToolOutputSchema) MarshalJSON() ([]byte, error) {
	return toolArgumentsSchemaMarshalJSON(ToolArgumentsSchema(tis))
}

// MarshalJSON implements the json.Marshaler interface for ToolArgumentsSchema.
func (tis ToolArgumentsSchema) MarshalJSON() ([]byte, error) {
	return toolArgumentsSchemaMarshalJSON(tis)
}

// UnmarshalJSON implements the json.Unmarshaler interface for ToolInputSchema.
func (tis *ToolInputSchema) UnmarshalJSON(data []byte) error {
	return toolArgumentsSchemaUnmarshalJSON(data, (*ToolArgumentsSchema)(tis))
}

// UnmarshalJSON implements the json.Unmarshaler interface for ToolOutputSchema.
func (tis *ToolOutputSchema) UnmarshalJSON(data []byte) error {
	return toolArgumentsSchemaUnmarshalJSON(data, (*ToolArgumentsSchema)(tis))
}

// UnmarshalJSON implements the json.Unmarshaler interface for ToolArgumentsSchema.
func (tis *ToolArgumentsSchema) UnmarshalJSON(data []byte) error {
	return toolArgumentsSchemaUnmarshalJSON(data, tis)
}

// toolArgumentsSchemaMarshalJSON handles the fields stored in ToolArgumentsSchema when json.Marshaler is called
func toolArgumentsSchemaMarshalJSON(tis ToolArgumentsSchema) ([]byte, error) {
	m := make(map[string]any)
	m["type"] = tis.Type

	if tis.Defs != nil {
		m["$defs"] = tis.Defs
	}

	// Marshal Properties to '{}' rather than `nil` when its length equals zero
	if tis.Properties != nil {
		m["properties"] = tis.Properties
	} else {
		m["properties"] = map[string]any{}
	}

	// Marshal Required to '[]' rather than `nil` when its length equals zero
	if len(tis.Required) > 0 {
		m["required"] = tis.Required
	} else {
		m["required"] = []string{}
	}

	if tis.AdditionalProperties != nil {
		m["additionalProperties"] = tis.AdditionalProperties
	}

	return json.Marshal(m)
}

// It handles both "$defs" (JSON Schema 2019-09+) and "definitions" (JSON Schema draft-07)
// by reading either field and storing it in the Defs field.
func toolArgumentsSchemaUnmarshalJSON(data []byte, tis *ToolArgumentsSchema) error {
	// Use a temporary type to avoid infinite recursion
	type Alias ToolArgumentsSchema
	aux := &struct {
		Definitions map[string]any `json:"definitions,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(tis),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If $defs wasn't provided but definitions was, use definitions
	if tis.Defs == nil && aux.Definitions != nil {
		tis.Defs = aux.Definitions
	}

	return nil
}
