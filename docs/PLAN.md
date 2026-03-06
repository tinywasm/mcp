# Plan: Absorb `mcpserve` into `tinywasm/mcp`

## Development Rules

- **SRP:** Every file must have a single, well-defined purpose reflected in its name.
- **Flat Hierarchy:** No subdirectories. All new files go in the package root.
- **Max 500 lines:** Split files by domain if limit is exceeded.
- **Standard Library Only:** No external assertion libs in tests.
- **DI (Dependency Injection):** `tinywasm/mcp` MUST NOT import `tinywasm/sse` directly.
  The SSE transport is defined as an interface in `tinywasm/mcp` and injected by the consumer (`app`).
  The consumer uses `tinywasm/sse` to create the concrete implementation and passes it in.
- **No Global State:** Avoid direct system calls in logic; use injected interfaces.

---

## Objective

Replace `github.com/tinywasm/mcpserve` across all consumers (`app`, `devtui`, `devbrowser`) with
`github.com/tinywasm/mcp`. All functionality currently in `mcpserve` must be absorbed into `tinywasm/mcp`
as a flat set of new files. Once consumers are updated, `mcpserve` becomes a dead package.

---

## Gap Analysis: What `tinywasm/mcp` is Missing

`mcpserve` imports `github.com/tinywasm/mcp/server` and `github.com/tinywasm/mcp/mcp`
(sub-packages of the **published** v0.0.6 that no longer exist locally). The local `tinywasm/mcp`
is a flat single package. The following capabilities must be added:

| Gap | Source in `mcpserve` | New file in `tinywasm/mcp` |
|-----|----------------------|---------------------------|
| `NewToolResultImage` helper | `executor.go:72` | `utils.go` (extend) |
| Tool builder API (`ToolOption`, `WithString`, etc.) | `tools.go:buildMCPTool()` | `tool_builders.go` |
| HTTP server transport (`StreamableHTTPServer`) | `handler.go:188` | `server_http.go` |
| Provider types (`ToolProvider`, `ToolMetadata`, `ToolExecutor`, `Loggable`, `BinaryData`) | `tools.go` | `provider.go` |
| SSE publisher interface (DI boundary) | `handler.go:sseHub` | `handler.go` |
| Handler (HTTP server + injected SSE + endpoints) | `handler.go` | `handler.go` |
| Tool execution adapter (`mcpExecuteTool`) | `executor.go` | `handler_executor.go` |
| IDE auto-configuration | `ide.go` + `config.go` | `handler_ide.go` |

---

## Step 1 — Extend `utils.go`: Add `NewToolResultImage`

Add the missing helper function. `mcpserve/executor.go` calls `mcp.NewToolResultImage(text, base64Data, mimeType)`.

```go
func NewToolResultImage(text, base64Data, mimeType string) *CallToolResult {
    return &CallToolResult{
        Content: []Content{
            TextContent{Type: "text", Text: text},
            ImageContent{Type: "image", Data: base64Data, MIMEType: mimeType},
        },
    }
}
```

---

## Step 2 — New file `tool_builders.go`: Tool Option Helpers

These helpers allow building `Tool` and its `ToolInputSchema` with a functional-options pattern.
`mcpserve/tools.go:buildMCPTool()` depends on them.

```go
package mcp

// ToolOption configures a Tool during construction.
type ToolOption func(*Tool)

// PropertyOption configures a property within a ToolInputSchema.
type PropertyOption func(map[string]any)

// WithDescription sets the tool description.
func WithDescription(desc string) ToolOption { ... }

// WithString adds a string property to the tool's input schema.
func WithString(name string, opts ...PropertyOption) ToolOption { ... }

// WithNumber adds a number property.
func WithNumber(name string, opts ...PropertyOption) ToolOption { ... }

// WithBoolean adds a boolean property.
func WithBoolean(name string, opts ...PropertyOption) ToolOption { ... }

// Required marks the property as required in the schema.
func Required() PropertyOption { ... }

// Description sets the property description.
func Description(desc string) PropertyOption { ... }

// Enum restricts allowed values.
func Enum(values ...string) PropertyOption { ... }

// DefaultString sets the default string value.
func DefaultString(v string) PropertyOption { ... }

// DefaultNumber sets the default number value.
func DefaultNumber(v float64) PropertyOption { ... }
```

Also update the existing `NewTool` in `tools.go` to accept `...ToolOption` instead of positional args:

```go
// Old: NewTool(name, description string, inputSchema ToolInputSchema) Tool
// New:
func NewTool(name string, opts ...ToolOption) Tool { ... }
```

**NOTE:** This is a breaking change. Update call sites: `mcp/tests/server_test.go` and `mcp/README.md`.

---

## Step 3 — New file `server_http.go`: StreamableHTTPServer (http.Handler)

