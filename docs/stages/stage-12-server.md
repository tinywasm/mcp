# Stage 12 — server.go

## 12.1 — base64: eliminate or isolate

`encoding/base64` is used in two places:

**Pagination cursor (`server.go`)** — base64 adds no value here; cursor is
already a string. Remove the encode/decode entirely:

```go
// Before
nextCursor = Cursor(base64.StdEncoding.EncodeToString([]byte(name)))
c, err := base64.StdEncoding.DecodeString(string(cursor))

// After — cursor is the raw string
nextCursor = Cursor(name)
c := string(cursor)
```

**Image result (`handler_executor.go`)** — base64 is required by the MCP
protocol for binary image content. Tools returning images are server-only.
Move `handler_executor.go` to `handler_executor.back.go` (`//go:build !wasm`).
`encoding/base64` stays in that file — isolated behind the build tag.

Result: zero `encoding/base64` in WASM-compiled code.

## 12.2 — slices: replace stdlib slices / sort

```go
// Before
import "slices"
slices.Contains(list, item)

// After — inline, or use tinywasm/fmt helpers if available
// For small lists (tools, sessions), a simple loop is acceptable:
for _, v := range list {
    if v == item { ... }
}
```

Do not import `"sort"` or `"slices"` — neither compiles with TinyGo.

## 12.3 — stdlib context in HTTP handlers

Server-side HTTP handlers receive `context.Context` from the standard library
(passed by `net/http`). This code lives in `.back.go` files only.

Within `.back.go`, use an alias to avoid collision:

```go
//go:build !wasm

import (
    stdctx "context"
    twctx  "github.com/tinywasm/context"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx := twctx.New()
    // populate from stdlib context if needed, then pass twctx forward
    s.handle(&ctx, r.Body)
}
```

Never pass `stdctx.Context` to any function that is also compiled for WASM.

## 12.4 — Remove imports

Delete from WASM-safe files: `"context"`, `"encoding/base64"`, `"sort"`, `"slices"` (stdlib)
`"encoding/base64"` remains only in `handler_executor.back.go` (`//go:build !wasm`).
Add: `"github.com/tinywasm/context"`, `"github.com/tinywasm/fmt"`
