package mcp

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strconv"
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

/* JSON-RPC types */

// JSONRPCMessage represents either a JSONRPCRequest, JSONRPCNotification, JSONRPCResponse, or JSONRPCError
type JSONRPCMessage any

const LATEST_PROTOCOL_VERSION = "2024-11-05"

var ValidProtocolVersions = []string{
	LATEST_PROTOCOL_VERSION,
}

const JSONRPC_VERSION = "2.0"

type ProgressToken any

type Cursor string

type Meta struct {
	ProgressToken    ProgressToken  `json:"progressToken,omitempty"`
	AdditionalFields map[string]any `json:"-"`
}

func (m *Meta) MarshalJSON() ([]byte, error) {
	raw := make(map[string]any)
	if m.ProgressToken != nil {
		raw["progressToken"] = m.ProgressToken
	}
	maps.Copy(raw, m.AdditionalFields)
	return json.Marshal(raw)
}

func (m *Meta) UnmarshalJSON(data []byte) error {
	raw := make(map[string]any)
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.ProgressToken = raw["progressToken"]
	delete(raw, "progressToken")
	m.AdditionalFields = raw
	return nil
}

func NewMetaFromMap(m map[string]any) *Meta {
	progressToken := m["progressToken"]
	if progressToken != nil {
		delete(m, "progressToken")
	}
	return &Meta{
		ProgressToken:    progressToken,
		AdditionalFields: m,
	}
}

type Request struct {
	Method string        `json:"method"`
	Params RequestParams `json:"params,omitempty"`
}

type RequestParams struct {
	Meta *Meta `json:"_meta,omitempty"`
}

type Notification struct {
	Method string             `json:"method"`
	Params NotificationParams `json:"params,omitempty"`
}

type NotificationParams struct {
	Meta             map[string]any `json:"_meta,omitempty"`
	AdditionalFields map[string]any `json:"-"`
}

func (p NotificationParams) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	if p.Meta != nil {
		m["_meta"] = p.Meta
	}
	for k, v := range p.AdditionalFields {
		if k != "_meta" {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

func (p *NotificationParams) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if p.Meta == nil {
		p.Meta = make(map[string]any)
	}
	if p.AdditionalFields == nil {
		p.AdditionalFields = make(map[string]any)
	}
	for k, v := range m {
		if k == "_meta" {
			if meta, ok := v.(map[string]any); ok {
				p.Meta = meta
			}
		} else {
			p.AdditionalFields[k] = v
		}
	}
	return nil
}

type Result struct {
	Meta *Meta `json:"_meta,omitempty"`
}

type RequestId struct {
	value any
}

func NewRequestId(value any) RequestId {
	return RequestId{value: value}
}

func (r RequestId) Value() any {
	return r.value
}

func (r RequestId) String() string {
	switch v := r.value.(type) {
	case string:
		return "string:" + v
	case int64:
		return "int64:" + strconv.FormatInt(v, 10)
	case float64:
		if v == float64(int64(v)) {
			return "int64:" + strconv.FormatInt(int64(v), 10)
		}
		return "float64:" + strconv.FormatFloat(v, 'f', -1, 64)
	case nil:
		return "<nil>"
	default:
		return "unknown:" + fmt.Sprintf("%v", v)
	}
}

func (r RequestId) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.value)
}

func (r *RequestId) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		r.value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		r.value = s
		return nil
	}
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		if f == float64(int64(f)) {
			r.value = int64(f)
		} else {
			r.value = f
		}
		return nil
	}
	return fmt.Errorf("invalid request id: %s", string(data))
}

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      RequestId   `json:"id"`
	Params  any         `json:"params,omitempty"`
	Header  http.Header `json:"-"`
	Request
}

type JSONRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Params  any    `json:"params,omitempty"`
	Notification
}

type JSONRPCResponse struct {
	JSONRPC string               `json:"jsonrpc"`
	ID      RequestId            `json:"id"`
	Result  any                  `json:"result,omitempty"`
	Error   *JSONRPCErrorDetails `json:"error,omitempty"`
}

type JSONRPCError struct {
	JSONRPC string              `json:"jsonrpc"`
	ID      RequestId           `json:"id"`
	Error   JSONRPCErrorDetails `json:"error"`
}

type JSONRPCErrorDetails struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

const (
	PARSE_ERROR         = -32700
	INVALID_REQUEST     = -32600
	METHOD_NOT_FOUND     = -32601
	INVALID_PARAMS      = -32602
	INTERNAL_ERROR      = -32603
	REQUEST_INTERRUPTED = -32800
)

type EmptyResult Result

type InitializeRequest struct {
	Request
	Params InitializeParams `json:"params"`
	Header http.Header      `json:"-"`
}

type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

type InitializeResult struct {
	Result
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

type ClientCapabilities struct {
	Experimental map[string]any `json:"experimental,omitempty"`
}

type ServerCapabilities struct {
	Experimental map[string]any    `json:"experimental,omitempty"`
	Tools        *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type PingRequest struct {
	Request
	Header http.Header `json:"-"`
}

type PaginatedRequest struct {
	Request
	Params PaginatedParams `json:"params,omitempty"`
}

type PaginatedParams struct {
	Cursor Cursor `json:"cursor,omitempty"`
}

type PaginatedResult struct {
	Result
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

type Content interface {
	isContent()
}

type TextContent struct {
	Annotated
	Meta *Meta  `json:"_meta,omitempty"`
	Type string `json:"type"` // Must be "text"
	Text string `json:"text"`
}

func (TextContent) isContent() {}

type ImageContent struct {
	Annotated
	Meta *Meta  `json:"_meta,omitempty"`
	Type string `json:"type"` // Must be "image"
	Data string `json:"data"`
	MIMEType string `json:"mimeType"`
}

func (ImageContent) isContent() {}

type AudioContent struct {
	Annotated
	Meta *Meta  `json:"_meta,omitempty"`
	Type string `json:"type"` // Must be "audio"
	Data string `json:"data"`
	MIMEType string `json:"mimeType"`
}

func (AudioContent) isContent() {}

type ClientRequest any
type ClientNotification any
type ClientResult any
type ServerRequest any
type ServerNotification any
type ServerResult any
