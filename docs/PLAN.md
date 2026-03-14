# Fix tinywasm/json Usage in MCP Client — Iteration 2

## Critical Constraint

**NEVER import `encoding/json`**. This package is FORBIDDEN in tinywasm projects because it uses reflection, which inflates WASM binaries by ~90KB. ALL JSON operations MUST use `tinywasm/json` exclusively.

## Context

The previous iteration incorrectly replaced `tinywasm/json` with `encoding/json`. This must be reverted. The real fix is to change the `any` fields in `rpcRequest` and `rpcResponse` to `string` so they can be handled by `tinywasm/json` (which requires `fmt.Fielder`).

### tinywasm/json API

```go
// Encode serializes a Fielder to JSON.
func Encode(data fmt.Fielder, output any) error  // output: *[]byte, *string, or io.Writer

// Decode parses JSON into a Fielder.
func Decode(input any, data fmt.Fielder) error    // input: []byte, string, or io.Reader
```

### ormc:formonly Directive

Structs annotated with `// ormc:formonly` implement `fmt.Fielder` but NOT `orm.Model`:
- Generated: `FormName()`, `Schema()`, `Pointers()`
- NOT generated: `TableName()`, `ReadOne*`, `ReadAll*`, `T_` descriptor

---

## Stage 1 — Update model.go

**File**: `model.go`

Replace the current structs with `string` typed fields and `ormc:formonly` annotation:

```go
// ormc:formonly
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  string `json:"params"` // pre-serialized JSON string
}

// ormc:formonly
type rpcResponse struct {
	Result string `json:"result"` // raw JSON string from server
}
```

**Do NOT use `any` for Params or Result.** They must be `string` so `tinywasm/json` can encode/decode them via `fmt.Fielder`.

---

## Stage 2 — Regenerate model_orm.go

1. Run `ormc` from project root
2. Verify generated file only contains `FormName()`, `Schema()`, `Pointers()` (no `TableName`, no `ReadOne*`, no `ReadAll*`)
3. Verify import is `tinywasm/fmt` only (no `tinywasm/orm`)

---

## Stage 3 — Update client.go

**File**: `client.go`

### 3.1 — Imports

```go
import (
	"context"
	"strings"

	"github.com/tinywasm/fetch"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
)
```

**Do NOT import `encoding/json`.** This is forbidden.

### 3.2 — buildBody()

Serialize `params` to a JSON string first using `tinywasm/json`, then embed it in the request envelope:

```go
func (c *Client) buildBody(method string, params any) []byte {
	var paramsJSON string
	if params != nil {
		if f, ok := params.(fmt.Fielder); ok {
			if err := json.Encode(f, &paramsJSON); err != nil {
				return nil
			}
		}
	}
	req := rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: paramsJSON}
	var body []byte
	if err := json.Encode(&req, &body); err != nil {
		return nil
	}
	return body
}
```

### 3.3 — Call() response handling

Decode the response envelope using `tinywasm/json`, then pass `Result` (which is already a raw JSON string) as bytes to the callback:

```go
var envelope rpcResponse
if err := json.Decode(resp.Body(), &envelope); err != nil {
	callback(nil, err)
	return
}
if envelope.Result == "" {
	callback(nil, nil)
	return
}
callback([]byte(envelope.Result), nil)
```

This eliminates the second `json.Encode` call that existed before.

---

## Stage 4 — Remove orm dependency

**File**: `go.mod`

1. Run `go mod tidy`
2. Verify `tinywasm/orm` is no longer in go.mod (only `tinywasm/fmt`, `tinywasm/json`, `tinywasm/fetch`)

---

## Verification

```bash
gotest
```

## Forbidden Actions

- Do NOT import `encoding/json` anywhere
- Do NOT use `json.Marshal`, `json.Unmarshal`, `json.RawMessage`
- Do NOT use `any` type for struct fields that need JSON encoding via `tinywasm/json`