`mcpserve/handler.go` calls `server.NewStreamableHTTPServer(s, ...)`. This HTTP transport must live
in `tinywasm/mcp`. It wraps `MCPServer.HandleMessage` behind `http.Handler`.

```go
package mcp

import "net/http"

type StreamableHTTPSOption func(*StreamableHTTPServer)

func WithEndpointPath(path string) StreamableHTTPSOption { ... }
func WithStateLess(stateless bool) StreamableHTTPSOption  { ... }

type StreamableHTTPServer struct {
    mcpServer    *MCPServer
    endpointPath string
    stateless    bool
}

// NewStreamableHTTPServer creates an http.Handler for the MCP JSON-RPC protocol.
func NewStreamableHTTPServer(s *MCPServer, opts ...StreamableHTTPSOption) *StreamableHTTPServer { ... }

// ServeHTTP implements http.Handler.
// Reads the JSON-RPC body, calls MCPServer.HandleMessage, writes the JSON response.
func (s *StreamableHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) { ... }
```

---

## Step 4 — New file `provider.go`: Provider Types

Move the tool-provider abstraction from `mcpserve/tools.go`.

```go
package mcp

// Loggable is implemented by handlers that support log injection.
type Loggable interface {
    Name() string
    SetLog(logger func(message ...any))
}

// ToolExecutor is a simplified handler that receives plain args.
type ToolExecutor func(args map[string]any)

// ParameterMetadata describes a single tool parameter.
type ParameterMetadata struct {
    Name        string
    Description string
    Required    bool
    Type        string   // "string", "number", "boolean"
    EnumValues  []string
    Default     any
}

// ToolMetadata is the high-level tool descriptor used by ToolProvider.
type ToolMetadata struct {
    Name        string
    Description string
    Parameters  []ParameterMetadata
    Execute     ToolExecutor
}

// ToolProvider exposes MCP tools to the Handler.
type ToolProvider interface {
    GetMCPToolsMetadata() []ToolMetadata
}

// BinaryData carries binary content (e.g. screenshot PNG) through the logger.
type BinaryData struct {
    MimeType string
    Data     []byte
}

// BuildTool constructs an mcp.Tool from ToolMetadata using the builder API (Step 2).
// Replaces mcpserve.buildMCPTool with identical logic.
func BuildTool(meta ToolMetadata) Tool { ... }
```

---

## Step 5 — New file `handler_executor.go`: Tool Execution Adapter

Move `mcpExecuteTool` from `mcpserve/executor.go`. Bridges `ToolExecutor` → `ToolHandlerFunc`.

```go
package mcp

import (
    "context"
    "encoding/base64"
    "fmt"
    "strings"
)

// toolExecutorAdapter converts a ToolExecutor + Loggable handler into a ToolHandlerFunc.
func toolExecutorAdapter(targetHandler any, executor ToolExecutor, tui tuiRefresher) ToolHandlerFunc {
    return func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
        args := req.GetArguments()

        var messages []string
        var binaryResponse *BinaryData

        if loggable, ok := targetHandler.(Loggable); ok {
            loggable.SetLog(func(message ...any) {
                for _, m := range message {
                    switch v := m.(type) {
                    case BinaryData:
                        binaryResponse = &v
                    case string:
                        messages = append(messages, v)
                    default:
                        messages = append(messages, fmt.Sprintf("%v", v))
                    }
                }
            })
        }

        executor(args)

        if tui != nil {
            tui.RefreshUI()
        }

        if binaryResponse != nil {
            base64Data := base64.StdEncoding.EncodeToString(binaryResponse.Data)
            return NewToolResultImage(strings.Join(messages, "\n"), base64Data, binaryResponse.MimeType), nil
        }
        if len(messages) == 0 {
            return NewToolResultText("Operation completed successfully"), nil
        }
        return NewToolResultText(strings.Join(messages, "\n")), nil
    }
}

// tuiRefresher is a minimal local interface to avoid importing devtui.
type tuiRefresher interface {
    RefreshUI()
}
```

---

## Step 6 — New file `handler.go`: Main Handler with Injected SSE

### DI Boundary: `SSEHub` Interface

`tinywasm/mcp` does NOT import `tinywasm/sse`. Instead it defines a minimal interface:

```go
// SSEHub is the DI interface for the SSE log transport.
// The consumer (app) creates the real implementation using tinywasm/sse
// and injects it via NewHandler.
// *sse.SSEServer from tinywasm/sse satisfies this interface automatically.
type SSEHub interface {
    http.Handler                          // serves the /logs endpoint
    Publish(data []byte, channel string)  // publishes structured log entries
}
```

### Handler

