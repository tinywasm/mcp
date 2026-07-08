# PLAN — Bearer-token auth for `mcp.Client`

> This plan is dispatched via the CodeJob workflow. See skill: `agents-workflow`.

You are an external agent with **zero prior context** about this project. Everything you need
is in this file. Read it fully before writing code.

---

## 1. Problem

The daemon's `/mcp` JSON-RPC endpoint **requires authentication** for the `tinywasm/state` and
`tinywasm/action` methods when the daemon runs in secured mode (an API key is configured). The
server extracts the token from the `Authorization: Bearer <token>` request header and rejects
unauthenticated calls with a JSON-RPC `Unauthorized` error:

```go
// app/daemon.go (server side — for context only, do NOT edit here)
authHeader := r.Header.Get("Authorization")
token := authHeader
if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
	token = authHeader[7:]
}
// ...
if _, err := auth.Authorize(token); err != nil {
	// -> {"jsonrpc":"2.0","id":..,"error":{"code":-32000,"message":"Unauthorized"}}
	return
}
```

But **`mcp.Client` cannot send an `Authorization` header** — it has no auth support at all
(`client.go`). Its only consumer, `tinywasm/devtui`, therefore calls `tinywasm/state` and
`tinywasm/action` **without credentials**, so on a secured daemon:

- `fetchAndReconstructState()` (GET-like `state` call) → `Unauthorized` → client-mode state
  reconstruction silently fails.
- `Dispatch("tinywasm/action", …)` (e.g. the Ctrl+C "stop" action, remote field actions) →
  `Unauthorized` → the action never runs.

This plan adds **optional Bearer-token authentication** to `mcp.Client`, backward compatible.

> Downstream (separate `devtui` repo, do NOT edit here — informational): after this ships,
> `devtui`'s `mcpClient()` must build the client with the token, and its flaky
> `TestSSEClient_Authentication` must be made deterministic. See Section 7.

---

## 2. Current code (`client.go`, in this repo — reuse, do not rewrite wholesale)

```go
package mcp

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/fetch"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
)

type Client struct {
	endpoint string
}

func NewClient(baseURL string) *Client {
	return &Client{
		endpoint: fmt.Convert(baseURL).TrimSuffix("/").String() + "/mcp",
	}
}

func (c *Client) Call(ctx *context.Context, method string, params any, callback func([]byte, error)) {
	body := c.buildBody(method, params)
	if body == nil { /* … */ return }
	fetch.Post(c.endpoint).ContentTypeJSON().Body(body).Send(func(resp *fetch.Response, err error) {
		// … response handling …
	})
}

func (c *Client) Dispatch(ctx *context.Context, method string, params any) {
	body := c.buildBody(method, params)
	if body == nil { return }
	fetch.Post(c.endpoint).ContentTypeJSON().Body(body).Send(func(*fetch.Response, error) {})
}
```

Key facts:
- `github.com/tinywasm/fetch` supports request headers: `func (r *fetch.Request) Header(key, value string) *fetch.Request` (chainable).
- The **only** caller of `NewClient` in the whole ecosystem is `devtui/sse_client.go`. There is
  no reason to keep a backward-compatible overload — change the signature directly and update
  that single caller. Do NOT add functional-option cruft to preserve the old form.

---

## 3. Required change — make the token a required constructor parameter

### 3.1 Named constants (no hardcoded strings)

Add to `client.go`:

```go
const (
	headerAuthorization = "Authorization"
	bearerPrefix        = "Bearer "
)
```

### 3.2 Client field + new signature

Change `NewClient` to take the auth token directly. An empty token means open /
unauthenticated mode (no `Authorization` header sent). No options, no overload.

```go
type Client struct {
	endpoint  string
	authToken string // when non-empty, sent as "Authorization: Bearer <token>"
}

// NewClient targets baseURL + "/mcp". authToken is sent as a Bearer token on
// every request; pass "" for open/unauthenticated daemons.
func NewClient(baseURL, authToken string) *Client {
	return &Client{
		endpoint:  fmt.Convert(baseURL).TrimSuffix("/").String() + "/mcp",
		authToken: authToken,
	}
}
```

### 3.3 Attach the header on every request

Add a single private helper so `Call` and `Dispatch` share the header logic (no duplication):

```go
// newPost builds the POST request for this client, attaching the Authorization
// header when a token is configured.
func (c *Client) newPost(body []byte) *fetch.Request {
	r := fetch.Post(c.endpoint).ContentTypeJSON().Body(body)
	if c.authToken != "" {
		r = r.Header(headerAuthorization, bearerPrefix+c.authToken)
	}
	return r
}
```

