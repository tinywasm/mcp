package mcp

// ormc:formonly
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  string `json:"params"` // pre-serialized JSON string
}

// ormc:formonly
type rpcResponse struct {
	Result string `json:"result"` // raw JSON string from server
}
