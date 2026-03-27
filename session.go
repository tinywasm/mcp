package mcp

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
)

type ClientSession interface {
	SessionID() string
	NotificationChannel() chan<- JSONRPCNotification
	Initialized() bool
	Initialize()
}

func ContextWithSessionID(ctx *context.Context, sessionID string) {
	ctx.Set(CtxKeySessionID, sessionID)
}

func SessionIDFromContext(ctx *context.Context) string {
	return ctx.Value(CtxKeySessionID)
}

func SendNotification(ctx *context.Context, session ClientSession, method string, params fmt.Fielder) error {
	if session == nil || !session.Initialized() {
		return fmt.Err("mcp", "session not initialized")
	}
	msg := JSONRPCNotification{
		JSONRPC: JSONRPC_VERSION,
		Method:  method,
	}
	select {
	case session.NotificationChannel() <- msg:
		return nil
	default:
		return fmt.Err("mcp", "notification channel blocked")
	}
}
