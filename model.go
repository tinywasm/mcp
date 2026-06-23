package mcp

import "github.com/tinywasm/fmt"

// ormc:formonly
type rpcRequest struct {
	JSONRPC string
	ID      string
	Method  string
	Params  string
}

// ormc:formonly
type rpcResponse struct {
	JSONRPC string
	ID      string
	Result  fmt.RawJSON
	Error   string `omitempty:"true"`
}

// ormc:formonly
type jsonRPCError struct {
	Code    int64
	Message string
	Data    string
}

// ormc:formonly
type initializeParams struct {
	ProtocolVersion string             ``
	ClientInfo      implementationInfo ``
}

// ormc:formonly
type implementationInfo struct {
	Name    string
	Version string
}

// ormc:formonly
type initializeResult struct {
	ProtocolVersion string             ``
	ServerInfo      implementationInfo ``
	Capabilities    fmt.RawJSON
}

// ormc:formonly
type CallToolParams struct {
	Name      string
	Arguments fmt.RawJSON ``
}

// ormc:formonly
type Result struct {
	IsError bool        ``
	Content fmt.RawJSON
}

// ormc:formonly
type TextContent struct {
	Type string
	Text string
}

// ormc:formonly
type toolEntry struct {
	Name        string
	Description string ``
	InputSchema fmt.RawJSON ``
}

// ormc:formonly
type listToolsResult struct {
	Tools      fmt.RawJSON
	NextCursor string ``
}

// ormc:formonly
type errorResponse struct {
	JSONRPC string
	ID      string ``
	Error   jsonRPCError
}

// ormc:formonly
type Meta struct {
	ProgressToken string ``
}

// ormc:formonly
type NotificationParams struct {
	Meta string ``
}

// ormc:formonly
type EmptyResult struct {
	Result string ``
}

// ormc:formonly
type JSONRPCRequest struct {
	JSONRPC string
	ID      RequestId
	Method  string
	Params  string ``
}

// ormc:formonly
type JSONRPCNotification struct {
	JSONRPC string
	Method  string
	Params  string ``
}

// ormc:formonly
type JSONRPCResponseStruct struct {
	JSONRPC string
	ID      string
	Result  fmt.RawJSON `omitempty:"true"`
	Error   fmt.RawJSON `omitempty:"true"`
}

func (r *JSONRPCResponseStruct) jsonrpcMessage() {}

// ormc:formonly
type JSONRPCError struct {
	JSONRPC string
	ID      string
	Error   fmt.RawJSON
}

func (e *JSONRPCError) jsonrpcMessage() {}

// ormc:formonly
type JSONRPCErrorDetails struct {
	Code    int64
	Message string
	Data    string ``
}
