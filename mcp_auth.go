package mcp

import (
	"context"
	"net/http"
	"strings"
)

// Authorizer is the DI contract for authentication and authorization.
// tinywasm/user.Module implements this structurally — no import of mcp needed
// in user (Go implicit interface satisfaction).
//
// Design: context-based identity — mcp never knows the concrete user type.
// The implementor injects its identity under its own private context key.
type Authorizer interface {
	// InjectIdentity reads the request (Bearer token, cookie, etc.) and
	// injects the authenticated identity into ctx. Returns ctx unchanged
	// if auth fails — CanExecute will deny based on missing identity.
	InjectIdentity(ctx context.Context, r *http.Request) context.Context

	// CanExecute checks if the identity in ctx has permission to perform
	// action ('c','r','u','d') on resource. Returns true if auth passes
	// or if the tool has no Resource constraint (empty string).
	CanExecute(ctx context.Context, resource string, action byte) bool
}

// denyAllAuthorizer is the private default — always denies every request.
//
// Secure-by-default: forgetting to call SetAuth() causes all tool calls to be
// rejected immediately (visible failure), rather than silently leaving the
// endpoint open. The caller MUST explicitly choose an Authorizer.
type denyAllAuthorizer struct{}

func (denyAllAuthorizer) InjectIdentity(ctx context.Context, _ *http.Request) context.Context {
	return ctx
}
func (denyAllAuthorizer) CanExecute(_ context.Context, _ string, _ byte) bool { return false }

// OpenAuthorizer returns an Authorizer that grants full access to every request.
// Use this as an explicit, conscious opt-in for local / trusted environments
// where network-level security is sufficient and auth adds unnecessary friction.
// Calling SetAuth(mcp.OpenAuthorizer()) makes the decision visible in code review.
func OpenAuthorizer() Authorizer { return openAuthorizer{} }

type openAuthorizer struct{}

func (openAuthorizer) InjectIdentity(ctx context.Context, _ *http.Request) context.Context {
	return ctx
}
func (openAuthorizer) CanExecute(_ context.Context, _ string, _ byte) bool { return true }

// tokenKey is the context key for tokenAuthorizer identity.
type tokenKey struct{}

// tokenAuthorizer validates a static Bearer token.
// Use when a lightweight auth is needed without a full user.Module
// (e.g., local daemon securing its MCP endpoint for IDE clients).
type tokenAuthorizer struct{ token string }

// NewTokenAuthorizer returns an Authorizer that accepts only the given static token.
// The token is validated against the "Authorization: Bearer <token>" request header.
// Empty token is always rejected — use OpenAuthorizer() for no-auth mode.
func NewTokenAuthorizer(token string) Authorizer {
	return &tokenAuthorizer{token: token}
}

func (a *tokenAuthorizer) InjectIdentity(ctx context.Context, r *http.Request) context.Context {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, prefix) && auth[len(prefix):] == a.token && a.token != "" {
		return context.WithValue(ctx, tokenKey{}, true)
	}
	return ctx // identity not injected → CanExecute will deny
}

func (a *tokenAuthorizer) CanExecute(ctx context.Context, _ string, _ byte) bool {
	v, _ := ctx.Value(tokenKey{}).(bool)
	return v
}
