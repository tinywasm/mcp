# Stage 1 — Cleanup and Deletions

← Back to [PLAN.md](PLAN.md) | Next → [Stage 3](PLAN_STAGE_3.md)

> **Strategy:** Do NOT use shell scripts or sed to edit existing files. Delete all out-of-scope feature files and deprecated transports directly.

---

## 1. Delete Deprecated Feature Files

Delete the following files directly from the `/home/cesar/Dev/Project/tinywasm/mcp` directory (they are no longer part of the target lean protocol):
- `resources.go`
- `prompts.go`
- `tasks.go`
- `task_hooks.go`
- `hooks.go`
- `sampling.go`
- `elicitation.go`
- `roots.go`
- `completion.go`
- `typed_tools.go`

## 2. Delete Deprecated Transports

Delete the following files related to `stdio`, `sse`, and `in-process` transports since we are converting to HTTP-only:
- `stdio.go`
- `transport_stdio.go`
- `inprocess.go`
- `inprocess_session.go`
- `transport_inprocess.go`

## 3. Review `internal/` Deprecated Packages

We want to delete the remaining unused internal packages in `internal/`, but you CANNOT do this yet.
`internal/jsonschema`, `internal/go-ordered-map`, `internal/generic-list-go`, and `internal/uritemplate` are still imported in `types.go`, `tools.go`, and `resources.go`. 

Therefore:
1. First, **complete Stage 3 (`PLAN_STAGE_3.md`)**, which will gracefully strip these imports and rewrite the references out of the core protocol files.
2. After Stage 3 is executed, verify if `internal/jsonschema/`, `internal/go-ordered-map/`, `internal/generic-list-go/`, and `internal/uritemplate/` are orphaned.
3. Only then, delete those remaining internal directories and their internal packages, followed by a `go mod tidy` verification.

---

← Back to [PLAN.md](PLAN.md) | Next → [Stage 3](PLAN_STAGE_3.md)
