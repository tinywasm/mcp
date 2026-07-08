package mcp

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
	"github.com/tinywasm/model"
	"github.com/tinywasm/router"
)

type mcpCaller struct {
	client *Client
}

// NewCaller adapts *Client to the router.Caller contract. It is the single
// place that knows the tools/call envelope. Consumers (views) depend on
// router.Caller only and pass logical operation names (tool names).
func NewCaller(c *Client) router.Caller {
	return &mcpCaller{client: c}
}

func (c *mcpCaller) Call(op string, args model.Encodable, callback func(result []byte, err error)) {
	var argsJSON string
	if args != nil {
		if err := json.Encode(args, &argsJSON); err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
	}

	params := &CallToolParams{
		Name:      op,
		Arguments: argsJSON,
	}

	c.client.Call(context.Background(), string(MethodToolsCall), params, func(raw []byte, err error) {
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}

		res, err := ParseResult(raw)
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}

		if res.IsError {
			if callback != nil {
				callback(nil, fmt.Err("mcp: tool execution failed: "+string(res.Content)))
			}
			return
		}

		if callback != nil {
			callback([]byte(res.Content), nil)
		}
	})
}

func (c *mcpCaller) Dispatch(op string, args model.Encodable) {
	var argsJSON string
	if args != nil {
		json.Encode(args, &argsJSON)
	}

	params := &CallToolParams{
		Name:      op,
		Arguments: argsJSON,
	}

	c.client.Dispatch(context.Background(), string(MethodToolsCall), params)
}
