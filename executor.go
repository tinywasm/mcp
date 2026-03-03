package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// mcpExecuteTool creates a generic tool executor that works for any handler tool.
// It extracts args, collects logs via SetLog, executes the tool, and returns results.
func mcpExecuteTool(targetHandler any, executor ToolExecutor) ToolHandlerFunc {
	return func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		args := make(map[string]any)
		if err := req.BindArguments(&args); err != nil {
			return NewToolResultError("invalid arguments: " + err.Error()), nil
		}

		var messages []string

		if loggable, ok := targetHandler.(Loggable); ok {
			loggable.SetLog(func(message ...any) {
				for _, m := range message {
					if str, ok := m.(string); ok {
						messages = append(messages, str)
					} else {
						messages = append(messages, fmt.Sprintf("%v", m))
					}
				}
			})
		}

		result, err := executor(ctx, args)
		if err != nil {
			return NewToolResultError(err.Error()), nil
		}

		if bd, ok := result.(BinaryData); ok {
			textSummary := strings.Join(messages, "\n")
			b64 := base64.StdEncoding.EncodeToString(bd.Data)
			return NewToolResultImage(textSummary, b64, bd.MimeType), nil
		}

		if result != nil {
			return NewToolResultStructuredOnly(result), nil
		}

		if len(messages) > 0 {
			return NewToolResultText(strings.Join(messages, "\n")), nil
		}

		return NewToolResultText("Operation completed successfully"), nil
	}
}
