package mcp

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
	Result  string
	Error   string
}

// ormc:formonly
type jsonRPCError struct {
	Code    int64
	Message string
	Data    string
}

// ormc:formonly
type initializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      implementationInfo `json:"clientInfo"`
}

// ormc:formonly
type implementationInfo struct {
	Name    string
	Version string
}

// ormc:formonly
type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      implementationInfo `json:"serverInfo"`
	Capabilities    string             `json:"omitempty"`
}

// ormc:formonly
type CallToolParams struct {
	Name      string
	Arguments string `json:"omitempty"`
}

// ormc:formonly
type Result struct {
	IsError bool   `json:"isError,omitempty"`
	Content string `json:"raw"`
}

// ormc:formonly
type TextContent struct {
	Type string
	Text string
}

// ormc:formonly
type toolEntry struct {
	Name        string
	Description string `json:"omitempty"`
	InputSchema string `json:"inputSchema,omitempty"`
}

// ormc:formonly
type listToolsResult struct {
	Tools      string `json:"raw"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// ormc:formonly
type errorResponse struct {
	JSONRPC string
	ID      string `json:"omitempty"`
	Error   jsonRPCError
}

// ormc:formonly
type Meta struct {
	ProgressToken string `json:"progressToken,omitempty"`
}

// ormc:formonly
type NotificationParams struct {
	Meta string `json:"omitempty"`
}

// ormc:formonly
type EmptyResult struct {
	Result string `json:"omitempty"`
}

// ormc:formonly
type JSONRPCRequest struct {
	JSONRPC string
	ID      RequestId
	Method  string
	Params  string `json:"omitempty"`
}

// ormc:formonly
type JSONRPCNotification struct {
	JSONRPC string
	Method  string
	Params  string `json:"omitempty"`
}

// ormc:formonly
type JSONRPCResponseStruct struct {
	JSONRPC string
	ID      string
	Result  string
	Error   string
}

func (r *JSONRPCResponseStruct) jsonrpcMessage() {}

// ormc:formonly
type JSONRPCError struct {
	JSONRPC string
	ID      string `json:"omitempty"`
	Error   string `json:"raw"`
}

func (e *JSONRPCError) jsonrpcMessage() {}

// ormc:formonly
type JSONRPCErrorDetails struct {
	Code    int64
	Message string
	Data    string `json:"omitempty"`
}
