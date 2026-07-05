# Plan — `mcp`: threadear la identidad en `MountAPI` y dejar el token a `user`

> Autocontenido, en español. Rige el **arnés de construcción** (reglas en el
> `AGENTS.md` de la raíz de esta librería): el mal uso no debe compilar y ningún
> fallo debe ser silencioso en runtime.
>
> Depende de [`router/docs/PLAN.md`](../../router/docs/PLAN.md) (identidad tipada
> `Context.UserID()`). Coordinado por
> [`ROUTER_ADAPTER_MASTER_PLAN.md`](../../docs/ROUTER_ADAPTER_MASTER_PLAN.md);
> consumidor de referencia: `veltylabs/mjosefa-cms`.

---

## El problema (dos huecos de arnés)

### 1. `MountAPI` descarta la identidad → RBAC por-tool silenciosamente inerte

`mount.go` monta `/mcp` así (resumido):

```go
r.Post(MCPPath, func(ctx router.Context) {
	resp := s.HandleMessage(context.Background(), ctx.Body()) // ← descarta ctx
	...
})
```

`HandleMessage` recibe `context.Background()`: **se tira el `router.Context`** con la
cabecera `Authorization` / la identidad. Aguas abajo,
`handleToolCall` hace `userID := ctx.Value(CtxKeyUserID)` → **siempre vacío**, y
`auth.Can(userID, tool.Resource, tool.Action)` con `OpenAuthorizer` devuelve `true`.
Resultado: **el RBAC por-tool no protege nada** y falla en silencio (el peor modo según
el arnés). El `ARCHITECTURE.md` de esta librería describe un flujo
(`extract token from ctx → Authorize → Can`) que `MountAPI` **no alimenta**.

### 2. `Authorizer.Authorize(token)` mezcla responsabilidades

```go
type Authorizer interface {
	Authorize(token string) (userID string, err error) // ← identidad: NO es de mcp
	Can(userID, resource string, action byte) bool     //   RBAC: sí es de mcp
}
```

