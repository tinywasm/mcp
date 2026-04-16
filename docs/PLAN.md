# PLAN: Eliminar double-encoding en respuestas JSON-RPC

## Problema

Las respuestas MCP tienen el campo `result` double-encoded (JSON string en lugar de JSON object), lo que impide que Claude Code parsee las tools correctamente.

**Respuesta actual:**
```json
{"jsonrpc":"2.0","id":"1","result":"{\"tools\":\"[{\\\"name\\\":\\\"start_development\\\"}]\"}"}
```

**Respuesta esperada por spec MCP:**
```json
{"jsonrpc":"2.0","id":"1","result":{"tools":[{"name":"start_development",...}]}}
```

## Causa raíz

Dos puntos de double-encoding en cadena:

### 1. `utils.go` — `newResultResponse` encodea el resultado a `string`

```go
func newResultResponse(id RequestId, result any) JSONRPCMessage {
    var resJSON string
    json.Encode(f, &resJSON)          // encodea a string
    return &JSONRPCResponseStruct{
        Result: resJSON,              // string guardado en campo string
    }
}
```

Cuando `daemon.go` encodea `JSONRPCResponseStruct` completo, el campo `Result string` se vuelve a encodear → string dentro de string.

### 2. `listToolsResult.Tools` y campos similares son `string` conteniendo JSON

```go
type listToolsResult struct {
    Tools      string  // contiene "[{...}]" como string
    NextCursor string
}
```

El array de tools se serializa a string antes de meterse en `listToolsResult`, luego ese string se encodea de nuevo al serializar la respuesta completa.

## Fix

### Opción A — `result` como JSON raw en la respuesta (recomendada)

`JSONRPCResponseStruct.Result` debe ser `json.RawMessage` (bytes raw) en lugar de `string`, para que `tinywasm/json` lo emita tal cual sin re-encodear:

Pero `tinywasm/json` trabaja con `fmt.Fielder` y `string`. La solución dentro del ecosistema es que `newResultResponse` construya el JSON de la respuesta completa directamente como bytes, sin pasar por un struct intermedio:

```go
func newResultResponse(id RequestId, result fmt.Fielder) []byte {
    var resultJSON []byte
    json.Encode(result, &resultJSON)
    // construir la respuesta manualmente con el result como objeto inline
    idJSON := fmt.Sprintf("%q", id)
    return fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":%s}`, idJSON, resultJSON)
}
```

`JSONRPCMessage` pasaría a retornar `[]byte` en lugar de un `fmt.Fielder`.

### Opción B — Adaptar `tinywasm/json` para emitir un campo string como raw JSON

Añadir en `tinywasm/fmt` un tipo de campo `FieldRaw` que `tinywasm/json` emita sin quotes. Los campos `Result`, `Error`, `Tools` usarían `FieldRaw` en su schema. El ORM generaría el tipo correcto cuando el campo tenga tag `json:",raw"` o similar.

## Impacto adicional: `error` también está double-encoded

```go
func newErrorResponse(...) JSONRPCMessage {
    json.Encode(det, &detJSON)   // string
    return &JSONRPCError{
        Error: detJSON,          // string re-encodeado
    }
}
```

Mismo patrón — debe emitirse como objeto inline.

## Recomendación

**Opción A** — construir las respuestas JSON-RPC directamente como bytes en `newResultResponse` y `newErrorResponse`, sin struct intermedio. Es el cambio más quirúrgico: solo afecta `utils.go` y el tipo de retorno de `JSONRPCMessage`. No requiere cambios en `tinywasm/json` ni en el ORM.

El tipo `JSONRPCMessage` cambiaría de interfaz a `[]byte`:

```go
// Antes
type JSONRPCMessage interface { jsonrpcMessage() }

// Después  
type JSONRPCMessage []byte
```

Y `HandleMessage` retornaría `[]byte` directamente, simplificando también el handler en `daemon.go`.
