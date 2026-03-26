# Stage 15 — Final Verification

## 15.1 — No stdlib banned imports

```bash
grep -rn '"context"'         . --include="*.go" | grep -v "_test.go" | grep -v ".back.go"
grep -rn '"encoding/json"'   . --include="*.go"
grep -rn '"encoding/base64"' . --include="*.go"
grep -rn '"sort"'            . --include="*.go"
grep -rn '"slices"'          . --include="*.go"
grep -rn '"strings"'         . --include="*.go"
grep -rn '"fmt"'             . --include="*.go"
```

All results must be empty.
`"context"` is allowed **only** in `.back.go` files (with alias `stdctx`).

## 15.2 — No map[string]any / []any / interface{}

```bash
grep -rn 'map\[string\]any\|map\[string\]interface{}\|\[\]any\|\[\]interface{}' . --include="*.go"
```

Must be empty.

## 15.3 — No json.RawMessage

```bash
grep -rn 'json\.RawMessage\|RawMessage' . --include="*.go"
```

Must be empty.

## 15.4 — ormc output is up to date

```bash
cd /path/to/mcp && ormc
git diff model_orm.go
```

No diff — generated file matches source structs.



