---
PLAN: "feat: harvest router.OpModule.MountOps into MCP tools; Caller decodes typed"
TAG: v0.2.0
---

> Este plan se despacha vía el flujo CodeJob. Ver skill: **agents-workflow**.
> Orquestado por `tinywasm/app-releases/docs/REUSABLE_MODULES_MASTER_PLAN.md` — **Fase B1**.

# PLAN — `mcp`: cosechar `OpModule.MountOps` como tools + `Caller` tipado

Autocontenido, en español. Eres un agente **sin contexto previo** y **solo tienes este repo**
(`tinywasm/mcp`). Todo el contrato y el código exacto van inline.

## 1. Qué cambia y por qué

Hoy un módulo de dominio que quiere exponerse por MCP implementa `mcp.ToolProvider{ Tools() []Tool
}` — **importando `tinywasm/mcp` directamente**. Es justo el acoplamiento que la ola de módulos
reutilizables elimina: un módulo de dominio declara sus operaciones contra el contrato **neutral**
`router.OpModule { ModelName(); MountOps(r router.OpRegistry) }`, llamando
`r.Op(name, h).Requires(res, act).Accepts(args)` — **sin saber cuál es el transporte final**. Este
plan hace que `mcp` **coseche** esas llamadas `Op` y las convierta en `Tool` internamente, y arregla
la rotura ya en curso: `router.Caller.Call` cambió de firma (decodifica en un destino tipado) y
`mcp.NewCaller` todavía implementa la vieja.

> **Nota de diseño (importante, evita el olor que este plan corrige).** `Op` **NO** es un método de
> `router.Router` (la interfaz HTTP con `Get/Post/…`). Vive en su propia interfaz mínima
> `router.OpRegistry { Op(name string, h HandlerFunc) Route }` — el espejo de `router.Caller`. Por eso
> el cosechador de este plan implementa `OpRegistry` (**un** método) y **no** finge ser un `Router`
> con `Get/Post` que revientan. Si te ves escribiendo `panic("nunca por aquí")` en `Get/Post/…`, estás
> implementando la interfaz equivocada: usa `OpRegistry`, no `Router`.

**Dos cambios independientes, en el mismo plan:**

1. **Nuevo:** `mcp.HarvestOps(modules ...router.OpModule) ToolProvider` — un `router.OpRegistry`
   interno (un método) usado para "ejecutar" el `MountOps` de cada módulo y recoger lo que registra
   como `[]Tool`.
2. **Migración obligatoria:** `mcpCaller.Call` pasa de `func(result []byte, err error)` a `into
   model.Decodable, done func(err error)` — **`router@v0.1.14` ya lo exige para compilar**.

## 2. Estado actual exacto (verificado, no supuesto)

- `mcp.Tool` (`tools.go:17-25`) **ya** tiene `Args model.Fielder` — mismo tipo que
  `router.Route.Accepts(args model.Fielder)`. Encaja sin fricción.
  ```go
  type Tool struct {
  	Name        string
  	Description string
  	Args        model.Fielder
  	Resource    model.Resource
  	Action      model.Action
  	Access      model.Access
  	Execute     func(ctx *context.Context, req Request) (*Result, error)
  }
  ```
- `mcp.ToolProvider` (`tools.go:57-59`): `type ToolProvider interface { Tools() []Tool }`.
- `mcp.NewServer(config Config, providers []ToolProvider)` (`server.go:32`) itera `providers`, hace
  `p.Tools()`, registra cada `Tool` en `s.tools map[string]Tool`. **No cambia.**
- `handleToolCall` (`server.go:150-185`) hace el chequeo RBAC (`Resource`/`Action`/`Access` del
  `Tool`), arma `req := Request{Params: params, Action: tool.actionByte()}`, llama
  `tool.Execute(ctx, req)`. `ctx` es `*context.Context` (paquete `tinywasm/context`, NO
  `router.Context`).
