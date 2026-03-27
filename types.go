package mcp

import (
	"github.com/tinywasm/fmt"
)

type MCPMethod string

const (
	MethodInitialize MCPMethod = "initialize"
	MethodPing       MCPMethod = "ping"
	MethodToolsList  MCPMethod = "tools/list"
	MethodToolsCall  MCPMethod = "tools/call"

	MethodNotificationToolsListChanged = "notifications/tools/list_changed"
	MethodNotificationInitialized      = "notifications/initialized"
)

type JSONRPCMessage any

const LATEST_PROTOCOL_VERSION = "2024-11-05"

var validProtocolVersions = []string{
	LATEST_PROTOCOL_VERSION,
}

const JSONRPC_VERSION = "2.0"

type Cursor string

type RequestId = string

type JSONRPCRequest struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      RequestId `json:"id"`
	Method  string    `json:"method"`
	Params  string    `json:"params,omitempty"`
}

func (m *JSONRPCRequest) Schema() []fmt.Field {
	return []fmt.Field{
		{Name: "jsonrpc", Type: fmt.FieldText},
		{Name: "id", Type: fmt.FieldText},
		{Name: "method", Type: fmt.FieldText},
		{Name: "params", Type: fmt.FieldText, OmitEmpty: true},
	}
}

func (m *JSONRPCRequest) Pointers() []any {
	return []any{&m.JSONRPC, &m.ID, &m.Method, &m.Params}
}

type JSONRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  string `json:"params,omitempty"`
}

func (m *JSONRPCNotification) Schema() []fmt.Field {
	return []fmt.Field{
		{Name: "jsonrpc", Type: fmt.FieldText},
		{Name: "method", Type: fmt.FieldText},
		{Name: "params", Type: fmt.FieldText, OmitEmpty: true},
	}
}

func (m *JSONRPCNotification) Pointers() []any {
	return []any{&m.JSONRPC, &m.Method, &m.Params}
}

type JSONRPCResponseStruct struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Result  string `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (m *JSONRPCResponseStruct) Schema() []fmt.Field {
	return []fmt.Field{
		{Name: "jsonrpc", Type: fmt.FieldText},
		{Name: "id", Type: fmt.FieldText, PK: true, OmitEmpty: true},
		{Name: "result", Type: fmt.FieldText, OmitEmpty: true},
		{Name: "error", Type: fmt.FieldText, OmitEmpty: true},
	}
}

func (m *JSONRPCResponseStruct) Pointers() []any {
	return []any{&m.JSONRPC, &m.ID, &m.Result, &m.Error}
}

type JSONRPCError struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id,omitempty"`
	Error   string `json:"error"`
}

func (m *JSONRPCError) Schema() []fmt.Field {
	return []fmt.Field{
		{Name: "jsonrpc", Type: fmt.FieldText},
		{Name: "id", Type: fmt.FieldText, PK: true, OmitEmpty: true},
		{Name: "error", Type: fmt.FieldText},
	}
}

func (m *JSONRPCError) Pointers() []any {
	return []any{&m.JSONRPC, &m.ID, &m.Error}
}

type JSONRPCErrorDetails struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (m *JSONRPCErrorDetails) Schema() []fmt.Field {
	return []fmt.Field{
		{Name: "code", Type: fmt.FieldInt},
		{Name: "message", Type: fmt.FieldText},
		{Name: "data", Type: fmt.FieldText, OmitEmpty: true},
	}
}

func (m *JSONRPCErrorDetails) Pointers() []any {
	return []any{&m.Code, &m.Message, &m.Data}
}

const (
	PARSE_ERROR         = -32700
	INVALID_REQUEST     = -32600
	METHOD_NOT_FOUND     = -32601
	INVALID_PARAMS      = -32602
	INTERNAL_ERROR      = -32603
	REQUEST_INTERRUPTED = -32800
)

type PaginatedParams struct {
	Cursor Cursor `json:"cursor,omitempty"`
}

func (m *PaginatedParams) Schema() []fmt.Field {
	return []fmt.Field{
		{Name: "cursor", Type: fmt.FieldText, OmitEmpty: true},
	}
}

func (m *PaginatedParams) Pointers() []any {
	return []any{&m.Cursor}
}

type PaginatedResult struct {
	NextCursor Cursor `json:"nextCursor,omitempty"`
}

type Named interface {
	GetName() string
}

type Annotated struct {
	Annotations *Annotations `json:"annotations,omitempty"`
}

type Annotations struct {
	Audience     []string `json:"audience,omitempty"`
	Priority     *float64 `json:"priority,omitempty"`
	LastModified string   `json:"lastModified,omitempty"`
}

type EmptyResult struct{}

func (m *EmptyResult) Schema() []fmt.Field { return nil }
func (m *EmptyResult) Pointers() []any    { return nil }
