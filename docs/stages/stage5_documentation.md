# Stage 5 — Documentation

### Problem
- `ARCHITECTURE.md` describes an API that doesn't exist
- README mentions `user.NewBearerAuth(secret)` which doesn't exist
- `ormc` without link or instructions
- Incomplete API table

### README Changes

Add `ormc` installation section:
```bash
go install github.com/tinywasm/orm/cmd/ormc@latest
ormc gen ./...  # generates Schema(), Pointers(), Validate() for tagged structs
```

Replace Authorizer section:
```go
// Static API key — all authenticated users can use all tools
srv, err := mcp.NewServer(mcp.Config{
    Auth: mcp.NewTokenAuthorizer(os.Getenv("MCP_API_KEY")),
    ...
}, providers)

// tinywasm/user integration (requires user implementation to be complete)
srv, err := mcp.NewServer(mcp.Config{
    Auth: userModule.MCPAuthorizer(),
    ...
}, providers)

// Local/trusted environment — explicit opt-in
srv, err := mcp.NewServer(mcp.Config{
    Auth: mcp.OpenAuthorizer(),
    ...
}, providers)
```

Add SSE section:
```go
tinySSE := sse.New(&sse.Config{})
sseServer := tinySSE.Server(&sse.ServerConfig{...})

srv, err := mcp.NewServer(mcp.Config{
    SSE: sseServer,   // *sse.SSEServer satisfies mcp.SSEPublisher directly
    ...
}, providers)

// Consumer (tinywasm/app) owns HTTP routing:
mux.Handle("POST /mcp", appHTTPHandler(srv))
mux.Handle("GET /events", sseServer)
```

### Rewrite ARCHITECTURE.md
- Remove references to `Handler`, `MCPServer`, `ProtocolTool`, `SSEHub`, `Parameter`
- Document current file structure
- Authentication + RBAC flow diagram
- Streamable HTTP flow diagram with SSE

### Steps
- [ ] Rewrite ARCHITECTURE.md with current files
- [ ] Update README: auth, SSE, ormc
- [ ] Complete API reference table
- [ ] Add auth+RBAC flow diagram
