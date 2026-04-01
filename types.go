package mcp

type MCPMethod string

const (
	MethodInitialize MCPMethod = "initialize"
	MethodPing       MCPMethod = "ping"
	MethodToolsList  MCPMethod = "tools/list"
	MethodToolsCall  MCPMethod = "tools/call"

	MethodNotificationToolsListChanged = "notifications/tools/list_changed"
	MethodNotificationInitialized      = "notifications/initialized"
)

type JSONRPCMessage any

const LATEST_PROTOCOL_VERSION = "2024-11-05"

var validProtocolVersions = []string{
	LATEST_PROTOCOL_VERSION,
}

const JSONRPC_VERSION = "2.0"

type Cursor string

type RequestId = string

const (
	PARSE_ERROR         = -32700
	INVALID_REQUEST     = -32600
	METHOD_NOT_FOUND    = -32601
	INVALID_PARAMS      = -32602
	INTERNAL_ERROR      = -32603
	REQUEST_INTERRUPTED = -32800
)