Then replace the inline `fetch.Post(...)` chains in **both** `Call` and `Dispatch`:

```go
// in Call:
c.newPost(body).Send(func(resp *fetch.Response, err error) { /* unchanged body */ })

// in Dispatch:
c.newPost(body).Send(func(*fetch.Response, error) {})
```

Do NOT change the response-handling logic, `buildBody`, or the rpc types.

---

## 4. Tests (`client_test.go` or a new `client_auth_test.go` in this repo)

Add unit tests that assert the header is sent (and omitted). Use a local HTTP test server and
point the client at it. Required cases:

1. **Auth header present**: `NewClient(server.URL, "k")` → the server receives
   `Authorization: Bearer k` for both `Call` and `Dispatch`.
2. **Empty token is open mode**: `NewClient(server.URL, "")` → the server receives an **empty**
   `Authorization` header (no header set).

Since `Call`/`Dispatch` are asynchronous (callback-based via `fetch`), capture the observed
header through a buffered channel and read it with a timeout (see the pattern below).

```go
gotAuth := make(chan string, 1)
srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	gotAuth <- r.Header.Get("Authorization")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":null}`))
}))
defer srv.Close()
// ... NewClient(srv.URL, "k").Call(...) ...
select {
case a := <-gotAuth:
	// assert a == "Bearer k"
case <-time.After(2 * time.Second):
	t.Fatal("timed out")
}
```

Run `go test ./...` — all existing tests MUST still pass.

---

## 5. Code-quality rules (enforced)

- **No hardcoded strings in logic**: use the `headerAuthorization` / `bearerPrefix` constants
  (Section 3.1). No `"Authorization"` / `"Bearer "` literals in `Call`/`Dispatch`/helpers.
- **No standard library** where a tinywasm package is the convention: this package already uses
  `github.com/tinywasm/fmt`, `github.com/tinywasm/json`, `github.com/tinywasm/fetch`,
  `github.com/tinywasm/context`. Test files may use stdlib `net/http`, `net/http/httptest`,
  `testing`, `time` (consistent with typical `_test.go` in the ecosystem).
- **Clean break, no cruft**: change `NewClient`'s signature directly to `NewClient(baseURL,
  authToken string)`. Do NOT keep an old overload or add functional options to preserve the
  previous form — the single caller is updated in the follow-up (Section 7). Leave
  `Call`/`Dispatch`/`buildBody` signatures unchanged.
- **No logic duplication**: `Call` and `Dispatch` build their request through the shared
  `newPost` helper.

---

## 6. Documentation

- Update `README.md` (if it documents `NewClient`/the client) to the new signature:
  `mcp.NewClient(baseURL, token)` (pass `""` for open mode).
- If `docs/` has an API/architecture doc that lists the client surface, update the `NewClient`
  signature there too.

---

## 7. Downstream follow-up (NOT part of this repo/dispatch — for the maintainer)

After this ships and `mcp` is published, `tinywasm/devtui` needs a small consumer change
(separate `docs/PLAN.md` in the devtui repo):

1. `sse_client.go` `mcpClient()`: build with the key — `mcp.NewClient(baseURL, h.apiKey)`
   (this is the required update for the new mandatory `authToken` parameter).
2. `client_mode_test.go` `TestSSEClient_Authentication` is **flaky** today: the test server
   captures the `Authorization` header of *any* request into a size-1 channel, and the
   pre-connect `fetchAndReconstructState()` fires an **unauthenticated** `/mcp` request that
   races the `/logs` SSE request — the test reads whichever lands first, sometimes getting `""`.
   Fix by only capturing the `/logs` request: `if r.URL.Path != "/logs" { return }` before the
   channel send. (With this mcp change wired in, the `/mcp` request also carries the token, so
   the race is doubly resolved.)

---

## 8. Execution stages

| Stage | File | Deliverable |
|---|---|---|
| 1 | `client.go` | Constants, `authToken` field, new `NewClient(baseURL, authToken string)` signature (Section 3.1–3.2). |
| 2 | `client.go` | `newPost` helper; route `Call` and `Dispatch` through it (Section 3.3). |
| 3 | `client_auth_test.go` | Auth-present / no-auth / empty-token tests (Section 4); `go test ./...` green. |
| 4 | `README.md` (+ `docs/` if applicable) | Document `WithBearerToken` (Section 6). |

Do NOT run `gopush` or `codejob` — those are local developer tools managed outside this task.
