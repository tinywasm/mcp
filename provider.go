package mcp

// Loggable is implemented by handlers that support log injection.
type Loggable interface {
	Name() string
	SetLog(logger func(message ...any))
	GetLog() func(message ...any)
}