```go
package mcp

import (
    "encoding/json"
    "net/http"
    "sync"
    "time"
    "context"
    "fmt"
)

const HandlerTypeLoggable = 4

type LogEntry struct {
    Id           string `json:"id"`
    Timestamp    string `json:"timestamp"`
    Content      string `json:"content"`
    Type         uint8  `json:"type"`
    TabTitle     string `json:"tab_title"`
    HandlerName  string `json:"handler_name"`
    HandlerColor string `json:"handler_color"`
    HandlerType  int    `json:"handler_type"`
}

type Config struct {
    Port          string
    ServerName    string
    ServerVersion string
    AppName       string
    AppVersion    string
}

type Handler struct {
    config        Config
    toolHandlers  []ToolProvider
    tui           tuiRefresher
    sseHub        SSEHub          // injected — never created internally
    exitChan      chan bool
    log           func(messages ...any)
    ideStatus     string
    actionFunc    func(string, string)
    stateProvider func() []byte
    httpServer    *http.Server
    mu            sync.Mutex
    running       bool
}

// NewHandler creates a Handler. sseHub is the injected SSE transport (from tinywasm/sse).
func NewHandler(config Config, toolHandlers []ToolProvider, tui tuiRefresher, sseHub SSEHub, exitChan chan bool) *Handler

func (h *Handler) Name() string
func (h *Handler) SetLog(f func(message ...any))
func (h *Handler) URL() string
func (h *Handler) OnUIAction(actionFunc func(key, value string))
func (h *Handler) RegisterStateProvider(fn func() []byte)
func (h *Handler) Serve()   // mounts /mcp, /logs, /action, /state, /version; blocks on exitChan
func (h *Handler) Stop() error
func (h *Handler) PublishLog(msg string)
func (h *Handler) PublishTabLog(tabTitle, handlerName, handlerColor, msg string)
```

### HTTP mux layout

```
/mcp     → NewStreamableHTTPServer(mcpServer) [JSON-RPC]
/logs    → h.sseHub                           [SSE stream — injected]
/action  → handleActionPOST
/state   → handleStateGET
/version → handleVersion
```

---

## Step 7 — New file `handler_ide.go`: IDE Configuration

Move `mcpserve/ide.go` + `mcpserve/config.go` verbatim. No logic changes needed.

Exported: `IDEInfo`, `(h *Handler) ConfigureIDEs()`
Private: all path helpers, `writeMCPConfig`, `validateAppName`, `needsUpdate`.

---

## Step 8 — `go.mod` of `tinywasm/mcp`: No New Dependencies

`tinywasm/mcp` does NOT add `tinywasm/sse` to its `go.mod`.
The `SSEHub` interface is satisfied at the consumer level.

`tinywasm/mcp/go.mod` remains:
```
module github.com/tinywasm/mcp
go 1.25.2
```
(No new requires.)

---

## Step 9 — Consumer Wiring: `app/bootstrap.go`

The consumer (`app`) is responsible for creating the SSE server and injecting it.
`app` already has `tinywasm/sse` as a transitive dependency — promote it to direct.

```go
// app/bootstrap.go — inside runDaemon()

import (
    "github.com/tinywasm/mcp"
    "github.com/tinywasm/sse"
)

// 1. Create the SSE server (tinywasm/sse)
tinySSE := sse.New(&sse.Config{})
sseHub := tinySSE.Server(&sse.ServerConfig{
    ChannelProvider:     &logChannelProvider{},  // resolves channel → "logs"
    ClientChannelBuffer: 256,
    HistoryReplayBuffer: 100,
    ReplayAllOnConnect:  true,
})

// 2. Inject into Handler — sseHub satisfies mcp.SSEHub (duck typing)
mcpHandler := mcp.NewHandler(mcpConfig, nil, ui, sseHub, exitChan)
```

The `logChannelProvider` struct (currently private in `mcpserve/handler.go`) moves to `app/bootstrap.go`:

```go
type logChannelProvider struct{}
func (p *logChannelProvider) ResolveChannels(r *http.Request) ([]string, error) {
    return []string{"logs"}, nil
}
```

`app/go.mod`: remove `mcpserve`, promote `mcp` and `sse` to direct dependencies.

---

## Step 10 — Update Remaining Consumers

Replace all `mcpserve` imports with `mcp` across the three packages.

### 10.1 `devtui` (`devtui/mcp.go`)

```diff
-import "github.com/tinywasm/mcpserve"
+import "github.com/tinywasm/mcp"

-func (d *DevTUI) GetMCPToolsMetadata() []mcpserve.ToolMetadata {
+func (d *DevTUI) GetMCPToolsMetadata() []mcp.ToolMetadata {
     return []mcp.ToolMetadata{
         {
-            Parameters: []mcpserve.ParameterMetadata{ ... },
+            Parameters: []mcp.ParameterMetadata{ ... },
         },
     }
}
```

`devtui/go.mod`: remove `mcpserve`, ensure `mcp` is a direct dependency.

### 10.2 `devbrowser` (all `mcp-*.go` files)

