# Stage 8 — utils.go

## 8.1 — JSON helper: fmt.Fielder → Result

```go
// Before
func NewToolResultJSON[T any](data T) (*CallToolResult, error)

// After
func JSON(data fmt.Fielder) (*Result, error) {
    var s string
    if err := json.Encode(data, &s); err != nil {
        return nil, err
    }
    return &Result{Content: s}, nil
}
```

## 8.2 — ParseResult: json.RawMessage → []byte

```go
// Before
func ParseCallToolResult(raw *json.RawMessage) (*CallToolResult, error)

// After
func ParseResult(raw []byte) (*Result, error) {
    var result callToolResult
    if err := json.Decode(raw, &result); err != nil {
        return nil, err
    }
    return &Result{Content: result.Content, IsError: result.IsError}, nil
}
```

## 8.3 — GetText: extract text from Result

```go
// Before
func GetTextFromContent(content []Content) string
func ExtractString(m map[string]any, key string) string

// After
func GetText(r *Result) (string, error) {
    var c textContent
    if err := json.Decode([]byte(r.Content), &c); err != nil {
        return "", err
    }
    return c.Text, nil
}
```

## 8.4 — Delete parallel Content constructors

```go
// DELETE — replaced by mcp.Text() / mcp.JSON() in Stage 6
func NewTextContent(text string) TextContent
func NewImageContent(data, mime string) ImageContent
func NewAudioContent(data, mime string) AudioContent
func AsTextContent(c Content) (TextContent, bool)
func AsImageContent(c Content) (ImageContent, bool)
func AsAudioContent(c Content) (AudioContent, bool)
```

`NewToolResultImage` moves to `utils.back.go` (`//go:build !wasm`) and is renamed:
```go
func Image(text, base64Data, mimeType string) *Result
```

## 8.5 — Delete JSONRPC response constructors

```go
// DELETE or UNEXPORT — internal protocol detail, not caller-facing
func NewJSONRPCResultResponse(id, result) → unexport to newResultResponse
func NewJSONRPCErrorDetails(code, msg)    → unexport to newErrorDetails
func NewJSONRPCError(id, code, msg)       → unexport to newErrorResponse
```

## 8.6 — Remove imports

Delete: `"encoding/json"`, `"fmt"` (stdlib)
