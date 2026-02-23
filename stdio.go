package mcp

import (
	"context"
	"fmt"
	"io"
)

// NewStdioMCPClient creates a new stdio-based MCP client that communicates with a subprocess.
// It launches the specified command with given arguments and sets up stdin/stdout pipes for communication.
// Returns an error if the subprocess cannot be started or the pipes cannot be created.
//
// NOTICE: NewStdioMCPClient will start the connection automatically.
// This is for backward compatibility.
func NewStdioMCPClient(
	command string,
	env []string,
	args ...string,
) (*Client, error) {
	return NewStdioMCPClientWithOptions(command, env, args)
}

// NewStdioMCPClientWithOptions creates a new stdio-based MCP client that communicates with a subprocess.
// It launches the specified command with given arguments and sets up stdin/stdout pipes for communication.
// Optional configuration functions can be provided to customize the transport before it starts,
// such as setting a custom command function.
//
// NOTICE: NewStdioMCPClientWithOptions automatically starts the underlying 
// This is for backward compatibility.
func NewStdioMCPClientWithOptions(
	command string,
	env []string,
	args []string,
	opts ...StdioOption,
) (*Client, error) {
	stdioTransport := NewStdioWithOptions(command, env, args, opts...)

	if err := stdioTransport.Start(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to start stdio transport: %w", err)
	}

	return NewClient(stdioTransport), nil
}

// GetStderr returns a reader for the stderr output of the subprocess.
// This can be used to capture error messages or logs from the subprocess.
func GetStderr(c *Client) (io.Reader, bool) {
	t := c.GetTransport()

	stdio, ok := t.(*Stdio)
	if !ok {
		return nil, false
	}

	return stdio.Stderr(), true
}
