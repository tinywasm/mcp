package mcp

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/json"
	"github.com/tinywasm/router"
)

// MCPPath is the canonical route for the JSON-RPC endpoint.
const MCPPath = "/mcp"

// ModelName identifies the module (identity contract reused by APIModule).
func (s *Server) ModelName() string { return s.name }

// MountAPI publishes the MCP endpoint on the host's Router.
func (s *Server) MountAPI(r router.Router) {
	r.Post(MCPPath, func(ctx router.Context) {
		reqCtx := context.Background()
		reqCtx.Set(CtxKeyUserID, ctx.UserID())
		resp := s.HandleMessage(reqCtx, ctx.Body())
		ctx.SetHeader(headerContentType, mimeJSON)
		var out string
		var err error
		switch m := resp.(type) {
		case *JSONRPCResponseStruct:
			err = json.Encode(m, &out)
		case *JSONRPCError:
			err = json.Encode(m, &out)
		default:
			ctx.WriteStatus(500)
			ctx.Write([]byte(`{"error":"mcp: unknown response type"}`))
			return
		}
		if err != nil {
			ctx.WriteStatus(500)
			ctx.Write([]byte(`{"error":"mcp: encode failed"}`))
			return
		}
		ctx.Write([]byte(out))
	})
}
