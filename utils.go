package mcp

import (
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
)

func JSON(data fmt.Fielder) (*Result, error) {
	var s string
	if err := json.Encode(data, &s); err != nil {
		return nil, err
	}
	return &Result{Content: s}, nil
}

func ParseResult(raw []byte) (*Result, error) {
	var result Result
	if err := json.Decode(raw, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func GetText(r *Result) (string, error) {
	var c TextContent
	if err := json.Decode([]byte(r.Content), &c); err != nil {
		return "", err
	}
	return c.Text, nil
}

func newResultResponse(id RequestId, result any) JSONRPCMessage {
	var resJSON string
	if f, ok := result.(fmt.Fielder); ok {
		json.Encode(f, &resJSON)
	}
	return &JSONRPCResponseStruct{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Result:  resJSON,
	}
}

func newErrorDetails(code int, message string, data any) *JSONRPCErrorDetails {
	var dataJSON string
	if f, ok := data.(fmt.Fielder); ok {
		json.Encode(f, &dataJSON)
	}
	return &JSONRPCErrorDetails{
		Code:    int64(code),
		Message: message,
		Data:    dataJSON,
	}
}

func newErrorResponse(id RequestId, code int, message string, data any) JSONRPCMessage {
	det := newErrorDetails(code, message, data)
	var detJSON string
	json.Encode(det, &detJSON)
	return &JSONRPCError{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Error:   detJSON,
	}
}
