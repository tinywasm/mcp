package mcp

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
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
		mcpCtx := context.Background()
		token := ctx.GetHeader("Authorization")
		if token != "" {
			mcpCtx.Set(CtxKeyAuthToken, token)
		}

		resp := s.HandleMessage(mcpCtx, ctx.Body())
		if resp == nil {
			// Notifications (id is null) return nil in HandleMessage
			ctx.WriteStatus(204)
			return
		}

		ctx.SetHeader(headerContentType, mimeJSON)
		var out []byte
		var err error

		// Handle single response vs batch (if HandleMessage were to support batches in the future)
		// For now HandleMessage returns JSONRPCMessage which is an interface.
		if enc, ok := resp.(fmt.Encodable); ok {
			err = json.Encode(enc, &out)
		} else {
			// Fallback for types that might not implement fmt.Encodable but are valid responses
			// Though in this library they should.
			ctx.WriteStatus(500)
			ctx.Write([]byte(`{"error":"mcp: response not encodable"}`))
			return
		}

		if err != nil {
			ctx.WriteStatus(500)
			ctx.Write([]byte(`{"error":"mcp: encode failed"}`))
			return
		}
		ctx.Write(out)
	})
}
