package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/tinywasm/fetch"
	"github.com/tinywasm/fmt"
)

// Client is a lightweight stateless JSON-RPC 2.0 client for tinywasm/mcp endpoints.
// Thread-safe (no mutable state after construction).
// Compatible with stdlib and WASM (browser) environments.
type Client struct {
	endpoint string // always points to /mcp
	apiKey   string // optional Bearer token; empty = no auth header
}

// NewClient creates a Client targeting baseURL/mcp.
// baseURL: e.g. "http://localhost:3030" — the /mcp path is appended automatically.
// apiKey: Bearer token for secured endpoints; empty = open/local daemon.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		endpoint: strings.TrimSuffix(baseURL, "/") + "/mcp",
		apiKey:   apiKey,
	}
}

// Call sends a stateless JSON-RPC 2.0 POST and delivers the raw result bytes via callback.
// callback(nil, nil) when response has no result field.
// Uses tinywasm/fetch (async, WASM+stdlib compatible).
func (c *Client) Call(ctx context.Context, method string, params any, callback func([]byte, error)) {
	body := c.buildBody(method, params)
	if body == nil {
		if callback != nil {
			callback(nil, fmt.Err("mcp: failed to encode request"))
		}
		return
	}
	r := fetch.Post(c.endpoint).ContentTypeJSON().Body(body)
	if c.apiKey != "" {
		r = r.Header("Authorization", "Bearer "+c.apiKey)
	}
	r.Send(func(resp *fetch.Response, err error) {
		if err != nil {
			if callback != nil {
				callback(nil, err)
			}
			return
		}
		if callback == nil {
			return
		}

		// Use stdlib json for envelope decoding because tinywasm/json
		// doesn't support raw JSON capturing into strings yet.
		var envelope struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(resp.Body(), &envelope); err != nil {
			callback(nil, err)
			return
		}
		if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
			callback(nil, nil)
			return
		}
		callback([]byte(envelope.Result), nil)
	})
}

// Dispatch sends a JSON-RPC 2.0 POST and ignores the response (fire-and-forget).
// Used for tinywasm/action calls where no return value is needed.
func (c *Client) Dispatch(ctx context.Context, method string, params any) {
	body := c.buildBody(method, params)
	if body == nil {
		return
	}
	r := fetch.Post(c.endpoint).ContentTypeJSON().Body(body)
	if c.apiKey != "" {
		r = r.Header("Authorization", "Bearer "+c.apiKey)
	}
	r.Send(func(*fetch.Response, error) {}) // ignore response
}

func (c *Client) buildBody(method string, params any) []byte {
	// Use stdlib json for request building because tinywasm/json
	// doesn't support arbitrary map/struct Params well yet.
	req := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil
	}
	return body
}
