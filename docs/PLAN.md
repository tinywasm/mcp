# PLAN: Fix MCP `content` field — object → array

## Problema

El protocolo MCP (2024-11-05 y 2025-11-25) exige que el campo `content` en la respuesta de `tools/call` sea un **array**:

```json
{"content": [{"type": "text", "text": "..."}]}
```

El servidor actualmente envía un **objeto** (y con claves en PascalCase):

```json
{"content": {"Type":"text","Text":"..."}}
```

Esto causa el error que reporta Claude Code:
```
Invalid input: expected array, received object — path: content
```

## Causa raíz

**Archivo:** `tools.go:30-35`

```go
func Text(text string) *Result {
    c := &TextContent{Type: "text", Text: text}
    var s string
    _ = json.Encode(c, &s)
    return &Result{Content: s}  // ← s es un objeto, no un array
}
```

Dos bugs encadenados:

1. `Content` debe ser un array JSON (`[{...}]`), no un objeto (`{...}`)
2. `TextContent` usa PascalCase → `json.Encode` produce `{"Type":"text","Text":"..."}` en lugar de `{"type":"text","text":"..."}`

## Archivos a modificar

- `model.go` — `TextContent`: agregar tags `json:"type"` / `json:"text"`
- `tools.go` — función `Text()`: envolver el content en array `[...]`

## Pasos

### Paso 1 — Corregir tags JSON en `TextContent` (`model.go`)

```go
// ANTES
type TextContent struct {
    Type string
    Text string
}

// DESPUÉS
type TextContent struct {
    Type string `json:"type"`
    Text string `json:"text"`
}
```

### Paso 2 — Envolver content en array (`tools.go`)

```go
// ANTES
func Text(text string) *Result {
    c := &TextContent{Type: "text", Text: text}
    var s string
    _ = json.Encode(c, &s)
    return &Result{Content: s}
}

// DESPUÉS
func Text(text string) *Result {
    c := &TextContent{Type: "text", Text: text}
    var item string
    _ = json.Encode(c, &item)
    return &Result{Content: "[" + item + "]"}
}
```

### Paso 3 — Agregar tests que documenten el bug (`tests/protocol_test.go`)

Los tests deben fallar **antes** del fix y pasar **después**, siguiendo el patrón de Bug A y Bug B ya existentes.

Agregar al final de `tests/protocol_test.go`:

```go
// TestToolCall_ContentIsArray verifica que la respuesta de tools/call
// devuelva content como array JSON, no como objeto.
// Este test FALLA antes del fix (tools.go Text() envuelve objeto, no array).
func TestToolCall_ContentIsArray(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "ping",
		Resource: "test",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("pong"), nil
		},
	})
	var ctx context.Context

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"ping","arguments":{}}}`)
	resp := srv.HandleMessage(&ctx, req)

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	body := encodeResponse(resp)

	// El protocolo MCP exige: "content":[{...}] — array, no objeto
	if contains(body, `"content":{`) {
		t.Fatalf("Bug C not fixed: content is an object instead of an array.\nResponse: %s", body)
	}
	if !contains(body, `"content":[`) {
		t.Fatalf("expected content as JSON array, got: %s", body)
	}
}

// TestToolCall_ContentItemLowercaseKeys verifica que los items dentro de content
// usen claves en minúsculas ("type", "text") según el protocolo MCP.
// Este test FALLA antes del fix (TextContent sin json tags produce PascalCase).
func TestToolCall_ContentItemLowercaseKeys(t *testing.T) {
	srv, _ := mcp.NewServer(mcp.Config{Name: "test", Version: "1.0.0", Auth: mcp.OpenAuthorizer()}, nil)
	srv.AddTool(mcp.Tool{
		Name:     "ping",
		Resource: "test",
		Action:   'r',
		Execute: func(ctx *context.Context, req mcp.Request) (*mcp.Result, error) {
			return mcp.Text("pong"), nil
		},
	})
	var ctx context.Context

	req := []byte(`{"jsonrpc":"2.0","id":"1","method":"tools/call","params":{"name":"ping","arguments":{}}}`)
	resp := srv.HandleMessage(&ctx, req)

	body := encodeResponse(resp)

	// Claves PascalCase ("Type", "Text") son inválidas para el protocolo MCP
	if contains(body, `"Type":`) || contains(body, `"Text":`) {
		t.Fatalf("Bug C not fixed: content item uses PascalCase keys instead of lowercase.\nResponse: %s", body)
	}
	if !contains(body, `"type":"text"`) || !contains(body, `"text":"pong"`) {
		t.Fatalf("expected lowercase 'type' and 'text' keys in content item, got: %s", body)
	}
}

// TestText_GetText_RoundTrip verifica que mcp.Text() + mcp.GetText() funcionen
// correctamente después del fix al formato array.
func TestText_GetText_RoundTrip(t *testing.T) {
	r := mcp.Text("hello protocol")
	text, err := mcp.GetText(r)
	if err != nil {
		t.Fatalf("GetText error after array fix: %v — Content was: %s", err, r.Content)
	}
	if text != "hello protocol" {
		t.Fatalf("got %q, expected 'hello protocol'", text)
	}
}
```

**Nota:** `GetText` en `utils.go` también deberá actualizarse para extraer el primer item del array:

```go
// ANTES
func GetText(r *Result) (string, error) {
    var c TextContent
    if err := json.Decode([]byte(r.Content), &c); err != nil {
        return "", err
    }
    return c.Text, nil
}

// DESPUÉS — extrae primer elemento del array
func GetText(r *Result) (string, error) {
    // Content es "[{...}]", extraer primer elemento
    raw := []byte(r.Content)
    first := mcp.ExtractJSONValue(raw, "0")  // alternativa: parse manual del array
    var c TextContent
    if err := json.Decode(first, &c); err != nil {
        return "", err
    }
    return c.Text, nil
}
```

O más simple, si `tinywasm/json` soporta decode de slice:

```go
func GetText(r *Result) (string, error) {
    var items []TextContent
    if err := json.Decode([]byte(r.Content), &items); err != nil {
        return "", err
    }
    if len(items) == 0 {
        return "", fmt.Err("mcp", "empty content array")
    }
    return items[0].Text, nil
}
```

### Paso 4 — Revisar función `JSON()` en `utils.go`

`JSON()` también asigna directamente a `Content` sin envolver en array. Si se usa para respuestas estructuradas, debe evaluarse si necesita el mismo tratamiento o si es para uso interno.

### Paso 4 — Verificar `Result.Content` en `model.go`

```go
type Result struct {
    IsError bool        `json:"isError,omitempty"`
    Content fmt.RawJSON  // ← debe serializar como array
}
```

Verificar que el encoder de `tinywasm/json` respeta `fmt.RawJSON` como raw (sin re-escapar).