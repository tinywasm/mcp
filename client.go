package mcp

import (
	"github.com/tinywasm/model"
	"github.com/tinywasm/context"
	"github.com/tinywasm/fetch"
	"github.com/tinywasm/fmt"
	"github.com/tinywasm/json"
)

const (
	headerAuthorization = "Authorization"
	bearerPrefix        = "Bearer "
)

type Client struct {
	endpoint  string
	authToken string // when non-empty, sent as "Authorization: Bearer <token>"
}

// NewClient targets baseURL + "/mcp". authToken is sent as a Bearer token on
// every request; pass "" for open/unauthenticated daemons.
func NewClient(baseURL, authToken string) *Client {
	return &Client{
		endpoint:  fmt.Convert(baseURL).TrimSuffix("/").String() + "/mcp",
		authToken: authToken,
	}
}

// newPost builds the POST request for this client, attaching the Authorization
// header when a token is configured.
func (c *Client) newPost(body []byte) *fetch.Request {
	r := fetch.Post(c.endpoint).ContentTypeJSON().Body(body)
	if c.authToken != "" {
		r = r.Header(headerAuthorization, bearerPrefix+c.authToken)
	}
	return r
}

func (c *Client) Call(ctx *context.Context, method string, params any, callback func([]byte, error)) {
	body := c.buildBody(method, params)
	if body == nil {
		if callback != nil {
			callback(nil, fmt.Err("mcp: failed to encode request"))
		}
		return
	}
	c.newPost(body).Send(func(resp *fetch.Response, err error) {
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
		if len(envelope.Error) != 0 {
			callback(nil, fmt.Err("mcp: "+string(envelope.Error)))
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
	c.newPost(body).Send(func(*fetch.Response, error) {})
}

func (c *Client) buildBody(method string, params any) []byte {
	var paramsJSON string
	if params != nil {
		if f, ok := params.(model.Encodable); ok {
			if err := json.Encode(f, &paramsJSON); err != nil {
				return nil
			}
		}
	}
	req := rpcRequest{Jsonrpc: "2.0", Id: RequestId("\"1\""), Method: method, Params: paramsJSON}
	var body []byte
	if err := json.Encode(&req, &body); err != nil {
		return nil
	}
	return body
}
