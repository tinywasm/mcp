package mcp_test

import (
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
	"github.com/tinywasm/model"
	"github.com/tinywasm/router/mock"
)

// The /mcp route must declare its access EXPLICITLY. Left unannotated it falls into the
// zero value AccessGuarded with no Resource — a contradiction that makes httpd refuse to
// start (and which, before httpd validated it, would have denied every request while
// looking protected).
//
// Public is the only coherent verdict for the transport: /mcp is an envelope carrying
// calls to many tools, each with its own resource, so naming one on the route would be a
// lie. The real gate is per-tool, and it is closed by default.
func TestMountAPI_RouteIsExplicitlyPublic(t *testing.T) {
	srv, err := mcp.NewServer(mcp.Config{Name: "t", Version: "1", Authorize: mcp.AllowAll}, nil)
	if err != nil {
		t.Fatal(err)
	}

	r := &mock.Router{}
	srv.MountAPI(r)

	routes := r.Routes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	route := routes[0]

	if route.Path != mcp.MCPPath || route.Method != "POST" {
		t.Errorf("got %s %s, want POST %s", route.Method, route.Path, mcp.MCPPath)
	}
	if route.Access != model.AccessPublic {
		t.Errorf("the /mcp route is %v, not public: an unannotated route is AccessGuarded "+
			"with no Resource, which httpd refuses to start with", route.Access)
	}
	if route.Resource != "" {
		t.Errorf("the transport route declares Resource %q: /mcp carries calls to many "+
			"tools, each with its own resource — naming one here is a lie", route.Resource)
	}
}

// The other half: making the transport public must NOT open the tools behind it. If this
// ever goes green-by-accident, the route being public would be a real hole.
func TestMountAPI_PublicRouteStillGuardsTools(t *testing.T) {
	auth := &rbacAuth{denyResource: "secrets", denyAction: model.Read}
	srv, err := mcp.NewServer(mcp.Config{Name: "t", Version: "1", Authorize: auth.Can}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.AddTool(okTool("guarded", model.AccessGuarded, "secrets")); err != nil {
		t.Fatal(err)
	}

	// Anonymous: no identity in the context, exactly as the public route delivers it.
	if _, err := callTool(srv, context.Background(), "guarded"); err == nil {
		t.Error("a public transport route let an anonymous caller reach a guarded tool")
	}
}
