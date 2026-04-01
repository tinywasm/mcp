//go:build ignore

package mcp

import (
	"github.com/tinywasm/context"
)

// HTTPHeaderFunc is a function that extracts header entries from the given context
// and returns them as key-value pairs. This is typically used to add context values
// as HTTP headers in outgoing requests.
type HTTPHeaderFunc func(*context.Context) map[string]string
