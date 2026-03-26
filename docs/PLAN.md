# MCP: Eliminación total de encoding/json → tinywasm/json

## Contexto

`tinywasm/mcp` debe ser portable al navegador (sin HTTP, consultando herramientas localmente). Por eso **no puede depender de `encoding/json`** en ningún archivo fuente — ni test ni producción. Toda serialización/deserialización debe ir por `tinywasm/json`, que opera sobre `fmt.Fielder`.

La migración del cliente (`client.go`) ya está completa. El resto del paquete aún usa `encoding/json`.

### Regla absoluta

> Ningún archivo `.go` en `tinywasm/mcp` puede importar `encoding/json`.
> Usar `tinywasm/json` con structs generados por `ormc`.

---

## Estado actual

| Archivo | encoding/json | stdlib pendiente | Estado |
|---|---|---|---|
| `client.go` | ✗ | `context`, `strings` | ❌ Incompleto |
| `types.go` | ✓ | — | ❌ Pendiente |
| `tools.go` | ✓ | — | ❌ Pendiente |
| `utils.go` | ✓ | — | ❌ Pendiente |
| `handler.go` | ✓ | — | ❌ Pendiente |
| `handler_ide.go` | ✓ | — | ❌ Pendiente |
| `request_handler.go` | ✓ | — | ❌ Pendiente |
| `server_http.go` | ✓ | — | ❌ Pendiente |
| `transport_streamable_http.go` | ✓ | — | ❌ Pendiente |
| `server.go` | ✓ | — | ❌ Pendiente |
| `tests/model.go` | — | ✅ Creado |
| `tests/model_orm.go` | — | ✅ Generado |
| `tests/handler_test.go` | ✓ (permitido) | ⏳ Actualizar calls |

---

## Dependencia bloqueante

> **tinywasm/json v0.4.0 ✅**
> `parser.go` ya no usa `map[string]any` ni `[]any`. Compilable con TinyGo.
> Ver: `tinywasm/json/docs/PLAN.md`

---

## Stage 0 — Completar migración de client.go ❌

`client.go` aún usa dos paquetes stdlib que deben reemplazarse:

### 0.1 — Reemplazar `"strings"` con `tinywasm/fmt`

```go
// Antes (NewClient)
endpoint: strings.TrimSuffix(baseURL, "/") + "/mcp",

// Después
endpoint: fmt.Convert(baseURL).TrimSuffix("/").String() + "/mcp",
```

Eliminar import `"strings"`.

### 0.2 — Reemplazar `"context"` con `tinywasm/context`

```go
// Antes
import "context"
func (c *Client) Call(ctx context.Context, ...) {
func (c *Client) Dispatch(ctx context.Context, ...) {

// Después
import "github.com/tinywasm/context"
func (c *Client) Call(ctx *context.Context, ...) {
func (c *Client) Dispatch(ctx *context.Context, ...) {
```

Eliminar import `"context"`.

---

## Stage 1 — Actualizar test calls ⏳

Los tests aún usan `map[string]string` / `map[string]any` como params. Los structs ya existen en `tests/model.go` y sus métodos en `tests/model_orm.go`.

### 1.1 — TestHandler_OnAction_JSONRPCMethod (~línea 83)

```go
// Antes
client.Dispatch(ctx, "tinywasm/action", map[string]string{"key": "foo", "value": "bar"})

// Después
client.Dispatch(ctx, "tinywasm/action", &actionParams{Key: "foo", Value: "bar"})
```

### 1.2 — TestHandler_ToolOutput_TextResult (~línea 166)

```go
// Antes
client.Call(ctx, "tools/call", map[string]any{"name": "echo_tool"}, func(...) {

// Después
client.Call(ctx, "tools/call", &toolCallParams{Name: "echo_tool"}, func(...) {
```

### 1.3 — TestHandler_ToolRBAC_Denied (~línea 205)

```go
// Antes
client.Call(ctx, "tools/call", map[string]any{"name": "secure_tool"}, func(...) {

// Después
client.Call(ctx, "tools/call", &toolCallParams{Name: "secure_tool"}, func(...) {
```

---

## Stage 2 — Modelos para protocol types

