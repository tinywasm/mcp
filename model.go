package mcp

// ormc:formonly
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  string `json:"params,omitempty"`
}

// ormc:formonly
type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Result  string `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ormc:formonly
type jsonRPCError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// ormc:formonly
type initializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ClientInfo      implementationInfo `json:"clientInfo"`
}

// ormc:formonly
type implementationInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ormc:formonly
type initializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	ServerInfo      implementationInfo `json:"serverInfo"`
	Capabilities    string             `json:"capabilities,omitempty"`
}

// ormc:formonly
type CallToolParams struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments,omitempty"`
}

// ormc:formonly
type Result struct {
	IsError bool   `json:"isError,omitempty"`
	Content string `json:"content"`
}

// ormc:formonly
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ormc:formonly
type toolEntry struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema string `json:"inputSchema,omitempty"`
}

// ormc:formonly
type listToolsResult struct {
	Tools      string `json:"tools"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// ormc:formonly
type errorResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      string       `json:"id,omitempty"`
	Error   jsonRPCError `json:"error"`
}

// ormc:formonly
type Meta struct {
	ProgressToken string `json:"progressToken,omitempty"`
}

// ormc:formonly
type NotificationParams struct {
	Meta string `json:"meta,omitempty"`
}

// ormc:formonly
type EmptyResult struct {
	Result string `json:"result,omitempty"`
}

type JSONRPCRequest struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      RequestId `json:"id"`
	Method  string    `json:"method"`
	Params  string    `json:"params,omitempty"`
}

type JSONRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  string `json:"params,omitempty"`
}

type JSONRPCResponseStruct struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Result  string `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (r *JSONRPCResponseStruct) jsonrpcMessage() {}

type JSONRPCError struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error"`
}

func (e *JSONRPCError) jsonrpcMessage() {}

type JSONRPCErrorDetails struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}
