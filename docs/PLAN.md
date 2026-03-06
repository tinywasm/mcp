# tinywasm/mcp — Protocol-Only Refactoring Plan

> **Goal:** Strip `tinywasm/mcp` to a lean MCP protocol library — tools + JSON-RPC + HTTP transport only. No SSE, stdio, OAuth, Resources, Prompts, Tasks, or session state. Session lifecycle is managed by consumer.

---

## Development Rules
- **Testing Runner:** `go install github.com/tinywasm/devflow/cmd/gotest@latest`
- **Standard Library Only:** No external assertion libraries. No `testify`.
- **Max 500 lines per file.** Subdivide if exceeded.
- **SRP:** Each file has a single purpose, named by domain.
- **No third-party dependencies:** Only stdlib + `tinywasm/*` ecosystem packages.
- **Flat structure:** All source in root. Tests in `tests/`. No new subdirectories.

---

## Current Status and Execution Order

Phase 1 and Phrase 2 of the previous master plan were only **partially completed**. Some feature and transport files meant to be removed still exist. We need to finish the deletion of deprecated files and then rewrite the core files to remove the remaining legacy types.

### 1. [Stage 1: Cleanup and Deletions](PLAN_STAGE_1_CLEANUP.md)
Delete remaining out-of-scope feature files (Prompts, Resources, Tasks, etc.) and deprecated transports (stdio, inprocess).

### 2. [Stage 3: Core File Rewrite](PLAN_STAGE_3.md)
Surgically remove deprecated types from core files (`utils.go`, `tools.go`, `types.go`) and completely rewrite the `MCPServer` to be stateless. 

### 3. Verify and Submit
After executing Stage 1 and Stage 3:
1. Delete unused internal packages from `internal/`
2. Run `go mod tidy` to clean up dependencies
3. Run `gotest`
4. Run `gopush 'refactor: strip mcp to protocol-only HTTP transport'`
