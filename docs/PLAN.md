# Plan: Regenerar `model_orm.go` con `omitempty` y validar respuestas JSON-RPC

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
> **Plan autónomo y autocontenido.** Todo el trabajo es dentro de `github.com/tinywasm/mcp`.
> **Prerrequisito externo:** `github.com/tinywasm/orm` debe estar publicado con su `ormc` corregido
> (este plan hace `go get orm@latest`, así que ese fix ya debe existir). Coordinación general del
> monorepo (no necesaria para ejecutar este plan): `tinywasm/docs/REGRESSION_FIX_MASTER_PLAN.md`.

## Restricciones del repo (inline — este repo no tiene `AGENTS.md`)

- Ecosistema `tinywasm`: en código agnóstico (compila wasm y backend) usar `tinywasm/fmt`/`tinywasm/json`,
  no stdlib pesado. Aquí el cambio es **regeneración** + tests; no se escribe lógica nueva.
- `model_orm.go` es **generado por `ormc`** — nunca editar a mano.
- Mantener verdes los tests con `gotest ./...` (o `gotest`).

## Contexto (causa raíz)

El IDE rechaza las respuestas del servidor MCP con un error de unión Zod:

```
expected "object" at "error", received null
Unrecognized keys: "result", "error"
```

Porque `EncodeFields` (generado por `ormc`) emite **`error:null` junto a `result`** en respuestas
exitosas. JSON-RPC exige exactamente **uno** de `result`/`error`. El fix del generador vive en
`tinywasm/orm` (Stage 1). Aquí solo **regeneramos** con el `ormc` corregido y **verificamos**.

Los modelos ya declaran la intención correcta en `model.go`:

```go
type JSONRPCResponseStruct struct {
	JSONRPC string
	ID      string
	Result  fmt.RawJSON `omitempty:"true"`
	Error   fmt.RawJSON `omitempty:"true"`
}
```

## Restricciones

- Ver `AGENTS.md` del repo (reglas permanentes del ecosistema).
- **No editar `model_orm.go` a mano**: es generado por `ormc`. Solo regenerar.
- No cambiar `model.go` salvo que falte algún `omitempty:"true"` donde la semántica JSON-RPC lo exige
  (ver Paso 3).

## Paso 1 — Bump de `orm` + regenerar

```bash
# (ejecutar en la raíz del repo mcp)
go get github.com/tinywasm/orm@latest      # trae el ormc corregido (omitempty)
go mod tidy
go install github.com/tinywasm/orm/cmd/ormc@latest   # instala el CLI ormc en $GOBIN
ormc                                        # regenera model_orm.go (corre en la raíz del repo)
```

Tras regenerar, `model_orm.go` debe mostrar las guardas de zero-value, p. ej.:

```go
func (m *JSONRPCResponseStruct) EncodeFields(w fmt.FieldWriter) {
	w.String("jsonrpc", m.JSONRPC)
	w.String("id", m.ID)
	if len(m.Result) != 0 { w.Raw("result", m.Result) }
	if len(m.Error)  != 0 { w.Raw("error",  m.Error)  }
}
```

## Paso 2 — Verificar las formas JSON-RPC

Las respuestas se construyen en `utils.go`:
- `newResultResponse` → `JSONRPCResponseStruct{JSONRPC, ID, Result}` (Error vacío) → **NO** debe emitir
  la clave `error`.
- `newErrorResponse` → `JSONRPCError{JSONRPC, ID, Error}` → emite solo `error` (correcto).

Confirmar que `JSONRPCResponseStruct` (model.go:116) tiene `omitempty:"true"` en **ambos** `Result` y
`Error` (ya lo tiene). Revisar que no haya **otros** structs de respuesta que emitan campos opcionales
como `null` y rompan el protocolo (p. ej. `listToolsResult.NextCursor`, `toolEntry.Description`,
`Result.IsError`): marcar con `omitempty:"true"` los que el cliente MCP trate como opcionales, y
regenerar.

## Paso 3 — Test de forma de respuesta (anti-regresión)

En `tests/protocol_test.go` agregar un test que serialice una respuesta de éxito y asegure la forma:

```go
func TestSuccessResponseHasNoErrorKey(t *testing.T) {
	// Construir una respuesta de éxito real (p. ej. handlePing / handleListTools)
	// y serializarla a bytes como hace el transporte.
	msg := createResponse("1", /* result Encodable */)
	var out []byte
	json.Encode(msg.(fmt.Encodable), &out)

	s := string(out)
	if strings.Contains(s, `"error"`) {
		t.Fatalf("success response must not contain \"error\" key: %s", s)
	}
	if !strings.Contains(s, `"result"`) {
		t.Fatalf("success response must contain \"result\": %s", s)
	}
}

func TestErrorResponseHasNoResultKey(t *testing.T) {
	msg := createErrorResponse("1", INVALID_REQUEST, "boom")
	var out []byte
	json.Encode(msg.(fmt.Encodable), &out)
	s := string(out)
	if strings.Contains(s, `"result"`) {
		t.Fatalf("error response must not contain \"result\" key: %s", s)
	}
	if strings.Contains(s, `"error":null`) {
		t.Fatalf("error must be an object, not null: %s", s)
	}
}
```

(Ajustar a los helpers reales y a cómo el transporte serializa `JSONRPCMessage`. Si `createResponse`
no es exportado/accesible desde `tests/`, ubicar el test en el paquete `mcp` o usar el handler público
`HandleMessage` con un mensaje `initialize`/`ping` y validar el byte slice resultante.)

## Paso 4 — `gotest`

```bash
# (en la raíz del repo mcp)
gotest   # verde, incluidos los nuevos tests de forma
```

Verificación funcional (end-to-end con el IDE) es **local del usuario** tras publicar: reiniciar el
daemon y confirmar que el cliente MCP ya no reporta el error de unión Zod. No forma parte de la
ejecución del agente.
