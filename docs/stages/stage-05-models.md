# Stage 5 — Protocol models in model.go

Add all structs that need `fmt.Fielder` for `tinywasm/json` encode/decode.
All are `// ormc:formonly`. Run `ormc` once at the end.

## 5.1 — Add structs to model.go

```go
package mcp

// Already present — verify these exist:

// ormc:formonly
type rpcRequest struct {
    JSONRPC string `json:"jsonrpc"`
    ID      string `json:"id,omitempty"`
    Method  string
    Params  string `json:",omitempty"`
}

// ormc:formonly
type rpcResponse struct {
    JSONRPC string `json:"jsonrpc"`
    ID      string `json:"id,omitempty"`
    Result  string `json:",omitempty"`
    Error   string `json:",omitempty"`
}

// New structs:

// ormc:formonly
type jsonRPCError struct {
    Code    int64
    Message string
    Data    string `json:",omitempty"`
}

// ormc:formonly
type initializeParams struct {
    ProtocolVersion string             `json:"protocolVersion"`
    ClientInfo      implementationInfo `json:"clientInfo"`
}

// ormc:formonly
type implementationInfo struct {
    Name    string
    Version string
}

// ormc:formonly
type initializeResult struct {
    ProtocolVersion string             `json:"protocolVersion"`
    ServerInfo      implementationInfo `json:"serverInfo"`
    Capabilities    string             `json:",omitempty"`
}

// ormc:formonly
type callToolParams struct {
    Name      string
    Arguments string `json:",omitempty"`
}

// ormc:formonly
type callToolResult struct {
    IsError bool   `json:"isError,omitempty"`
    Content string
}

// ormc:formonly
type textContent struct {
    Type string
    Text string
}

// ormc:formonly
type toolEntry struct {
    Name        string
    Description string `json:",omitempty"`
    InputSchema string `json:"inputSchema,omitempty"`
}

// ormc:formonly
type listToolsResult struct {
    Tools      string
    NextCursor string `json:"nextCursor,omitempty"`
}

// ormc:formonly
type errorResponse struct {
    JSONRPC string       `json:"jsonrpc"`
    ID      string       `json:"id,omitempty"`
    Error   jsonRPCError
}

// ormc:formonly
type Meta struct {
    ProgressToken string `json:"progressToken,omitempty"`
}

// ormc:formonly
type NotificationParams struct {
    Meta string `json:"_meta,omitempty"`
}
```

## 5.2 — Run ormc

```bash
cd /path/to/mcp && ormc
```

Generates `model_orm.go` with `Schema()` and `Pointers()` for every struct above.

## 5.3 — Verify model_orm.go

Check that every struct in 5.1 has corresponding `Schema()` and `Pointers()` methods.
No `Validate()` should be generated — all are `formonly`.
