# Stage 1 — client.go

**File**: `client.go`

Two stdlib imports remain. Replace both.

## 1.1 — Replace `"strings"`

```go
// Before
endpoint: strings.TrimSuffix(baseURL, "/") + "/mcp",

// After
endpoint: fmt.Convert(baseURL).TrimSuffix("/").String() + "/mcp",
```

Remove import `"strings"`.

## 1.2 — Replace stdlib `"context"`

```go
// Before
import "context"
func (c *Client) Call(ctx context.Context, method string, params any, callback func([]byte, error))
func (c *Client) Dispatch(ctx context.Context, method string, params any)

// After
import "github.com/tinywasm/context"
func (c *Client) Call(ctx *context.Context, method string, params any, callback func([]byte, error))
func (c *Client) Dispatch(ctx *context.Context, method string, params any)
```

Remove import `"context"`.

## Result

`client.go` imports after this stage:

```go
import (
    "github.com/tinywasm/context"
    "github.com/tinywasm/fetch"
    "github.com/tinywasm/fmt"
    "github.com/tinywasm/json"
)
```
