# PLAN: Compatibilidad MCP con Claude Code

## Problema

Claude Code no puede conectarse al MCP server en `:3030/mcp`. Dos fallos propios de este módulo:

### Fallo 1 — `model_orm.go` genera nombres snake_case

El ORM produce `ColumnName` en snake_case. Los structs de `tinywasm/mcp` son serialización pura JSON y el protocolo MCP usa camelCase:

```
"protocol_version"  ← generado actualmente
"protocolVersion"   ← lo que envía Claude Code
```

`tinywasm/json` hace match exacto → campo no encontrado → `ProtocolVersion` vacío → error `"unsupported protocol version: "`. Lo mismo afecta la serialización de respuestas.

**Prerequisito:** `tinywasm/orm` debe publicar soporte para respetar el tag `json:` como `ColumnName` (ver su PLAN.md).

**Fix:** Añadir tags `json:` en `model.go` con los nombres exactos del protocolo MCP y regenerar `model_orm.go`:

```go
type initializeParams struct {
    ProtocolVersion string             `json:"protocolVersion"`
    ClientInfo      implementationInfo `json:"clientInfo"`
}

type initializeResult struct {
    ProtocolVersion string `json:"protocolVersion"`
    // ...
}
```

Los structs de respuesta (`JSONRPCResponseStruct`, etc.) también necesitan sus tags para producir `"jsonrpc"`, `"id"`, `"error"`, `"result"`.

### Fallo 2 — `handleInitialize` valida auth antes de procesar `initialize`

En `request_handler.go`, la autorización ocurre antes del switch de métodos. El spec MCP requiere que `initialize` sea accesible sin token previo (el cliente aún no tiene sesión).

**Fix:** Mover el case `MethodInitialize` antes del bloque de auth:

```go
// Primero manejar initialize sin auth
if MCPMethod(method) == MethodInitialize {
    ...
    return createResponse(id, result)
}

// Auth para el resto
userID, err := s.auth.Authorize(token)
...
```

## Verificación

```bash
curl -X POST http://localhost:3030/mcp -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"claude","version":"1.0"}}}'

# Esperado:
# {"jsonrpc":"2.0","id":"1","result":{"protocolVersion":"2024-11-05","serverInfo":{...}}}
```

## Orden de ejecución

1. Esperar `tinywasm/orm` con soporte de tag `json:` como `ColumnName`
2. Añadir tags `json:` en `model.go` y regenerar `model_orm.md`
3. Mover auth después de `initialize` en `request_handler.go`
4. Publicar nuevo tag de `tinywasm/mcp`
