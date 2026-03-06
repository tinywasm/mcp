package mcp

import (
	"errors"
	"fmt"
)

var (
	ErrUnsupported                = errors.New("not supported")
	ErrToolNotFound               = errors.New("tool not found")
	ErrNotificationNotInitialized = errors.New("notification channel not initialized")
	ErrNotificationChannelBlocked = errors.New("notification channel queue is full - client may not be processing notifications fast enough")
)

// NewError returns a new error that wraps the given error.
func NewError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("mcp error: %w", err)
}
