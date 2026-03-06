package mcp

import "context"

// ClientSession is the protocol contract for an active client connection.
// It is implemented by the consumer (e.g. tinywasm/user) and injected via context.
type ClientSession interface {
	// SessionID is a unique identifier for this connection.
	SessionID() string
	// NotificationChannel is the channel the server writes outbound notifications to.
	NotificationChannel() chan<- JSONRPCNotification
	// Initialized reports whether the MCP handshake is complete.
	Initialized() bool
	// Initialize marks the session as ready to receive notifications.
	Initialize()
}

// SessionWithStreamableHTTPConfig is an optional extension for streamable HTTP transport.
type SessionWithStreamableHTTPConfig interface {
	ClientSession
	UpgradeToSSEWhenReceiveNotification()
}

type clientSessionKey struct{}

// ContextWithSession stores a ClientSession in a context.
// Called by the consumer before invoking HandleMessage.
func ContextWithSession(ctx context.Context, session ClientSession) context.Context {
	return context.WithValue(ctx, clientSessionKey{}, session)
}

// SessionFromContext retrieves the ClientSession from a context.
func SessionFromContext(ctx context.Context) ClientSession {
	session, _ := ctx.Value(clientSessionKey{}).(ClientSession)
	return session
}

// SendNotification sends a notification to the session currently in ctx.
// Returns ErrNotificationNotInitialized if no initialized session is present.
func SendNotification(ctx context.Context, method string, params map[string]any) error {
	session := SessionFromContext(ctx)
	if session == nil || !session.Initialized() {
		return ErrNotificationNotInitialized
	}
	if sh, ok := session.(SessionWithStreamableHTTPConfig); ok {
		sh.UpgradeToSSEWhenReceiveNotification()
	}
	select {
	case session.NotificationChannel() <- JSONRPCNotification{
		JSONRPC: JSONRPC_VERSION,
		Notification: Notification{
			Method: method,
			Params: NotificationParams{AdditionalFields: params},
		},
	}:
		return nil
	default:
		return ErrNotificationChannelBlocked
	}
}
