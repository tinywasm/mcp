//go:build wasm

package mcp

type SSEPublisher interface {
	Publish(data []byte, channel string)
}
