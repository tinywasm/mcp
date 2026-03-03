# tinywasm/mcp — Refactoring Plan: Minimal MCP Server/Client Library

> **Goal:** Simplify `tinywasm/mcp` from a ~46-file fork of mark3labs/mcp-go into a lean, zero-external-dependency MCP library that will eventually replace `tinywasm/mcpserve` entirely.
>
> **Status:** Pending execution

---

## Development Rules

- **Testing Runner:** `go install github.com/tinywasm/devflow/cmd/gotest@latest`
- **Standard Library Only:** No external assertion libraries. No `testify`.
- **Max 500 lines per file.** Subdivide if exceeded.
- **SRP:** Each file has a single purpose, named by domain.
- **No third-party dependencies:** Only standard library + `tinywasm/*` ecosystem packages. Remove any non-tinywasm external dependencies from go.mod.
- **Flat structure:** All source in root. Tests in `tests/`. No new subdirectories.

---

## Context

`tinywasm/mcp` is a fork of `mark3labs/mcp-go` containing ~46 Go files, 20+ test files (all broken after migration to `tests/`), and 10 internal packages. The library implements the full MCP spec (tools, resources, prompts, tasks, sampling, elicitation, roots, OAuth, stdio, inprocess), but only **tools + SSE/HTTP transport** are needed.

The goal is a single importable package with a clean API:
```go
srv := mcp.NewServer("name", "1.0.0")
srv.RegisterProvider(myModule)
http.ListenAndServe(":8080", srv.HTTPHandler())
```

This will allow `tinywasm/mcpserve` to be eliminated after consumers are migrated.

**Key insight from exploration:** `mcpserve/handler.go` already imports `"github.com/tinywasm/mcp/server"` (a subpackage that does not yet exist). This plan keeps everything in the root package.

---

## Phase 1 — Delete Unnecessary Files

Delete the following files from `/home/cesar/Dev/Project/tinywasm/mcp/`:

**Feature files (not needed):**
- `handler.go` — app-level orchestration, belongs in mcpserve/app, not the library
- `resources.go` — resources not needed now
- `prompts.go` — prompts not needed now
- `tasks.go` + `task_hooks.go` — tasks removed
- `hooks.go` — lifecycle hooks removed
- `request_handler.go` — auto-generated dispatcher (will be rewritten to tool-only)
- `sampling.go` — not needed
- `elicitation.go` — not needed
- `roots.go` — not needed
- `completion.go` — not needed
- `oauth.go` + `transport_oauth.go` + `transport_oauth_utils.go` — no OAuth needed
- `transport_stdio.go` + `stdio.go` — no stdio transport
- `inprocess.go` + `inprocess_session.go` + `transport_inprocess.go` — no in-process
- `ide_config.go` — not library concern
- `typed_tools.go` — reflection-based tool building not needed
- `http_transport_options.go` — merge into `transport_streamable_http.go`
- `consts.go` — check for duplicates with `constants.go`, merge and delete

**Directories:**
- `e2e/` — delete entirely
- `tests/` — delete entirely (rewrite from scratch in Step 6)

---

## Phase 2 — Delete Unused Internal Packages

Audit and delete from `internal/`:

| Package | Used By | Decision |
|---------|---------|----------|
| `internal/jsonschema/` | `typed_tools.go` (deleted) | **DELETE** |
| `internal/go-ordered-map/` | `jsonschema` (deleted) | **DELETE** |
| `internal/generic-list-go/` | task lists (deleted) | **DELETE** |
| `internal/uritemplate/` | resource templates (deleted) | **DELETE** |
| `internal/tfmt/` | possibly tools/server | **CHECK** — delete if unused |
| `internal/ttime/` | possibly tools/server | **CHECK** — delete if unused |
| `internal/cast/` | possibly tools | **CHECK** — delete if unused |
| `internal/unixid/` | `server.go` (session IDs) | **KEEP** |
| `internal/testutils/` | tests | **KEEP** (for new tests) |

---

## Phase 3 — Simplify Core Files