Validar un token (JWT/sesión → `userID`) es responsabilidad de `tinywasm/user`, no de
`mcp`. Que `mcp` la incluya duplica identidad en dos sitios (viola SRP y "una sola
forma de hacer cada cosa").

---

## La corrección

**`mcp` deja de conocer tokens. Solo aplica RBAC por-tool sobre una identidad ya
resuelta que llega por el `Context`.**

### a) `MountAPI` propaga la identidad del `router.Context`

```go
func (s *Server) MountAPI(r router.Router) {
	r.Post(MCPPath, func(ctx router.Context) {
		// La identidad la puso un middleware de auth aguas arriba (user.Authenticate).
		reqCtx := context.Background()
		reqCtx.Set(CtxKeyUserID, ctx.UserID()) // ← tipado, sin clave mágica
		resp := s.HandleMessage(reqCtx, ctx.Body())
		...
	})
}
```

### b) El `Authorizer` se reduce a la decisión de permiso

```go
// Antes: interface con Authorize(token) + Can(...byte).
// Después: mcp consume el tipo del contrato, sin interfaz propia ni parseo de token.
//
// package router
//   type Authorize func(userID, resource, action string) bool

type Config struct {
	Name      string
	Version   string
	Authorize router.Authorize // requerido (fail-fast en NewServer si es nil)
	SSE       SSEPublisher
}
```

`handleToolCall` pasa a:

```go
userID := ctx.Value(CtxKeyUserID)
if !s.authorize(userID, tool.Resource, string(tool.Action)) { // byte→string en el borde
	return forbidden
}
```

- **`mcp.Tool.Action` sigue siendo `byte`** internamente (fuente: el Plan Maestro —
  el `byte` de tool/crudp es otro carril, no RBAC de ruta). La conversión a `string`
  ocurre **una sola vez**, en el borde donde se consulta `router.Authorize` (cuya
  `action` es `string`, la fuente de verdad `user.Permission.Action string`).
- `OpenAuthorizer()` se reemplaza por un `router.Authorize` trivial
  (`func(_, _, _ string) bool { return true }`) para dev/tests.
- `NewServer` sigue rechazando `Authorize == nil` (fail-fast en compilación de la
  configuración, no en runtime).

---

## Cambios

| Archivo | Cambio |
|---|---|
| `mount.go` | `MountAPI` inyecta `ctx.UserID()` en `CtxKeyUserID` antes de `HandleMessage`. |
| `mcp_auth.go` | Eliminar `Authorizer.Authorize`; `mcp` consume `router.Authorize`. `OpenAuthorizer` → helper `AllowAll router.Authorize`. |
| `server.go` | `Config.Auth Authorizer` → `Config.Authorize router.Authorize`; `handleToolCall` usa `string(tool.Action)`. |
| `tools.go` | `Tool` gana `Public bool` (ausencia = privado). |
| `server.go` | `handleToolCall`: si `!tool.Public && userID==""` ⇒ `Forbidden` antes de `Authorize`. |
| `request_handler.go` | Quitar el paso "extract token → Authorize"; la identidad ya viene resuelta en el ctx. |
| `docs/ARCHITECTURE.md` | Actualizar el flujo de auth: identidad la resuelve el host (middleware), `mcp` solo hace `Can`. |

---

## Estrategia de pruebas y criterios de aceptación

- **RBAC por-tool real:** un `router.Context` de mentira con `UserID()=="u1"` y un
  `Authorize` que niega `("u1","x","c")` → `tools/call` de una tool `{Resource:"x",
  Action:'c'}` responde `-32001 Forbidden`. Con `UserID()==""` (anónimo) también niega.
  Hoy este test **pasaría incorrectamente** (allow) — es la regresión que cierra.
- **Sin parseo de token en `mcp`:** `grep` de `Authorize(token` y de `jwt`/`Bearer` en
  la superficie de `mcp` no devuelve nada.
- `NewServer(Config{Authorize: nil})` devuelve error (fail-fast).
- Compila `!wasm` y el core del protocolo sigue WASM-safe.

---

## Endurecimiento de seguridad (cerrado por defecto) — cada punto con test

Responsabilidad de `mcp` en el principio *cerrado por defecto*:

- **Portón abierto es opt-in explícito, nunca el default.** `mcp` no expone un
  `OpenAuthorizer` ambiguo: `AllowAll` es un `router.Authorize` con nombre inequívoco de
  dev; no existe "si `Authorize` es nil, permitir" — `NewServer` con `Authorize == nil`
  **falla**.
  **Test:** `NewServer(Config{Authorize: nil})` → error. Con `AllowAll`, la llamada pasa
  (opt-in explícito).
- **Marcador de público tipado en la tool (ausencia = privado).** Una tool no es
  accesible al anónimo salvo que **declare** `Public: true`. `handleToolCall`: si la tool
  no es pública y `userID == ""`, deniega antes de consultar `Authorize`. Hacer pública
  una tool es un acto que se lee y se `grep`-ea; el default (no declarar) es cerrado.

  ```go
  type Tool struct {
      // ...Name, Description, InputSchema, Resource, Action, Execute...
      Public bool // explícito: accesible sin identidad. Ausencia = privado.
  }
  ```
  **Test:** tool con `Public:false` + `UserID()==""` → `Forbidden`; la misma tool con
  `Public:true` + anónimo → ejecuta. Una tool sin el campo (zero-value) es privada.

---

## Relación con el ecosistema

- **Provee identidad:** [`user`](../../user/docs/PLAN_MCP_AUTHZ.md) expone
  `Authenticate() router.Middleware` (deja `ctx.SetUserID`) y `Can` (`router.Authorize`).
- **Contrato:** [`router`](../../router/docs/PLAN.md) aporta `Context.UserID()` y el tipo
  `Authorize`.
- **Host:** el consumidor de referencia `veltylabs/mjosefa-cms` inyecta `user.Can`
  como `Config.Authorize` y monta el middleware `user.Authenticate`.
