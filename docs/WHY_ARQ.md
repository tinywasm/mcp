# Why this architecture

## Goals

1. Compile with TinyGo for browser execution (WASM)
2. Minimal public API — one way to do each thing
3. Type-safe arguments with validation at the boundary
4. Mandatory access control on every tool

---

## Pros

- **Minimal API** — ~45 exported symbols. Less to learn, less to break.
- **One way to register tools** — `AddTool` or `ToolProvider`. No parallel APIs.
- **Type-safe arguments** — `req.Bind(&args)` decodes and validates in one call. No `map[string]any`.
- **RBAC always required** — every tool declares who can use it. No accidental open access.
- **Automatic error wrapping** — `return nil, err` is enough. The server produces JSON-RPC errors.
- **Provider groups tools + dependencies** — database, config, and related tools in one struct.
- **WASM-ready** — the protocol core compiles with TinyGo for browser use.

---

## Cons

- **Breaking change** — existing callers must rewrite tool registration and argument handling.
- **`ormc` is required** — without running `ormc`, `Schema()` and `Validate()` don't exist. Add it to CI.
- **Schema is a runtime string** — `new(Args).Schema()` is opaque at compile time. If you change the struct and forget to run `ormc`, the schema is stale.
- **3 lines of boilerplate per handler** — `var args; req.Bind(&args); if err` repeats in every tool. This is explicit by design.
- **No prototyping without RBAC** — you must define `Resource` and `Action` even for throwaway tools. This is intentional.

---

## Key decisions

| Decision | Why |
|----------|-----|
| No `encoding/json` | stdlib json uses reflection — incompatible with TinyGo |
| No `map[string]any` | Not type-safe, not WASM-safe, hides bugs |
| `ormc` generates Schema/Validate | No hand-written boilerplate, single source of truth |
| `tinywasm/context` instead of stdlib | stdlib context uses `any` values and `Done()` channel — neither works in TinyGo |
| Auth is interface only | mcp is protocol, not identity management — `tinywasm/user` implements |
| Session ID only in context | mcp doesn't own sessions — `tinywasm/user` does |
| `//go:build !wasm` for HTTP | HTTP is server-only; browser uses `HandleMessage` directly |
| Tool.Run is the API | Single registration path: `AddTool` or `ToolProvider.Tools()` returns Tool with Run handler |
| Mandatory Resource/Action | Every tool declares RBAC intent, prevents "open by default" bugs |
| Result.Content is string | Avoids nested slice types that `tinywasm/json` doesn't support as top-level |
