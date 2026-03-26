# Stage 10 — interface.go + transport_interface.go

## 10.1 — interface.go: tag as !wasm

`interface.go` defines server-side abstractions that depend on `net/http`.
Rename to `interface.back.go` and add build tag:

```go
//go:build !wasm
```

No logic changes — only the build constraint is added.

## 10.2 — HTTPHeaderFunc: update context type only

`HTTPHeaderFunc` lives in `interface.back.go` (`!wasm`) — `map[string]string` is valid there.
Only the context parameter changes:

```go
// Before
import "context"
type HTTPHeaderFunc func(ctx context.Context) map[string]string

// After
import "github.com/tinywasm/context"
type HTTPHeaderFunc func(ctx *context.Context) map[string]string
```

## 10.3 — transport_interface.go: split by platform

`ClientTransport` has no platform-specific imports — keep in `transport_interface.go` (no build tag, WASM-safe).

`HTTPClientTransport` depends on `net/http` — move to `transport_interface.back.go`:

```go
//go:build !wasm
```

## 10.4 — Remove imports

In `interface.back.go`:
- Delete: `"context"` (stdlib)
- Add: `"github.com/tinywasm/context"`

In `transport_interface.back.go`:
- Delete: `"context"` (stdlib)
- Add: `"github.com/tinywasm/context"`