Files: `mcp-tools.go`, `mcp-management.go`, `mcp-console.go`, `mcp-screenshot.go`,
`mcp-structure.go`, `mcp-interaction.go`, `mcp-navigation.go`, `mcp-inspect.go`, `mcp-performance.go`.

```diff
-import "github.com/tinywasm/mcpserve"
+import "github.com/tinywasm/mcp"

-[]mcpserve.ToolMetadata      → []mcp.ToolMetadata
-[]mcpserve.ParameterMetadata → []mcp.ParameterMetadata
-mcpserve.BinaryData           → mcp.BinaryData
```

`devbrowser/go.mod`: remove `mcpserve`, ensure `mcp` is a direct dependency.

### 10.3 `app` (`mcp-tools.go`, `interface.go`, `bootstrap.go`)

**`app/mcp-tools.go`:**
```diff
-import "github.com/tinywasm/mcpserve"
+import "github.com/tinywasm/mcp"
-func (h *Handler) GetMCPToolsMetadata() []mcpserve.ToolMetadata {
+func (h *Handler) GetMCPToolsMetadata() []mcp.ToolMetadata {
```

**`app/interface.go`:**
```diff
-import "github.com/tinywasm/mcpserve"
+import "github.com/tinywasm/mcp"
 type BrowserInterface interface {
-    GetMCPToolsMetadata() []mcpserve.ToolMetadata
+    GetMCPToolsMetadata() []mcp.ToolMetadata
 }
```

**`app/bootstrap.go`:** (see Step 9 for the full wiring)
```diff
-import "github.com/tinywasm/mcpserve"
+import (
+    "github.com/tinywasm/mcp"
+    "github.com/tinywasm/sse"
+)

-McpToolHandlers []mcpserve.ToolProvider  →  McpToolHandlers []mcp.ToolProvider
-mcpHandler *mcpserve.Handler             →  mcpHandler *mcp.Handler
-[]mcpserve.ToolMetadata                  →  []mcp.ToolMetadata
```

---

## Step 11 — Update `mcp/tests/server_test.go`

`NewTool` signature changes (Step 2):
```diff
-tool := mcp.NewTool("test-tool", "A test tool", mcp.ToolInputSchema{Type: "object"})
+tool := mcp.NewTool("test-tool", mcp.WithDescription("A test tool"))
```

---

## Step 12 — Verify & Publish

Run in order:
```bash
cd /home/cesar/Dev/Project/tinywasm/mcp      && gotest
cd /home/cesar/Dev/Project/tinywasm/devtui   && gotest
cd /home/cesar/Dev/Project/tinywasm/devbrowser && gotest
cd /home/cesar/Dev/Project/tinywasm/app      && gotest
```

If all pass:
```bash
cd /home/cesar/Dev/Project/tinywasm/mcp       && gopush 'absorb mcpserve: Handler, SSEHub interface, provider types, tool builders, HTTP transport, IDE config'
cd /home/cesar/Dev/Project/tinywasm/devtui    && gopush 'replace mcpserve with mcp'
cd /home/cesar/Dev/Project/tinywasm/devbrowser && gopush 'replace mcpserve with mcp'
cd /home/cesar/Dev/Project/tinywasm/app       && gopush 'replace mcpserve with mcp; inject sse via SSEHub interface'
```

---

## File Checklist

### `tinywasm/mcp` — New/Modified Files

| File | Action |
|------|--------|
| `utils.go` | Add `NewToolResultImage` |
| `tool_builders.go` | New — ToolOption, PropertyOption, WithString/Number/Boolean, Required, Enum, DefaultString, DefaultNumber |
| `server_http.go` | New — StreamableHTTPServer (http.Handler) |
| `provider.go` | New — Loggable, ToolExecutor, ParameterMetadata, ToolMetadata, ToolProvider, BinaryData, BuildTool |
| `handler_executor.go` | New — toolExecutorAdapter, tuiRefresher |
| `handler.go` | New — SSEHub interface, Config, Handler, Serve, Stop, PublishLog, PublishTabLog |
| `handler_ide.go` | New — IDEInfo, ConfigureIDEs, writeMCPConfig |
| `tools.go` | Update `NewTool` signature → `(name string, opts ...ToolOption)` |
| `tests/server_test.go` | Update `NewTool` call sites |
| `go.mod` | **No changes** — SSEHub is an interface, no new imports |

### Consumers

| Package | Files | Change |
|---------|-------|--------|
| `devtui` | `mcp.go`, `go.mod` | Import swap + remove `mcpserve` |
| `devbrowser` | `mcp-*.go` (9 files), `go.mod` | Import swap + remove `mcpserve` |
| `app` | `mcp-tools.go`, `interface.go`, `bootstrap.go`, `go.mod` | Import swap + SSE wiring + `logChannelProvider` + remove `mcpserve` |
