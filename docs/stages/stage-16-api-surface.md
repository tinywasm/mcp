# Stage 16 — API surface reduction

Goal: from ~186 exported symbols to the minimum needed to use the library.

## 16.1 — Delete marker types (zero value, zero use)

```go
// DELETE — empty types, no runtime value
type ClientRequest      struct{}
type ClientNotification struct{}
type ClientResult       struct{}
type ServerRequest      struct{}
type ServerNotification struct{}
type ServerResult       struct{}
```

## 16.2 — Delete tool_builders.go entirely

Builder DSL for the old `ProtocolTool` — obsolete with `Tool.InputSchema string` (ormc).

Delete all: `ToolOption`, `PropertyOption`, `WithDescription`, `WithString`,
`WithNumber`, `WithBoolean`, `Required`, `Description`, `Enum`, `DefaultString`,
`DefaultNumber`.

## 16.3 — Delete Content interface + implementations

`Result.Content` is a `string` (JSON encoded). Callers use `mcp.Text()` / `mcp.JSON()`.

Delete: `Content` interface, `TextContent`, `ImageContent`, `AudioContent`,
`AsTextContent`, `AsImageContent`, `AsAudioContent`, `Annotated`, `Annotations`.

## 16.4 — Delete map-based helpers

All replaced by `req.Bind(&args)`:

```go
// DELETE
func NewMetaFromMap(m map[string]any) Meta
func ExtractString(m map[string]any, key string) string
func GetTextFromContent(content []Content) string
```

## 16.5 — Delete or unexport duplicate/internal symbols

```go
// DELETE — replaced by Tool (Stage 6)
type ProtocolTool struct{}
func NewProtocolTool(...)
type ToolExecutor func(args map[string]any)
type Parameter struct{}

// UNEXPORT
type ServerTool → serverTool
```

## 16.6 — Delete auth implementations from mcp

Move to `tinywasm/user`:

```go
// DELETE from mcp
func OpenAuthorizer() Authorizer
func NewTokenAuthorizer(token string) Authorizer
```

## 16.7 — Unexport ValidProtocolVersions

```go
var ValidProtocolVersions → var validProtocolVersions
```

## 16.8 — Unify MCPServer + Handler into Server

```go
type Server struct { ... }
func NewServer(config Config, providers []ToolProvider) *Server
func (s *Server) AddTool(tool Tool) error
func (s *Server) HTTPHandler() http.Handler  // .back.go only
```

`MCPServer` and `Handler` disappear as public types.
`AddTool` validates: `Resource`, `Action`, and `Run` must be non-empty/non-nil.

## 16.9 — Rename exported types for clarity

| Before | After | Reason |
|--------|-------|--------|
| `CallToolRequest` | `Request` | used in every tool handler, context is obvious |
| `CallToolResult` | `Result` | same |
| `ToolHandlerFunc` | deleted | replaced by `Tool.Run` |
| `ToolFilterFunc` | `FilterFunc` | shorter |
| `ToolProvider` | `ToolProvider` | stays — already clear |
| `NewToolResultText` | `Text` | `mcp.Text("ok")` is self-explanatory |
| `NewToolResultJSON` | `JSON` | `mcp.JSON(data)` |
| `NewToolResultImage` | `Image` | `mcp.Image(text, b64, mime)` — .back.go only |
| `ParseCallToolResult` | `ParseResult` | shorter |
| `GetTextFromResult` | `GetText` | shorter |
| `BindArguments` | `Bind` | "arguments" is redundant |

## Expected result

| Category | Before | After |
|----------|--------|-------|
| Total exported symbols | ~186 | ~45 |
| Types | ~53 | ~12 |
| Functions/constructors | ~47 | ~10 |
| Interfaces | ~15 | ~6 |
| Constants | ~17 | ~12 |
