package mcp

import "github.com/tinywasm/model"

type MCPMethod string

const (
	MethodInitialize MCPMethod = "initialize"
	MethodPing       MCPMethod = "ping"
	MethodToolsList  MCPMethod = "tools/list"
	MethodToolsCall  MCPMethod = "tools/call"

	MethodNotificationToolsListChanged = "notifications/tools/list_changed"
	MethodNotificationInitialized      = "notifications/initialized"
)

type JSONRPCMessage interface {
	jsonrpcMessage()
}

const (
	ProtocolVersion20241105 = "2024-11-05"
	ProtocolVersion20250326 = "2025-03-26"
	ProtocolVersion20250618 = "2025-06-18"
	ProtocolVersion20251125 = "2025-11-25"
)

var SupportedProtocolVersions = []string{
	ProtocolVersion20251125,
	ProtocolVersion20250618,
	ProtocolVersion20250326,
	ProtocolVersion20241105,
}

const LATEST_PROTOCOL_VERSION = ProtocolVersion20251125

const JSONRPC_VERSION = "2.0"

type RequestId = model.RawJSON

const (
	PARSE_ERROR         = -32700
	INVALID_REQUEST     = -32600
	METHOD_NOT_FOUND    = -32601
	INVALID_PARAMS      = -32602
	INTERNAL_ERROR      = -32603
	REQUEST_INTERRUPTED = -32800
)
