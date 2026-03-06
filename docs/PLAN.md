# Plan: MCP Package — Remove Business Logic, Expose http.Handler

## References

- [ARCHITECTURE.md](ARCHITECTURE.md)
- `handler.go` — current monolith (to be trimmed)
- `handler_executor.go` — adapter that passes `tuiRefresher` (to be cleaned)
- `interface.go` — interfaces to trim
- `tinywasm/app` — the orchestrator that will own the HTTP server

---

## Development Rules

- **SRP:** Every file must have a single, well-defined purpose.
- **Max 500 lines per file.**
- **Standard library only** in test assertions.
- **Test runner:** `gotest`. **Publish:** `gopush`.
- **Language:** Plans in English, chat in Spanish.
- **No code changes** until the user says "ejecuta" or "ok".

---

## Problem

`mcp/handler.go` owns the full HTTP server lifecycle AND injects tinywasm-specific
business logic (UI actions, state provider, tab log publishing). This violates SRP:
`tinywasm/mcp` is a generic MCP JSON-RPC 2.0 library — it must not know about tinywasm's
`/action`, `/state`, `/version` endpoints or hardcoded section names and colors.

### What does NOT belong in `mcp`:
| Symbol | Reason |
|--------|--------|
| `OnUIAction(func(key, value string))` | Business routing — belongs in `app` |
| `RegisterStateProvider(fn func() []byte)` | Application state — belongs in `app` |
| `handleActionPOST` | tinywasm custom endpoint — belongs in `app` |
| `handleStateGET` | tinywasm custom endpoint — belongs in `app` |
| `handleVersion` | tinywasm custom endpoint — belongs in `app` |
| `PublishTabLog(tab, handler, color, msg)` | tinywasm SSE publishing — belongs in `app` |
| `PublishLog(msg)` | Hardcoded `"BUILD"`, `"MCP"`, `"#f97316"` — pure app logic |
| `SSEHub` interface + field | Transport concern — app injects SSE directly |
| `tuiRefresher` interface + field | TUI concern — belongs in `app` |
| Full `http.Server` lifecycle in `Serve()` | App is the orchestrator, not mcp |

### What DOES belong in `mcp`:
| Symbol | Reason |
|--------|--------|
| Tool registration (`ToolProvider`, `AddTool`) | Core MCP protocol |
| JSON-RPC 2.0 request handling | Core MCP protocol |
| `ConfigureIDEs()` | IDE integration config — generic utility |
| `Config` (port, name, version) | Kept for `ConfigureIDEs` only |
| `SetLog`, `Name` | Loggable interface — used by app to register in TUI |

---

## Solution: `mcp.Handler` → pure JSON-RPC handler

### New API

```go
// Handler is a pure MCP JSON-RPC 2.0 handler.
// It only knows how to serve the /mcp endpoint.
// The caller (app) mounts it in an http.ServeMux and owns the HTTP server.
type Handler struct { ... }

// NewHandler creates a Handler. No sseHub, no exitChan, no tuiRefresher.
func NewHandler(config Config, toolHandlers []ToolProvider) *Handler

// HTTPHandler returns the http.Handler for the /mcp endpoint only.
// Mount it as: mux.Handle("/mcp", mcpHandler.HTTPHandler())
func (h *Handler) HTTPHandler() http.Handler

// SetLog sets the logger (used by ConfigureIDEs and internal errors).
func (h *Handler) SetLog(f func(message ...any))

// ConfigureIDEs stays — it is a generic utility.
func (h *Handler) ConfigureIDEs()

// URL returns the MCP endpoint address.
func (h *Handler) URL() string

// Name returns "MCP" — satisfies the Loggable interface used by devtui/app.
func (h *Handler) Name() string
```

### Removed from `mcp.Handler`:
- `Serve()` — app starts the HTTP server
- `Stop()` — app manages server lifecycle
- `OnUIAction(...)` — moved to app
- `RegisterStateProvider(...)` — moved to app
- `handleActionPOST` — moved to app
- `handleStateGET` — moved to app
- `handleVersion` — moved to app
- `PublishTabLog(...)` — moved to app
- `PublishLog(...)` — moved to app
- `SSEHub` interface — removed; app imports `tinywasm/sse` directly
- `tuiRefresher` interface — removed; app owns TUI interaction
- `exitChan chan bool` field — removed

### `HTTPHandler()` implementation

```go
func (h *Handler) HTTPHandler() http.Handler {
    s := NewMCPServer(h.config.ServerName, h.config.ServerVersion, WithToolCapabilities(true))
    for _, provider := range h.toolHandlers {
        if provider == nil {
            continue
        }
        for _, tool := range provider.GetMCPTools() {
            s.AddToolFromUser(tool, toolExecutorAdapter(tool.Execute))
        }
    }
    return NewStreamableHTTPServer(s, WithEndpointPath("/mcp"), WithStateLess(true))
}
```

---

## Files to Modify

### `handler.go` (trim from ~270 to ~80 lines)

- Remove: `Serve`, `Stop`, `OnUIAction`, `RegisterStateProvider`, `handleActionPOST`,
  `handleStateGET`, `handleVersion`, `PublishTabLog`, `PublishLog`
- Remove fields: `sseHub`, `tui`, `exitChan`, `actionFunc`, `stateProvider`, `httpServer`, `running`
- Add: `HTTPHandler() http.Handler`
- Update `NewHandler` signature: remove `tui`, `sseHub`, `exitChan` parameters

### `interface.go`

- Remove `tuiRefresher` interface
- Remove `SSEHub` interface

### `handler_executor.go`

Current signature passes `tui tuiRefresher` to the adapter. Remove that parameter.
The adapter only needs the `Execute func(map[string]any)` — no TUI refresh needed
(app handles UI updates via its own logger injection).

```go
// Before:
func toolExecutorAdapter(provider ToolProvider, execute func(map[string]any), tui tuiRefresher) mcp.ToolHandlerFunc

// After:
func toolExecutorAdapter(execute func(map[string]any)) mcp.ToolHandlerFunc
```

Verify that the adapter body does not call `tui` methods — if it does, that logic moves to app.

---

## Execution Steps

### Step 1 — Trim `handler.go`
Remove all non-MCP methods and fields. Add `HTTPHandler()`.
Update `NewHandler` to `NewHandler(config Config, toolHandlers []ToolProvider)`.

### Step 2 — Clean `interface.go`
Remove `tuiRefresher` and `SSEHub`.

### Step 3 — Clean `handler_executor.go`
Remove `tui tuiRefresher` parameter from `toolExecutorAdapter`.
Verify nil safety — no panic if TUI-related calls were removed.

### Step 4 — Run tests and publish
```bash
gotest
gopush 'refactor: mcp becomes pure JSON-RPC handler, removes business logic'
```

---

## Test Strategy

- Existing JSON-RPC protocol tests stay unchanged (`tests/server_test.go`, etc.)
- `TestHandler_HTTPHandler_RegistersTools` — `HTTPHandler()` returns a valid `http.Handler`
  that responds to `tools/list` with registered tools
- `TestHandler_ConfigureIDEs_DoesNotPanic` — stays

---

## Impact on Consumers

| Consumer | Change Required |
|----------|----------------|
| `tinywasm/app` | Build own `http.Server`; see app PLAN.md |
| `tinywasm/devtui` | No change — devtui does not import mcp directly |
