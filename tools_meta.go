package mcp

// Loggable defines the interface for handlers that support logging
type Loggable interface {
	Name() string
	SetLog(logger func(message ...any))
}

// ToolProvider defines the interface for handlers that support MCP tools
type ToolProvider interface {
	GetMCPToolsMetadata() []ToolMetadata
}

// ToolExecutor defines how a tool should be executed
// Handlers implement this to provide execution logic without exposing internals
// args: map of parameter name to value from MCP request
type ToolExecutor func(args map[string]any)

// ToolMetadata provides MCP tool configuration metadata
// This is the standard interface that all handlers should implement
type ToolMetadata struct {
	Name        string
	Description string
	Parameters  []ParameterMetadata
	Execute     ToolExecutor // Handler provides execution function
}

// ParameterMetadata describes a tool parameter
type ParameterMetadata struct {
	Name        string
	Description string
	Required    bool
	Type        string // "string", "number", "boolean"
	EnumValues  []string
	Default     any
}

// buildMCPTool constructs MCP tool from metadata
func buildMCPTool(meta ToolMetadata) *Tool {
	options := []ToolOption{
		WithDescription(meta.Description),
	}

	for _, param := range meta.Parameters {
		switch param.Type {
		case "string":
			// Build string parameter options directly
			var strOpts []PropertyOption

			if param.Required {
				strOpts = append(strOpts, Required())
			}
			if param.Description != "" {
				strOpts = append(strOpts, Description(param.Description))
			}
			if len(param.EnumValues) > 0 {
				strOpts = append(strOpts, Enum(param.EnumValues...))
			}
			if param.Default != nil {
				if defaultStr, ok := param.Default.(string); ok {
					strOpts = append(strOpts, DefaultString(defaultStr))
				}
			}

			options = append(options, WithString(param.Name, strOpts...))

		case "number":
			// Build number parameter options directly
			var numOpts []PropertyOption

			if param.Required {
				numOpts = append(numOpts, Required())
			}
			if param.Description != "" {
				numOpts = append(numOpts, Description(param.Description))
			}
			if param.Default != nil {
				if defaultNum, ok := param.Default.(float64); ok {
					numOpts = append(numOpts, DefaultNumber(defaultNum))
				}
			}

			options = append(options, WithNumber(param.Name, numOpts...))

		case "boolean":
			// Build boolean parameter options directly
			var boolOpts []PropertyOption

			if param.Required {
				boolOpts = append(boolOpts, Required())
			}
			if param.Description != "" {
				boolOpts = append(boolOpts, Description(param.Description))
			}
			// Note: DefaultBoolean might not exist in mcp-go, skip for now

			options = append(options, WithBoolean(param.Name, boolOpts...))
		}
	}

	tool := NewTool(meta.Name, options...)
	return &tool
}
