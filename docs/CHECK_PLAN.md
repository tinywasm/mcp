# tinywasm/mcp — Remaining Refactoring Plan

> **Goal:** Complete the simplification of `tinywasm/mcp` into a lean, zero-external-dependency MCP library (tools + SSE/HTTP transport only).
>
> **Status:** Partially complete. Files `provider.go`, `executor.go`, `tools_meta.go` exist and tests in `tests/` are present. The bulk deletion and core file simplification phases were NOT executed.

---

## Development Rules

- **Testing Runner:** `go install github.com/tinywasm/devflow/cmd/gotest@latest` (first prerequisite)
- **Standard Library Only:** No external assertion libraries. No `testify`.
- **Max 500 lines per file.** Subdivide if exceeded.
- **SRP:** Each file has a single purpose, named by domain.
- **No third-party dependencies:** Only standard library + `tinywasm/*` ecosystem packages.
- **Flat structure:** All source in root. Tests in `tests/`. No new subdirectories.

---

## Current State (After Partial Execution)

**Files that already exist and MUST be kept:**
- `provider.go` ✅ — RegisterProvider() implementation (NEW, keep)
- `executor.go` ✅ — mcpExecuteTool() (NEW, keep)
- `tools_meta.go` ✅ — ToolProvider, ToolMetadata, BinaryData (NEW, keep)
- `tests/provider_test.go` ✅
- `tests/server_test.go` ✅
- `tests/tools_meta_test.go` ✅
- `go.mod` ✅ — already clean (only `tinywasm/sse` + `tinywasm/fmt`)

**Files that still need to be DELETED (Phase 1 not executed):**
- `completion.go`, `consts.go`, `elicitation.go`, `hooks.go`, `http_transport_options.go`
- `inprocess.go`, `inprocess_session.go`, `prompts.go`, `request_handler.go`
- `resources.go`, `roots.go`, `sampling.go`, `stdio.go`, `task_hooks.go`, `tasks.go`
- `transport_inprocess.go`, `transport_oauth.go`, `transport_oauth_utils.go`
- `transport_stdio.go`, `typed_tools.go`
- Scratch files: `jules_changes.patch`, `remote_diff.txt`

**Internal packages to DELETE (Phase 2 not executed):**
- `internal/jsonschema/`
- `internal/go-ordered-map/`
- `internal/generic-list-go/`
- `internal/uritemplate/`
- `internal/cast/`
- `internal/tfmt/`
- `internal/ttime/`

**Core files that still need HEAVY SURGERY (Phase 3 not executed):**
- `server.go` — 2325 lines, still contains resources/prompts/tasks/hooks logic
- `types.go` — 54 KB, still contains all removed feature types
- `session.go` — 765 lines, still contains SessionWithResources/Prompts/Sampling/Elicitation/Roots
- `errors.go` — still contains ErrResourceNotFound, ErrPromptNotFound, ErrDynamicPathConfig, etc.

---

## Phase 1 — Delete Unnecessary Feature Files

Delete the following files from the project root:

```bash
rm -f completion.go consts.go elicitation.go hooks.go http_transport_options.go \
      inprocess.go inprocess_session.go prompts.go request_handler.go \
      resources.go roots.go sampling.go stdio.go task_hooks.go tasks.go \
      transport_inprocess.go transport_oauth.go transport_oauth_utils.go \
      transport_stdio.go typed_tools.go \
      jules_changes.patch remote_diff.txt
```

> **Why:** These files implement features (resources, prompts, tasks, OAuth, stdio, inprocess, completion) that are out of scope for this library. Deleting them eliminates dead code and compilation errors from missing type references.

---

## Phase 2 — Delete Unused Internal Packages

Delete these `internal/` subdirectories entirely:

```bash
rm -rf internal/jsonschema internal/go-ordered-map internal/generic-list-go \
       internal/uritemplate internal/cast internal/tfmt internal/ttime
```

Keep: `internal/unixid/` and `internal/testutils/`

> **Why:** These packages were only used by the deleted feature files. Removing them eliminates external dependency risks and reduces binary size.

---

## Phase 3 — Rewrite `server.go`

**Target:** Reduce from 2325 lines to ~300 lines. Keep ONLY tool and session management.

Replace the entire content of `server.go` with a minimal version that:

1. **Remove all struct fields** for resources/resourceTemplates/prompts/promptHandlers/tasks/taskTools/expiredTasks/hooks/taskHooks/activeTasks/maxConcurrentTasks
2. **Remove all mutex fields** except `toolsMu`, `toolMiddlewareMu`, `toolFiltersMu`, `notificationHandlersMu`, `capabilitiesMu`
3. **Remove all methods** for: `AddResource*`, `AddResourceTemplate*`, `AddPrompt*`, `AddTask*`, `AddSessionResource*`, `AddSessionResourceTemplate*`, `DeleteResource*`, `DeletePrompt*`, `RemoveResource*`, `SetResources`, `SetResourceTemplates`, `SetPrompts`, `AddTaskTool*`, `AddTaskTools*`
4. **Remove server options:** `WithResourceCapabilities`, `WithResourceHandlerMiddleware`, `WithResourceRecovery`, `WithPromptCapabilities`, `WithPromptCompletionProvider`, `WithResourceCompletionProvider`, `WithLogging`, `WithElicitation`, `WithRoots`, `WithTaskCapabilities`, `WithTaskHooks`, `WithMaxConcurrentTasks`, `WithCompletions`
5. **Remove type definitions:** `ResourceHandlerFunc`, `ResourceTemplateHandlerFunc`, `PromptHandlerFunc`, `TaskToolHandlerFunc`, `ResourceHandlerMiddleware`, `ServerPrompt`, `ServerResource`, `ServerResourceTemplate`, `ServerTaskTool`, `taskEntry`, `resourceEntry`, `resourceTemplateEntry`
6. **Remove helper types:** `serverCapabilities` fields: resources, prompts, logging, sampling, elicitation, roots, tasks, completions — keep only `tools`
7. **Remove:** `implicitlyRegisterResourceCapabilities`, `implicitlyRegisterPromptCapabilities`, `GenerateInProcessSessionID`

**Minimal `MCPServer` struct (target):**
```go
type MCPServer struct {
    toolsMu                sync.RWMutex
    toolMiddlewareMu       sync.RWMutex
    notificationHandlersMu sync.RWMutex
    capabilitiesMu         sync.RWMutex
    toolFiltersMu          sync.RWMutex

    name                   string
    version                string
    instructions           string
    tools                  map[string]ServerTool
    toolHandlerMiddlewares []ToolHandlerMiddleware
    toolFilters            []ToolFilterFunc
    notificationHandlers   map[string]NotificationHandlerFunc
    capabilities           serverCapabilities
    paginationLimit        *int
    sessions               sync.Map
}

type serverCapabilities struct {
    tools *toolCapabilities
}
```

**Keep these methods in server.go:**
- `NewMCPServer`
- `AddTool`, `AddTools`, `SetTools`, `GetTool`, `ListTools`, `DeleteTools`
- `WithToolCapabilities`, `WithToolHandlerMiddleware`, `WithRecovery`, `WithToolFilter`
- `WithInstructions`, `WithPaginationLimit`
- `implicitlyRegisterToolCapabilities`, `implicitlyRegisterCapabilities`
- `HandleMessage` (dispatches tool calls only)
- `handleInitialize`, `handleToolsList`, `handleToolsCall`, `handlePing`
- `SendNotificationToAllClients`, `SendNotificationToClient`, `SendNotificationToSpecificClient`
- `sendNotificationToAllClients`, `sendNotificationToSpecificClient`, `sendNotificationCore`
- `AddNotificationHandler`
- `ServerFromContext`
- `requestError`, `UnparsableMessageError`

> **IMPORTANT:** The `HandleMessage` function in `request_handler.go` (which is being deleted) must be reproduced as a minimal version directly in `server.go`. It should only switch on these methods: `initialize`, `ping`, `tools/list`, `tools/call`. All other methods should return a JSON-RPC "method not found" error.

---

## Phase 4 — Rewrite `types.go`

**Target:** Reduce from 54 KB to ~300 lines. Keep ONLY JSON-RPC core + tool-related types.

Delete all type definitions for:
- Resources: `Resource`, `ResourceContents`, `TextResourceContents`, `BlobResourceContents`, `ResourceTemplate`, `ReadResourceRequest`, `ReadResourceResult`, `ListResourcesRequest`, `ListResourcesResult`, `ListResourceTemplatesRequest`, `ListResourceTemplatesResult`, `SubscribeRequest`, `UnsubscribeRequest`, `ResourceListChangedNotification`
- Prompts: `Prompt`, `PromptArgument`, `PromptMessage`, `GetPromptRequest`, `GetPromptResult`, `ListPromptsRequest`, `ListPromptsResult`, `PromptListChangedNotification`, `PromptCompletionProvider`, `DefaultPromptCompletionProvider`
- Tasks: `Task`, `TaskStatus`, `CreateTaskResult`, `TaskResultRequest`, `TaskListRequest`, `TaskListResult`, `TaskCancelRequest`
- Sampling: `CreateMessageRequest`, `CreateMessageResult`, `SamplingMessage`, `ModelPreferences`, `ModelHint`
- Elicitation: `ElicitationRequest`, `ElicitationResult`, `ElicitationSchema`
- Roots: `Root`, `ListRootsRequest`, `ListRootsResult`
- Logging: `LoggingLevel`, `LoggingMessageNotification`, `SetLevelRequest`
- Completion: `CompletionArgument`, `CompleteRequest`, `CompleteResult`, `CompletionResult`, `ResourceCompletionProvider`, `DefaultResourceCompletionProvider`

