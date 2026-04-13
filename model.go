package mcp

// ormc:formonly
type rpcRequest struct {
	JSONRPC string
	ID      string `json:",omitempty"`
	Method  string
	Params  string `json:",omitempty"`
}

// ormc:formonly
type rpcResponse struct {
	JSONRPC string
	ID      string `json:",omitempty"`
	Result  string `json:",omitempty"`
	Error   string `json:",omitempty"`
}

// ormc:formonly
type jsonRPCError struct {
	Code    int64
	Message string
	Data    string `json:",omitempty"`
}

// ormc:formonly
type initializeParams struct {
	ProtocolVersion string
	ClientInfo      implementationInfo
}

// ormc:formonly
type implementationInfo struct {
	Name    string
	Version string
}

// ormc:formonly
type initializeResult struct {
	ProtocolVersion string
	ServerInfo      implementationInfo
	Capabilities    string `json:",omitempty"`
}

// ormc:formonly
type CallToolParams struct {
	Name      string
	Arguments string `json:",omitempty"`
}

// ormc:formonly
type Result struct {
	IsError bool `json:",omitempty"`
	Content string
}

// ormc:formonly
type TextContent struct {
	Type string
	Text string
}

// ormc:formonly
type toolEntry struct {
	Name        string
	Description string `json:",omitempty"`
	InputSchema string `json:",omitempty"`
}

// ormc:formonly
type listToolsResult struct {
	Tools      string
	NextCursor string `json:",omitempty"`
}

// ormc:formonly
type errorResponse struct {
	JSONRPC string
	ID      string `json:",omitempty"`
	Error   jsonRPCError
}

// ormc:formonly
type Meta struct {
	ProgressToken string `json:",omitempty"`
}

// ormc:formonly
type NotificationParams struct {
	Meta string `json:",omitempty"`
}

// ormc:formonly
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
	ID      string `json:",omitempty"`
	Result  string `json:",omitempty"`
	Error   string `json:",omitempty"`
}

func (r *JSONRPCResponseStruct) jsonrpcMessage() {}

type JSONRPCError struct {
	JSONRPC string
	ID      string `json:",omitempty"`
	Error   string
}

func (e *JSONRPCError) jsonrpcMessage() {}

type JSONRPCErrorDetails struct {
	Code    int64
	Message string
	Data    string `json:",omitempty"`
}
