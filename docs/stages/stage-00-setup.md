# Stage 0 — Setup

## Install ormc

```bash
go install github.com/tinywasm/orm/cmd/ormc@latest
```

## Verify go.mod dependencies

`go.mod` must contain:

```
github.com/tinywasm/fetch  v0.1.23+
github.com/tinywasm/fmt    v0.20.0+
github.com/tinywasm/json   v0.4.0+
github.com/tinywasm/context (latest)
github.com/tinywasm/unixid (latest)
```

Run `go mod tidy` after adding any missing dependency.

## ormc field name rule

`ormc` converts Go `CamelCase` field names to `snake_case` JSON keys automatically:
- `Method` → `"method"`
- `ProgressToken` → `"progress_token"`

Add `json:` tag **only** when the MCP protocol requires a different name:

| Field | Auto key | Protocol key | Tag needed? |
|---|---|---|---|
| `JSONRPC` | (unpredictable) | `"jsonrpc"` | ✅ `json:"jsonrpc"` |
| `ProtocolVersion` | `"protocol_version"` | `"protocolVersion"` | ✅ `json:"protocolVersion"` |
| `IsError` | `"is_error"` | `"isError"` | ✅ `json:"isError"` |
| `NextCursor` | `"next_cursor"` | `"nextCursor"` | ✅ `json:"nextCursor"` |
| `InputSchema` | `"input_schema"` | `"inputSchema"` | ✅ `json:"inputSchema"` |
| `ClientInfo` | `"client_info"` | `"clientInfo"` | ✅ `json:"clientInfo"` |
| `ServerInfo` | `"server_info"` | `"serverInfo"` | ✅ `json:"serverInfo"` |
| `Name` | `"name"` | `"name"` | ❌ no tag |
| `Method` | `"method"` | `"method"` | ❌ no tag |
| `Code` | `"code"` | `"code"` | ❌ no tag |
| `Message` | `"message"` | `"message"` | ❌ no tag |
| `Meta` | `"_meta"` ← wrong | `"_meta"` | ✅ `json:"_meta"` |

## Empty struct rule

`ormc` skips structs with zero exported fields — nothing is generated.
For notifications with no params, pass `nil`:

```go
SendNotification(ctx, "notifications/tools/list_changed", nil)
```

`SendNotification` must accept `nil` params (see Stage 9).
