package mcp

import (
	"encoding/json"
	"fmt"
)

// ClientRequest types
var (
	_ ClientRequest = (*PingRequest)(nil)
	_ ClientRequest = (*InitializeRequest)(nil)
	_ ClientRequest = (*CallToolRequest)(nil)
	_ ClientRequest = (*ListToolsRequest)(nil)
)

// ClientNotification types
var (
	_ ClientNotification = (*JSONRPCNotification)(nil)
)

// ClientResult types
var (
	_ ClientResult = (*EmptyResult)(nil)
)

// ServerRequest types
var (
	_ ServerRequest = (*PingRequest)(nil)
)

// ServerNotification types
var (
	_ ServerNotification = (*JSONRPCNotification)(nil)
)

// ServerResult types
var (
	_ ServerResult = (*EmptyResult)(nil)
	_ ServerResult = (*InitializeResult)(nil)
	_ ServerResult = (*CallToolResult)(nil)
	_ ServerResult = (*ListToolsResult)(nil)
)

// Helper functions for type assertions

func asType[T any](content any) (*T, bool) {
	tc, ok := content.(T)
	if !ok {
		return nil, false
	}
	return &tc, true
}

func AsTextContent(content any) (*TextContent, bool) {
	return asType[TextContent](content)
}

func AsImageContent(content any) (*ImageContent, bool) {
	return asType[ImageContent](content)
}

func AsAudioContent(content any) (*AudioContent, bool) {
	return asType[AudioContent](content)
}

// Helper function for JSON-RPC

func NewJSONRPCResultResponse(id RequestId, result any) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Result:  result,
	}
}

func NewJSONRPCErrorDetails(code int, message string, data any) JSONRPCErrorDetails {
	return JSONRPCErrorDetails{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func NewJSONRPCError(id RequestId, code int, message string, data any) JSONRPCError {
	return JSONRPCError{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Error:   NewJSONRPCErrorDetails(code, message, data),
	}
}

func NewTextContent(text string) TextContent {
	return TextContent{
		Type: "text",
		Text: text,
	}
}

func NewImageContent(data, mimeType string) ImageContent {
	return ImageContent{
		Type:     "image",
		Data:     data,
		MIMEType: mimeType,
	}
}

func NewAudioContent(data, mimeType string) AudioContent {
	return AudioContent{
		Type:     "audio",
		Data:     data,
		MIMEType: mimeType,
	}
}

func NewToolResultText(text string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
		},
	}
}

func NewToolResultJSON[T any](data T) (*CallToolResult, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal JSON: %w", err)
	}
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: string(b),
			},
		},
		StructuredContent: data,
	}, nil
}

func NewToolResultError(text string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
		},
		IsError: true,
	}
}

func ParseCallToolResult(rawMessage *json.RawMessage) (*CallToolResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}
	var result CallToolResult
	if err := json.Unmarshal(*rawMessage, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &result, nil
}

func ExtractString(data map[string]any, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func GetTextFromContent(content any) string {
	switch c := content.(type) {
	case TextContent:
		return c.Text
	case map[string]any:
		if contentType, exists := c["type"]; exists && contentType == "text" {
			if text, exists := c["text"].(string); exists {
				return text
			}
		}
		return fmt.Sprintf("%v", content)
	case string:
		return c
	default:
		return fmt.Sprintf("%v", content)
	}
}
