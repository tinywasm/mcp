package mcp

import (
	"context"
)

// HTTPHeaderFunc is a function that extracts header entries from the given context
// and returns them as key-value pairs. This is typically used to add context values
// as HTTP headers in outgoing requests.
type HTTPHeaderFunc func(context.Context) map[string]string

// Interface for the transport layer.
type Interface interface {
	// Start the connection. Start should only be called once.
	Start(ctx context.Context) error

	// SendRequest sends a json RPC request and returns the response synchronously.
	SendRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error)

	// SendNotification sends a json RPC Notification to the
	SendNotification(ctx context.Context, notification JSONRPCNotification) error

	// SetNotificationHandler sets the handler for notifications.
	// Any notification before the handler is set will be discarded.
	SetNotificationHandler(handler func(notification JSONRPCNotification))

	// Close the connection.
	Close() error

	// GetSessionId returns the session ID of the
	GetSessionId() string
}

// RequestHandler defines a function that handles incoming requests from the
type RequestHandler func(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error)

// BidirectionalInterface extends Interface to support incoming requests from the
// This is used for features like sampling where the server can send requests to the
type BidirectionalInterface interface {
	Interface

	// SetRequestHandler sets the handler for incoming requests from the
	// The handler should process the request and return a response.
	SetRequestHandler(handler RequestHandler)
}

// HTTPConnection is a Transport that runs over HTTP and supports
// protocol version headers.
type HTTPConnection interface {
	Interface
	SetProtocolVersion(version string)
}
