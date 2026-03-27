# MCP: Full Migration to tinywasm/json + TinyGo Compatibility

## Goal

`tinywasm/mcp` must compile with TinyGo to run in the browser as a local handler
(no HTTP). The protocol core must be WASM-safe. Server-only files (HTTP, filesystem)
are isolated with `//go:build !wasm`.

## Resolved dependencies

- `tinywasm/json v0.4.0` — no `map[string]any`, TinyGo-safe ✅
- `tinywasm/fmt v0.20.0` — string ops, errors, KeyValue ✅
- `tinywasm/context` — 16 string pairs, no reflection ✅
- `tinywasm/unixid` — unique string IDs with embedded timestamp ✅
- `tinywasm/orm (empty struct support)` — see [tinywasm/orm PLAN](../../orm/docs/PLAN-empty-struct.md) ⏳

## Build tag convention

```
//go:build wasm     → WASM/browser only
//go:build !wasm    → server only (net/http, os, sync available)
no tag              → compiled in both — MUST be WASM-safe
```

## JSON-in-string pattern

Fields that contain arrays or nested objects (tool list, capabilities, arguments)
are serialized to `string` first and embedded. This avoids `[]T` and nested structs
as direct fields, which `tinywasm/json` does not support as top-level slice types.

```go
// Wrong
type listToolsResult struct { Tools []toolEntry }

// Correct
type listToolsResult struct { Tools string } // JSON array serialized as string
```

## ormc rules

| Struct type | Annotation | Generates | Implements |
|---|---|---|---|
| Protocol types (internal) | `// ormc:formonly` | `Schema()`, `Pointers()` | `fmt.Fielder` |
| Tool argument types (external input) | none + `validate:` tags | `Schema()`, `Pointers()`, `Validate()` | `fmt.SafeFields` |

> `ormc` converts `CamelCase` fields to `snake_case` JSON keys automatically.
> Only add `json:` tag when the protocol requires a different name
> (e.g. `json:"protocolVersion"`, `json:"isError"`, `json:"_meta"`).
> Simple fields like `Name`, `Method`, `Code` need no `json:` tag.

> **Empty structs**: `ormc` skips structs with zero fields.
> For notifications with no params, pass `nil` — `SendNotification` accepts `nil`.

## Absolute rules

- NO `encoding/json` in any file — use `tinywasm/json` everywhere
- NO stdlib `"context"` — use `github.com/tinywasm/context`
- NO stdlib `"strings"` — use `tinywasm/fmt`
- NO `map[string]any`, `[]any`, `json.RawMessage` as struct fields
- NO structs or interfaces stored in `tinywasm/context` — strings only
- NO hand-written `Schema()`, `Pointers()`, `Validate()` — always use `ormc`
- NO `// ormc:formonly` on tool argument structs — they need `Validate()`
- NO `reflect` in any file
- Session management belongs to `tinywasm/user` — mcp only stores session ID (string)
- Auth is an interface only — implementation lives in `tinywasm/user`

---

## Stages

| Stage | File | Description |
|---|---|---|
| [0](stages/stage-00-setup.md) | Setup | Install ormc, verify dependencies |
| [1](stages/stage-01-client.md) | `client.go` | Replace stdlib strings + context |
| [2](stages/stage-02-ide-handler.md) | `handler_ide` | Split wasm/server, define IDE config structs |
| [3](stages/stage-03-http-split.md) | `server_http`, `transport` | Tag as server-only, migrate json |
| [4](stages/stage-04-types.md) | `types.go` | Simplify RequestId, Meta, remove http.Header + Experimental |
| [5](stages/stage-05-models.md) | `model.go` | Add all protocol structs, run ormc |
| [6](stages/stage-06-tools.md) | `tools.go`, `provider.go` | Simplify API: Tool.Run, req.Bind, mcp.Text/JSON, delete builders |
| [7](stages/stage-07-request-handler.md) | `request_handler.go` | 2 context keys only (user_id, session_id), replace json.Unmarshal |
| [8](stages/stage-08-utils.md) | `utils.go` | JSON/ParseResult/GetText, delete Content constructors, unexport JSONRPC helpers |
| [9](stages/stage-09-session.md) | `session.go` | ID-only context, remove registry from mcp |
| [10](stages/stage-10-interfaces.md) | `interface.go`, `transport_interface.go` | Tag server-only, update context type |
| [11](stages/stage-11-handler.md) | `handler.go` | Split HTTP parts to .back.go, remove duplicate MethodFunc, replace anon structs |
| [12](stages/stage-12-server.md) | `server.go` | Remove cursor base64, isolate image base64 in .back.go, remove slices/sort |
| [13](stages/stage-13-auth.md) | `mcp_auth.go` | Replace with Authorizer interface |
| [14](stages/stage-14-tests.md) | `tests/` | Rewrite broken tests, add new cases |
| [15](stages/stage-15-verify.md) | — | Final grep + gotest |
| [16](stages/stage-16-api-surface.md) | global | Reduce ~186 → ~45 exported symbols, unify MCPServer+Handler→Server |
| [17](stages/stage-17-docs.md) | `README.md` | Rewrite docs: usage example, API table, WASM note |
