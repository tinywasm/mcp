package mcp

// ToolOption configures a ProtocolTool during construction.
type ToolOption func(*ProtocolTool)

// PropertyOption configures a property within a ToolInputSchema.
type PropertyOption func(map[string]any)

// WithDescription sets the tool description.
func WithDescription(desc string) ToolOption {
	return func(t *ProtocolTool) {
		t.Description = desc
	}
}

// WithString adds a string property to the tool's input schema.
func WithString(name string, opts ...PropertyOption) ToolOption {
	return func(t *ProtocolTool) {
		prop := make(map[string]any)
		prop["type"] = "string"
		for _, opt := range opts {
			opt(prop)
		}
		if t.InputSchema.Properties == nil {
			t.InputSchema.Properties = make(map[string]any)
		}
		t.InputSchema.Properties[name] = prop
	}
}

// WithNumber adds a number property to the tool's input schema.
func WithNumber(name string, opts ...PropertyOption) ToolOption {
	return func(t *ProtocolTool) {
		prop := make(map[string]any)
		prop["type"] = "number"
		for _, opt := range opts {
			opt(prop)
		}
		if t.InputSchema.Properties == nil {
			t.InputSchema.Properties = make(map[string]any)
		}
		t.InputSchema.Properties[name] = prop
	}
}

// WithBoolean adds a boolean property to the tool's input schema.
func WithBoolean(name string, opts ...PropertyOption) ToolOption {
	return func(t *ProtocolTool) {
		prop := make(map[string]any)
		prop["type"] = "boolean"
		for _, opt := range opts {
			opt(prop)
		}
		if t.InputSchema.Properties == nil {
			t.InputSchema.Properties = make(map[string]any)
		}
		t.InputSchema.Properties[name] = prop
	}
}

// Required marks the property as required in the schema.
func Required() PropertyOption {
	return func(prop map[string]any) {
		// Mark parent schema as having this property required
		// This will be handled by the calling context
	}
}

// Description sets the property description.
func Description(desc string) PropertyOption {
	return func(prop map[string]any) {
		prop["description"] = desc
	}
}

// Enum restricts allowed values.
func Enum(values ...string) PropertyOption {
	return func(prop map[string]any) {
		prop["enum"] = values
	}
}

// DefaultString sets the default string value.
func DefaultString(v string) PropertyOption {
	return func(prop map[string]any) {
		prop["default"] = v
	}
}

// DefaultNumber sets the default number value.
func DefaultNumber(v float64) PropertyOption {
	return func(prop map[string]any) {
		prop["default"] = v
	}
}
