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

## Phase 4 — Update `go.mod`

After deleting unnecessary files, run:

```bash
go mod tidy
```

This will remove any unused dependencies automatically. `tinywasm/sse` and `tinywasm/fmt` may remain if still referenced by other files.

---

## Phase 5 — Verify & Submit

1. Run the test suite: `gotest` — 100% pass required before proceeding.
2. `gopush 'refactor: complete mcp file pruning and server/types simplification'`

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
