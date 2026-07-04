> Este plan se despacha vía el flujo CodeJob. Ver skill: agents-workflow.
> Orquestado por `tinywasm/docs/ROUTER_ADAPTER_MASTER_PLAN.md` — **Fase 1** (paralela a serverd).
> Depende de la **Fase 0** (`tinywasm/router` con cookies) ya publicada.

# Plan — `github.com/tinywasm/mcp`: montar el servidor como `router.APIModule`

> Hace que `*mcp.Server` publique su endpoint JSON-RPC **montándose** sobre un
> `router.Router`, en lugar de que cada consumidor reescriba a mano el adaptador
> `POST /mcp` con `net/http`. Convierte el boilerplate (leer body → `HandleMessage` →
> `json.Encode` → escribir) en una línea: `server.Mount(mcpServer)`.

---

## Reglas de Desarrollo

Las reglas del arnés viven en el `AGENTS.md`/`docs/` de esta librería — respétalas. Esta
librería compila también a WASM (cliente MCP): **sin stdlib de red** en la superficie.
El adaptador es transporte-agnóstico ya (`HandleMessage` recibe `[]byte` y devuelve un
`JSONRPCMessage`), así que montar sobre `router.Context` **no** añade `net/http`.

---

## Problema

`mjosefa-cms/web/server.go` (y cualquier host MCP) reescribe hoy este adaptador:

```go
func mcpHandler(s *mcp.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Body != nil { body, _ = io.ReadAll(r.Body) }
		resp := s.HandleMessage(context.Background(), body)
		w.Header().Set("Content-Type", "application/json")
		var out string
		switch m := resp.(type) {
		case *mcp.JSONRPCResponseStruct: json.Encode(m, &out)
		case *mcp.JSONRPCError:          json.Encode(m, &out)
		default: http.Error(w, "mcp: unknown response type", 500); return
		}
		w.Write([]byte(out))
	}
}
mux.HandleFunc("POST /mcp", mcpHandler(mcpServer))
```

Es `net/http` puro, copiado en cada host, y filtra el detalle del switch de tipos. El
`mcp.Server` ya sabe responder a un `[]byte`; solo le falta hablar el contrato `router`.

---

## Cambio objetivo — `mcp.Server` implementa `router.APIModule`

`router.APIModule` requiere `ModelName() string` (vía `fmt.ModuleNaming`) + `MountAPI(router.Router)`.

Nuevo archivo `mount.go` (`//go:build !wasm` si conviene mantener el cliente WASM libre
de la dependencia `router`; ver nota al final):

```go
package mcp

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/json"
	"github.com/tinywasm/router"
)

// MCPPath es la ruta canónica del endpoint JSON-RPC. Sin literales sueltos.
const MCPPath = "/mcp"

// ModelName identifica al módulo (contrato de identidad reutilizado por APIModule).
func (s *Server) ModelName() string { return s.name }

// MountAPI publica el endpoint MCP sobre el Router del host. Reemplaza el adaptador
// net/http que hoy reescribe cada consumidor.
func (s *Server) MountAPI(r router.Router) {
	r.Post(MCPPath, func(ctx router.Context) {
		resp := s.HandleMessage(context.Background(), ctx.Body())
		ctx.SetHeader(headerContentType, mimeJSON)
		var out string
		var err error
		switch m := resp.(type) {
		case *JSONRPCResponseStruct:
			err = json.Encode(m, &out)
		case *JSONRPCError:
			err = json.Encode(m, &out)
		default:
			ctx.WriteStatus(500)
			ctx.Write([]byte(`{"error":"mcp: unknown response type"}`))
			return
		}
		if err != nil {
			ctx.WriteStatus(500)
			ctx.Write([]byte(`{"error":"mcp: encode failed"}`))
			return
		}
		ctx.Write([]byte(out))
	})
}
```

Con esto, `var _ router.APIModule = (*Server)(nil)` compila, y el host escribe
`serverd.New(...).Mount(mcpServer).ListenAndServe()`.

**Constantes** a declarar (o reutilizar si ya existen en el paquete):
`const headerContentType = "Content-Type"`, `const mimeJSON = "application/json"`.

**Nota sobre `s.name`:** hoy es un campo privado; `ModelName()` lo expone como identidad.
Confirmar que `Config.Name` lo alimenta (así es en `NewServer`).

---

## Pasos de implementación

1. `go.mod`: añadir dependencia `github.com/tinywasm/router` (Fase 0).
2. Crear `mount.go` con `MCPPath`, `ModelName()`, `MountAPI(router.Router)` y las
   constantes de cabecera/mime (reutilizar las que ya existan en `constants.go`).
3. Fijar en un test: `var _ router.APIModule = (*Server)(nil)`.
4. Actualizar `docs/ARCHITECTURE.md` y `README.md`: el endpoint MCP se publica montándose
   sobre `router.Router`; documentar `MCPPath` y `MountAPI`.

---

## Code Quality Checklist (obligatorio)

- **Sin literales repetidos.** `MCPPath`, `headerContentType`, `mimeJSON` son constantes.
  Reutiliza las de `constants.go` si ya existen; no dupliques.
- **Tipado sobre `any`.** El `switch` sobre `*JSONRPCResponseStruct`/`*JSONRPCError` se
  mantiene tipado; nada de `interface{}` en datos.
- **Superficie mínima.** Exporta solo `MCPPath`, `ModelName`, `MountAPI`. El switch de
  codificación queda encapsulado dentro de `MountAPI`.
- **Sin `net/http`.** El adaptador opera solo sobre `router.Context`.

---

## Estrategia de pruebas y criterios de aceptación

- `gotest`. Doble objetivo si `mount.go` no lleva `//go:build !wasm`; ver nota.
- **Contrato fijado:** `var _ router.APIModule = (*Server)(nil)`.
- **Test funcional con un `router.Router` de mentira** (registra la ruta en un mapa y
  ejecuta el handler con un `router.Context` de mentira cuyo `Body()` devuelve un
  `initialize` JSON-RPC): verifica que la respuesta se escribe con
  `Content-Type: application/json` y contiene el `result` esperado.

---

## Nota de dependencia (decisión de build tag)

`router` es prácticamente cero-dep e isomórfico, así que importarlo no contamina el
cliente WASM con `net/http`. **Preferencia:** `mount.go` **sin** build tag, para que el
contrato `APIModule` valga en ambos objetivos. Si al medir el binario WASM el símbolo
estorba, marcar `mount.go` con `//go:build !wasm` (el montaje solo lo usa el servidor).
Elegir sin tag salvo que la métrica de tamaño lo desaconseje.

---

## Tabla de etapas

| Etapa | Archivo | Acción | Rompe API |
|---|---|---|---|
| 1 | `go.mod` | Añadir dep `router` (Fase 0) | No |
| 2 | `mount.go` | `MCPPath` + `ModelName` + `MountAPI` + constantes | No (adición) |
| 3 | `mount_test.go` | `var _ router.APIModule` + test con Router de mentira | No |
| 4 | `README.md`, `docs/ARCHITECTURE.md` | Documentar montaje MCP vía router | No |
