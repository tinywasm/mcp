package mcp_test

import (
	"testing"

	"webtyp.com/mcp"
	"webtyp.com/model"
	"webtyp.com/router"
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

// fakeModule mimics a domain module: implements router.OperationModule, imports ONLY router+model.
type fakeModule struct{}

func (fakeModule) ModelName() string { return "fake" }
func (fakeModule) MountOperations(r router.OperationRegistry) {
	r.Operation("do_thing", func(ctx router.Context) {
		var in fakeArgs
		if err := ctx.Decode(&in); err != nil {
			ctx.WriteStatus(500)
			return
		}
		_ = ctx.Encode(&fakeArgs{Value: "echo:" + in.Value})
	}).Requires("fake_resource", model.Read).Accepts(&fakeArgs{})
}

var _ router.OperationModule = fakeModule{}

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
