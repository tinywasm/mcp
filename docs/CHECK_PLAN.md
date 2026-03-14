# Fix tinywasm/json Usage in MCP Client

## Context

The MCP client uses `tinywasm/json` to encode/decode JSON-RPC envelopes (`rpcRequest`, `rpcResponse`). There are two problems:

1. **`rpcResponse.Result` is `any`**: `json.Encode(envelope.Result, &resultBytes)` passes an `any` value, but `json.Encode` requires `fmt.Fielder`. When `Result` holds a raw value (not a Fielder), this fails.

2. **`rpcRequest.Params` is `any`**: Same issue — `Params` can hold arbitrary data that doesn't implement `Fielder`.

3. **Unnecessary ORM code**: `model_orm.go` generates `TableName()`, `ReadOne*`, `ReadAll*` for RPC envelope structs that are **never persisted to a database**. These structs should use `// ormc:formonly` or be hand-written with just `Schema()`/`Pointers()`.

### tinywasm/json API

```go
// Encode serializes a Fielder to JSON.
// data: must implement fmt.Fielder.
// output: *[]byte, *string, or io.Writer.
func Encode(data fmt.Fielder, output any) error

// Decode parses JSON into a Fielder.
// input: []byte, string, or io.Reader.
// data: must implement fmt.Fielder.
func Decode(input any, data fmt.Fielder) error
```

### ormc:formonly Directive

Structs annotated with `// ormc:formonly` implement `fmt.Fielder` but NOT `orm.Model`:
- Generated: `FormName()`, `Schema()`, `Pointers()`
- NOT generated: `TableName()`, `ReadOne*`, `ReadAll*`, `T_` descriptor

---

## Stage 1 — Fix rpcRequest and rpcResponse Models

**Files**: `model.go`, `model_orm.go`

The `any` typed fields (`Params`, `Result`) cannot be represented as `fmt.FieldStruct` because `any` does not implement `Fielder`. Two approaches:

### Option A — Custom Schema (Recommended)

Since these structs have `any` fields that `ormc` can't handle properly, remove them from `ormc` generation and write `Schema()`/`Pointers()` by hand in `model.go`:

1. Add `db:"-"` tags to `Params` and `Result` fields in the struct definitions, or better: annotate both structs with `// ormc:formonly`
2. For `rpcRequest`: the `Params` field should be encoded as raw JSON bytes before building the envelope. Store `Params` as `string` (pre-serialized JSON) instead of `any`
3. For `rpcResponse`: the `Result` field should be decoded as raw JSON bytes. Store `Result` as `string` (raw JSON) instead of `any`

Updated structs:
```go
// ormc:formonly
type rpcRequest struct {
    JSONRPC string `json:"jsonrpc"`
    ID      int    `json:"id"`
    Method  string `json:"method"`
    Params  string `json:"params"` // pre-serialized JSON
}

// ormc:formonly
type rpcResponse struct {
    Result string `json:"result"` // raw JSON to be decoded by caller
}
```

4. Re-run `ormc` — it will generate `FormName()`, `Schema()`, `Pointers()` only (no TableName, no ReadOne/ReadAll)
5. Delete or let `ormc` overwrite `model_orm.go`

---

## Stage 2 — Update client.go Encoding Logic

**File**: `client.go`

1. Update `buildBody()`: Serialize `params` to JSON string first, then build the envelope:
   ```go
   func (c *Client) buildBody(method string, params any) []byte {
       var paramsJSON string
       if params != nil {
           if f, ok := params.(fmt.Fielder); ok {
               if err := json.Encode(f, &paramsJSON); err != nil {
                   return nil
               }
           }
           // handle other param types as needed
       }
       req := rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: paramsJSON}
       var body []byte
       if err := json.Encode(&req, &body); err != nil {
           return nil
       }
       return body
   }
   ```

2. Update `Call()` response handling: `envelope.Result` is now `string` (raw JSON bytes), pass directly to callback:
   ```go
   callback([]byte(envelope.Result), nil)
   ```
   This eliminates the second `json.Encode` call entirely.

---

## Stage 3 — Update go.mod

1. Run `go mod tidy`
2. Remove `tinywasm/orm` dependency if no longer needed (only `tinywasm/fmt` and `tinywasm/json` should remain)

---

## Verification

```bash
gotest
```

## Linked Documents

- [ARCHITECTURE.md](ARCHITECTURE.md)
