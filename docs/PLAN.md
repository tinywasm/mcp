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

## Stage 2 — Regenerate model_orm.go with ormc

The `ormc` code generator reads struct definitions from `model.go`/`models.go` and produces `model_orm.go` with `Schema()`, `Pointers()`, and other methods based on the struct fields and tags.

### 2.1 — Install ormc

```bash
go install github.com/tinywasm/orm/cmd/ormc@latest
```

### 2.2 — Run ormc

From the project root directory:

```bash
ormc
```

This will scan for `model.go` and regenerate `model_orm.go`.

### 2.3 — Expected output in model_orm.go

Because both structs use `// ormc:formonly`, the generated file should contain ONLY:

For `rpcRequest`:
- `func (m *rpcRequest) FormName() string` → returns `"rpc_request"`
- `func (m *rpcRequest) Schema() []fmt.Field` → returns fields with types `fmt.FieldText` (for JSONRPC, Method, Params), `fmt.FieldInt` (for ID), and `JSON` tags from struct tags
- `func (m *rpcRequest) Pointers() []any` → returns `[]any{&m.JSONRPC, &m.ID, &m.Method, &m.Params}`

For `rpcResponse`:
- `func (m *rpcResponse) FormName() string` → returns `"rpc_response"`
- `func (m *rpcResponse) Schema() []fmt.Field` → returns `[]fmt.Field{{Name: "result", Type: fmt.FieldText, JSON: "result"}}`
- `func (m *rpcResponse) Pointers() []any` → returns `[]any{&m.Result}`

### 2.4 — Verify

- Import must be `"github.com/tinywasm/fmt"` only (NO `"github.com/tinywasm/orm"`)
- There must be NO `TableName()`, `ReadOne*`, `ReadAll*`, or `T_` descriptor generated
- If `model_orm.go` still contains ORM methods, delete it and re-run `ormc`

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
