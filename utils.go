package mcp

import (
	"encoding/json"
	"fmt"
)


// Helper functions for type assertions

// asType attempts to cast the given interface to the given type
func asType[T any](content any) (*T, bool) {
	tc, ok := content.(T)
	if !ok {
		return nil, false
	}
	return &tc, true
}

// AsTextContent attempts to cast the given interface to TextContent
func AsTextContent(content any) (*TextContent, bool) {
	return asType[TextContent](content)
}

// AsImageContent attempts to cast the given interface to ImageContent
func AsImageContent(content any) (*ImageContent, bool) {
	return asType[ImageContent](content)
}

// AsAudioContent attempts to cast the given interface to AudioContent
func AsAudioContent(content any) (*AudioContent, bool) {
	return asType[AudioContent](content)
}

// Helper function for JSON-RPC

// NewJSONRPCResponse creates a new JSONRPCResponse with the given id and result.
// NOTE: This function expects a Result struct, but JSONRPCResponse.Result is typed as `any`.
// The Result struct wraps the actual result data with optional metadata.
// For direct result assignment, use NewJSONRPCResultResponse instead.
func NewJSONRPCResponse(id RequestId, result Result) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Result:  result,
	}
}

// NewJSONRPCResultResponse creates a new JSONRPCResponse with the given id and result.
// This function accepts any type for the result, matching the JSONRPCResponse.Result field type.
func NewJSONRPCResultResponse(id RequestId, result any) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Result:  result,
	}
}

// NewJSONRPCErrorDetails creates a new JSONRPCErrorDetails with the given code, message, and data.
func NewJSONRPCErrorDetails(code int, message string, data any) JSONRPCErrorDetails {
	return JSONRPCErrorDetails{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewJSONRPCError creates a new JSONRPCResponse with the given id, code, and message
func NewJSONRPCError(
	id RequestId,
	code int,
	message string,
	data any,
) JSONRPCError {
	return JSONRPCError{
		JSONRPC: JSONRPC_VERSION,
		ID:      id,
		Error:   NewJSONRPCErrorDetails(code, message, data),
	}
}

// NewTextContent
// Helper function to create a new TextContent
func NewTextContent(text string) TextContent {
	return TextContent{
		Type: "text",
		Text: text,
	}
}

// NewImageContent
// Helper function to create a new ImageContent
func NewImageContent(data, mimeType string) ImageContent {
	return ImageContent{
		Type:     "image",
		Data:     data,
		MIMEType: mimeType,
	}
}

// Helper function to create a new AudioContent
func NewAudioContent(data, mimeType string) AudioContent {
	return AudioContent{
		Type:     "audio",
		Data:     data,
		MIMEType: mimeType,
	}
}

// NewToolResultText creates a new CallToolResult with a text content
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

// NewToolResultJSON creates a new CallToolResult with a JSON content.
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

// NewToolResultStructured creates a new CallToolResult with structured content.
// It includes both the structured content and a text representation for backward compatibility.
func NewToolResultStructured(structured any, fallbackText string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: fallbackText,
			},
		},
		StructuredContent: structured,
	}
}

// NewToolResultStructuredOnly creates a new CallToolResult with structured
// content and creates a JSON string fallback for backwards compatibility.
// This is useful when you want to provide structured data without any specific text fallback.
func NewToolResultStructuredOnly(structured any) *CallToolResult {
	var fallbackText string
	// Convert to JSON string for backward compatibility
	jsonBytes, err := json.Marshal(structured)
	if err != nil {
		fallbackText = fmt.Sprintf("Error serializing structured content: %v", err)
	} else {
		fallbackText = string(jsonBytes)
	}

	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: fallbackText,
			},
		},
		StructuredContent: structured,
	}
}

// NewToolResultImage creates a new CallToolResult with both text and image content
func NewToolResultImage(text, imageData, mimeType string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
			ImageContent{
				Type:     "image",
				Data:     imageData,
				MIMEType: mimeType,
			},
		},
	}
}

// NewToolResultAudio creates a new CallToolResult with both text and audio content
func NewToolResultAudio(text, audioData, mimeType string) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: text,
			},
			AudioContent{
				Type:     "audio",
				Data:     audioData,
				MIMEType: mimeType,
			},
		},
	}
}

// NewToolResultError creates a new CallToolResult with an error message.
// Any errors that originate from the tool SHOULD be reported inside the result object.
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

