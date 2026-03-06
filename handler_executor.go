package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// toolExecutorAdapter converts a ToolExecutor into a ToolHandlerFunc.
func toolExecutorAdapter(executor ToolExecutor) ToolHandlerFunc {
	return func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		args := req.GetArguments()

		var messages []string
		var binaryResponse *BinaryData

		// Collectors for tool output. The executor may call a logger callback
		// (passed by app via executor's closure) to publish results.
		_ = func(message ...any) {
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
