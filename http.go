package mcp

import (
	"context"
	"fmt"
	"net/http"
)

// HTTPContextFunc is a function that takes an existing context and the current
// request and returns a potentially modified context based on the request
// content. This can be used to inject context values from headers, for example.
type HTTPContextFunc func(ctx context.Context, r *http.Request) context.Context

// NewStreamableHttpClient is a convenience method that creates a new streamable-http-based MCP client
// with the given base URL. Returns an error if the URL is invalid.
func NewStreamableHttpClient(baseURL string, options ...StreamableHTTPCOption) (*Client, error) {
	trans, err := NewStreamableHTTP(baseURL, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE transport: %w", err)
	}
	clientOptions := make([]ClientOption, 0)
	sessionID := trans.GetSessionId()
	if sessionID != "" {
		clientOptions = append(clientOptions, WithInitializedSession())
	}
	return NewClient(trans, clientOptions...), nil
}