// NewToolResultErrorFromErr creates a new CallToolResult with an error message.
// If an error is provided, its details will be appended to the text message.
// Any errors that originate from the tool SHOULD be reported inside the result object.
func NewToolResultErrorFromErr(text string, err error) *CallToolResult {
	if err != nil {
		text = fmt.Sprintf("%s: %v", text, err)
	}
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

// NewToolResultErrorf creates a new CallToolResult with an error message.
// The error message is formatted using the fmt package.
// Any errors that originate from the tool SHOULD be reported inside the result object.
func NewToolResultErrorf(format string, a ...any) *CallToolResult {
	return &CallToolResult{
		Content: []Content{
			TextContent{
				Type: "text",
				Text: fmt.Sprintf(format, a...),
			},
		},
		IsError: true,
	}
}

// NewListToolsResult creates a new ListToolsResult
func NewListToolsResult(tools []Tool, nextCursor Cursor) *ListToolsResult {
	return &ListToolsResult{
		PaginatedResult: PaginatedResult{
			NextCursor: nextCursor,
		},
		Tools: tools,
	}
}

// NewInitializeResult creates a new InitializeResult
func NewInitializeResult(
	protocolVersion string,
	capabilities ServerCapabilities,
	serverInfo Implementation,
	instructions string,
) *InitializeResult {
	return &InitializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities:    capabilities,
		ServerInfo:      serverInfo,
		Instructions:    instructions,
	}
}

// FormatNumberResult
// Helper for formatting numbers in tool results
func FormatNumberResult(value float64) *CallToolResult {
	return NewToolResultText(fmt.Sprintf("%.2f", value))
}

func ExtractString(data map[string]any, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func ExtractMap(data map[string]any, key string) map[string]any {
	if value, ok := data[key]; ok {
		if m, ok := value.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func ParseContent(contentMap map[string]any) (Content, error) {
	contentType := ExtractString(contentMap, "type")

	var annotations *Annotations

	switch contentType {
	case "text":
		text := ExtractString(contentMap, "text")
		c := NewTextContent(text)
		c.Annotations = annotations
		return c, nil

	case "image":
		data := ExtractString(contentMap, "data")
		mimeType := ExtractString(contentMap, "mimeType")
		if data == "" || mimeType == "" {
			return nil, fmt.Errorf("image data or mimeType is missing")
		}
		c := NewImageContent(data, mimeType)
		c.Annotations = annotations
		return c, nil

	case "audio":
		data := ExtractString(contentMap, "data")
		mimeType := ExtractString(contentMap, "mimeType")
		if data == "" || mimeType == "" {
			return nil, fmt.Errorf("audio data or mimeType is missing")
		}
		c := NewAudioContent(data, mimeType)
		c.Annotations = annotations
		return c, nil

	}

	return nil, fmt.Errorf("unsupported content type: %s", contentType)
}

func ParseCallToolResult(rawMessage *json.RawMessage) (*CallToolResult, error) {
	if rawMessage == nil {
		return nil, fmt.Errorf("response is nil")
	}

	var jsonContent map[string]any
	if err := json.Unmarshal(*rawMessage, &jsonContent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	var result CallToolResult

	meta, ok := jsonContent["_meta"]
	if ok {
		if metaMap, ok := meta.(map[string]any); ok {
			result.Meta = NewMetaFromMap(metaMap)
		}
	}

	isError, ok := jsonContent["isError"]
	if ok {
		if isErrorBool, ok := isError.(bool); ok {
			result.IsError = isErrorBool
		}
	}

	contents, ok := jsonContent["content"]
	if !ok {
		return nil, fmt.Errorf("content is missing")
	}

	contentArr, ok := contents.([]any)
	if !ok {
		return nil, fmt.Errorf("content is not an array")
	}

	for _, content := range contentArr {
		// Extract content
		contentMap, ok := content.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("content is not an object")
		}

		// Process content
		content, err := ParseContent(contentMap)
		if err != nil {
			return nil, err
		}

		result.Content = append(result.Content, content)
	}

	// Handle structured content
	structuredContent, ok := jsonContent["structuredContent"]
	if ok {
		result.StructuredContent = structuredContent
	}

	return &result, nil
}

func ParseArgument(request CallToolRequest, key string, defaultVal any) any {
	args := request.GetArguments()
	if _, ok := args[key]; !ok {
		return defaultVal
	} else {
		return args[key]
	}
}

// ToBoolPtr returns a pointer to the given boolean value
func ToBoolPtr(b bool) *bool {
	return &b
}

// ToInt64Ptr returns a pointer to the given int64 value
func ToInt64Ptr(i int64) *int64 {
	return &i
}

// GetTextFromContent extracts text from a Content interface that might be a TextContent struct
// or a map[string]any that was unmarshaled from JSON. This is useful when dealing with content
// that comes from different transport layers that may handle JSON differently.
//
// This function uses fallback behavior for non-text content - it returns a string representation
// via fmt.Sprintf for any content that cannot be extracted as text. This is a lossy operation
// intended for convenience in logging and display scenarios.
//
// For strict type validation, use ParseContent() instead, which returns an error for invalid content.
func GetTextFromContent(content any) string {
	switch c := content.(type) {
	case TextContent:
		return c.Text
	case map[string]any:
		// Handle JSON unmarshaled content
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
