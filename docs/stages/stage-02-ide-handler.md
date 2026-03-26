# Stage 2 — handler_ide split

`handler_ide.go` reads/writes IDE config files from the filesystem.
Meaningless in the browser. Split into server-only + WASM stub.

## 2.1 — Define IDE config structs in model.go

The current code uses `map[string]any` to parse IDE JSON configs.
Replace with typed structs so `tinywasm/json` can handle them.

Add to `model.go` (root package, not tests/):

```go
// ormc:formonly
type ideServerEntry struct {
    URL     string `json:"url"`
    Headers string `json:"headers,omitempty"`
}

// ormc:formonly
type vscodeConfig struct {
    Servers string `json:"servers,omitempty"`
}

// ormc:formonly
type claudeCodeConfig struct {
    MCPServers string `json:"mcpServers,omitempty"`
}
```

Note: nested objects (servers map, headers map) are stored as `string`
(JSON-in-string pattern) because `tinywasm/json` does not support `map` fields.

Run `ormc` after adding structs (see Stage 5 for full ormc run).

## 2.2 — Create handler_ide.back.go

Move ALL content of `handler_ide.go` to `handler_ide.back.go`.
Add as first line:

```go
//go:build !wasm
```

Replace in this file:
- `encoding/json` → `tinywasm/json` using the structs from 2.1
- `strings.TrimSpace` → `fmt.Convert(s).TrimSpace().String()`
- `strings.Join` → `fmt.Join(slice, sep)` or manual loop
- `strings.ToLower` → `fmt.Convert(s).ToLower().String()`

Keep `"os"`, `"path/filepath"`, `"runtime"` — valid server-only deps.

## 2.3 — Create handler_ide.front.go

```go
//go:build wasm

package mcp

// ConfigureIDEs is a no-op in WASM environments.
func (h *Handler) ConfigureIDEs() error { return nil }
```

## 2.4 — Delete handler_ide.go

The original file is fully replaced by the two new files.
