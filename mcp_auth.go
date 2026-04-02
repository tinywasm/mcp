package mcp

import (
	"github.com/tinywasm/fmt"
)

type Authorizer interface {
	Authorize(token string) (userID string, err error)
	Can(userID, resource string, action byte) bool
}

func NewTokenAuthorizer(apiKey string) Authorizer {
	return &tokenAuthorizer{apiKey: apiKey}
}

type tokenAuthorizer struct {
	apiKey string
}

func (a *tokenAuthorizer) Authorize(token string) (string, error) {
	if constantTimeEqual(token, a.apiKey) {
		return "user", nil
	}
	return "", fmt.Err("mcp", "auth", "unauthorized")
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}

func (a *tokenAuthorizer) Can(userID, resource string, action byte) bool {
	return true
}

func OpenAuthorizer() Authorizer {
	return &openAuthorizer{}
}

type openAuthorizer struct{}

func (a *openAuthorizer) Authorize(token string) (string, error) {
	return "guest", nil
}

func (a *openAuthorizer) Can(userID, resource string, action byte) bool {
	return true
}
