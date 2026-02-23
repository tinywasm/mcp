package mcp

import (
	"errors"
	"fmt"
)

// NewOAuthStreamableHttpClient creates a new streamable-http-based MCP client with OAuth support.
// Returns an error if the URL is invalid.
func NewOAuthStreamableHttpClient(baseURL string, oauthConfig OAuthConfig, options ...StreamableHTTPCOption) (*Client, error) {
	// Add OAuth option to the list of options
	options = append(options, WithHTTPOAuth(oauthConfig))

	trans, err := NewStreamableHTTP(baseURL, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP transport: %w", err)
	}
	return NewClient(trans), nil
}

// NewOAuthSSEClient creates a new streamable-http-based MCP client with OAuth support.
// Returns an error if the URL is invalid.
func NewOAuthSSEClient(baseURL string, oauthConfig OAuthConfig, options ...SSEOption) (*Client, error) {
	// Add OAuth option to the list of options
	options = append(options, WithSSEOAuth(oauthConfig))

	trans, err := NewSSE(baseURL, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE transport: %w", err)
	}
	return NewClient(trans), nil
}

// WithOAuth enables OAuth authentication for the SSE client.
func WithOAuth(config OAuthConfig) SSEOption {
	return WithSSEOAuth(config)
}

// NewOAuthSSE creates a new SSE-based MCP client with the given base URL and OAuth configuration.
func NewOAuthSSE(serverURL string, options ...SSEOption) (*Client, error) {
	sseTransport, err := NewSSE(serverURL, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE transport: %w", err)
	}
	return NewClient(sseTransport), nil
}

// IsOAuthAuthorizationRequiredError checks if an error is an OAuthAuthorizationRequiredError
func IsOAuthAuthorizationRequiredError(err error) bool {
	var target *OAuthAuthorizationRequiredError
	return errors.As(err, &target)
}

// GetOAuthHandler extracts the OAuthHandler from an OAuthAuthorizationRequiredError
func GetOAuthHandler(err error) *OAuthHandler {
	var oauthErr *OAuthAuthorizationRequiredError
	if errors.As(err, &oauthErr) {
		return oauthErr.Handler
	}
	return nil
}
