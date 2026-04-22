# PLAN: Fix MCP `content` field — object → array

## Problema

El protocolo MCP (2024-11-05 y 2025-11-25) exige que el campo `content` en la respuesta de `tools/call` sea un **array**:

```json
{"content": [{"type": "text", "text": "..."}]}
```

El servidor actualmente envía un **objeto**:

```json
{"content": {"type":"text","text":"..."}}
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
    return &Result{Content: s}  // ← s es un objeto {"type":"text","text":"..."}, no un array
}
```

**Un solo bug:** `Content` debe ser un array JSON (`[{...}]`), no un objeto (`{...}`).

> **Nota sobre ormc y PascalCase:** `json.Encode` de `tinywasm/json` usa `Schema()` (generado por ormc) para obtener los nombres de campo, NO los nombres de la struct Go. El `model_orm.go` ya define `_schemaTextContent` con `"type"` y `"text"` en minúsculas. Las claves son correctas. **ormc no requiere ningún cambio.**

## Archivos a modificar

- `tools.go` — función `Text()`: envolver el content en array usando `TextContentList`

## Pasos

### Paso 1 — Envolver content en array usando `TextContentList` (`tools.go`)

`json.Encode` ya soporta `FielderSlice` — escribe un array `[...]` cuando recibe un tipo que implemente esa interfaz. `TextContentList` es generado por ormc en `model_orm.go` y ya implementa `FielderSlice`.

```go
// ANTES
func Text(text string) *Result {
    c := &TextContent{Type: "text", Text: text}
    var s string
    _ = json.Encode(c, &s)
    return &Result{Content: s}  // objeto: {"type":"text","text":"..."}
}

// DESPUÉS — usa TextContentList para que json.Encode produzca un array
func Text(text string) *Result {
    list := TextContentList{&TextContent{Type: "text", Text: text}}
    var s string
    _ = json.Encode(&list, &s)
    return &Result{Content: s}  // array: [{"type":"text","text":"..."}]
}
```

> `TextContentList` implementa `fmt.FielderSlice` (generado por ormc), por lo que `json.Encode` llama a `encodeSlice` internamente y produce `[...]`. No se necesita concatenación manual ni cambios en ormc.

### Paso 2 — Actualizar `GetText` en `utils.go`

Al cambiar `Content` de objeto a array, `GetText` debe extraer el primer elemento:

```go
// ANTES — decodifica un objeto
func GetText(r *Result) (string, error) {
    var c TextContent
    if err := json.Decode([]byte(r.Content), &c); err != nil {
        return "", err
    }
    return c.Text, nil
}

// DESPUÉS — decodifica el primer elemento del array via TextContentList
func GetText(r *Result) (string, error) {
    var list TextContentList
    if err := json.Decode([]byte(r.Content), &list); err != nil {
        return "", err
    }
    if len(list) == 0 {
        return "", fmt.Err("mcp", "empty content array")
    }
    return list[0].Text, nil
}
```

> `json.Decode` soporta `FielderSlice` (verificado en `decode.go:37-39` — es simétrico al encode). `TextContentList` implementa esa interfaz, por lo que el decode del array funciona directamente.

### Paso 3 — Tests (ya escritos en `tests/mcp_test.go`)

Los tres tests están en `tests/mcp_test.go` y se pueden correr con `go test ./tests/`:

- `TestToolCall_ContentIsArray` — **falla antes del fix** con: `Bug C not fixed: content is an object instead of an array`
- `TestToolCall_ContentItemLowercaseKeys` — pasa (regresión ormc, claves ya correctas)
- `TestText_GetText_RoundTrip` — pasa (actualmente `GetText` decodifica objeto; fallará tras el fix del Paso 2)

### Paso 4 — Revisar función `JSON()` en `utils.go`

`JSON()` también asigna directamente a `Content` sin envolver en array. Evaluar si se usa para respuestas externas (requiere array) o solo internamente.