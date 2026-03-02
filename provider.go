package mcp

import "context"

// RegisterProvider registers all MCP tools declared by a ToolProvider.
// The provider's GetMCPToolsMetadata() is called once at registration time.
// Each ToolMetadata.Execute is automatically adapted to ToolHandlerFunc.
//
// This is the standard registration entry point for tinywasm ecosystem modules.
// Modules implement GetMCPToolsMetadata() and call srv.RegisterProvider(m) — one line.
//
// Example (in a domain module):
//
//   func (m *Module) RegisterTools(srv *mcp.MCPServer) {
//       srv.RegisterProvider(m)
//   }
//
//   func (m *Module) GetMCPToolsMetadata() []mcp.ToolMetadata {
//       return []mcp.ToolMetadata{
//           {Name: "get_status", Description: "...", Execute: m.GetStatus},
//           {Name: "toggle",     Description: "...", Execute: m.Toggle,
//               Parameters: []mcp.ParameterMetadata{
//                   {Name: "is_enabled", Description: "Enable or disable", Required: true, Type: "boolean"},
//               },
//           },
//       }
//   }
func (s *MCPServer) RegisterProvider(provider ToolProvider) {
	for _, meta := range provider.GetMCPToolsMetadata() {
		tool := buildMCPTool(meta)
		exec := meta.Execute // capture for closure

		s.AddTool(*tool, func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
			if exec == nil {
				return NewToolResultError("tool has no executor"), nil
			}
			args := make(map[string]any)
			if err := req.BindArguments(&args); err != nil {
				return NewToolResultError("invalid arguments: " + err.Error()), nil
			}
			result, err := exec(ctx, args)
			if err != nil {
				return NewToolResultError(err.Error()), nil
			}
			return NewToolResultStructuredOnly(result), nil
		})
	}
}
