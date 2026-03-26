# Stage 3 — server_http and transport split

Both files depend on `net/http` and are server-only.

## 3.1 — Rename server_http.go → server_http.back.go

Add as first line:
```go
//go:build !wasm
```

Replace in this file:
- `json.NewDecoder(r.Body).Decode(&msg)` → `json.Decode(r.Body, &msg)`
- `json.NewEncoder(w).Encode(resp)` → encode to `[]byte` then `w.Write(b)`
- Error response `map[string]any{...}` → use `errorResponse` struct from model.go
- `strings.*` → `tinywasm/fmt` equivalents

No `.front.go` stub needed — this type is not referenced from WASM-safe code.

## 3.2 — Rename transport_streamable_http.go → transport_streamable_http.back.go

Add as first line:
```go
//go:build !wasm
```

This file uses stdlib `context.Context` for HTTP request cancellation
(`http.NewRequestWithContext`). HTTP cancellation requires `context.Done()` which
`tinywasm/context` does not provide.

Strategy: accept `*context.Context` (tinywasm) at the public boundary,
convert internally for stdlib HTTP calls only:

```go
// At the boundary — public method signature
func (t *StreamableHTTP) SendRequest(ctx *context.Context, req JSONRPCRequest) (*JSONRPCResponse, error) {
    // Internal stdlib context for HTTP cancellation only
    httpCtx := stdctx.Background()
    httpReq, _ := stdhttp.NewRequestWithContext(httpCtx, ...)
    ...
}
```

Import stdlib context with alias to avoid collision:
```go
import (
    stdctx "context"
    "github.com/tinywasm/context"
)
```

Replace:
- `encoding/json` → `tinywasm/json`
- `"strings"` → `tinywasm/fmt`
- `map[string]string` headers field → `[]fmt.KeyValue` (see Stage 10)
