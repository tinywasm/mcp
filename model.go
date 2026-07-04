package mcp

import "github.com/tinywasm/fmt"

type rpcRequest struct {
	JSONRPC string
	ID      string
	Method  string
	Params  string
}

type rpcResponse struct {
	JSONRPC string
	ID      string
	Result  fmt.RawJSON `json:",omitempty"`
	Error   string      `json:",omitempty"`
}

type jsonRPCError struct {
	Code    int64
	Message string
	Data    string `json:",omitempty"`
}

type initializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      implementationInfo `json:"clientInfo"`
}

type implementationInfo struct {
	Name    string
	Version string
}

type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      implementationInfo `json:"serverInfo"`
	Capabilities    fmt.RawJSON        `json:",omitempty"`
}

type CallToolParams struct {
	Name      string
	Arguments fmt.RawJSON
}

type Result struct {
	IsError bool        `json:"isError,omitempty"`
	Content fmt.RawJSON `json:"content,omitempty"`
}

type TextContent struct {
	Type string
	Text string
}

type toolEntry struct {
	Name        string
	Description string      `json:",omitempty"`
	InputSchema fmt.RawJSON `json:"inputSchema"`
}

type listToolsResult struct {
	Tools      fmt.RawJSON
	NextCursor string `json:"nextCursor,omitempty"`
}

type errorResponse struct {
	JSONRPC string
	ID      string `json:",omitempty"`
	Error   jsonRPCError
}

type Meta struct {
	ProgressToken string `json:"progressToken,omitempty"`
}

type NotificationParams struct {
	Meta string `json:",omitempty"`
}

type EmptyResult struct {
	Result string `json:",omitempty"`
}

type JSONRPCRequest struct {
	JSONRPC string
	ID      RequestId
	Method  string
	Params  string `json:",omitempty"`
}

type JSONRPCNotification struct {
	JSONRPC string
	Method  string
	Params  string `json:",omitempty"`
}

type JSONRPCResponseStruct struct {
	JSONRPC string
	ID      string
	Result  fmt.RawJSON `json:",omitempty"`
	Error   fmt.RawJSON `json:",omitempty"`
}

func (r *JSONRPCResponseStruct) jsonrpcMessage() {}

type JSONRPCError struct {
	JSONRPC string
	ID      string
	Error   fmt.RawJSON
}

func (e *JSONRPCError) jsonrpcMessage() {}

type JSONRPCErrorDetails struct {
	Code    int64
	Message string
	Data    string `json:",omitempty"`
}
