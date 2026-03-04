package mcp

import (
	"context"
	"encoding/base64"
)

// RegisterProvider registers all MCP tools declared by a ToolProvider.
// Calls provider.GetMCPToolsMetadata() once at registration time.
// Each ToolMetadata.Execute is automatically adapted to ToolHandlerFunc.
//
// This is the standard registration entry point for tinywasm ecosystem modules:
//
//   func (m *Module) RegisterTools(srv *mcp.MCPServer) {
//       srv.RegisterProvider(m)
//   }
func (s *MCPServer) RegisterProvider(provider ToolProvider) {
    for _, meta := range provider.GetMCPToolsMetadata() {
        tool := BuildMCPTool(meta)
        exec := meta.Execute // capture

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
            if bd, ok := result.(BinaryData); ok {
                b64 := base64.StdEncoding.EncodeToString(bd.Data)
                return NewToolResultImage("", b64, bd.MimeType), nil
            }
            return NewToolResultStructuredOnly(result), nil
        })
    }
}
