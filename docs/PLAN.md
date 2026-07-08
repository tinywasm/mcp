# PLAN — `mcp.NewCaller`: the router.Caller adapter (single owner of the tools/call envelope)

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.

## Context (zero-context summary)

`tinywasm/mcp` owns the MCP JSON-RPC protocol: `Client` (wasm-friendly HTTP
client), `Server` (dispatches `initialize`/`ping`/`tools/list`/`tools/call`),
`CallToolParams`, `ParseResult`.

Today, consumers' views import `mcp` directly and hand-build calls. That leaks
the wire protocol into code that must not know it, and already produced a real
silent bug in `veltylabs/mjosefa-cms`: the view called
`client.Call(ctx, "list_services", …)` — passing the **tool name** as the
JSON-RPC `method`. The server only dispatches `tools/call`, so every real call
returned `METHOD_NOT_FOUND` and the view silently rendered nothing.

`tinywasm/router` now defines the client-side contract (see
`router/docs/PLAN.md`, a **gate for this plan** — do not start until it is
merged and tagged):

```go
type Caller interface {
	Call(op string, args model.Encodable, callback func(result []byte, err error))
	Dispatch(op string, args model.Encodable)
}
```

**Goal:** ship the one and only adapter from `*mcp.Client` to `router.Caller`.
After this, `mcp` is the ONLY package that knows `"tools/call"`,
`CallToolParams` and `ParseResult`; consumers depend on `router.Caller` alone.

## Change — new file `caller.go` (additive, no break)

```go
package mcp

// NewCaller adapts *Client to the router.Caller contract. It is the single
// place that knows the tools/call envelope. Consumers (views) depend on
// router.Caller only and pass logical operation names (tool names).
func NewCaller(c *Client) router.Caller
```

Adapter rules (final, each one exists because its violation was a real bug):

1. **`Call(op, args, cb)`** builds `method = MethodToolsCall` (`"tools/call"`)
   with `CallToolParams{Name: op, Arguments: <args encoded>}`. The consumer can
   no longer pass a raw wire method — the METHOD_NOT_FOUND bug becomes
   unwritable.
2. **The callback receives fully unwrapped domain bytes.** `Client.Call`
   already strips the JSON-RPC envelope; the adapter additionally applies
   `ParseResult` and hands `Result.Content` bytes to the callback. The consumer
   only `json.Decode`s its own domain type.
3. **Every error propagates.** Transport errors, JSON-RPC error objects,
   `ParseResult` failures, and `IsError` tool results all reach the callback's
   `err` — never swallowed, never a silent empty result (this library itself
   swallowed JSON-RPC errors once; fixed in v0.1.17 — do not regress).
4. **`Dispatch(op, args)`** wraps the same envelope over `Client.Dispatch`
   (fire-and-forget).
5. The adapter supplies its own internal `*context.Context`
   (`context.Background()`); `router.Caller` deliberately has no ctx parameter.

Use existing typed constants (`MethodToolsCall`); no string literals in logic.

## Tests

- **Unit (native):** adapter against a `httptest` server built from this repo's
  own `Server` with a fake tool provider — asserts:
  - wire request has `method:"tools/call"` and `params.name == op`,
  - callback receives exactly the tool's Content bytes (decodable domain JSON),
  - a JSON-RPC error and an `IsError` result both surface via `err`,
  - `Dispatch` sends and ignores the response.
- Existing client/server tests must stay green: `gotest ./...` (never
  `go test`). Prerequisite:
  `go install github.com/tinywasm/devflow/cmd/gotest@latest`.

## Documentation (mandatory)

- `README.md`: "Calling tools from a view" section rewritten around
  `mcp.NewCaller` — show the composition-root wiring
  (`mcp.NewCaller(mcp.NewClient(origin))`) and state that views must depend on
  `router.Caller`, never on `*mcp.Client`.
- `docs/ARCHITECTURE.md`: add the adapter to the component map (Client →
  NewCaller → router.Caller) and note the "single owner of the envelope" rule.
- `docs/SKILL.md`: update usage conventions if they show direct `Client.Call`
  from views.

## Harness checklist (mandatory)

- No stdlib imports (`tinywasm/fmt`/`json`/`context`/`fetch` only).
- No `any`, no `map`, no generics in the public API (`args` is
  `model.Encodable`, already this library's encoding requirement).
- No hardcoded strings in logic: reuse `MethodToolsCall` and existing constants.
- Additive only: `Client`, `Server`, existing signatures untouched.
- Errors are values that reach the caller; no logging-instead-of-returning.

## Acceptance criteria

1. `gotest ./...` green (native + wasm targets).
2. A consumer view compiled against `router.Caller` + this adapter performs a
   real `tools/call` round-trip in the unit test described above.
3. `grep -rn "tools/call" .` outside this package's own files finds nothing in
   downstream repos after their migration (verified there; here: the constant
   is used, not the literal).
4. README/ARCHITECTURE/SKILL updated as specified.

## Stages

| Stage | File | Action | Gate |
|---|---|---|---|
| 1 | `caller.go` | `NewCaller` adapter per rules 1–5 | router plan merged+tagged |
| 2 | `caller_test.go` | round-trip + error-propagation tests vs own `Server` | 1 |
| 3 | `README.md`, `docs/ARCHITECTURE.md`, `docs/SKILL.md` | document adapter + wiring rule | 2 |
