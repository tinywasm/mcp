# Fix MCP Tests After tinywasm/json Migration

## Context

The client code was correctly migrated to use `tinywasm/json` with `fmt.Fielder` structs. However, the tests still pass `map[string]string` and `map[string]any` as params to `client.Call()` / `client.Dispatch()`. These types don't implement `fmt.Fielder`, so `buildBody()` silently produces empty params — causing test failures.

### Key Constraint

**`encoding/json` is allowed in test files** (they don't compile to WASM). Only the main package source files must use `tinywasm/json` exclusively.

---

## Stage 1 — Create test param structs

**File**: `tests/handler_test.go`

Add `ormc:formonly` structs to replace the anonymous maps used as params:

```go
// ormc:formonly
type actionParams struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ormc:formonly
type toolCallParams struct {
	Name string `json:"name"`
}
```

Then run `ormc` from the `tests/` directory to generate `tests/handler_test_orm.go` with `Schema()` and `Pointers()` for these structs.

**Important**: `ormc` scans for structs in `*model*.go` or `*_test.go` files. Since these are in a test file, you may need to create a `tests/model.go` file with the structs instead. After running `ormc`, the generated file will be `tests/model_orm.go`.

### 1.1 — Install ormc

```bash
go install github.com/tinywasm/orm/cmd/ormc@latest
```

### 1.2 — Create tests/model.go

```go
package mcp_test

// ormc:formonly
type actionParams struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ormc:formonly
type toolCallParams struct {
	Name string `json:"name"`
}
```

### 1.3 — Run ormc

```bash
cd tests && ormc
```

### 1.4 — Verify tests/model_orm.go

Should contain `FormName()`, `Schema()`, `Pointers()` for both structs. No `TableName()`, no ORM methods.

---

## Stage 2 — Update test calls

**File**: `tests/handler_test.go`

### 2.1 — TestHandler_OnAction_JSONRPCMethod (line ~83)

Replace:
```go
client.Dispatch(context.Background(), "tinywasm/action", map[string]string{"key": "foo", "value": "bar"})
```

With:
```go
client.Dispatch(context.Background(), "tinywasm/action", &actionParams{Key: "foo", Value: "bar"})
```

### 2.2 — TestHandler_ToolOutput_TextResult (line ~166)

Replace:
```go
client.Call(context.Background(), "tools/call", map[string]any{"name": "echo_tool"}, func(data []byte, err error) {
```

With:
```go
client.Call(context.Background(), "tools/call", &toolCallParams{Name: "echo_tool"}, func(data []byte, err error) {
```

### 2.3 — TestHandler_ToolRBAC_Denied (line ~205)

Replace:
```go
client.Call(context.Background(), "tools/call", map[string]any{"name": "secure_tool"}, func(data []byte, err error) {
```

With:
```go
client.Call(context.Background(), "tools/call", &toolCallParams{Name: "secure_tool"}, func(data []byte, err error) {
```

---

## Stage 3 — Verify

```bash
gotest
```

All tests must pass. Do NOT modify `client.go`, `model.go`, or `model_orm.go` — those are already correct.

## Forbidden Actions

- Do NOT modify any source files outside of `tests/`
- Do NOT import `encoding/json` in non-test source files
- Do NOT revert the client.go changes from the previous iteration
