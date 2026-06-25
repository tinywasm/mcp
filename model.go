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
	Result  fmt.RawJSON `omitempty:"true"`
	Error   string      `omitempty:"true"`
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
	Capabilities    fmt.RawJSON        `omitempty:"true"`
}

// ormc:formonly
type CallToolParams struct {
	Name      string
	Arguments fmt.RawJSON ``
}

// ormc:formonly
type Result struct {
	IsError bool        `omitempty:"true"`
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
	Description string `omitempty:"true"`
	InputSchema fmt.RawJSON ``
}

// ormc:formonly
type listToolsResult struct {
	Tools      fmt.RawJSON
	NextCursor string `omitempty:"true"`
}

// ormc:formonly
type errorResponse struct {
	JSONRPC string
	ID      string `omitempty:"true"`
	Error   jsonRPCError
}

// ormc:formonly
type Meta struct {
	ProgressToken string `omitempty:"true"`
}

// ormc:formonly
type NotificationParams struct {
	Meta string `omitempty:"true"`
}

// ormc:formonly
type EmptyResult struct {
	Result string `omitempty:"true"`
}

// ormc:formonly
type JSONRPCRequest struct {
	JSONRPC string
	ID      RequestId
	Method  string
	Params  string `omitempty:"true"`
}

// ormc:formonly
type JSONRPCNotification struct {
	JSONRPC string
	Method  string
	Params  string `omitempty:"true"`
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
	Data    string `omitempty:"true"`
}
