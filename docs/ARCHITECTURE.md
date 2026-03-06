# Architecture: tinywasm/mcp

## Purpose

Lean Go implementation of MCP (Model Context Protocol) over JSON-RPC 2.0. The library provides two levels of abstraction: a simple user-facing API for defining tools, and an internal protocol layer that handles JSON-RPC serialization and MCP session lifecycle.

## Two-Layer Design

```
┌─────────────────────────────────────┐
│  USER LAYER (public API)            │
│  Tool, Parameter, ToolProvider      │  ← consumers define their tools here
│  provider.go                        │
└──────────────┬──────────────────────┘
               │ AddToolFromUser()
               ▼ parametersToInputSchema()
┌─────────────────────────────────────┐
│  PROTOCOL LAYER (internal)          │
│  ProtocolTool, ToolInputSchema      │  ← JSON-RPC 2.0 wire format
│  tools.go, server.go                │
└──────────────┬──────────────────────┘
               │ JSON-RPC
               ▼
           MCP Client (LLM)
```

**Key principle:** Users never interact with `ProtocolTool` directly. They define `Tool` structs with `Parameters`, and the library auto-generates the `ToolInputSchema` needed by the MCP protocol.

## Files

| File | Responsibility |
|------|---------------|
| `provider.go` | Public API: `Tool`, `Parameter`, `ToolProvider`, `Loggable`, `BinaryData` |
| `tools.go` | Internal: `ProtocolTool`, `ToolInputSchema`, `parametersToInputSchema()`, `CallToolRequest/Result` |
| `server.go` | `MCPServer`: tool registry, JSON-RPC dispatch, `AddTool`, `AddToolFromUser`, `ListTools` |
| `handler.go` | `Handler`: HTTP server wiring, SSE injection, IDE config, tool lifecycle |
| `handler_executor.go` | `toolExecutorAdapter`: bridges `ToolExecutor func` into `ToolHandlerFunc` |
| `types.go` | JSON-RPC 2.0 primitives: `JSONRPCRequest`, `JSONRPCResponse`, `InitializeRequest`, etc. |
| `transport_streamable_http.go` | `StreamableHTTPServer`: SSE-based MCP transport |
| `request_handler.go` | `RequestHandler`: routes raw JSON-RPC messages to `MCPServer` |
| `tool_builders.go` | Legacy builder API (`WithString`, `WithNumber`, etc.) — kept for advanced use |
| `client.go` | `Client`: MCP client for calling tool servers |
| `utils.go` | Result helpers: `NewToolResultText`, `NewToolResultError`, `NewToolResultImage` |
| `session.go` | Session lifecycle helpers |
| `errors.go` | Error constants and helpers |

## Data Flow: Tool Registration

```
Consumer calls GetMCPTools() → []Tool
    │
    ▼
Handler.Serve() iterates tools
    │
    ▼
MCPServer.AddToolFromUser(userTool, handler)
    │
    ├─ parametersToInputSchema(userTool.Parameters)
    │      → builds ToolInputSchema{Properties, Required}
    │
    └─ stores as ServerTool{ProtocolTool, ToolHandlerFunc}
```

## Data Flow: Tool Call

```
LLM → POST /mcp (JSON-RPC: tools/call)
    │
    ▼
StreamableHTTPServer → RequestHandler
    │
    ▼
MCPServer.handleToolCall()
    │
    ▼
toolExecutorAdapter(handler, executor, tui)
    │
    ├─ injects SetLog() into handler if Loggable
    ├─ calls executor(args)
    ├─ collects string messages + BinaryData from logger
    └─ returns CallToolResult{Content: TextContent | ImageContent}
```

## SSE Injection (DI)

`Handler` depends on `SSEHub` interface, not on the concrete `sse.SSEServer`:

```go
type SSEHub interface {
    http.Handler
    Publish(data []byte, channel string)
}
```

The consumer creates a real `*sse.SSEServer` and injects it at construction time via `NewHandler(...)`. This keeps `mcp` decoupled from `sse`.

## Constraints

- No global state
- No external dependencies (stdlib only)
- `ProtocolTool` is internal: users always work with `Tool`
- `tool_builders.go` exists for advanced protocol-level use only
