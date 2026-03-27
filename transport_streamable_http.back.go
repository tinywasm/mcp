//go:build !wasm

package mcp

import (
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/tinywasm/context"
	"github.com/tinywasm/fmt"
)

type StreamableHTTPCOption func(*StreamableHTTP)

func WithHTTPBasicClient(client *http.Client) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.httpClient = client
	}
}

func WithHTTPHeaders(headers []fmt.KeyValue) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.headers = headers
	}
}

type StreamableHTTP struct {
	serverURL           *url.URL
	httpClient          *http.Client
	headers             []fmt.KeyValue
	sessionID           atomic.Value // string
	protocolVersion     atomic.Value // string
	initialized         chan struct{}
	initializedOnce     sync.Once
	notificationHandler func(JSONRPCNotification)
	notifyMu            sync.RWMutex
	requestHandler      RequestHandler
	requestMu           sync.RWMutex
	closed              chan struct{}
	closeOnce           sync.Once
}

func NewStreamableHTTP(serverURL string, options ...StreamableHTTPCOption) (*StreamableHTTP, error) {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Err("mcp", "invalid URL")
	}
	smc := &StreamableHTTP{
		serverURL:   parsedURL,
		httpClient:  &http.Client{},
		closed:      make(chan struct{}),
		initialized: make(chan struct{}),
	}
	smc.sessionID.Store("")
	for _, opt := range options {
		if opt != nil {
			opt(smc)
		}
	}
	return smc, nil
}

func (c *StreamableHTTP) Start(ctx *context.Context) error {
	return nil
}

func (c *StreamableHTTP) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *StreamableHTTP) SetProtocolVersion(version string) {
	c.protocolVersion.Store(version)
}

func (c *StreamableHTTP) SendRequest(ctx *context.Context, request JSONRPCRequest) (*JSONRPCResponseStruct, error) {
	return nil, nil
}

func (c *StreamableHTTP) SendNotification(ctx *context.Context, notification JSONRPCNotification) error {
	return nil
}

func (c *StreamableHTTP) SetNotificationHandler(handler func(JSONRPCNotification)) {
	c.notifyMu.Lock()
	defer c.notifyMu.Unlock()
	c.notificationHandler = handler
}

func (c *StreamableHTTP) SetRequestHandler(handler RequestHandler) {
	c.requestMu.Lock()
	defer c.requestMu.Unlock()
	c.requestHandler = handler
}

func (c *StreamableHTTP) GetSessionId() string {
	return c.sessionID.Load().(string)
}
