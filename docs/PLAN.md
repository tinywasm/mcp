# PLAN — execution queue for `tinywasm/mcp`

> If you were told to "execute the plan described in docs/PLAN.md", execute
> **ALL the plans below, in order (top to bottom)**. Each plan is
> self-contained; finish one (its acceptance criteria green) before starting
> the next. Never mix changes from one plan into another.

| Order | Plan | Subject | Gate |
|-------|------|---------|------|
| 1 | [PLAN_JSONRPC_PROTOCOL_COMPLIANCE.md](PLAN_JSONRPC_PROTOCOL_COMPLIANCE.md) | Protocol version negotiation in `initialize` + echo request `id` preserving its JSON type | None — dispatchable now (MCP daemon hardening wave; its e2e gate waits on this) |
| 2 | [PLAN_KIND_UNIFICATION_INPUTSCHEMA.md](PLAN_KIND_UNIFICATION_INPUTSCHEMA.md) | Kind unification phase B: `inputSchema` derivation reads `Field.Type.Storage()` | Requires the phase-A `tinywasm/model` PUBLISHED (Kind + `Struct(ref)`) |
| 3 | [PLAN_bearer_auth_pending.md](PLAN_bearer_auth_pending.md) | Bearer auth (previous wave) | None, lowest priority |

Order rationale: the JSON-RPC fixes belong to the MCP daemon hardening wave
(no upstream gate, and that wave's final e2e verification waits on them); the
Kind migration is blocked until `model` publishes; bearer auth blocks nothing.
Cross-wave dispatch order lives in `tinywasm/docs/ROADMAPS_PLANS.md`.

After completing all plans, run `gotest ./...` one final time: everything green.
