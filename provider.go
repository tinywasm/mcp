package mcp

// Loggable is implemented by handlers that support log injection.
type Loggable interface {
	Name() string
	SetLog(logger func(message ...any))
}

// ToolExecutor is a simplified handler that receives plain args.
type ToolExecutor func(args map[string]any)

// ParameterMetadata describes a single tool parameter.
type ParameterMetadata struct {
	Name        string
	Description string
	Required    bool
	Type        string   // "string", "number", "boolean"
	EnumValues  []string
	Default     any
}

// ToolMetadata is the high-level tool descriptor used by ToolProvider.
type ToolMetadata struct {
	Name        string
	Description string
	Parameters  []ParameterMetadata
	Execute     func(args map[string]any)
}

// ToolProvider exposes MCP tools to the Handler.
type ToolProvider interface {
	GetMCPToolsMetadata() []ToolMetadata
}

// BinaryData carries binary content (e.g. screenshot PNG) through the logger.
type BinaryData struct {
	MimeType string
	Data     []byte
}

// BuildTool constructs an mcp.Tool from ToolMetadata using the builder API.
// Replaces mcpserve.buildMCPTool with identical logic.
func BuildTool(meta ToolMetadata) Tool {
	options := []ToolOption{
		WithDescription(meta.Description),
	}

	for _, param := range meta.Parameters {
		switch param.Type {
		case "string":
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
			var boolOpts []PropertyOption

			if param.Required {
				boolOpts = append(boolOpts, Required())
			}
			if param.Description != "" {
				boolOpts = append(boolOpts, Description(param.Description))
			}

			options = append(options, WithBoolean(param.Name, boolOpts...))
		}
	}

	tool := NewTool(meta.Name, options...)
	return tool
}
