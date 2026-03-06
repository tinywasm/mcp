# MCP Library Testing Plan

This plan aims to increase the test coverage of the `tinywasm/mcp` library to ensure it correctly implements the Model Context Protocol (MCP) and works reliably in both standard Go and WebAssembly environments.

## Prerequisites

Run the following command to install the testing runner:
```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

## Development Rules

- **Single Responsibility Principle (SRP):** Every file must have a single purpose.
- **Mandatory Dependency Injection (DI):** No global state; interfaces for external dependencies.
- **Framework-less Development:** Use only the Go Standard Library for tests.
- **Max 500 lines:** Subdivision of files if they exceed this limit.
- **Testing Runner (`gotest`):** ALWAYS use `gotest` CLI command.
- **WASM/Stlib Dual Testing Pattern:** Use build tags for isomorphic code and shared test runners.
- **Diagram-Driven Testing (DDT):** Integration tests must cover branches defined in diagrams.
- **Language Protocol:** Plan in English, Chat in Spanish.

## Proposed Changes

### [Component] JSON-RPC Layer
Add tests for low-level protocol message handling.

#### [NEW] [types_test.go](tests/types_test.go)
- Test `RequestId` marshalling/unmarshalling (string vs int vs float).
- Test `Meta` additional fields handling.
- Test `NotificationParams` custom marshalling.

### [Component] Server Logic
Expand server verification.

#### [MODIFY] [server_test.go](tests/server_test.go)
- Add `TestServer_Initialize` to verify version negotiation and capabilities.
- Add `TestServer_Ping` to verify basic responsiveness.
- Add `TestServer_ListTools_Pagination` to verify cursor-based listing and limit handling.
- Add `TestServer_ToolMiddleware` to verify call chain interception.
- Add `TestServer_ToolFilter` to verify tool visibility based on context.

### [Component] Client Implementation
Implement the first set of tests for the client.

#### [NEW] [client_test.go](tests/client_test.go)
- Create a `MockTransport` that satisfies `mcp.Interface`.
- Test `Client.Initialize` and protocol version validation.
- Test `Client.ListTools` with automatic pagination.
- Test `Client.CallTool` with success and error responses.

### [Component] Integration / End-to-End
Verify the full communication loop.

#### [NEW] [integration_test.go](tests/integration_test.go)
- Implement a test that runs a real `MCPServer` and `Client` connected via an in-memory transport (e.g., `net.Pipe` or a custom mock).
- Verify a full lifecycle: Start -> Initialize -> List Tools -> Call Tool -> Close.

## Verification Plan

### Automated Tests
Run the following command to verify all tests pass:
```bash
gotest
```

### Manual Verification
- Verify that `gotest` reports 100% success for all new tests.
- Inspect the generated coverage report if available.
