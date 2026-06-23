# PLAN — Corregir doble-serialización y clave `isError` en `tinywasm/mcp`

> **Repo:** `github.com/tinywasm/mcp`
> **Archivos:** `model.go`, luego `ormc` regenera `model_orm.go`
> **Tipo:** bug fix + regeneración de código
> **Prerequisito:** `tinywasm/fmt`, `tinywasm/json` y `tinywasm/orm` publicados con
> soporte de `Raw()` y fix de `FieldRaw` en generate.go

## Contexto

Los tests de `tinywasm/mcp` fallan porque:

1. **Doble-serialización** — todos los campos `fmt.RawJSON` (Arguments, Content,
   Result, InputSchema, Tools, Capabilities) son emitidos como strings entrecomillados
   por `model_orm.go` porque el `ormc` anterior usaba `w.String()` para `FieldRaw`.
   Con el `ormc` corregido, regenerar `model_orm.go` produce `w.Raw()` / `r.Raw()`.

2. **Clave `is_error` vs `isError`** — `ormc` convierte `IsError` a snake_case
   (`is_error`), pero el protocolo MCP exige `isError` (camelCase). Requiere tag
   `json:"isError"` en `model.go`.

## Cambios en `model.go`

### Agregar tag `json:"isError"` a `Result.IsError`

```go
// ANTES:
type Result struct {
    IsError bool
    Content fmt.RawJSON
}

// DESPUÉS:
type Result struct {
    IsError bool        `json:"isError"`
    Content fmt.RawJSON
}
```

## Actualizar dependencias y regenerar

```bash
go get github.com/tinywasm/fmt@latest
go get github.com/tinywasm/json@latest
go get github.com/tinywasm/orm@latest
go mod tidy

# Regenerar model_orm.go con el ormc corregido:
ormc
```

## Verificación

```bash
go vet ./...
gotest   # todos los tests deben pasar
```

Tests clave que deben quedar verdes:
- `TestToolCall_ContentIsArray` — content como array JSON inline
- `TestToolCall_ContentItemLowercaseKeys` — claves `type`/`text` en minúsculas
- `TestListTools_InputSchemaIsObject` — inputSchema como objeto inline
- `TestToolCall_ArgumentsAsObject` — arguments como objeto, no string
- `TestHandleToolCall_ExecuteReturnsError_IsErrorTrue` — clave `isError`
- `TestInitialize_ValidVersion_ReturnsServerInfo` — decode correcto de params
- `TestInitialize_GeneratesSessionID` — sesión generada
- `TestInitialize_OlderSupportedVersion_Accepted` — versión 2024-11-05
- `TestInitialize_NewerUnknownVersion_DowngradesGracefully` — downgrade a 2025-11-25

## Checklist

- [ ] `json:"isError"` agregado a `Result.IsError` en `model.go`
- [ ] `go get` a versiones nuevas de `fmt`, `json`, `orm`
- [ ] `ormc` ejecutado → `model_orm.go` regenerado
- [ ] `go vet ./...` sin errores
- [ ] `gotest` verde (todos los tests listados arriba pasan)
