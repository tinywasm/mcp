# PLAN: Usar `fmt.RawJSON` en `rpcResponse.Result`

## Problema

`mcp/model.go` define `rpcResponse.Result` como `string` (`FieldText`).
Cuando el daemon responde con el resultado como JSON crudo (array `[{...}]`),
`tinywasm/json` intenta decodificarlo como un string JSON `"..."` — falla silenciosamente
o requiere que el daemon haga doble encoding — lo que provoca que el footer TUI quede vacío.

```go
// mcp/model.go — estado actual (incorrecto)
type rpcResponse struct {
    JSONRPC string
    ID      string
    Result  string   // FieldText: espera "..." → no sirve para JSON crudo
    Error   string
}
```

## Fix

Cambiar `Result string` → `Result fmt.RawJSON`. `ormc` detecta `fmt.RawJSON`
automáticamente y genera `FieldRaw` en `Schema()`, haciendo que `tinywasm/json`
lea el valor JSON crudo verbatim como string, sin necesitar doble encoding.

---

## Archivos afectados

| Archivo | Acción |
|---|---|
| `mcp/model.go` | Cambiar `Result string` → `Result fmt.RawJSON` |
| `mcp/model_orm.go` | **Regenerar** con `ormc` |
| `mcp/client.go` | `envelope.Result == ""` → `len(envelope.Result) == 0` |

---

## Paso 1 — Instalar `ormc`

```bash
go install github.com/tinywasm/orm/cmd/ormc@latest
```

## Paso 2 — Editar `mcp/model.go`

```go
// mcp/model.go — después
// ormc:formonly
type rpcResponse struct {
    JSONRPC string
    ID      string
    Result  fmt.RawJSON  // ← FieldRaw: lee JSON crudo [{}] directo como string
    Error   string
}
```

> `fmt.RawJSON` es un alias de `string`. El campo en memoria es idéntico.
> Solo cambia cómo `ormc` lo categoriza en `Schema()` y cómo `tinywasm/json` lo procesa.

## Paso 3 — Regenerar `mcp/model_orm.go`

```bash
cd tinywasm/mcp
ormc
```

El campo `result` en `_schemarpcResponse` cambia de `fmt.FieldText` a `fmt.FieldRaw`:

```go
// mcp/model_orm.go — después de ormc
var _schemarpcResponse = []fmt.Field{
    {Name: "jsonrpc", Type: fmt.FieldText, Widget: input.Text()},
    {Name: "id",      Type: fmt.FieldText, Widget: input.Text()},
    {Name: "result",  Type: fmt.FieldRaw},  // ← cambia de FieldText
    {Name: "error",   Type: fmt.FieldText, Widget: input.Text()},
}
```

## Paso 4 — Actualizar `mcp/client.go`

`fmt.RawJSON = string`, por lo que el check de vacío cambia:

```go
// mcp/client.go — antes
if envelope.Result == "" {
    callback(nil, nil)
    return
}
callback([]byte(envelope.Result), nil)

// mcp/client.go — después
if len(envelope.Result) == 0 {
    callback(nil, nil)
    return
}
callback([]byte(envelope.Result), nil)
```

`[]byte(envelope.Result)` funciona igual porque `RawJSON = string`.

---

## Pasos de ejecución (orden estricto)

1. Editar `mcp/model.go`: `Result string` → `Result fmt.RawJSON`
2. `cd tinywasm/mcp && ormc` → regenera `mcp/model_orm.go`
3. Editar `mcp/client.go`: check de vacío (ver Paso 4)
4. `go build ./...`
5. `go test ./...`

---

## Cobertura de tests requerida

Agregar en `mcp/tests/protocol_test.go`:

```go
func TestRpcResponse_DecodeRawResult(t *testing.T) {
    // Respuesta con result como array JSON directo (no string JSON)
    raw := `{"jsonrpc":"2.0","id":"1","result":[{"tab_title":"BUILD","handler_type":1}]}`

    var resp rpcResponse
    if err := twjson.Decode([]byte(raw), &resp); err != nil {
        t.Fatalf("Decode failed: %v", err)
    }

    want := `[{"tab_title":"BUILD","handler_type":1}]`
    if string(resp.Result) != want {
        t.Errorf("Result\n got:  %s\n want: %s", resp.Result, want)
    }
}
```

> El test verifica que `FieldRaw` lee el array JSON crudo sin doble encoding.
> Falla con `FieldText` (estado actual) → confirma el bug antes del fix.