- `request_handler.go` construye ese `ctx` una sola vez en `mount.go:30-33` (`MountAPI`):
  ```go
  reqCtx := context.Background()
  reqCtx.Set(CtxKeyUserID, ctx.UserID())   // ctx aquí SÍ es router.Context, el de la petición HTTP
  resp := s.HandleMessage(reqCtx, ctx.Body())
  ```
  y ese `reqCtx` fluye sin reemplazo hasta `Execute`. La identidad se lee de vuelta así
  (`server.go:158`, patrón YA usado, cópialo tal cual):
  ```go
  userID := ctx.Value(CtxKeyUserID) // *context.Context.Value(key string) string — YA es string, sin type assertion
  ```
  `CtxKeyUserID = "mcp.user_id"` está en `request_handler.go:9`.
- `mcp.Request` (`tools.go:9-15`): `{ Params CallToolParams; Action byte }`. `CallToolParams.Arguments`
  es **`string`** (JSON crudo, `model.Raw()`), no `[]byte` — ver `model_orm.go:231-262`.
- `mcp.Result` (`model_orm.go:265-267`): `{ IsError bool; Content string }`.
- `mcp.Server.MountAPI(r router.Router)` (`mount.go:15-52`) registra **una sola ruta**:
  `r.Post(MCPPath, handler).Public()` — el sobre `/mcp`. **No usa `Op`, no debe usarlo**: `/mcp` es
  el transporte único que lleva `tools/call` para MUCHOS tools, cada uno con su propio RBAC — eso ya
  está resuelto y correcto, no lo toques.
- `mcp.NewCaller(c *Client) router.Caller` (`caller.go`, completo):
  ```go
  func NewCaller(c *Client) router.Caller { return &mcpCaller{client: c} }

  func (c *mcpCaller) Call(op string, args model.Encodable, callback func(result []byte, err error)) {
  	params := ... // arma CallToolParams{Name: op, Arguments: <args codificado>}
  	c.client.Call(context.Background(), string(MethodToolsCall), params, func(raw []byte, err error) {
  		if err != nil { callback(nil, err); return }
  		res, err := ParseResult(raw)
  		if err != nil { callback(nil, err); return }
  		if res.IsError {
  			callback(nil, fmt.Err("mcp: tool execution failed: "+string(res.Content)))
  			return
  		}
  		callback([]byte(res.Content), nil)
  	})
  }

  func (c *mcpCaller) Dispatch(op string, args model.Encodable) { ... c.client.Dispatch(...) }
  ```
  Implementa la firma **vieja** de 3 argumentos. `router@v0.1.14` exige:
  ```go
  type Caller interface {
  	Call(op string, args model.Encodable, into model.Decodable, done func(err error))
  	Dispatch(op string, args model.Encodable)
  }
  ```
  → `mcpCaller` **no compila** contra `router@v0.1.14` sin este cambio.
- `go.mod` actual: `github.com/tinywasm/router v0.1.12`. Súbelo a `v0.1.14`, que trae:
  `Route.Accepts`/`Context.Decode`/`Encode` (desde `v0.1.12`), `Caller.Call` tipado (`v0.1.13`), y
  **`Op` movido de `Router` a la nueva interfaz `router.OpRegistry` + `router.OpModule`** (`v0.1.14`).
  Este plan usa `OpRegistry`/`OpModule`, NO `Router.Op` (que ya no existe).

## 3. El cambio exacto

### 3.1 `go.mod`

```
go get github.com/tinywasm/router@v0.1.14
```

### 3.2 `mcpCaller.Call` — firma tipada (`caller.go`)

```go
func (c *mcpCaller) Call(op string, args model.Encodable, into model.Decodable, done func(err error)) {
	params, err := buildParams(op, args) // lo que ya arma internamente — no lo reinventes, solo léelo
	if err != nil {
		done(err)
		return
	}
	c.client.Call(context.Background(), string(MethodToolsCall), params, func(raw []byte, err error) {
		if err != nil {
			done(err)
			return
		}
		res, err := ParseResult(raw)
		if err != nil {
			done(err)
			return
		}
		if res.IsError {
			done(fmt.Err("mcp: tool execution failed: " + res.Content))
			return
		}
		if into == nil {
			done(nil)
			return
		}
		done(json.Decode([]byte(res.Content), into))
	})
}
```