Los tipos del protocolo JSON-RPC que hoy usan `map[string]any` necesitan structs con `ormc`.

### 2.1 — Crear `model.go` en el paquete raíz

```go
package mcp

// ormc:formonly
type metaFields struct {
    ProgressToken string `json:"progressToken,omitempty"`
}

// ormc:formonly
type notificationParamsMeta struct {
    Meta metaFields `json:"_meta,omitempty"`
}

// ormc:formonly
type initializeParams struct {
    ProtocolVersion string         `json:"protocolVersion"`
    ClientInfo      implementationInfo `json:"clientInfo"`
}

// ormc:formonly
type implementationInfo struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

// ormc:formonly
type toolCallArguments struct {
    Name      string `json:"name"`
    Arguments string `json:"arguments,omitempty"`
}

// ormc:formonly
type toolResultContent struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
}
```

### 2.2 — Ejecutar ormc

```bash
cd . && ormc
```

Genera `model_orm.go` con `Schema()` y `Pointers()` para todos los structs.

---

## Stage 3 — Migrar types.go

**Problema actual**: `Meta` y `NotificationParams` usan `MarshalJSON`/`UnmarshalJSON` con `map[string]any`.

**Estrategia**: Reemplazar los custom marshalers con los structs de Stage 2.

- `Meta.MarshalJSON` → `tinywasm/json.Encode(&metaFields{...})`
- `Meta.UnmarshalJSON` → `tinywasm/json.Decode(data, &metaFields{})`
- `NotificationParams.MarshalJSON` → `tinywasm/json.Encode(&notificationParamsMeta{...})`
- `NotificationParams.UnmarshalJSON` → `tinywasm/json.Decode(data, &notificationParamsMeta{})`

---

## Stage 4 — Migrar tools.go

**Problema actual**: `CallToolRequest.BindArguments` hace `json.Unmarshal` a un `any` target. `CallToolResult` tiene custom marshalers.

**Estrategia**:
- `BindArguments(target fmt.Fielder)` — cambiar firma para aceptar `fmt.Fielder` en lugar de `any`
- `CallToolResult` → usar `toolResultContent` struct para `MarshalJSON`/`UnmarshalJSON`
- `NewToolResultJSON` → aceptar `fmt.Fielder` en lugar de genérico `T`

---

## Stage 5 — Migrar request_handler.go

**Problema actual**: `json.Unmarshal` a structs del protocolo (`InitializeRequest`, `PingRequest`, `ListToolsRequest`, etc.).

**Estrategia**:
- Todos los tipos de request/response del protocolo deben implementar `fmt.Fielder`
- Generarlos con `ormc` desde `model.go`
- Reemplazar `json.Unmarshal(msg, &req)` con `tinywasm/json.Decode(msg, &req)`

---

## Stage 6 — Migrar server_http.go y transport_streamable_http.go

**Problema actual**: `json.NewDecoder(r.Body).Decode(...)` y `json.NewEncoder(w).Encode(...)`.

**Estrategia**:
- `Decode`: `tinywasm/json.Decode(r.Body, target)` — soporta `io.Reader`
- `Encode` de error response: crear `errorResponse` struct con `ormc:formonly`
- `Encode` de response: el objeto de response debe implementar `fmt.Fielder`

---

## Stage 7 — Migrar utils.go y handler.go / handler_ide.go

- `utils.go NewToolResultJSON`: refactorizar para recibir `fmt.Fielder`
- `utils.go ParseCallToolResult`: usar `tinywasm/json.Decode`
- `handler.go` / `handler_ide.go`: identificar usos residuales y migrar

---

## Stage 8 — Verificar

```bash
gotest
```

Ningún archivo debe importar `encoding/json`. Todos los tests deben pasar.

---

## Acciones prohibidas

- NO importar `encoding/json` en ningún archivo del paquete
- NO importar `"context"` stdlib — usar `github.com/tinywasm/context`
- NO importar `"strings"` stdlib — usar `tinywasm/fmt`
- NO usar `map[string]any` ni `map[string]string` como destino de deserialización
- NO revertir los cambios de `client.go`
- NO modificar `fmt.Fielder` ni `tinywasm/json` para acomodar comportamientos de `encoding/json`
