# PLAN: Fix MCP Tool Output — Wire Loggable to Response Collector

## Related Docs
- [ARCHITECTURE.md](ARCHITECTURE.md)

---

## Development Rules
- No external libraries; standard library only.
- Max 500 lines per file.
- No global state; use interfaces and DI.
- Tests must use mocks, no I/O.
- Use `gotest` (not `go test`) to run tests.
- Use `gopush` (not `git commit/push`) to publish.

---

## Root Cause

In `handler_executor.go`, `toolExecutorAdapter` creates a result-collector closure but **discards it** (`_ = func(...)`). The executor calls `b.Logger(...)` → `b.log(...)` which correctly outputs to the TUI/SSE, but the MCP response collector never captures anything. The LLM always receives `"Operation completed successfully"`.

```
// BROKEN — collector created but discarded
_ = func(message ...any) { ... }   // ← dead code
executor(args)                      // → b.Logger → TUI ✓, MCP response ✗
```

The fix is **wiring**, not API change. `rebuildMCPServer` already has access to each `ToolProvider`. If the provider implements `Loggable`, we temporarily swap its logger to the MCP collector during execution, then restore it.

---

## Output Routing Design

| Caller | `b.log` during execution | TUI sees | LLM receives |
|---|---|---|---|
| LLM (default) | collector only | nothing | the data |
| LLM (debug mode) | wrapper(original + collector) | the data | the data |
| Human (TUI shortcut) | original TUI logger | the data | nothing |

---

## Step 1 — Add `GetLog()` to `Loggable` interface

**File:** `provider.go`

```go
// BEFORE
type Loggable interface {
    Name() string
    SetLog(logger func(message ...any))
}

// AFTER
type Loggable interface {
    Name() string
    SetLog(logger func(message ...any))
    GetLog() func(message ...any)
}
```

---

## Step 2 — Add `DebugToolOutput` to `Config`

**File:** `handler.go`

```go
type Config struct {
    Port            string
    ServerName      string
    ServerVersion   string
    AppName         string
    AppVersion      string
    DebugToolOutput bool   // if true, tool output goes to both TUI and MCP response
}
```

---

## Step 3 — Update `toolExecutorAdapter` signature

**File:** `handler_executor.go`

Add `loggable Loggable` and `debug bool` parameters. Before executing, swap the logger to the collector (or a wrapper in debug mode). Restore after execution with `defer`.

```go
func toolExecutorAdapter(executor ToolExecutor, loggable Loggable, debug bool) ToolHandlerFunc {
    return func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
        args := req.GetArguments()

        var messages []string
        var binaryResponse *BinaryData

        collector := func(message ...any) {
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
        }

        // Swap logger: route output to MCP response collector
        if loggable != nil {
            original := loggable.GetLog()
            if debug {
                loggable.SetLog(func(msg ...any) {
                    original(msg...)    // → TUI (developer sees output)
                    collector(msg...)   // → MCP response (LLM receives)
                })
            } else {
                loggable.SetLog(collector)  // → MCP response only
            }
            defer loggable.SetLog(original)
        } else {
            // No Loggable — executor must call collector directly (fallback)
            executor(args)
            goto buildResponse
        }

        executor(args)

    buildResponse:
        if binaryResponse != nil {
            base64Data := base64.StdEncoding.EncodeToString(binaryResponse.Data)
            text := strings.Join(messages, "\n")
            return NewToolResultImage(text, base64Data, binaryResponse.MimeType), nil
        }
        if len(messages) == 0 {
            return NewToolResultText("Operation completed successfully"), nil
        }
        return NewToolResultText(strings.Join(messages, "\n")), nil
    }
}
```

---

## Step 4 — Update `rebuildMCPServer` to pass `Loggable`

**File:** `handler.go`

```go
func (h *Handler) rebuildMCPServer() {
    s := NewMCPServer(h.config.ServerName, h.config.ServerVersion,
        WithToolCapabilities(true),
        WithToolHandlerMiddleware(h.toolAuthMiddleware()),
    )
    newMeta := make(map[string]Tool)

    h.mu.RLock()
    all := append(append([]ToolProvider{}, h.fixed...), h.dynamic...)
    h.mu.RUnlock()

    for _, p := range all {
        if p == nil {
            continue
        }
        loggable, _ := p.(Loggable)
        for _, tool := range p.GetMCPTools() {
            s.AddToolFromUser(tool, toolExecutorAdapter(tool.Execute, loggable, h.config.DebugToolOutput))
            newMeta[tool.Name] = tool
        }
    }

    h.mu.Lock()
    h.mcpServer = s
    h.toolMeta = newMeta
    h.mu.Unlock()
}
```

---

## Step 5 — Implement `GetLog()` on all Loggable consumers

Every struct that implements `Loggable` (DevBrowser, Handler, etc.) must add:

```go
func (b *DevBrowser) GetLog() func(message ...any) {
    return b.log
}
```

This is the **only change needed in downstream libraries** (devbrowser, app).

---

## Step 6 — Update tests

**File:** `tests/handler_test.go`

- **Test 1 — Text result:** Create a tool whose `Execute` calls `b.Logger("hello")`. Call it via MCP. Assert response content equals `"hello"` (not `"Operation completed successfully"`).
- **Test 2 — Binary result:** Create a tool whose `Execute` calls `b.Logger(mcp.BinaryData{...})`. Assert response is an image result.
- **Test 3 — Debug mode:** Set `DebugToolOutput: true`. Assert both TUI logger AND MCP response receive the output.
- **Test 4 — Non-Loggable provider:** A provider that does NOT implement `Loggable`. Assert it still returns `"Operation completed successfully"` (no crash).

---

## Impact Summary

| Library | Files changed | Nature |
|---|---|---|
| `tinywasm/mcp` | `provider.go`, `handler.go`, `handler_executor.go`, `tests/handler_test.go` | Core fix — all changes here |
| `tinywasm/devbrowser` | `devbrowser.go` | Add `GetLog()` (1 method, 3 lines) |
| `tinywasm/app` | `handler.go` or equivalent | Add `GetLog()` (1 method, 3 lines) |

**No changes to `Execute` signature. No changes to any tool definition.**

---

## Publish
After tests pass: `gopush 'fix: wire Loggable logger to MCP response collector'`
