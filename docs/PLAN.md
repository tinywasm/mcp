# tinywasm/mcp — Enhancement Plan (ToolProvider Registration)

> **Goal:** Fix `ToolExecutor` signature and add `RegisterProvider(ToolProvider)` to
> `*MCPServer`. This eliminates all handler adapter boilerplate in every module that
> uses `tinywasm/mcp`, making tool registration a zero-friction one-liner.
>
> **Status:** Pending execution

---

## Development Rules

- **Testing Runner:** `go install github.com/tinywasm/devflow/cmd/gotest@latest`
- **SRP:** Changes are surgical — only `tools_meta.go` and one new file.
- **Max 500 lines per file.** Subdivide if exceeded.
- **Backwards compatibility:** `AddTool(tool, ToolHandlerFunc)` remains unchanged.
  `RegisterProvider` is additive. Only `ToolExecutor` is a breaking change.

---

## Context

`tools_meta.go` already defines the correct structural foundation:

```go
type ToolProvider interface {
    GetMCPToolsMetadata() []ToolMetadata
}

type ToolMetadata struct {
    Name        string
    Description string
    Parameters  []ParameterMetadata
    Execute     ToolExecutor         // ← this is the only problem
}

func buildMCPTool(meta ToolMetadata) *Tool { ... } // already exists
```

The only problem: `ToolExecutor` is `func(args map[string]any)` — no context, no return,
no error. Every module must write its own adapter to bridge this to its actual handler methods.

**Target state:** `ToolExecutor` matches the ecosystem handler signature exactly, so module
methods can be assigned directly with no intermediate code.

---

## Step 1 — Fix `ToolExecutor` Signature

**Target File:** `tools_meta.go`

Change `ToolExecutor` from:

```go
// BEFORE
type ToolExecutor func(args map[string]any)
```

To:

```go
// AFTER
type ToolExecutor func(ctx context.Context, args map[string]any) (any, error)
```

This is a breaking change for any existing `Execute` field assignments. The compiler will
catch all callsites. Update `buildMCPTool` to no longer use `Execute` (it only builds the
`*Tool` schema — execution happens in `RegisterProvider`).

---

## Step 2 — Add `RegisterProvider` to `*MCPServer`

**Target File:** `provider.go` (create new)

```go
package mcp

import "context"

// RegisterProvider registers all MCP tools declared by a ToolProvider.
// The provider's GetMCPToolsMetadata() is called once at registration time.
// Each ToolMetadata.Execute is automatically adapted to ToolHandlerFunc.
//
// This is the standard registration entry point for tinywasm ecosystem modules.
// Modules implement GetMCPToolsMetadata() and call srv.RegisterProvider(m) — one line.
//
// Example (in a domain module):
//
//   func (m *Module) RegisterTools(srv *mcp.MCPServer) {
//       srv.RegisterProvider(m)
//   }
//
//   func (m *Module) GetMCPToolsMetadata() []mcp.ToolMetadata {
//       return []mcp.ToolMetadata{
//           {Name: "get_status", Description: "...", Execute: m.GetStatus},
//           {Name: "toggle",     Description: "...", Execute: m.Toggle,
//               Parameters: []mcp.ParameterMetadata{
//                   {Name: "is_enabled", Description: "Enable or disable", Required: true, Type: "boolean"},
//               },
//           },
//       }
//   }
func (s *MCPServer) RegisterProvider(provider ToolProvider) {
    for _, meta := range provider.GetMCPToolsMetadata() {
        tool := buildMCPTool(meta)
        exec := meta.Execute // capture for closure

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
            return NewToolResultStructuredOnly(result), nil
        })
    }
}
```

---

## Step 3 — Update Tests

- Fix any existing tests that set `ToolMetadata.Execute` to match the new signature.
- Add a test for `RegisterProvider`:
  - Create a mock `ToolProvider` with 2 tools
  - Call `srv.RegisterProvider(mock)`
  - Verify both tools are registered on the server
  - Verify each tool's handler receives `ctx` and `args` correctly
  - Verify errors from `Execute` are returned as `NewToolResultError`
- Run `gotest` — 100% pass required.

---

## Step 4 — Verify & Submit

1. Run `gotest` from project root.
2. Run `gopush 'feat: fix ToolExecutor signature, add RegisterProvider for zero-friction module registration'`
