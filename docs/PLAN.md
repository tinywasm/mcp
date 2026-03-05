# tinywasm/mcp — Refactoring Plan: Protocol-Only MCP Library

> **Goal:** Strip `tinywasm/mcp` to a lean MCP protocol library — tools + JSON-RPC + HTTP transport only.
> No SSE, no stdio, no OAuth, no Resources, no Prompts, no Tasks, no session management state.
> Session lifecycle is managed by the consumer (e.g. `tinywasm/user`) and injected via context.
>
> **Status:** Phases 1–2 complete. Phase 3 pending.

---

## Development Rules

- **Testing Runner:** `go install github.com/tinywasm/devflow/cmd/gotest@latest`
- **Standard Library Only:** No external assertion libraries. No `testify`.
- **Max 500 lines per file.** Subdivide if exceeded.
- **SRP:** Each file has a single purpose, named by domain.
- **No third-party dependencies:** Only stdlib + `tinywasm/*` ecosystem packages.
- **Flat structure:** All source in root. Tests in `tests/`. No new subdirectories.

---

## Architecture

```
[ tinywasm/user ]  — creates + manages sessions, injects via context
        │
        ▼
[ tinywasm/mcp ]   — MCP protocol: types, tools, JSON-RPC dispatch, HTTP transport
        │
        ▼
[ Standard HTTP handler: srv.HTTPHandler() ]
```

### What lives in `tinywasm/mcp`

| Concern | Decision |
|---------|----------|
| JSON-RPC types | ✅ Keep |
| Tool registration + dispatch | ✅ Keep |
| HTTP transport (streamable HTTP) | ✅ Keep — stripped of OAuth |
| SSE transport | ❌ Delete — superseded by streamable HTTP |
| Stdio transport | ❌ Delete — HTTP only |
| OAuth / security | ❌ Delete — injected by consumer |
| Session state (sync.Map, AddSessionTools) | ❌ Delete — managed by tinywasm/user |
| Resources, Prompts, Tasks, Sampling | ❌ Delete — out of scope |

### Session design

The `ClientSession` **interface** stays as the protocol contract. The **management** of sessions
(creating, storing, routing) is extracted from `MCPServer`. The consumer calls `HandleMessage`
with a context that already contains the resolved session, placed there by `tinywasm/user`.

```go
// consumer code (pseudocode)
session := userPkg.GetSession(r)
ctx = mcp.ContextWithSession(ctx, session)
response := mcpServer.HandleMessage(ctx, body)
```

`MCPServer` never holds a `sync.Map` of sessions. It is stateless regarding session lifecycle.

---

## Final File List (after refactor)

| File | Purpose |
|------|---------|
| `server.go` | MCPServer — tool registration, HandleMessage dispatch |
| `client.go` | Client — Initialize, ListTools, CallTool (HTTP transport) |
| `provider.go` | RegisterProvider() |
| `executor.go` | mcpExecuteTool() — Loggable + BinaryData handling |
| `tools.go` | Tool, CallToolRequest/Result, argument helpers |
| `tools_meta.go` | ToolProvider, ToolMetadata, ParameterMetadata, BinaryData |
| `types.go` | JSON-RPC types, capabilities, content types |
| `session.go` | `ClientSession` interface only (~15 lines) |
| `errors.go` | Standard errors |
| `ctx.go` | Context utilities |
| `constants.go` | Protocol constants |
| `interface.go` | MCPClient interface |
| `utils.go` | Helpers (NewTool*, ParseXxx, content helpers) |
| `logger.go` | Logger |
| `transport_interface.go` | Transport contract (Interface, BidirectionalInterface) |
| `transport_streamable_http.go` | HTTP streamable transport — stripped of OAuth |
| `transport_error.go` | Transport errors |
| `transport_utils.go` | Transport utilities |
| `http.go` | HTTP server handler — stripped of OAuth |
| `internal/unixid/` | Session ID generation |
| `tests/` | Integration tests |

---

## Phase 1 — Delete Files ✅ (complete or verify)

**Feature files:**
- `handler.go` ✅
- `resources.go`
- `prompts.go`
- `tasks.go` + `task_hooks.go`
- `hooks.go`
- `sampling.go`
- `elicitation.go`
- `roots.go`
- `completion.go`
- `oauth.go` + `transport_oauth.go` + `transport_oauth_utils.go`
- `ide_config.go` ✅
- `typed_tools.go`
- `http_transport_options.go` ✅
- `consts.go` ✅

**Eliminated transport implementations:**
- `transport_sse.go` ← DELETE (superseded by streamable HTTP)
- `transport_stdio.go` + `stdio.go` ← DELETE (HTTP only)
- `inprocess.go` + `inprocess_session.go` + `transport_inprocess.go` ← DELETE
- `request_handler.go` ← will be **replaced** in Phase 3 (not deleted here)

**Directories:**
- `e2e/` ✅
- `testdata/` ✅
- `util/` ✅ (logger.go moved to root)

---

## Phase 2 — Delete Unused Internal Packages ✅ (complete or verify)

| Package | Decision |
|---------|----------|
| `internal/jsonschema/` | DELETE |
| `internal/go-ordered-map/` | DELETE |
| `internal/generic-list-go/` | DELETE |
| `internal/uritemplate/` | DELETE |
| `internal/tfmt/` | DELETE ✅ |
| `internal/ttime/` | DELETE ✅ |
| `internal/cast/` | KEEP — used by `utils.go` ParseXxx helpers |
| `internal/unixid/` | KEEP ✅ |
| `internal/testutils/` | DELETE ✅ |

---

## Phase 3 — Core File Rewrites

> ⚠️ Do NOT edit large files with shell scripts. Follow [PLAN_STAGE_3.md](PLAN_STAGE_3.md) exactly.
> That document provides complete replacement content or exact deletion lists per file.

**Execution order:**

1. **Section A** — Write complete replacements:
   - `errors.go`, `session.go` (interface only), `interface.go`, `client.go`, `request_handler.go`

2. **Section B** — Surgical deletions from `utils.go`

3. **Section C** — Structural rewrite of `server.go`
   (new stateless MCPServer struct + exact deletion list)

4. **Section D** — Surgical deletions from `tools.go`

5. **Section E** — Type deletion list for `types.go`

6. **Section E2** — OAuth/SSE cleanup in `transport_streamable_http.go` and `http.go`

7. **Section F** — Build, test, push

→ **[Open PLAN_STAGE_3.md](PLAN_STAGE_3.md)**

---

## Phase 4 — `go mod tidy`

```bash
go mod tidy
```

---

## Phase 5 — Verify & Submit

```bash
gotest
gopush 'refactor: strip mcp to protocol-only HTTP transport, extract session management'
```

---

## Migration Note (Out of Scope)

After this plan executes, a separate plan will:
1. Implement session management in `tinywasm/user` and wire it to `tinywasm/mcp`.
2. Migrate `app`, `devbrowser`, `devtui` to import `github.com/tinywasm/mcp` directly.
3. Delete `tinywasm/mcpserve`.