`Dispatch` **no cambia** (sigue fire-and-forget, misma firma). Actualiza los tests de
`tests/caller_test.go` que usan la forma vieja `caller.Call("echo", args, func(result []byte, err
error) {...})` → deben pasar un destino `model.Decodable` y un `done func(error)`; sigue el patrón ya
publicado en `router/mock/caller.go` (referencia, no la copies literal, es de otro repo):
```go
// router/mock/caller.go — la forma de referencia para leer, ya publicada
func (c *Caller) Call(op string, args model.Encodable, into model.Decodable, done func(err error)) {
	c.Calls = append(c.Calls, Call{Op: op, Args: args})
	err := c.CannedError
	if err == nil && into != nil && len(c.CannedResult) > 0 {
		err = json.Decode(c.CannedResult, into)
	}
	if done != nil {
		done(err)
	}
}
```

### 3.3 El cosechador de operaciones — nuevo archivo `harvest.go`

```go
package mcp

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/json"
	"github.com/tinywasm/model"
	"github.com/tinywasm/router"
)

// HarvestOps builds a ToolProvider from one or more router.OpModule implementations. It runs
// each module's MountOps against an internal router.OpRegistry — the transport-neutral surface a
// domain module registers against without importing mcp — and converts every harvested Op into a
// Tool. This is the ONLY supported way for a domain module to reach the MCP transport; NewServer
// keeps accepting []ToolProvider for MCP-native providers (mcp's own tools, or a raw ToolProvider
// a repo still hand-writes). A composition root passes both:
//
//	providers := []mcp.ToolProvider{mcp.HarvestOps(catalogModule, userModule), rawProvider}
func HarvestOps(modules ...router.OpModule) ToolProvider {
	reg := &opRegistry{}
	for _, m := range modules {
		m.MountOps(reg)
	}
	return staticProvider(reg.tools)
}

type staticProvider []Tool

func (s staticProvider) Tools() []Tool { return []Tool(s) }

// opRegistry implements router.OpRegistry — a ONE-method surface. It does NOT implement (nor
// pretend to be) router.Router: an op-only transport must never carry Get/Post/… it can neither
// honour nor need. There is nothing to panic on, because there is nothing to leave unimplemented.
type opRegistry struct {
	tools []Tool
}

func (r *opRegistry) Op(name string, h router.HandlerFunc) router.Route {
	idx := len(r.tools)
	r.tools = append(r.tools, Tool{Name: name, Execute: harvestExecute(name, h)})
	return &opRoute{owner: r, idx: idx}
}

var _ router.OpRegistry = (*opRegistry)(nil)

// opRoute implements router.Route, writing straight into the Tool opRegistry already appended —
// no copy-then-writeback: Requires/Accepts/etc mutate the SAME Tool by index.
type opRoute struct {
	owner *opRegistry
	idx   int
}

func (rt *opRoute) Requires(resource model.Resource, action model.Action) router.Route {
	t := &rt.owner.tools[rt.idx]
	t.Access, t.Resource, t.Action = model.AccessGuarded, resource, action
	return rt
}
func (rt *opRoute) Authenticated() router.Route {
	rt.owner.tools[rt.idx].Access = model.AccessAuthenticated
	return rt
}
func (rt *opRoute) Public() router.Route {
	rt.owner.tools[rt.idx].Access = model.AccessPublic
	return rt
}
func (rt *opRoute) Accepts(args model.Fielder) router.Route {
	rt.owner.tools[rt.idx].Args = args
	return rt
}

var _ router.Route = (*opRoute)(nil)
```

### 3.4 El puente `router.Context` ↔ `mcp.Request`/`Result` — el mismo archivo

Un módulo escribe su handler contra `router.Context` (`ctx.Decode(&in)` / `ctx.Encode(&out)`); MCP
necesita adaptar eso a `(ctx *context.Context, req Request) (*Result, error)`:

```go
// opContext adapts one mcp.Request into router.Context so a router.HandlerFunc (registered via
// Op) can run unmodified against the MCP transport — the same handler would run verbatim under any
// future op transport that harvests the SAME module's MountOps.
type opContext struct {
	userID string
	body   []byte // request: raw Arguments; after the handler runs: what it wrote via Encode/Write
	status int
}

func (c *opContext) Method() string                          { return "POST" }
func (c *opContext) Path() string                             { return "" }
func (c *opContext) Body() []byte                             { return c.body }
func (c *opContext) GetHeader(string) string                  { return "" }
func (c *opContext) SetHeader(string, string)                 {}
func (c *opContext) WriteStatus(code int)                     { c.status = code }
func (c *opContext) Write(b []byte) (int, error)              { c.body = append([]byte{}, b...); return len(b), nil }
func (c *opContext) SetValue(string, any)                     {}
func (c *opContext) Value(string) any                         { return nil }
func (c *opContext) SetCookie(router.Cookie)                  {}
func (c *opContext) Cookie(string) (router.Cookie, bool)      { return router.Cookie{}, false }
func (c *opContext) SetUserID(id string)                      { c.userID = id }
func (c *opContext) UserID() string                           { return c.userID }
func (c *opContext) Decode(into model.Decodable) error        { return json.Decode(c.body, into) }
func (c *opContext) Encode(v model.Encodable) error {
	var out []byte
	if err := json.Encode(v, &out); err != nil {
		return err
	}
	c.body = out
	return nil
}

var _ router.Context = (*opContext)(nil)

// harvestExecute wraps a router.HandlerFunc as a Tool.Execute. name is unused today (kept for a
// future error-message improvement) — silence the unused-param lint with _ if your toolchain
// complains, do not delete the parameter (keeps the call site self-documenting).
func harvestExecute(name string, h router.HandlerFunc) func(ctx *context.Context, req Request) (*Result, error) {
	return func(ctx *context.Context, req Request) (*Result, error) {
		oc := &opContext{userID: ctx.Value(CtxKeyUserID), body: []byte(req.Params.Arguments)}
		h(oc)
		return &Result{IsError: oc.status >= 400, Content: string(oc.body)}, nil
	}
}
```

`ctx.Value(CtxKeyUserID)` ya es `string` directo (`context.Context.Value(key string) string`,
`/home/cesar/Dev/Project/tinywasm/context/*.go`) — **sin** type assertion, no repitas el patrón de
otros ecosistemas que devuelven `any`.

## 4. Fuera de alcance

- **No** toques `mount.go` — el sobre `/mcp` sigue registrado con `r.Post(...)`.Public()`, sigue
  codificando la respuesta JSON-RPC a mano. Es un mecanismo distinto (el transporte del sobre, no el
  de una operación individual) y ya es correcto.
- **No** elimines `NewServer(config, []ToolProvider)` ni `ToolProvider` — siguen siendo el camino para
  proveedores nativos de MCP. `HarvestOps` es **aditivo**: devuelve un `ToolProvider` más para
  combinar con los que ya existan.
- **No** conviertas `opRegistry` en un `router.Router`. Implementa **solo** `router.OpRegistry` (un
  método `Op`). No añadas `Get/Post/…` ni con panics ni con no-ops: un módulo de dominio recibe un
  `OpRegistry`, no un `Router`, así que esos métodos no existen en su superficie y no hay nada que
  fallar. (Ese `opRouter` con panics fue precisamente el olor que este plan elimina.)
- **No** cambies `Tool`, `ToolProvider`, `Request`, `Result`, ni `handleToolCall` — su forma ya
  encaja.

## 5. Test con forma de consumidor (obligatorio, arnés de construcción)

Prueba, DENTRO de este repo, que un "módulo" que solo conoce `router`+`model` llega a MCP:

```go
// tests/harvest_test.go
package tests

import (
	"testing"

	"github.com/tinywasm/mcp"
	"github.com/tinywasm/model"
	"github.com/tinywasm/router"
)

type fakeArgs struct{ Value string }

