package mcp

// ormc:formonly
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Method  string
	Params  string `json:",omitempty"`
}

// ormc:formonly
type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Result  string `json:",omitempty"`
	Error   string `json:",omitempty"`
}

// ormc:formonly
type ideServerEntry struct {
	URL     string `json:"url"`
	Headers string `json:"headers,omitempty"`
}

// ormc:formonly
type vscodeConfig struct {
	Servers string `json:"servers,omitempty"`
}

// ormc:formonly
type claudeCodeConfig struct {
	MCPServers string `json:"mcpServers,omitempty"`
}

// ormc:formonly
type jsonRPCError struct {
	Code    int64
	Message string
	Data    string `json:",omitempty"`
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
	Capabilities    string             `json:",omitempty"`
}

// ormc:formonly
type callToolParams struct {
	Name      string
	Arguments string `json:",omitempty"`
}

// ormc:formonly
type callToolResult struct {
	IsError bool   `json:"isError,omitempty"`
	Content string
}

// ormc:formonly
type textContent struct {
	Type string
	Text string
}

// ormc:formonly
type toolEntry struct {
	Name        string
	Description string `json:",omitempty"`
	InputSchema string `json:"inputSchema,omitempty"`
}

// ormc:formonly
type listToolsResult struct {
	Tools      string
	NextCursor string `json:"nextCursor,omitempty"`
}

// ormc:formonly
type errorResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      string       `json:"id,omitempty"`
	Error   jsonRPCError
}

// ormc:formonly
type Meta struct {
	ProgressToken string `json:"progressToken,omitempty"`
}

// ormc:formonly
type NotificationParams struct {
	Meta string `json:"_meta,omitempty"`
}
