# PLAN: Usar FieldRaw para campos JSON inline

## Contexto

`tinywasm/fmt` v0.23.3 añadió `FieldRaw`. `tinywasm/json` y `tinywasm/orm` lo implementarán. Una vez publicados, `tinywasm/mcp` solo necesita añadir el tag `raw` a los campos que contienen JSON pre-serializado y regenerar `model_orm.go`.

## Prerequisitos

- `tinywasm/json` publicado con soporte `FieldRaw` en encode/decode
- `tinywasm/orm` publicado con detección de `json:",raw"`

## Campos a actualizar en `model.go`

Los campos que contienen JSON y hoy causan double-encoding:

```go
// JSONRPCResponseStruct — result y error son objetos JSON
type JSONRPCResponseStruct struct {
    JSONRPC string `json:"jsonrpc"`
    ID      string `json:"id,omitempty"`
    Result  string `json:"result,omitempty,raw"`
    Error   string `json:"error,omitempty,raw"`
}

// JSONRPCError — error es un objeto JSON
type JSONRPCError struct {
    JSONRPC string `json:"jsonrpc"`
    ID      string `json:"id,omitempty"`
    Error   string `json:"error,raw"`
}

// listToolsResult — tools es un array JSON
type listToolsResult struct {
    Tools      string `json:"tools,raw"`
    NextCursor string `json:"nextCursor,omitempty"`
}

// Result — content es un objeto/array JSON
type Result struct {
    IsError bool   `json:"isError,omitempty"`
    Content string `json:"content,raw"`
}
```

## Acción

1. Actualizar dependencias a las nuevas versiones de `json` y `orm`
2. Añadir `,raw` a los campos listados en `model.go`
3. Regenerar: `go generate ./...`
4. Verificar respuesta MCP sin double-encoding:

```bash
curl -s -X POST http://localhost:3030/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# Esperado:
# {"jsonrpc":"2.0","id":"1","result":{"tools":[{"name":"start_development",...}]}}
```
