package mcp

import (
	"encoding/json"
)

type ToolAnnotation struct {
	// Human-readable title for the tool
	Title string `json:"title,omitempty"`
	// If true, the tool does not modify its environment
	ReadOnlyHint *bool `json:"readOnlyHint,omitempty"`
	// If true, the tool may perform destructive updates
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	// If true, repeated calls with same args have no additional effect
	IdempotentHint *bool `json:"idempotentHint,omitempty"`
	// If true, tool interacts with external entities
	OpenWorldHint *bool `json:"openWorldHint,omitempty"`
}

// ToolOption is a function that configures a Tool.
// It provides a flexible way to set various properties of a Tool using the functional options pattern.
type ToolOption func(*Tool)

// PropertyOption is a function that configures a property in a Tool's input schema.
// It allows for flexible configuration of JSON Schema properties using the functional options pattern.
type PropertyOption func(map[string]any)

//
// Core Tool Functions
//

// NewTool creates a new Tool with the given name and options.
// The tool will have an object-type input schema with configurable properties.
// Options are applied in order, allowing for flexible tool configuration.
func NewTool(name string, opts ...ToolOption) Tool {
	tool := Tool{
		Name: name,
		InputSchema: ToolInputSchema{
			Type:       "object",
			Properties: make(map[string]any),
			Required:   nil, // Will be omitted from JSON if empty
		},
		Annotations: ToolAnnotation{
			Title:           "",
			ReadOnlyHint:    ToBoolPtr(false),
			DestructiveHint: ToBoolPtr(true),
			IdempotentHint:  ToBoolPtr(false),
			OpenWorldHint:   ToBoolPtr(true),
		},
	}

	for _, opt := range opts {
		opt(&tool)
	}

	return tool
}

// NewToolWithRawSchema creates a new Tool with the given name and a raw JSON
// Schema. This allows for arbitrary JSON Schema to be used for the tool's input
// schema.
//
// NOTE a [Tool] built in such a way is incompatible with the [ToolOption] and
// runtime errors will result from supplying a [ToolOption] to a [Tool] built
// with this function.
func NewToolWithRawSchema(name, description string, schema json.RawMessage) Tool {
	tool := Tool{
		Name:           name,
		Description:    description,
		RawInputSchema: schema,
	}

	return tool
}

// WithDescription adds a description to the Tool.
// The description should provide a clear, human-readable explanation of what the tool does.
func WithDescription(description string) ToolOption {
	return func(t *Tool) {
		t.Description = description
	}
}

// WithDeferLoading sets the defer_loading flag for the tool.
// This is used to implement dynamic tool loading/searching patterns.
func WithDeferLoading(deferLoading bool) ToolOption {
	return func(t *Tool) {
		t.DeferLoading = deferLoading
	}
}

// WithToolIcons adds icons to the Tool.
// Icons provide visual identifiers for the tool.
func WithToolIcons(icons ...Icon) ToolOption {
	return func(t *Tool) {
		t.Icons = icons
	}
}

// WithRawInputSchema sets a raw JSON schema for the tool's input.
// Use this when you need full control over the schema or when working with
// complex schemas that can't be generated from Go types. The jsonschema library
// can handle complex schemas and provides nice extension points, so be sure to
// check that out before using this.
func WithRawInputSchema(schema json.RawMessage) ToolOption {
	return func(t *Tool) {
		t.RawInputSchema = schema
	}
}

// WithRawOutputSchema sets a raw JSON schema for the tool's output.
// Use this when you need full control over the schema or when working with
// complex schemas that can't be generated from Go types. The jsonschema library
// can handle complex schemas and provides nice extension points, so be sure to
// check that out before using this.
func WithRawOutputSchema(schema json.RawMessage) ToolOption {
	return func(t *Tool) {
		t.RawOutputSchema = schema
	}
}

// WithToolAnnotation adds optional hints about the Tool.
func WithToolAnnotation(annotation ToolAnnotation) ToolOption {
	return func(t *Tool) {
		t.Annotations = annotation
	}
}

// WithTitleAnnotation sets the Title field of the Tool's Annotations.
// It provides a human-readable title for the tool.
func WithTitleAnnotation(title string) ToolOption {
	return func(t *Tool) {
		t.Annotations.Title = title
	}
}

// WithReadOnlyHintAnnotation sets the ReadOnlyHint field of the Tool's Annotations.
// If true, it indicates the tool does not modify its environment.
func WithReadOnlyHintAnnotation(value bool) ToolOption {
	return func(t *Tool) {
		t.Annotations.ReadOnlyHint = &value
	}
}

// WithDestructiveHintAnnotation sets the DestructiveHint field of the Tool's Annotations.
// If true, it indicates the tool may perform destructive updates.
func WithDestructiveHintAnnotation(value bool) ToolOption {
	return func(t *Tool) {
		t.Annotations.DestructiveHint = &value
	}
}

// WithIdempotentHintAnnotation sets the IdempotentHint field of the Tool's Annotations.
// If true, it indicates repeated calls with the same arguments have no additional effect.
func WithIdempotentHintAnnotation(value bool) ToolOption {
	return func(t *Tool) {
		t.Annotations.IdempotentHint = &value
	}
}

// WithOpenWorldHintAnnotation sets the OpenWorldHint field of the Tool's Annotations.
// If true, it indicates the tool interacts with external entities.
func WithOpenWorldHintAnnotation(value bool) ToolOption {
	return func(t *Tool) {
		t.Annotations.OpenWorldHint = &value
	}
}

// WithSchemaAdditionalProperties sets the additionalProperties field on the tool's input schema.
// It accepts false (disallow extra properties), true (allow any), or a schema map
// to validate additional properties against.
func WithSchemaAdditionalProperties(schema any) ToolOption {
	return func(t *Tool) {
		t.InputSchema.AdditionalProperties = schema
	}
}
