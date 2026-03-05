package mcp

// Common HTTP header constants used across server transports
const (
	HeaderKeySessionID       = "Mcp-Session-Id"
	HeaderKeyProtocolVersion = "Mcp-Protocol-Version"
)

type MCPMethod string

const (
	// MethodInitialize initiates connection and negotiates protocol capabilities.
	MethodInitialize MCPMethod = "initialize"

	// MethodPing verifies connection liveness between client and server.
	MethodPing MCPMethod = "ping"

	// MethodToolsList lists all available executable tools.
	MethodToolsList MCPMethod = "tools/list"

	// MethodToolsCall invokes a specific tool with provided parameters.
	MethodToolsCall MCPMethod = "tools/call"

	// MethodNotificationToolsListChanged notifies when the list of available tools changes.
	MethodNotificationToolsListChanged = "notifications/tools/list_changed"
)

// LATEST_PROTOCOL_VERSION is the most recent version of the MCP protocol.
const LATEST_PROTOCOL_VERSION = "2025-11-25"

// ValidProtocolVersions lists all known valid MCP protocol versions.
var ValidProtocolVersions = []string{
	LATEST_PROTOCOL_VERSION,
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}

// JSONRPC_VERSION is the version of JSON-RPC used by MCP.
const JSONRPC_VERSION = "2.0"