### 3a. `server.go`
Remove:
- All resource-related fields: `resources`, `resourceTemplates`, `resourceHandlers`
- All prompt-related fields: `prompts`, `promptHandlers`
- All task-related fields: `tasks`, `taskTools`, `expiredTasks`
- `hooks` and `taskHooks` fields
- All methods: `AddResource`, `AddPrompt`, `AddTask`, `SetHooks` and their variants
- Session extension handlers: `SessionWithResources`, `SessionWithResourceTemplates`, `SessionWithSampling`, `SessionWithElicitation`, `SessionWithRoots`, `SessionWithLogging`

Keep:
- `MCPServer` struct (minimal fields: `name`, `version`, `tools`, `sessions`)
- `NewMCPServer(name, version string, opts ...ServerOption) *MCPServer`
- `AddTool(tool Tool, handler ToolHandlerFunc)`
- `AddTools(tools ...ServerTool)`
- `RegisterSession` / `UnregisterSession`
- `AddSessionTool(s)` / `DeleteSessionTools`
- `WithContext` / session context helpers
- HTTP handler creation (currently in `http.go`)

### 3b. `types.go`
Remove type definitions for:
- Resources, ResourceTemplates
- Prompts, PromptArguments
- Tasks, TaskStatus
- Sampling (CreateMessage*)
- Elicitation
- Roots
- Logging (LoggingLevel)
- Completion (CompletionArgument, etc.)

Keep:
- JSON-RPC types: `JSONRPCRequest`, `JSONRPCResponse`, `JSONRPCNotification`, `JSONRPCErrorDetails`
- Capability types (only tool-related capabilities)
- `Implementation`, `Meta`, `Content`, `TextContent`, `ImageContent`
- `InitializeRequest/Result`, `PaginatedRequest/Result`
- Session and client info types

### 3c. `session.go`
Remove:
- `SessionWithResources`
- `SessionWithResourceTemplates`
- `SessionWithSampling`
- `SessionWithElicitation`
- `SessionWithRoots`
- `SessionWithLogging`

Keep:
- `ClientSession` interface
- `SessionWithTools`
- `SessionWithClientInfo`
- `SessionWithStreamableHTTPConfig`

### 3d. `mcptest.go`
Remove:
- `prompts`, `resources`, `resourceTemplates` fields
- `AddPrompt`, `AddResource`, `AddResourceTemplate` methods

Keep:
- `Server` struct (tools only)
- `NewServer(t, tools...)` / `NewUnstartedServer(t)`
- `AddTool`, `AddTools`, `Start`, `Close`, `Client`

### 3e. `errors.go`
Remove:
- `ErrResourceNotFound`, `ErrPromptNotFound`
- `ErrSessionDoesNotSupportResources`, `ErrSessionDoesNotSupportResourceTemplates`
- `ErrSessionDoesNotSupportLogging`
- `ErrDynamicPathConfig`

Keep all tool/session errors.

---

## Phase 4 — Fix `ToolExecutor` Signature

**File:** `tools_meta.go`

Current (broken — no context, no return):
```go
type ToolExecutor func(args map[string]any)
```

New (matches ecosystem handler signature):
```go
type ToolExecutor func(ctx context.Context, args map[string]any) (any, error)
```

Also add `BinaryData` type to this file (moved from mcpserve/executor.go):
```go
// BinaryData represents a binary tool result (e.g., screenshots).
// When passed as result from ToolExecutor, it is base64-encoded and returned as image content.
type BinaryData struct {
    MimeType string
    Data     []byte
}
```

---

## Phase 5 — Create `provider.go` (New File)

New file `/home/cesar/Dev/Project/tinywasm/mcp/provider.go`:

```go
package mcp

import "context"

// RegisterProvider registers all MCP tools declared by a ToolProvider.
// Calls provider.GetMCPToolsMetadata() once at registration time.
// Each ToolMetadata.Execute is automatically adapted to ToolHandlerFunc.
//
// This is the standard registration entry point for tinywasm ecosystem modules:
//
//   func (m *Module) RegisterTools(srv *mcp.MCPServer) {
//       srv.RegisterProvider(m)
//   }
func (s *MCPServer) RegisterProvider(provider ToolProvider) {
    for _, meta := range provider.GetMCPToolsMetadata() {
        tool := buildMCPTool(meta)
        exec := meta.Execute // capture

        s.AddTool(*tool, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
            if exec == nil {
                return NewToolResultError("tool has no executor"), nil
            }
            args := make(map[string]any)
            if err := req.BindArguments(&args); err != nil {
                return NewToolResultError("invalid arguments: " + err.Error()), nil
            }
            result, err := exec(ctx, args)
            if err != nil {
                return NewToolResultError(err.Error()), nil
            }
            if bd, ok := result.(BinaryData); ok {
                return NewToolResultImage("", bd.Data, bd.MimeType), nil
            }
            return NewToolResultStructuredOnly(result), nil
        })
    }
}
```

