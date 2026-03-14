package mcp

// rpcResponse is the JSON-RPC 2.0 response envelope.
// Struct avoids map allocations (WASM binary size concern).
type rpcResponse struct {
	Result any `json:"result"`
}

// rpcRequest is the JSON-RPC 2.0 request envelope.
// Struct avoids map allocations (WASM binary size concern).
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}
