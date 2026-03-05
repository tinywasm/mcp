package mcp

import (
	"errors"
)

var (
	// Common server errors
	ErrUnsupported      = errors.New("not supported")
	ErrToolNotFound     = errors.New("tool not found")

	// Session-related errors
	ErrSessionNotFound                        = errors.New("session not found")
	ErrSessionExists                          = errors.New("session already exists")
	ErrSessionNotInitialized                  = errors.New("session not properly initialized")
	ErrSessionDoesNotSupportTools             = errors.New("session does not support per-session tools")

	// Notification-related errors
	ErrNotificationNotInitialized = errors.New("notification channel not initialized")
	ErrNotificationChannelBlocked = errors.New("notification channel queue is full - client may not be processing notifications fast enough")
)
