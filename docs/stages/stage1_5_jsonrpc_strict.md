# Stage 1.5 — Strict JSON-RPC 2.0 Responses

### Problem
`HandleMessage` has three gaps that break JSON-RPC 2.0 compliance:

1. **`JSONRPCMessage` is `type any`** — the consumer has no compile-time guarantee that the returned value is a valid JSON-RPC 2.0 response. Any struct (or nil) can leak through.
2. **`json.Decode` errors on params are silently ignored** — if a client sends malformed `params`, the handler receives a zero-value struct instead of returning `INVALID_PARAMS (-32602)`.
3. **No single contract** — success and error responses are different structs (`JSONRPCResponseStruct`, `JSONRPCError`) with no common interface, making downstream handling fragile.

### Design

#### A. Sealed `JSONRPCMessage` interface

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

#### B. Validate `json.Decode` on params

Every `json.Decode` of params must check for errors:

```go
case MethodToolsCall:
    var p callToolParams
    if err := json.Decode(message, &p); err != nil {
        return createErrorResponse(id, INVALID_PARAMS, "Invalid params: "+err.Error())
    }
    result, reqErr := s.handleToolCall(ctx, id, p)
    if reqErr != nil {
        return reqErr.ToJSONRPCError()
    }
    return createResponse(id, result)
```

Apply the same pattern to `MethodInitialize`, `MethodToolsList`, and any future method.

#### C. Guarantee response on every code path

| Input condition | Response |
|----------------|----------|
| Unparseable JSON (no valid `jsonrpc` field) | `PARSE_ERROR (-32700)`, id from best-effort extraction or empty |
| `jsonrpc != "2.0"` | `INVALID_REQUEST (-32600)` |
| Valid notification (no `id`) | `nil` — correct per spec |
| Unknown method | `METHOD_NOT_FOUND (-32601)` |
| Malformed params | `INVALID_PARAMS (-32602)` |
| Handler error | Error via `requestError.ToJSONRPCError()` |
| Handler success | `JSONRPCResponseStruct` with result |

### Tests

```
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

### Steps
- [ ] Replace `type JSONRPCMessage any` with sealed interface + marker methods
- [ ] Add `json.Decode` error checks to all param-parsing cases in `HandleMessage`
- [ ] Add tests listed above
- [ ] Verify all existing tests still compile with the new `JSONRPCMessage` type
