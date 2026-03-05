# tinywasm/mcp — Remaining Tasks Plan

> **Context:** The refactoring plan from `CHECK_PLAN.md` was successfully executed for Phases 1–7.
> The core simplification is complete. This plan addresses the remaining work.
>
> **Related docs:** [CHECK_PLAN.md](CHECK_PLAN.md)

---

## Development Rules

- **Testing Runner:** `go install github.com/tinywasm/devflow/cmd/gotest@latest` (first prerequisite — required in isolated environments)
- **Standard Library Only:** No external assertion libraries. No `testify`.
- **Max 500 lines per file.** Subdivide if exceeded.
- **SRP:** Each file has a single purpose, named by domain.
- **No third-party dependencies:** Only standard library + `tinywasm/*` ecosystem packages.
- **Flat structure:** All source in root. Tests in `tests/`. No new subdirectories.

---

## Status After CHECK_PLAN Execution

| Phase | Description | Status |
|-------|-------------|--------|
| Phase 1 | Delete unnecessary feature files | ✅ Complete |
| Phase 2 | Delete unused internal packages | ✅ Complete |
| Phase 3 | Rewrite `server.go` | ✅ Complete (758 lines, tools-only) |
| Phase 4 | Rewrite `types.go` | ✅ Complete (487 lines, clean) |
| Phase 5 | Simplify `session.go` | ✅ Complete (294 lines, tools-only) |
| Phase 6 | Clean `errors.go` | ✅ Complete (22 lines, minimal) |
| Phase 7 | Clean `constants.go` | ✅ Complete (41 lines, 4 methods only) |
| **NEW** | Split `tools.go` (1338 lines → violates 500-line rule) | ❌ Pending |
| Phase 8 | `go mod tidy` | ⚠️ Not verified |
| Phase 9 | `gotest` + `gopush` | ⚠️ Not verified |

---

## Task 1 — Split `tools.go` into Domain Files

**Problem:** `tools.go` has **1338 lines**, violating the 500-line-per-file rule.

**Analysis of `tools.go` content:**
- Lines 1–98: `CallToolRequest`/`CallToolResult`/`ListToolsRequest`/`ListToolsResult` types + argument getters (`GetString`, `GetInt`, `GetBool`, etc.)
- Lines 471–545: `CallToolResult` JSON marshal/unmarshal
- Lines 547–728: `Tool` struct + `ToolInputSchema`/`ToolOutputSchema`/`ToolArgumentsSchema` types + JSON marshal/unmarshal
- Lines 729–780: `NewTool`, `NewToolWithRawSchema` constructors
- Lines 781+: `ToolOption` helper functions (`WithDescription`, `WithString`, `WithNumber`, etc.) and `NewToolResult*` helper functions

**Split strategy:** Divide into 3 files by domain responsibility:

### File: `tools.go` (~250 lines — keep)
Contains only the **core request/response types** used by the MCP protocol:
- `ListToolsRequest`, `ListToolsResult`
- `CallToolRequest`, `CallToolParams`, `CallToolResult` + their JSON marshal/unmarshal
- `ToolListChangedNotification`
- `GetArguments()`, `GetRawArguments()`, `BindArguments()`, and all `Get*`/`Require*` argument helper methods on `CallToolRequest`

### File: `tool_schema.go` (NEW, ~300 lines)
Contains **tool schema and definition types**:
- `Tool` struct + `GetName()` + `MarshalJSON()`
- `ToolAnnotation`
- `ToolArgumentsSchema`, `ToolInputSchema`, `ToolOutputSchema`
- `toolArgumentsSchemaMarshalJSON`, `toolArgumentsSchemaUnmarshalJSON`

### File: `tool_builder.go` (NEW, ~350 lines)
Contains **tool construction helpers** (builder pattern):
- `ToolOption`, `PropertyOption` types
- `NewTool`, `NewToolWithRawSchema`
- All `With*` option functions (`WithDescription`, `WithString`, `WithNumber`, `WithBoolean`, `WithArray`, `WithObject`, `WithRequired`, `WithToolAnnotation`, `WithReadOnly`, etc.)
- All `NewToolResult*` helper functions (`NewToolResultText`, `NewToolResultImage`, `NewToolResultError`, etc.)
- Helper: `ToBoolPtr`

**Implementation steps:**

1. Create `tool_schema.go` with the `Tool`, `ToolAnnotation`, schema types and their JSON marshal/unmarshal methods. Move them out of `tools.go`.
2. Create `tool_builder.go` with `ToolOption`, `PropertyOption`, `NewTool`, `NewToolWithRawSchema`, all `With*` functions, all `NewToolResult*` functions, and `ToBoolPtr`. Move them out of `tools.go`.
3. Remove the moved code from `tools.go`, keeping only the request/response types and argument accessor methods.
4. Verify all files stay under 500 lines.
5. Run `go build ./...` to confirm no compilation errors.

---

## Task 2 — Run `go mod tidy`

```bash
go mod tidy
```

Verify that `go.mod` and `go.sum` are clean after all deletions.

Expected `go.mod` content (approximately):
```
module github.com/tinywasm/mcp

go X.XX

require (
    github.com/tinywasm/sse vX.X.X
    github.com/tinywasm/fmt vX.X.X
)
```

---

## Task 3 — Verify & Submit

1. Install test runner:
   ```bash
   go install github.com/tinywasm/devflow/cmd/gotest@latest
   ```
2. Run all tests:
   ```bash
   gotest
   ```
   All tests in `tests/` must pass (0 failures).

3. If all tests pass, push:
   ```bash
   gopush 'refactor: split tools.go by domain, complete mcp simplification'
   ```

---

## Expected Final File Set

| File | Lines (approx) | Purpose |
|------|----------------|---------|
| `server.go` | ~758 | MCPServer — tool registration, dispatch |
| `client.go` | ~368 | MCPClient — ListTools, CallTool |
| `provider.go` | ~60 | RegisterProvider() |
| `executor.go` | ~55 | mcpExecuteTool() |
| `tools.go` | ~250 | Request/response types + argument accessors |
| `tool_schema.go` | ~182 | Tool struct + schema types |
| `tool_builder.go` | ~175 | NewTool + option/result helpers |
| `tool_property.go` | ~440 | PropertyOption helpers |
| `tools_meta.go` | ~105 | ToolProvider, ToolMetadata, BinaryData |
| `types.go` | ~487 | JSON-RPC types, capabilities, content |
| `session.go` | ~294 | ClientSession, SessionWithTools |
| `errors.go` | ~22 | Minimal errors |
| `ctx.go` | ~5 | Context utilities |
| `constants.go` | ~41 | Protocol constants |
| `logger.go` | ~34 | Minimal logging interface |
| `interface.go` | ~40 | MCPClient interface |
| `utils.go` | ~451 | Internal helpers |
| `http.go` | ~25 | HTTP server handler |
| `transport_error.go` | ~12 | Transport errors |
| `transport_interface.go` | ~65 | Transport interface |
| `transport_sse.go` | ~567 | SSE client transport |
| `transport_streamable_http.go` | ~732 | HTTP streamable transport |
| `transport_utils.go` | ~12 | Transport utilities |
| `internal/testutils/` | — | Test assertion helpers |
| `tests/` | — | Existing minimal tests |

> **Note:** `transport_sse.go` (567) and `transport_streamable_http.go` (732) exceed 500 lines
> but are complex transport implementations that are difficult to split without losing cohesion.
> These are acceptable exceptions given their single clear responsibility (one transport protocol each).
> If the executing agent has time, splitting them is a bonus but not required for this task.
