# Stage 1.5 — Strict JSON-RPC 2.0 Responses

### Problem
`HandleMessage` has four gaps that break JSON-RPC 2.0 compliance:

1. **`JSONRPCMessage` is `type any`** — the consumer has no compile-time guarantee that the returned value is a valid JSON-RPC 2.0 response. Any struct (or nil) can leak through.
2. **Params are decoded from the full message, not from the `params` field** — `json.Decode(message, &p)` receives the entire JSON-RPC envelope. The ORM schema for `callToolParams` looks for `"name"` in the JSON, but the root has `"method"`, `"id"`, `"jsonrpc"` — not `"name"`. The field `"name"` is nested inside `"params"`. Result: `p.Name` is always empty, and `s.tools[""]` fails with "tool not found".
3. **`json.Decode` errors on params are silently ignored** — if a client sends malformed `params`, the handler receives a zero-value struct instead of returning `INVALID_PARAMS (-32602)`.
4. **No single contract** — success and error responses are different structs (`JSONRPCResponseStruct`, `JSONRPCError`) with no common interface, making downstream handling fragile.

### Critical: Params Extraction Bug

This is the root cause of tool call failures. A JSON-RPC message looks like:

```json
{"jsonrpc":"2.0", "id":"1", "method":"tools/call", "params":{"name":"myTool","arguments":"{}"}}
```

The current code does:
```go
// WRONG — decodes the full envelope, not the params object
var p callToolParams
json.Decode(message, &p)  // p.Name = "" because root has no "name" field
```

The fix requires extracting the raw `params` value BEFORE decoding:
```go
// CORRECT — extract params first, then decode the inner object
raw := extractJSONValue(message, "params")
var p callToolParams
if err := json.Decode(raw, &p); err != nil {
    return createErrorResponse(id, INVALID_PARAMS, "Invalid params: "+err.Error())
}
```

---

### Design

#### A. `extractJSONValue` — extract raw JSON field values

Add a function that extracts the raw value of a JSON field (objects `{...}`, arrays `[...]`, strings `"..."`, or primitives). This is needed because `extractJSONString` only handles string values.

```go
// extractJSONValue returns the raw JSON bytes of a field value.
// Handles objects {}, arrays [], quoted strings, and primitives.
// Returns nil if the key is not found.
func extractJSONValue(data []byte, key string) []byte
```

Implementation must handle:
- Nested braces/brackets (count depth)
- Quoted strings (skip escaped quotes)
- Return the complete value including delimiters

#### B. Sealed `JSONRPCMessage` interface

Replace `type JSONRPCMessage any` with a sealed marker interface:

```go
// JSONRPCMessage is the return type of HandleMessage.
// Only JSONRPCResponseStruct and JSONRPCError implement it.
// nil is valid only for notifications (no id).
type JSONRPCMessage interface {
    jsonrpcMessage() // unexported marker — seals the interface to this package
}

func (r *JSONRPCResponseStruct) jsonrpcMessage() {}
func (e *JSONRPCError) jsonrpcMessage()          {}
```

This makes it impossible for HandleMessage to return an unstructured value.

#### C. Fix all param decoding in `HandleMessage`

Every method that has params must: (1) extract raw params, (2) decode, (3) check error.

```go
case MethodInitialize:
    raw := extractJSONValue(message, "params")
    var p initializeParams
    if err := json.Decode(raw, &p); err != nil {
        return createErrorResponse(id, INVALID_PARAMS, "Invalid params: "+err.Error())
    }
    result, reqErr := s.handleInitialize(ctx, id, p)
    // ...

case MethodToolsList:
    raw := extractJSONValue(message, "params")
    var p PaginatedParams
    if err := json.Decode(raw, &p); err != nil {
        return createErrorResponse(id, INVALID_PARAMS, "Invalid params: "+err.Error())
    }
    result, reqErr := s.handleListTools(ctx, id, p)
    // ...

case MethodToolsCall:
    raw := extractJSONValue(message, "params")
    var p callToolParams
    if err := json.Decode(raw, &p); err != nil {
        return createErrorResponse(id, INVALID_PARAMS, "Invalid params: "+err.Error())
    }
    result, reqErr := s.handleToolCall(ctx, id, p)
    // ...
```

#### D. Guarantee response on every code path

| Input condition | Response |
|----------------|----------|
| Unparseable JSON (no valid `jsonrpc` field) | `PARSE_ERROR (-32700)`, id from best-effort extraction or empty |
| `jsonrpc != "2.0"` | `INVALID_REQUEST (-32600)` |
| Valid notification (no `id`) | `nil` — correct per spec |
| Unknown method | `METHOD_NOT_FOUND (-32601)` |
| Malformed params | `INVALID_PARAMS (-32602)` |
| Handler error | Error via `requestError.ToJSONRPCError()` |
| Handler success | `JSONRPCResponseStruct` with result |

---

### Tests

```
TestExtractJSONValue_Object
    → {"params":{"name":"test"}} → returns {"name":"test"}

TestExtractJSONValue_NestedObject
    → {"params":{"name":"test","args":{"x":1}}} → returns full nested object

TestExtractJSONValue_Missing
    → {"method":"ping"} → returns nil

TestExtractJSONValue_String
    → {"id":"abc"} → returns "abc" (with quotes)

TestHandleMessage_ToolCall_ParamsDecoded
    → full JSON-RPC tools/call message → p.Name is correctly decoded from params, not from root

TestHandleMessage_MalformedJSON_ParseError
    → body "not json" → PARSE_ERROR -32700

TestHandleMessage_WrongVersion_InvalidRequest
    → jsonrpc: "1.0" → INVALID_REQUEST -32600

TestHandleMessage_EmptyBody_ParseError
    → body "" → PARSE_ERROR -32700

TestHandleMessage_InvalidToolCallParams_InvalidParams
    → tools/call with params: "not an object" → INVALID_PARAMS -32602

TestHandleMessage_InvalidInitializeParams_InvalidParams
    → initialize with broken params → INVALID_PARAMS -32602

TestHandleMessage_InvalidListToolsParams_InvalidParams
    → tools/list with broken params → INVALID_PARAMS -32602

TestHandleMessage_Notification_ReturnsNil
    → valid jsonrpc, no id → nil

TestHandleMessage_ValidPing_ReturnsResponse
    → ping → JSONRPCResponseStruct with jsonrpc "2.0"

TestResponseType_IsJSONRPCMessage
    → JSONRPCResponseStruct and JSONRPCError both satisfy JSONRPCMessage interface (compile-time)
```

---

### Steps (execute in order)
- [ ] 1. Add `extractJSONValue(data []byte, key string) []byte` to request_handler.go
- [ ] 2. Add tests for `extractJSONValue` (object, nested, missing, string cases)
- [ ] 3. Replace `type JSONRPCMessage any` with sealed interface + marker methods in types.go
- [ ] 4. Update ALL `json.Decode` calls in `HandleMessage` to use `extractJSONValue(message, "params")` first
- [ ] 5. Add error checks to all `json.Decode` calls → return `INVALID_PARAMS` on failure
- [ ] 6. Add all remaining tests listed above
- [ ] 7. Verify all existing tests still compile with the new `JSONRPCMessage` type
