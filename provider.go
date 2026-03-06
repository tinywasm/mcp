package mcp

// Loggable is implemented by handlers that support log injection.
type Loggable interface {
	Name() string
	SetLog(logger func(message ...any))
}

// ToolExecutor is a simplified handler that receives plain args.
type ToolExecutor func(args map[string]any)

// Parameter describes a single tool parameter.
type Parameter struct {
	Name        string
	Description string
	Required    bool
	Type        string // "string", "number", "boolean"
	EnumValues  []string
	Default     any
}

// Tool is the high-level tool descriptor used by ToolProvider.
type Tool struct {
	Name        string
	Description string
	Parameters  []Parameter
	Execute     func(args map[string]any)
}

// ToolProvider exposes MCP tools to the Handler.
type ToolProvider interface {
	GetMCPTools() []Tool
}

// BinaryData carries binary content (e.g. screenshot PNG) through the logger.
type BinaryData struct {
	MimeType string
	Data     []byte
}