func (a *fakeArgs) IsNil() bool                      { return a == nil }
func (a *fakeArgs) Schema() []model.Field            { return nil }
func (a *fakeArgs) Pointers() []any                  { return []any{&a.Value} }
func (a *fakeArgs) EncodeFields(w model.FieldWriter) { w.String("value", a.Value) }
func (a *fakeArgs) DecodeFields(r model.FieldReader) {
	if v, ok := r.String("value"); ok {
		a.Value = v
	}
}

// fakeModule mimics a domain module: implements router.OpModule, imports ONLY router+model.
type fakeModule struct{}

func (fakeModule) ModelName() string { return "fake" }
func (fakeModule) MountOps(r router.OpRegistry) {
	r.Op("do_thing", func(ctx router.Context) {
		var in fakeArgs
		if err := ctx.Decode(&in); err != nil {
			ctx.WriteStatus(500)
			return
		}
		_ = ctx.Encode(&fakeArgs{Value: "echo:" + in.Value})
	}).Requires("fake_resource", model.Read).Accepts(&fakeArgs{})
}

var _ router.OpModule = fakeModule{}

func TestHarvestOps_ModuleReachesMCP(t *testing.T) {
	provider := mcp.HarvestOps(fakeModule{})
	tools := provider.Tools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 harvested tool, got %d", len(tools))
	}
	tool := tools[0]
	if tool.Name != "do_thing" || tool.Resource != "fake_resource" || tool.Action != model.Read {
		t.Fatalf("harvested tool metadata mismatch: %+v", tool)
	}
	if tool.Args == nil {
		t.Fatal("expected Accepts(...) to populate Tool.Args")
	}

	res, err := tool.Execute(nil, mcp.Request{Params: mcp.CallToolParams{Arguments: `{"value":"x"}`}})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError, content: %s", res.Content)
	}
	if res.Content != `{"value":"echo:x"}` {
		t.Errorf("unexpected content: %s", res.Content)
	}
}
```

> Ajusta el JSON esperado exacto (`{"value":"echo:x"}`) al que produzca `tinywasm/json.Encode`
> realmente — corre el test y usa la salida real si difiere en espaciado/orden; no adivines el
> formato exacto del codec.

## 6. Criterios de aceptación

- `go build ./...` verde con `router@v0.1.14`.
- `mcpCaller` satisface `router.Caller` (firma nueva) — `var _ router.Caller = (*mcpCaller)(nil)` ya
  existe o se añade.
- `mcp.HarvestOps(modules ...router.OpModule) ToolProvider` existe y funciona según §5.
- `grep -rn "panic(" harvest.go` **vacío** — el cosechador implementa `OpRegistry` (un método), no
  finge ser un `Router`. Si hay un panic, se implementó la interfaz equivocada.
- `grep -rn "router.Router" harvest.go` **vacío** — `harvest.go` no menciona `Router`, solo
  `OpRegistry`/`Route`/`Context`.
- `grep -rn "tinywasm/mcp" tests/harvest_test.go` **vacío salvo el propio import de `mcp`** — el
  `fakeModule` de prueba no importa nada de `mcp` más que a través del `ToolProvider` resultante.
- Tests existentes de `tests/caller_test.go` migrados a la firma nueva, verdes.
- `gotest ./...` (o `go test ./...`) verde.

## 7. Etapas

| # | Etapa | Archivo(s) | Criterio |
|---|---|---|---|
| 1 | Bump dependencia | `go.mod`, `go.sum` | `router@v0.1.14` |
| 2 | `Caller.Call` tipado | `caller.go` | firma nueva, `into`/`done` |
| 3 | Migrar tests de Caller | `tests/caller_test.go` | firma nueva, verdes |
| 4 | Cosechador de ops | `harvest.go` (nuevo) | `HarvestOps(...OpModule)`, `opRegistry` (implementa `OpRegistry`, sin panics), `opRoute` |
| 5 | Puente Context↔Request | `harvest.go` (mismo archivo) | `opContext`, `harvestExecute` |
| 6 | Test consumidor | `tests/harvest_test.go` (nuevo) | §5, verde |
| 7 | Verificación | — | `go build`/`gotest` verdes |
