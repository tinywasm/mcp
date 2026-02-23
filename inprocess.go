package mcp

import (
	"context"
)

// NewInProcessClient connect directly to a mcp server object in the same process
func NewInProcessClient(server *MCPServer) (*Client, error) {
	inProcessTransport := NewInProcessTransport(server)
	return NewClient(inProcessTransport), nil
}

// NewInProcessClientWithSamplingHandler creates an in-process client with sampling support
func NewInProcessClientWithSamplingHandler(server *MCPServer, handler SamplingHandler) (*Client, error) {
	// Create a wrapper that implements SamplingHandler
	serverHandler := &inProcessSamplingHandlerWrapper{handler: handler}

	inProcessTransport := NewInProcessTransportWithOptions(server,
		WithInProcessSamplingHandler(serverHandler))

	client := NewClient(inProcessTransport, WithSamplingHandler(handler))

	return client, nil
}

// inProcessSamplingHandlerWrapper wraps SamplingHandler to implement SamplingHandler
type inProcessSamplingHandlerWrapper struct {
	handler SamplingHandler
}

func (w *inProcessSamplingHandlerWrapper) CreateMessage(ctx context.Context, request CreateMessageRequest) (*CreateMessageResult, error) {
	return w.handler.CreateMessage(ctx, request)
}
