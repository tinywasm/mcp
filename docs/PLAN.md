# PLAN: tinywasm/mcp — Corrections and Enhancements

> Date: 2026-03-28
> Updated: 2026-03-31
> Status: In progress — partially completed (see progress by stage)

---

## Confirmed Design Decisions

| Decision | Resolution |
|----------|-----------|
| Auth nil | `NewServer` returns error if `Config.Auth == nil`. No silent open access. |
| RBAC Interface name | Merged into `Authorizer` (see Stage 3) — a single interface, not two. |
| HTTP Transport | Streamable HTTP via `tinywasm/sse` injected by the consumer. |
| SSE injection | `Config.SSE SSETransport` — consumer passes `*sse.SSEServer` directly. |
| Dead code | Removed: `session.go`, `transport_interface.go`, `handler_ide.front.go`. |
| `.back.go` files | Marked `//go:build ignore` — they don't compile, kept as reference. |
| `ConfigureIDEs` | Reverting: extracted into `tinywasm/app` to preserve protocol purity. |
| `RegisterMethod` | Deleted. App uses its own HTTP router for custom routes. |
| `tinywasm/app` | Requires migration — separate plan in `tinywasm/app/docs/PLAN.md`. |
| **JSON-RPC 2.0 strict** | `HandleMessage` ALWAYS returns a valid JSON-RPC 2.0 response or nil (notification). No exceptions. See Stage 1.5. |
| **No HTTP ownership** | `mcp` never owns HTTP routing. `HTTPHandler`, `RegisterRoutes`, `HTTPEngine` belong to `tinywasm/app`. `mcp` only exposes `HandleMessage(ctx, []byte) JSONRPCMessage`. |

---

## Compatibility Status: tinywasm/user

The functions `GenerateAPIToken`, `ValidateJWT`, `CanExecute`, `InjectIdentity` are **documented but not implemented** in `tinywasm/user`. Only the following exists:
- `m.HasPermission(userID, resource string, action byte) (bool, error)` ✅

Stage 3 of this plan and the `tinywasm/user` plan must be coordinated.

---

## Implementation Stages

For an orderly execution and review, the plan has been divided into the following modules:

### [Stage 1 — Cleanup ✅ COMPLETED](stages/stage1_cleanup.md)
Removal of dead code and recovery of critical functions.

### [Stage 1.5 — Strict JSON-RPC 2.0 Responses](stages/stage1_5_jsonrpc_strict.md)
Guarantee every `HandleMessage` call returns valid JSON-RPC 2.0. Typed response interface, param validation, no silent failures.

### [Stage 2 — Startup Security](stages/stage2_security.md)
Ensuring the server doesn't start without an explicit security configuration.

### [Stage 3 — Mandatory RBAC (Unified Authorizer)](stages/stage3_rbac.md)
Implementation of role-based access control integrating Auth and RBAC.

### [Stage 4 — SSE Transport (Streamable HTTP)](stages/stage4_sse.md)
Support for streaming responses via Server-Sent Events.

### [Stage 5 — Documentation](stages/stage5_documentation.md)
Updating README and ARCHITECTURE.md to reflect the new state of the package.

---

## Execution Order

```
Stage 1 ✅ → Stage 1.5 → Stage 2 → Stage 3 → Stage 4 → Stage 5
```

Stages 2 and 3 are blockers for Stage 4 because `HTTPHandler` depends on `Authorizer` having `Can()`.

---

## External Dependencies Added

| Package | Stage | Reason |
|---------|-------|--------|
| `github.com/tinywasm/sse` | Stage 4 | `SSETransport` interface + `HTTPHandler` |

Note: the dependency is only in `!wasm` builds. The protocol core remains without external dependencies.
