package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
)

type StreamableHTTPCOption func(*StreamableHTTP)

func WithHTTPBasicClient(client *http.Client) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.httpClient = client
	}
}

func WithHTTPHeaders(headers map[string]string) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.headers = headers
	}
}

func WithHTTPLogger(logger Logger) StreamableHTTPCOption {
	return func(sc *StreamableHTTP) {
		sc.logger = logger
	}
}

type StreamableHTTP struct {
	serverURL           *url.URL
	httpClient          *http.Client
	headers             map[string]string
	logger              Logger
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
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	smc := &StreamableHTTP{
		serverURL:   parsedURL,
		httpClient:  &http.Client{},
		headers:     make(map[string]string),
		closed:      make(chan struct{}),
		logger:      DefaultLogger(),
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

func (c *StreamableHTTP) Start(ctx context.Context) error {
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

func (c *StreamableHTTP) SendRequest(ctx context.Context, request JSONRPCRequest) (*JSONRPCResponse, error) {
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	resp, err := c.sendHTTP(ctx, http.MethodPost, bytes.NewReader(requestBody), "application/json", request.Header)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	if request.Method == string(MethodInitialize) {
		if sessionID := resp.Header.Get(HeaderKeySessionID); sessionID != "" {
			c.sessionID.Store(sessionID)
		}
		c.initializedOnce.Do(func() {
			close(c.initialized)
		})
	}
	var response JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &response, nil
}

func (c *StreamableHTTP) sendHTTP(ctx context.Context, method string, body io.Reader, acceptType string, header http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.serverURL.String(), body)
	if err != nil {
		return nil, err
	}
	if header != nil {
		req.Header = header
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", acceptType)
	if sid := c.sessionID.Load().(string); sid != "" {
		req.Header.Set(HeaderKeySessionID, sid)
	}
	return c.httpClient.Do(req)
}

func (c *StreamableHTTP) SendNotification(ctx context.Context, notification JSONRPCNotification) error {
	requestBody, _ := json.Marshal(notification)
	resp, err := c.sendHTTP(ctx, http.MethodPost, bytes.NewReader(requestBody), "application/json", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
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
