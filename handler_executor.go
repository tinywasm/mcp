package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// toolExecutorAdapter converts a ToolExecutor into a ToolHandlerFunc.
// Optionally wraps a Loggable to route output to MCP response collector.
func toolExecutorAdapter(executor ToolExecutor, loggable Loggable, debug bool) ToolHandlerFunc {
	return func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		args := req.GetArguments()

		var messages []string
		var binaryResponse *BinaryData

		// Collector for tool output destined for MCP response.
		collector := func(message ...any) {
			for _, m := range message {
				switch v := m.(type) {
				case BinaryData:
					binaryResponse = &v
				case string:
					messages = append(messages, v)
				default:
					messages = append(messages, fmt.Sprintf("%v", v))
				}
			}
		}

		// Swap logger: route output to MCP response collector.
		if loggable != nil {
			original := loggable.GetLog()
			if debug {
				// Debug mode: output goes to both TUI and MCP response.
				loggable.SetLog(func(msg ...any) {
					original(msg...)    // → TUI
					collector(msg...)   // → MCP response
				})
			} else {
				// Normal mode: output goes to MCP response only.
				loggable.SetLog(collector)
			}
			defer loggable.SetLog(original)
		}

		executor(args)

		if binaryResponse != nil {
			base64Data := base64.StdEncoding.EncodeToString(binaryResponse.Data)
			text := strings.Join(messages, "\n")
			return NewToolResultImage(text, base64Data, binaryResponse.MimeType), nil
		}
		if len(messages) == 0 {
			return NewToolResultText("Operation completed successfully"), nil
		}
		return NewToolResultText(strings.Join(messages, "\n")), nil
	}
}
