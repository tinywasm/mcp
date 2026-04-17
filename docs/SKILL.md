---
module: github.com/tinywasm/mcp
version: see go.mod
protocol: MCP (Model Context Protocol) over JSON-RPC 2.0
---

# SKILL: tinywasm/mcp

Lean Go MCP server + client library. Protocol-only, WASM-safe, no HTTP ownership.

## What it does

- Handles JSON-RPC 2.0 MCP messages: `initialize`, `ping`, `tools/list`, `tools/call`
- Mandatory RBAC on every tool call via `Authorizer` interface
- Optional SSE streaming via injected `SSEPublisher`
- Dual-use: server (backend) + client (WASM frontend via `tinywasm/fetch`)

## Core types

```go
// Server
mcp.NewServer(Config{Name, Version, Auth, SSE}, []ToolProvider) (*Server, error)
srv.HandleMessage(ctx, []byte) JSONRPCMessage   // main entry point
srv.AddTool(Tool)                                // register at runtime

// Tool definition
mcp.Tool{
    Name, Description string
    InputSchema string   // JSON schema from ormc: new(Args).Schema()
    Resource    string   // RBAC resource e.g. "catalog"
    Action      byte     // 'c','r','u','d'
    Execute     func(ctx, Request) (*Result, error)
}

// Tool handler pattern
func handle(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
    var args MyArgs
    if err := req.Bind(&args); err != nil { return nil, err }
    return mcp.Text("ok"), nil
}

// Result helpers
mcp.Text("string")         // *Result with text content
mcp.JSON(&myFielder{})     // *Result with JSON content
mcp.GetText(result)        // extract text string

// Auth
mcp.NewTokenAuthorizer("apikey")  // Bearer token, Can() always true
mcp.OpenAuthorizer()               // no auth, open access

// Client (WASM-safe, uses tinywasm/fetch)
c := mcp.NewClient("http://host", "apikey")
c.Call(ctx, "tools/call", params, func(data []byte, err error){})
c.Dispatch(ctx, "tools/call", params)  // fire-and-forget

// Context keys
mcp.CtxKeyAuthToken   // set before HandleMessage: ctx.Set(CtxKeyAuthToken, token)
mcp.CtxKeyUserID      // set by server after Authorize
mcp.CtxKeySessionID   // set by server on initialize
```

## ToolProvider pattern

```go
type MyProvider struct{ db *postgres.DB }

func (p *MyProvider) Tools() []mcp.Tool {
    return []mcp.Tool{{
        Name: "my_tool", Resource: "items", Action: 'r',
        InputSchema: new(MyArgs).Schema(),
        Execute: p.handle,
    }}
}

srv, _ := mcp.NewServer(config, []mcp.ToolProvider{&MyProvider{db: db}})
```

## HTTP integration (consumer owns routing)

```go
http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    ctx := context.New()
    ctx.Set(mcp.CtxKeyAuthToken, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
    resp := srv.HandleMessage(&ctx, body)
    // encode resp (fmt.Fielder) to JSON and write
})
```

## Code generation (ormc)

```bash
go install github.com/tinywasm/orm/cmd/ormc@latest
# annotate struct with // ormc:formonly
# run: go generate ./...
```

## Constraints

- `Config.Auth` must not be nil — use `OpenAuthorizer()` for open access
- Every `Tool` must have `Resource` and `Action` set
- No stdlib `encoding/json` — uses `tinywasm/json` (TinyGo compatible)
- SSE is optional; when nil no streaming occurs
- WASM build excludes server-only files via `//go:build !wasm`

## Files

| File | Role |
|------|------|
| `server.go` | `Server`, `NewServer`, handlers (init/ping/list/call) |
| `request_handler.go` | `HandleMessage` dispatch, JSON extraction, context keys |
| `mcp_auth.go` | `Authorizer` interface + built-in implementations |
| `tools.go` | `Tool`, `Request`, `Request.Bind`, `Text`, `ToolProvider` |
| `client.go` | `Client` for WASM frontend use |
| `publish_sse.go` | `SSEPublisher` interface |
| `model.go` | JSON-RPC 2.0 type definitions (ormc annotated) |
| `model_orm.go` | Generated: `Schema()`, `Pointers()`, `Validate()` |
| `utils.go` | `JSON`, `GetText`, `ParseResult`, response builders |
| `constants.go` | HTTP header constants, content type constants |
| `types.go` | MCP method constants, error codes, `JSONRPCMessage` interface |
