package mcp

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/json"
	"github.com/tinywasm/model"
)

type Request struct {
	Params CallToolParams
	Action byte
}

type Tool struct {
	Name        string
	Description string
	Args        model.Fielder // model of tool arguments (ormc-generated); nil = no args
	Resource    string        // required — RBAC resource identifier
	Action      byte          // required — 'c','r','u','d'
	Public      bool          // explicit: accessible without identity. Absence = private.
	Execute     func(ctx *context.Context, req Request) (*Result, error)
}

// DecodableFields combines Decodable (codec) with Validate (validation).
// ormc-generated model types satisfy this interface.
type DecodableFields interface {
	model.Decodable
	Validate(action byte) error
}

func (r *Request) Bind(target DecodableFields) error {
	if err := json.Decode([]byte(r.Params.Arguments), target); err != nil {
		return err
	}
	return target.Validate(r.Action)
}

func Text(text string) *Result {
	list := TextContentList{&TextContent{Type: "text", Text: text}}
	var s string
	_ = json.Encode(&list, &s)
	return &Result{Content: s}
}

type ToolProvider interface {
	Tools() []Tool
}
