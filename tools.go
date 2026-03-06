package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
)

var errToolSchemaConflict = errors.New("provide either InputSchema or RawInputSchema, not both")

type ListToolsRequest struct {
	PaginatedRequest
	Header http.Header `json:"-"`
}

type ListToolsResult struct {
	PaginatedResult
	Tools []Tool `json:"tools"`
}

type CallToolResult struct {
	Result
	Content           []Content `json:"content"`
	StructuredContent any       `json:"structuredContent,omitempty"`
	IsError           bool      `json:"isError,omitempty"`
}

type CallToolRequest struct {
	Request
	Header http.Header    `json:"-"`
	Params CallToolParams `json:"params"`
}

type CallToolParams struct {
	Name      string `json:"name"`
	Arguments any    `json:"arguments,omitempty"`
	Meta      *Meta  `json:"_meta,omitempty"`
}

func (r CallToolRequest) GetArguments() map[string]any {
	if args, ok := r.Params.Arguments.(map[string]any); ok {
		return args
	}
	return nil
}

func (r CallToolRequest) GetRawArguments() any {
	return r.Params.Arguments
}

func (r CallToolRequest) BindArguments(target any) error {
	if target == nil || reflect.ValueOf(target).Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a non-nil pointer")
	}
	if raw, ok := r.Params.Arguments.(json.RawMessage); ok {
		return json.Unmarshal(raw, target)
	}
	data, err := json.Marshal(r.Params.Arguments)
	if err != nil {
		return fmt.Errorf("failed to marshal arguments: %w", err)
	}
	return json.Unmarshal(data, target)
}

func (r CallToolRequest) GetString(key string, defaultValue string) string {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func (r CallToolResult) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	if r.Meta != nil {
		m["_meta"] = r.Meta
	}
	m["content"] = r.Content
	if r.StructuredContent != nil {
		m["structuredContent"] = r.StructuredContent
	}
	if r.IsError {
		m["isError"] = r.IsError
	}
	return json.Marshal(m)
}

func (r *CallToolResult) UnmarshalJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if meta, ok := raw["_meta"]; ok {
		if metaMap, ok := meta.(map[string]any); ok {
			r.Meta = NewMetaFromMap(metaMap)
		}
	}
	if contentRaw, ok := raw["content"]; ok {
		if contentArray, ok := contentRaw.([]any); ok {
			r.Content = make([]Content, len(contentArray))
			for i, item := range contentArray {
				itemBytes, _ := json.Marshal(item)
				var typeInfo struct {
					Type string `json:"type"`
				}
				json.Unmarshal(itemBytes, &typeInfo)
				switch typeInfo.Type {
				case "text":
					var c TextContent
					json.Unmarshal(itemBytes, &c)
					r.Content[i] = c
				case "image":
					var c ImageContent
					json.Unmarshal(itemBytes, &c)
					r.Content[i] = c
				case "audio":
					var c AudioContent
					json.Unmarshal(itemBytes, &c)
					r.Content[i] = c
				}
			}
		}
	}
	if structured, ok := raw["structuredContent"]; ok {
		r.StructuredContent = structured
	}
	if isError, ok := raw["isError"]; ok {
		if isErrorBool, ok := isError.(bool); ok {
			r.IsError = isErrorBool
		}
	}
	return nil
}

type ToolListChangedNotification struct {
	Notification
}

type Tool struct {
	Meta            *Meta            `json:"_meta,omitempty"`
	Name            string           `json:"name"`
	Description     string           `json:"description,omitempty"`
	InputSchema     ToolInputSchema  `json:"inputSchema"`
	RawInputSchema  json.RawMessage  `json:"-"`
	OutputSchema    ToolOutputSchema `json:"outputSchema,omitempty"`
	RawOutputSchema json.RawMessage  `json:"-"`
}

func (t Tool) GetName() string {
	return t.Name
}

func (t Tool) MarshalJSON() ([]byte, error) {
	m := make(map[string]any)
	m["name"] = t.Name
	if t.Description != "" {
		m["description"] = t.Description
	}
	if t.RawInputSchema != nil {
		m["inputSchema"] = t.RawInputSchema
	} else {
		m["inputSchema"] = t.InputSchema
	}
	if t.RawOutputSchema != nil {
		m["outputSchema"] = t.RawOutputSchema
	} else if t.OutputSchema.Type != "" {
		m["outputSchema"] = t.OutputSchema
	}
	if t.Meta != nil {
		m["_meta"] = t.Meta
	}
	return json.Marshal(m)
}

type ToolArgumentsSchema struct {
	Type                 string         `json:"type"`
	Properties           map[string]any `json:"properties,omitempty"`
	Required             []string       `json:"required,omitempty"`
	AdditionalProperties any            `json:"additionalProperties,omitempty"`
}

type ToolInputSchema ToolArgumentsSchema
type ToolOutputSchema ToolArgumentsSchema

func NewTool(name string, opts ...ToolOption) Tool {
	tool := Tool{
		Name:        name,
		InputSchema: ToolInputSchema{Type: "object"},
	}
	for _, opt := range opts {
		opt(&tool)
	}
	return tool
}

func (r CallToolRequest) GetInt(key string, defaultValue int) int {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case int: return v
		case float64: return int(v)
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return defaultValue
}

func (r CallToolRequest) GetBool(key string, defaultValue bool) bool {
	args := r.GetArguments()
	if val, ok := args[key]; ok {
		switch v := val.(type) {
		case bool: return v
		case string:
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		}
	}
	return defaultValue
}
