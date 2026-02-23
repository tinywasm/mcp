package mcp

// NewJSONRPCErrorResponse creates a new JSONRPCResponse with an error.
func NewJSONRPCErrorResponse(id RequestId, code int, message string, data any) *JSONRPCResponse {
	details := NewJSONRPCErrorDetails(code, message, data)
	return &JSONRPCResponse{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Error:   &details,
	}
}
