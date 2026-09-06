# PLAN — Kind unification (phase B): tool `inputSchema` derivation reads `Field.Type.Storage()`

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
> Phase B of `webtyp/docs/KIND_UNIFICATION_MASTER_PLAN.md` (Kind unification wave). Requires
> the published phase-A `webtyp/model`. Runs parallel to orm/form/sqlt/postgres.
>
> NOTE: `docs/PLAN_bearer_auth_pending.md` is a separate, unrelated pending
> plan from a previous wave — do NOT touch, merge, or delete it.

## Context (zero-context summary)

Phase A changed `webtyp/model`: `Field.Type` is no longer the `FieldType`
enum but the interface

```go
type Kind interface {
    Storage() FieldType   // the enum survives here — same values, same meaning
    Name() string
    Validate(value string) error
}
```

This repo derives each tool's MCP `inputSchema` (JSON Schema) from
`Tool.Args`'s generated `Schema() []model.Field`. `tool_schema.go` declares
`jsonSchemaType(t model.FieldType) string` and switches over the enum. That
signature is **correct and stays**; every call site now feeds it
`f.Type.Storage()`.

## Stage 1 — mechanical migration

- Bump `webtyp/model` to the phase-A version.
- `jsonSchemaType` keeps its `model.FieldType` parameter; call sites in
  `tool_schema.go` (and anywhere else the compiler flags) pass
  `f.Type.Storage()`.
- Test fixtures building `model.Field` literals by hand (e.g. `model.go`,
  schema tests) switch to the phase-A base kind constructors
  (`Type: model.Text()`, `model.Int()`, …). Composition fields migrate BOTH
  slots: `{Type: model.FieldStruct, Ref: &implementationInfoModel}` becomes
  `{Type: model.Struct(&implementationInfoModel)}` — the `Ref:` is DELETED
  (it now means scalar FK only; leaving it alongside a composition kind is a
  contradiction and an ormc generation error). Then regenerate via
  `//go:generate ormc` where applicable (requires the phase-B ormc — if
  ormc is not yet published when this dispatches, hand-migrate the literals
  and leave generated files to a regen commit; note it in the summary).

## Stage 2 — tests

- `gotest ./...` green with no weakened assertions: the emitted
  `inputSchema` JSON for existing fixtures must be byte-identical to before
  the migration (this guards the Claude Code Zod validation contract — an
  invalid `inputSchema` silently hides ALL tools).

## Harness checklist (mandatory)

- No behavior change: call-site migration only. If the `Kind` contract is
  insufficient here, **STOP and report** to the master plan.
- Only `webtyp/json` for JSON — stdlib `encoding/json` forbidden.
- No unrelated refactors; `gotest` only.
- Breaking dependency bump: next minor version.

## Acceptance criteria

1. Module compiles against phase-A model; all enum access goes through
   `.Storage()`; `jsonSchemaType` signature unchanged.
2. `inputSchema` output byte-identical for existing fixtures;
   `gotest ./...` green.

## Stages

| Stage | File(s) | Action |
|---|---|---|
| 1 | `tool_schema.go` call sites, `model.go` + fixtures | `.Storage()` migration, base-kind literals |
| 2 | `tests/*_test.go` | inputSchema byte-identical regression |
