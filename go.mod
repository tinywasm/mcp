module github.com/tinywasm/mcp

go 1.25.2

require (
	github.com/tinywasm/base64 v0.0.5
	github.com/tinywasm/context v0.0.18
	github.com/tinywasm/fetch v0.1.24
	github.com/tinywasm/fmt v0.25.7
	github.com/tinywasm/json v0.5.17
	github.com/tinywasm/model v0.1.4
	github.com/tinywasm/router v0.1.22
	github.com/tinywasm/unixid v0.2.24
)

require github.com/tinywasm/time v0.5.0

// Local dev: Encode/Decode (standard, padded base64) were just added and
// aren't in the v0.0.3 tag yet.
