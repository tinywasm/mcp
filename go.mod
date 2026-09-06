module webtyp.com/mcp

go 1.25.2

require (
	webtyp.com/base64 v0.0.5
	webtyp.com/context v0.0.18
	webtyp.com/fetch v0.1.24
	webtyp.com/fmt v0.25.7
	webtyp.com/json v0.5.23
	webtyp.com/model v0.1.7
	webtyp.com/router v0.1.30
	webtyp.com/unixid v0.2.24
)

require webtyp.com/time v0.5.4

// Local dev: Encode/Decode (standard, padded base64) were just added and
// aren't in the v0.0.3 tag yet.
