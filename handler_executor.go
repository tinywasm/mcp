package mcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// tuiRefresher is a minimal local interface to avoid importing devtui.
type tuiRefresher interface {
	RefreshUI()
}

// toolExecutorAdapter converts a ToolExecutor + Loggable handler into a ToolHandlerFunc.
func toolExecutorAdapter(targetHandler any, executor ToolExecutor, tui tuiRefresher) ToolHandlerFunc {
	return func(ctx context.Context, req CallToolRequest) (*CallToolResult, error) {
		args := req.GetArguments()

		var messages []string
		var binaryResponse *BinaryData

		if loggable, ok := targetHandler.(Loggable); ok {
			loggable.SetLog(func(message ...any) {
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
			})
		}

		executor(args)

		if tui != nil {
			tui.RefreshUI()
		}

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
