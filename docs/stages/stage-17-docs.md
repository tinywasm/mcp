# Stage 17 — Documentation

## 17.1 — README.md: complete usage example

```go
package main

import (
    "github.com/tinywasm/context"
    "github.com/tinywasm/mcp"
)

// 1. Define tool arguments — ormc generates Schema(), Pointers(), Validate()
type SearchArgs struct {
    Query string `validate:"required,min=1,max=255"`
    Limit int64  `validate:"min=1,max=100"`
}

func main() {
    // 2. Create server
    srv := mcp.NewServer(mcp.Config{
        Name:    "my-server",
        Version: "1.0.0",
        Auth:    myAuth, // implements mcp.Authorizer
    }, nil)

    // 3. Register tool — one object, one call
    srv.AddTool(mcp.Tool{
        Name:        "search",
        Description: "Search items by query",
        InputSchema: new(SearchArgs).Schema(),
        Resource:    "items",
        Action:      'r',
        Run: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
            var args SearchArgs
            if err := req.Bind(&args); err != nil {
                return nil, err // server wraps as JSON-RPC error
            }
            return mcp.Text("found 3 results"), nil
        },
    })

    // 4. Serve (server-only)
    http.Handle("/mcp", srv.HTTPHandler())
    http.ListenAndServe(":3030", nil)
}
```

### Authorizer

mcp defines the interface, `tinywasm/user` implements it:

```go
// mcp.Authorizer — validates a bearer token, returns user ID
type Authorizer interface {
    Authorize(token string) (userID string, err error)
}

// In your app:
auth := user.NewBearerAuth(secret) // satisfies mcp.Authorizer
```

### ToolProvider

Group related tools in a provider:

```go
type CatalogProvider struct { db *postgres.DB }

// Arguments live next to the provider — ormc generates Schema/Pointers/Validate
type CatalogSearchArgs struct {
    Query    string `validate:"required,min=1,max=255"`
    Category string `validate:"max=50"`
}

type CatalogUpdateArgs struct {
    ProductID string  `validate:"required"`
    Price     float64 `validate:"required,min=0"`
}

func (p *CatalogProvider) Tools() []mcp.Tool {
    return []mcp.Tool{
        {
            Name:        "catalog_search",
            Description: "Search product catalog",
            InputSchema: new(CatalogSearchArgs).Schema(),
            Resource:    "catalog",
            Action:      'r',
            Run:         p.handleSearch,
        },
        {
            Name:        "catalog_update",
            Description: "Update product price",
            InputSchema: new(CatalogUpdateArgs).Schema(),
            Resource:    "catalog",
            Action:      'u',
            Run:         p.handleUpdate,
        },
    }
}

func (p *CatalogProvider) handleSearch(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
    var args CatalogSearchArgs
    if err := req.Bind(&args); err != nil {
        return nil, err
    }
    // args.Query and args.Category are validated
    // ... query p.db
    return mcp.Text("found 3 products"), nil
}

func (p *CatalogProvider) handleUpdate(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
    var args CatalogUpdateArgs
    if err := req.Bind(&args); err != nil {
        return nil, err
    }
    // ... update p.db
    return mcp.Text("price updated"), nil
}

// Pass providers to NewServer
srv := mcp.NewServer(config, []mcp.ToolProvider{&CatalogProvider{db: db}})
```

## 17.2 — WASM usage note

```markdown
## WASM / Browser

The protocol core compiles with TinyGo. Server-only files (`//go:build !wasm`)
are excluded automatically.

In browser mode, call the handler directly — no HTTP server needed:

    var handler = mcp.NewServer(config, providers)
    response := handler.HandleMessage(&ctx, message)
```

## 17.3 — Create docs/WHY_ARQ.md

Architecture decisions and trade-offs go in a dedicated file, not in README.

```markdown
# Why this architecture

## Goals

1. Compile with TinyGo for browser execution (WASM)
2. Minimal public API — one way to do each thing
3. Type-safe arguments with validation at the boundary
4. Mandatory access control on every tool

## Pros

- **Minimal API** — ~45 exported symbols. Less to learn, less to break.
- **One way to register tools** — `AddTool` or `ToolProvider`. No parallel APIs.
- **Type-safe arguments** — `req.Bind(&args)` decodes and validates in one call. No `map[string]any`.
- **RBAC always required** — every tool declares who can use it. No accidental open access.
- **Automatic error wrapping** — `return nil, err` is enough. The server produces JSON-RPC errors.
- **Provider groups tools + dependencies** — database, config, and related tools in one struct.
- **WASM-ready** — the protocol core compiles with TinyGo for browser use.

## Cons

- **Breaking change** — existing callers must rewrite tool registration and argument handling.
- **`ormc` is required** — without running `ormc`, `Schema()` and `Validate()` don't exist. Add it to CI.
- **Schema is a runtime string** — `new(Args).Schema()` is opaque at compile time. If you change the struct and forget to run `ormc`, the schema is stale.
- **3 lines of boilerplate per handler** — `var args; req.Bind(&args); if err` repeats in every tool. This is explicit by design.
- **No prototyping without RBAC** — you must define `Resource` and `Action` even for throwaway tools. This is intentional.

## Key decisions

| Decision | Why |
|----------|-----|
| No `encoding/json` | stdlib json uses reflection — incompatible with TinyGo |
| No `map[string]any` | Not type-safe, not WASM-safe, hides bugs |
| `ormc` generates Schema/Validate | No hand-written boilerplate, single source of truth |
| `tinywasm/context` instead of stdlib | stdlib context uses `any` values and `Done()` channel — neither works in TinyGo |
| Auth is interface only | mcp is protocol, not identity management — `tinywasm/user` implements |
| Session ID only in context | mcp doesn't own sessions — `tinywasm/user` does |
| `//go:build !wasm` for HTTP | HTTP is server-only; browser uses `HandleMessage` directly |
```

## 17.4 — Delete outdated README sections

Remove any reference to:
- `map[string]any`, `GetArguments()`, `GetString()`, `GetBool()`
- `ProtocolTool`, `NewProtocolTool`, builder DSL
- `CallToolRequest`, `CallToolResult` (renamed to `Request`, `Result`)
- `AddToolFromUser`, `ToolHandlerFunc`
- `OpenAuthorizer`, `NewTokenAuthorizer`
- `ContextWithSession`, `SessionFromContext`
- `NewTextContent`, `NewImageContent`, `NewAudioContent`

## 17.4 — API quick reference

Add a concise table of the full public API at the end of README:

```markdown
## API

| Symbol | Description |
|--------|-------------|
| `NewServer(config, providers)` | Create server |
| `Server.AddTool(tool)` | Register a tool |
| `Server.HTTPHandler()` | HTTP endpoint (server-only) |
| `Server.HandleMessage(ctx, msg)` | Process JSON-RPC message |
| `Tool{Name, Description, InputSchema, Resource, Action, Run}` | Tool definition |
| `ToolProvider` | Interface: `Tools() []Tool` |
| `Authorizer` | Interface: `Authorize(token) (userID, error)` |
| `Request` | Incoming tool call |
| `Request.Bind(target)` | Decode + validate arguments |
| `Result` | Tool call result |
| `Text(s)` | Create text result |
| `JSON(data)` | Create JSON result |
| `GetText(result)` | Extract text from result |
| `Config` | Server configuration |
| `FilterFunc` | Filter tools by context |
```
