package mcp_test

import (
	"encoding/json"
	"testing"

	"github.com/tinywasm/mcp"
	"github.com/tinywasm/mcp/internal/testutils/assert"
	"github.com/tinywasm/mcp/internal/testutils/require"
)

func TestPromptMessageWithMultipleContent(t *testing.T) {
	// This test verifies that we can create a PromptMessage with multiple content items
	// currently this is not supported by the type system directly as NewPromptMessage takes single Content

	// We want to achieve something like this:
	/*
	msg := mcp.PromptMessage{
		Role: mcp.RoleUser,
		Content: []mcp.Content{
			mcp.NewTextContent("Here is an image:"),
			mcp.NewImageContent("base64data", "image/png"),
		},
	}
	*/

	// But PromptMessage.Content is mcp.Content (interface), which []mcp.Content does not implement.
	// So we can't do the above.

	// If we use 'any' for Content, we could do it.

	// Let's try to manually create JSON and parse it using ParseGetPromptResult
	// to see if it fails (it should fail currently).

	jsonData := `{
		"description": "Test prompt",
		"messages": [
			{
				"role": "user",
				"content": [
					{
						"type": "text",
						"text": "Hello"
					},
					{
						"type": "image",
						"data": "base64",
						"mimeType": "image/png"
					}
				]
			}
		]
	}`

	raw := json.RawMessage(jsonData)
	result, err := mcp.ParseGetPromptResult(&raw)

	// We expect this to SUCCESS now
	require.NoError(t, err)
    assert.Equal(t, "Test prompt", result.Description)
    require.Len(t, result.Messages, 1)
    msg := result.Messages[0]
    assert.Equal(t, mcp.RoleUser, msg.Role)

    // Check content
    contents, ok := msg.Content.([]mcp.Content)
    require.True(t, ok)
    require.Len(t, contents, 2)

    text, ok := contents[0].(mcp.TextContent)
    require.True(t, ok)
    assert.Equal(t, "Hello", text.Text)

    img, ok := contents[1].(mcp.ImageContent)
    require.True(t, ok)
    assert.Equal(t, "base64", img.Data)
}
