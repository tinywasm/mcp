package mcp

// AllowAll is a helper that grants any permission. Use only for dev/tests.
func AllowAll(_, _, _ string) bool {
	return true
}
