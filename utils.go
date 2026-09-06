package mcp

import (
	"webtyp.com/model"
	"webtyp.com/fmt"
	"webtyp.com/json"
)

func JSON(data model.Encodable) (*Result, error) {
	var s string
	if err := json.Encode(data, &s); err != nil {
		return nil, err
	}
	if len(s) > 0 && s[0] != '[' {
		s = "[" + s + "]"
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
	var list TextContentList
	if err := json.Decode([]byte(r.Content), &list); err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", fmt.Err("mcp", "empty content array")
	}
	return list[0].Text, nil
}

func newResultResponse(id RequestId, result any) JSONRPCMessage {
	var resJSON []byte
	if f, ok := result.(model.Encodable); ok {
		json.Encode(f, &resJSON)
	}
	return &JSONRPCResponseStruct{
		Jsonrpc: JSONRPC_VERSION,
		Id:      id,
		Result:  model.RawJSON(resJSON),
	}
}

func newErrorDetails(code int, message string, data any) *JSONRPCErrorDetails {
	var dataJSON []byte
	if f, ok := data.(model.Encodable); ok {
		json.Encode(f, &dataJSON)
	}
	return &JSONRPCErrorDetails{
		Code:    int64(code),
		Message: message,
		Data:    string(dataJSON),
	}
}

func newErrorResponse(id RequestId, code int, message string, data any) JSONRPCMessage {
	det := newErrorDetails(code, message, data)
	var detJSON []byte
	json.Encode(det, &detJSON)
	return &JSONRPCError{
		Jsonrpc: JSONRPC_VERSION,
		Id:      id,
		Error:   model.RawJSON(detJSON),
	}
}
