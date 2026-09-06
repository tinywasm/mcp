package mcp

import (
	"webtyp.com/fmt"
	"webtyp.com/model"
)

const EmptyInputSchema = `{"type":"object","properties":{}}`

func inputSchemaOf(m model.Fielder) string {
	if m == nil {
		return EmptyInputSchema
	}
	fields := m.Schema()
	if len(fields) == 0 {
		return EmptyInputSchema
	}
	var b fmt.Conv
	b.Write(`{"type":"object","properties":{`)
	var required []string
	for i, f := range fields {
		if i > 0 {
			b.Write(",")
		}
		b.Write(`"`)
		b.Write(f.Name)
		b.Write(`":`)
		b.Write(jsonSchemaType(f.Type.Storage()))
		if f.NotNull {
			required = append(required, f.Name)
		}
	}
	b.Write("}")
	if len(required) > 0 {
		b.Write(`,"required":[`)
		for i, name := range required {
			if i > 0 {
				b.Write(",")
			}
			b.Write(`"`)
			b.Write(name)
			b.Write(`"`)
		}
		b.Write("]")
	}
	b.Write("}")
	return b.String()
}

func jsonSchemaType(t model.FieldType) string {
	switch t {
	case model.FieldInt:
		return `{"type":"integer"}`
	case model.FieldFloat:
		return `{"type":"number"}`
	case model.FieldBool:
		return `{"type":"boolean"}`
	case model.FieldIntSlice:
		return `{"type":"array","items":{"type":"integer"}}`
	case model.FieldStruct:
		return `{"type":"object"}`
	case model.FieldStructSlice:
		return `{"type":"array","items":{"type":"object"}}`
	default:
		return `{"type":"string"}`
	}
}