**Keep:**
- `JSONRPCRequest`, `JSONRPCResponse`, `JSONRPCNotification`, `JSONRPCError`, `JSONRPCErrorDetails`, `NewJSONRPCErrorDetails`
- `RequestId`, `NewRequestId`
- `MCPMethod` (string type + method constants for the 4 supported methods only: initialize, ping, tools/list, tools/call)
- `Implementation`, `ClientCapabilities`, `ServerCapabilities` (minimal)
- `InitializeRequest`, `InitializeResult`
- `PaginatedRequest`, `PaginatedResult`, `Cursor`
- `Meta`, `Content`, `TextContent`, `ImageContent`, `EmbeddedResource`
- `Notification`, `NotificationParams`
- `Tool`, `CallToolRequest`, `CallToolResult`, `ListToolsRequest`, `ListToolsResult`
- `ToolsListChangedNotification`
- Helper functions: `ToBoolPtr`, `NewToolResult*` functions (already in `tools.go` — move if needed)
- `JSONRPC_VERSION`, `LATEST_PROTOCOL_VERSION`

> **Note:** Many helper constants and notification method strings for deleted features can simply be removed. Scan for `MethodNotification*` constants in `constants.go` and remove all except `MethodNotificationToolsListChanged`.

---

## Phase 5 — Simplify `session.go`

**Target:** Reduce from 765 lines to ~150 lines.

**Remove these interfaces and all their usages:**
- `SessionWithLogging` (and `buildLogNotification`, `SendLogMessageToClient`, `SendLogMessageToSpecificClient`)
- `SessionWithResources` (and `AddSessionResources`, `AddSessionResource`, `DeleteSessionResources`)
- `SessionWithResourceTemplates` (and `AddSessionResourceTemplates`, `AddSessionResourceTemplate`, `DeleteSessionResourceTemplates`)
- `SessionWithSampling`
- `SessionWithElicitation`
- `SessionWithRoots`

**Keep:**
- `ClientSession` interface
- `SessionWithTools` interface + `AddSessionTools`, `AddSessionTool`, `DeleteSessionTools`
- `SessionWithClientInfo` interface
- `SessionWithStreamableHTTPConfig` interface
- `clientSessionKey`, `ClientSessionFromContext`
- `WithContext`, `RegisterSession`, `UnregisterSession`
- `SendNotificationToAllClients`, `SendNotificationToClient`, `SendNotificationToSpecificClient`

> **Note:** After removing `SessionWithLogging`, remove the `ErrSessionDoesNotSupportLogging` error from `errors.go` as well.

---

## Phase 6 — Clean `errors.go`

Remove:
- `ErrResourceNotFound`
- `ErrPromptNotFound`
- `ErrSessionDoesNotSupportResources`
- `ErrSessionDoesNotSupportResourceTemplates`
- `ErrSessionDoesNotSupportLogging`
- `ErrDynamicPathConfig` type + method

Keep all tool/session errors (`ErrToolNotFound`, `ErrSessionNotFound`, `ErrSessionExists`, `ErrSessionNotInitialized`, `ErrSessionDoesNotSupportTools`, `ErrNotification*`, `ErrUnsupported`).

---

## Phase 7 — Clean `constants.go` and merge `consts.go`

`consts.go` is being deleted (Phase 1). Before deletion, check if it contains anything not already in `constants.go`. If so, merge the unique entries to `constants.go` first.

From `constants.go`, remove notification method constants for deleted features (resources, prompts, tasks, logging, etc.). Keep only:
- `JSONRPC_VERSION`
- `LATEST_PROTOCOL_VERSION`
- `MethodToolsList`, `MethodToolsCall`, `MethodInitialize`, `MethodPing`
- `MethodNotificationToolsListChanged`

---

## Phase 8 — Run `go mod tidy`

```bash
go mod tidy
```

This removes any stale indirect dependencies automatically.

---

## Phase 9 — Verify & Submit

1. Install test runner: `go install github.com/tinywasm/devflow/cmd/gotest@latest`
2. Run: `gotest` — all tests in `tests/` must pass (0 failures).
3. If tests pass: `gopush 'refactor: complete mcp simplification — tools-only lean library'`

---

## Expected Final File Set (~20 files)

| File | Purpose |
|------|---------|
| `server.go` | MCPServer — tool registration, dispatch (tools only) |
| `client.go` | MCPClient — ListTools, CallTool |
| `provider.go` | RegisterProvider() ✅ already done |
| `executor.go` | mcpExecuteTool() ✅ already done |
| `tools.go` | Tool, CallToolRequest, CallToolResult, builder helpers |
| `tools_meta.go` | ToolProvider, ToolMetadata, BinaryData ✅ already done |
| `types.go` | JSON-RPC types, capabilities, content types (trimmed) |
| `session.go` | ClientSession, SessionWithTools (trimmed) |
| `errors.go` | Standard errors (trimmed) |
| `ctx.go` | Context utilities |
| `constants.go` | Protocol constants (trimmed) |
| `interface.go` | MCPClient interface |
| `utils.go` | Internal helpers |
| `transport_interface.go` | Transport interface |
| `transport_sse.go` | SSE client transport |
| `transport_streamable_http.go` | HTTP streamable transport (stripped of OAuth) |
| `transport_error.go` | Transport errors |
| `transport_utils.go` | Transport utilities |
| `http.go` | HTTP server handler |
| `internal/unixid/` | Session ID generation |
| `internal/testutils/` | Test assertion helpers |
| `tests/` | Existing minimal tests |
