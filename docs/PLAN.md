# PLAN: Migrar campos `raw` a `fmt.RawJSON`

## Contexto

El tag `json:",raw"` genera warnings de linter (`unknown JSON option "raw"`).
La solución es usar el type alias `fmt.RawJSON` introducido en tinywasm/fmt.

Ver plan principal en [tinywasm/fmt/docs/PLAN.md](../../fmt/docs/PLAN.md).

## Cambios requeridos

- [ ] Migrar todos los campos con `json:"...,raw"` en `model.go` a `fmt.RawJSON`
- [ ] Regenerar `model_orm.go` con ormc
