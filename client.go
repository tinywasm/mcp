package mcp

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/fetch"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
)

type Client struct {
	endpoint string
}

func NewClient(baseURL string) *Client {
	return &Client{
		endpoint: fmt.Convert(baseURL).TrimSuffix("/").String() + "/mcp",
	}
}

func (c *Client) Call(ctx *context.Context, method string, params any, callback func([]byte, error)) {
	body := c.buildBody(method, params)
	if body == nil {
		if callback != nil {
			callback(nil, fmt.Err("mcp: failed to encode request"))
		}
		return
	}
	fetch.Post(c.endpoint).ContentTypeJSON().Body(body).Send(func(resp *fetch.Response, err error) {
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		if callback == nil {
			return
		}

		var envelope rpcResponse
		if err := json.Decode(resp.Body(), &envelope); err != nil {
			callback(nil, err)
			return
		}
		if len(envelope.Result) == 0 {
			callback(nil, nil)
			return
		}
		callback([]byte(envelope.Result), nil)
	})
}

func (c *Client) Dispatch(ctx *context.Context, method string, params any) {
	body := c.buildBody(method, params)
	if body == nil {
		return
	}
	fetch.Post(c.endpoint).ContentTypeJSON().Body(body).Send(func(*fetch.Response, error) {})
}

func (c *Client) buildBody(method string, params any) []byte {
	var paramsJSON string
	if params != nil {
		if f, ok := params.(fmt.Encodable); ok {
			if err := json.Encode(f, &paramsJSON); err != nil {
				return nil
			}
		}
	}
	req := rpcRequest{JSONRPC: "2.0", ID: "1", Method: method, Params: paramsJSON}
	var body []byte
	if err := json.Encode(&req, &body); err != nil {
		return nil
	}
	return body
}