---

## Phase 6 — Update `executor.go`

Update `mcpExecuteTool` to use the new `ToolExecutor` signature. The pattern:
1. Extract args from `CallToolRequest` via `BindArguments`
2. Call `handler.SetLog(capturingLogger)` if handler implements `Loggable`
3. Call `exec(ctx, args)` → returns `(any, error)`
4. If result is `BinaryData`, return as image content
5. Otherwise return as structured/text content

Reference implementation: `/home/cesar/Dev/Project/tinywasm/mcpserve/executor.go`

---

## Phase 7 — Update `go.mod`

After deleting `handler.go` (which was the only file importing `github.com/tinywasm/sse` directly), run:

```bash
go mod tidy
```

This will remove any unused dependencies automatically. `tinywasm/sse` and `tinywasm/fmt` may remain if still referenced by other files — that is fine, they are ecosystem packages.

---

## Phase 8 — Write New Tests

Create new `tests/` directory with minimal, passing tests:

**`tests/setup_test.go`**
- Package declaration: `package mcp_test`
- Imports `github.com/tinywasm/mcp`
- Helper to create test server

**`tests/server_test.go`**
- `TestAddTool` — register a tool, verify it appears in ListTools
- `TestCallTool` — register a tool with handler, call it, verify result
- `TestCallTool_Error` — handler returns error, verify tool result is error

**`tests/provider_test.go`**
- `TestRegisterProvider` — mock ToolProvider with 2 tools, register, verify both registered
- `TestRegisterProvider_Execute` — verify ctx and args passed correctly to Execute
- `TestRegisterProvider_ExecuteError` — Execute returns error → NewToolResultError

**`tests/tools_meta_test.go`**
- `TestBuildMCPTool` — verify ParameterMetadata → Tool schema conversion

Run: `gotest` — 100% pass required before proceeding.

---

## Phase 9 — Verify & Submit

1. `gotest` — all tests pass
2. `gopush 'refactor: simplify to minimal tools-only MCP server/client, remove all unused features'`

---

## Files Remaining After Refactor (~15 files)

| File | Purpose |
|------|---------|
| `server.go` | MCPServer — tool registration, dispatch, sessions |
| `client.go` | MCPClient — ListTools, CallTool |
| `provider.go` | RegisterProvider() |
| `executor.go` | mcpExecuteTool() — Loggable + BinaryData handling |
| `tools.go` | Tool, CallToolRequest, CallToolResult, builder helpers |
| `tools_meta.go` | ToolProvider, ToolMetadata, ParameterMetadata, BinaryData |
| `types.go` | JSON-RPC types, capabilities, content types |
| `session.go` | ClientSession, SessionWithTools |
| `errors.go` | Standard errors |
| `ctx.go` | Context utilities |
| `constants.go` | Protocol constants |
| `interface.go` | MCPClient interface |
| `utils.go` | Internal helpers |
| `transport_interface.go` | Transport interface |
| `transport_sse.go` | SSE client transport |
| `transport_streamable_http.go` | HTTP streamable transport (client+server) |
| `transport_error.go` | Transport errors |
| `transport_utils.go` | Transport utilities |
| `http.go` | HTTP server handler (NewStreamableHTTPServer) |
| `mcptest.go` | Test harness |
| `internal/unixid/` | Session ID generation |
| `internal/testutils/` | Test assertion helpers |
| `tests/` | New minimal tests |

---

## Migration Note (Out of Scope for This Plan)

After this plan is executed, a separate plan will migrate `app`, `devbrowser`, `devtui` to import `github.com/tinywasm/mcp` directly instead of `github.com/tinywasm/mcpserve`, allowing `mcpserve` to be deleted. The `ToolExecutor` signature change is a breaking change for those consumers.
