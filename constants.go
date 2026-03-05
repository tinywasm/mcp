package mcp

type contextKey int

const (
	// This const is used as key for context value lookup
	requestHeader contextKey = iota
)

// Common HTTP header constants used across server transports
const (
	HeaderKeySessionID       = "Mcp-Session-Id"
	HeaderKeyProtocolVersion = "Mcp-Protocol-Version"
	ContentTypeText          = "text"
	ContentTypeImage         = "image"
	ContentTypeAudio         = "audio"
	ContentTypeLink          = "resource_link"
	ContentTypeResource      = "resource"
	ElicitationModeForm      = "form"
	ElicitationModeURL       = "url"
)
