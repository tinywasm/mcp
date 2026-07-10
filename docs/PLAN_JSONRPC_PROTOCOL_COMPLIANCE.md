# PLAN — mcp: cumplimiento JSON-RPC 2.0 (eco del `id`) y negociación de versión de protocolo MCP

> This plan is dispatched via the CodeJob workflow. See skill: agents-workflow.
> Parte de `tinywasm/docs/MCP_DAEMON_HARDENING_MASTER_PLAN.md`.
> Independiente de `PLAN_KIND_UNIFICATION_INPUTSCHEMA.md` (otra ola — no mezclar).
> Idioma: español (decisión del mantenedor). Autocontenido: el agente no tiene contexto previo.

## Prerequisito (correr primero)

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
```

Todos los tests con `gotest` (nunca `go test` a secas).

---

## 0. Diagnóstico (evidencia real contra un server en producción, 2026-07-09)

### Bug 1 — `initialize` rechaza versiones de protocolo que debería negociar

```
→ {"method":"initialize","params":{"protocolVersion":"2025-06-18",...}}
← {"error":{"code":-32602,"message":"mcp unsupported protocol version: 2025-06-18"}}

→ protocolVersion "2025-03-26"  ← mismo error
→ protocolVersion "2024-11-05"  ← OK, responde "2024-11-05"
→ protocolVersion "2025-11-25"  ← OK, responde "2025-11-25"
```

La spec MCP (sección *Version Negotiation* del lifecycle) dice: si el server no
soporta la versión pedida, **DEBE responder con otra versión que sí soporte**
(típicamente la más reciente) — el cliente decide entonces si continúa o
desconecta. Devolver error `-32602` rompe el handshake con cualquier cliente
que pida una versión intermedia (p. ej. `2025-06-18`, usada por clientes MCP
actuales). Consecuencia real: "server inalcanzable" para esos clientes.

### Bug 2 — el `id` de la request pierde su tipo JSON en la respuesta

```
→ {"jsonrpc":"2.0","id":3,"method":"tools/call",...}
← {"jsonrpc":"2.0","id":"3","result":{...}}
```

JSON-RPC 2.0: *"The Server MUST reply with the same value in the Response
object... This member is used to correlate the context"* — mismo **valor y
tipo**. `id:3` debe volver como `3`, no `"3"`. Clientes estrictos (y varios SDK
MCP) descartan la respuesta o fallan la correlación.

Nota: los errores tempranos (p. ej. el de versión del Bug 1) también responden
con id string (`"id":"1"`), así que el fix debe cubrir todas las rutas de
respuesta, no solo la feliz.

---

## 1. Reglas de código (obligatorias)

- Este paquete puede compilar a WASM en consumidores: **no stdlib nueva** —
  usar `tinywasm/fmt` / `tinywasm/json` como ya hace el paquete. Verificar con
  los imports existentes antes de agregar cualquiera.
- Cero strings de JSON schema a mano; el inputSchema se deriva de
  `Args.Schema()` (no tocar ese mecanismo).
- Strings repetidos → constantes tipadas (las versiones soportadas van en un
  slice/constantes exportadas, no literales sueltos).
- Errores se propagan; nada se traga en silencio.
- No tocar: `Tool`, `Request.Bind`, RBAC/`Authorize`, ni nada de
  `PLAN_KIND_UNIFICATION_INPUTSCHEMA.md`.

## 2. Etapa 1 — negociación de versión conforme a spec

Localizar el manejo de `initialize` (grep `unsupported protocol version`).

1. Definir las versiones soportadas como constantes exportadas + slice
   ordenado, p. ej.:

```go
const (
    ProtocolVersion20241105 = "2024-11-05"
    ProtocolVersion20251125 = "2025-11-25"
)
// LatestProtocolVersion es la que se ofrece cuando el cliente pide una no soportada.
const LatestProtocolVersion = ProtocolVersion20251125
```

2. Comportamiento nuevo de `initialize`:
   - versión pedida soportada → responder esa misma (como hoy).
   - versión pedida NO soportada → responder **resultado normal** con
     `protocolVersion: LatestProtocolVersion` (no error). El cliente decide.
   - `protocolVersion` ausente o vacío → tratar como no soportada (responder
     `LatestProtocolVersion`).
3. Evaluar (decisión del implementador, documentarla en el código): soportar
   directamente `2025-03-26` y `2025-06-18` si el server ya cumple sus
   requisitos de transporte; si se agregan, van al slice de soportadas.

**Tests**: initialize con cada caso de arriba; ninguno devuelve `error`; el
`protocolVersion` de la respuesta es siempre uno del slice soportado.

## 3. Etapa 2 — eco del `id` preservando el tipo JSON

Hoy el `id` se extrae como string y se re-serializa entre comillas. Fix:
conservar los **bytes crudos** del `id` tal como llegaron (número, string o
null) y escribirlos verbatim en la respuesta.

- Localizar dónde se parsea/almacena el id de la request (grep de cómo se
  construye la respuesta; el paquete ya usa `ExtractJSONValue` que devuelve
  los bytes crudos — el bug es que algo los envuelve en comillas o los pasa
  por string). Guardar `[]byte` y emitirlos sin transformar.
- Cubrir TODAS las rutas: resultado, error de protocolo (versión), error de
  tool, `-32602 tool not found`, respuestas de `notifications` (id ausente →
  no responder, según spec).

**Tests**:
- `id:3` → respuesta contiene `"id":3`.
- `id:"abc"` → `"id":"abc"`.
- `id` numérico en request con error (tool inexistente) → `"id":<n>` numérico.

## 4. Etapa 3 — documentación

- `docs/ARCHITECTURE.md`: sección de lifecycle/handshake actualizada con la
  tabla de versiones soportadas y la regla de negociación.
- `docs/SKILL.md`: si documenta el handshake o ejemplos de requests, alinear.

## 5. Criterios de aceptación

1. `gotest ./...` verde.
2. `initialize` jamás responde error por versión: siempre un resultado con una
   versión soportada.
3. Round-trip de `id` preserva tipo en todas las rutas de respuesta.
4. Cero regresiones en derivación de inputSchema (tests existentes verdes).

## 6. Tabla de etapas

| # | Etapa | Archivos | Gate |
|---|-------|----------|------|
| 1 | Negociación de versión | handler de `initialize` + constantes | tests §2 verdes |
| 2 | Eco de `id` con tipo | serialización de respuestas | tests §3 verdes |
| 3 | Docs | `docs/ARCHITECTURE.md`, `docs/SKILL.md` | 1–2 |
