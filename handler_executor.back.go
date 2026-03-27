//go:build !wasm

package mcp

import (
	"encoding/base64"
)

type BinaryData struct {
	MimeType string
	Data     []byte
}

func Image(text, dataBase64, mimeType string) *Result {
	// Implementation stub for minimal API
	return &Result{Content: "[]"}
}

func ImageFromBytes(text string, data []byte, mimeType string) *Result {
	b64 := base64.StdEncoding.EncodeToString(data)
	return Image(text, b64, mimeType)
}
