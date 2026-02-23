package mcp

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type InProcessSession struct {
	sessionID          string
	notifications      chan JSONRPCNotification
	initialized        atomic.Bool
	loggingLevel       atomic.Value
	clientInfo         atomic.Value
	clientCapabilities atomic.Value
	samplingHandler    SamplingHandler
	elicitationHandler ElicitationHandler
	rootsHandler       RootsHandler
	mu                 sync.RWMutex
}

func NewInProcessSession(sessionID string) *InProcessSession {
	return &InProcessSession{
		sessionID:     sessionID,
		notifications: make(chan JSONRPCNotification, 100),
	}
}

// NewInProcessSessionWithHandlers is no longer needed as handlers are managed by the transport.
// func NewInProcessSessionWithHandlers(sessionID string, samplingHandler SamplingHandler, elicitationHandler ElicitationHandler, rootsHandler RootsHandler) *InProcessSession {
// 	return &InProcessSession{
// 		sessionID:          sessionID,
// 		notifications:      make(chan JSONRPCNotification, 100),
// 		samplingHandler:    samplingHandler,
// 		elicitationHandler: elicitationHandler,
// 		rootsHandler:       rootsHandler,
// 	}
// }

func (s *InProcessSession) SessionID() string {
	return s.sessionID
}

func (s *InProcessSession) NotificationChannel() chan<- JSONRPCNotification {
	return s.notifications
}

func (s *InProcessSession) Initialize() {
	s.loggingLevel.Store(LoggingLevelError)
	s.initialized.Store(true)
}

func (s *InProcessSession) Initialized() bool {
	return s.initialized.Load()
}

func (s *InProcessSession) GetClientInfo() Implementation {
	if value := s.clientInfo.Load(); value != nil {
		if clientInfo, ok := value.(Implementation); ok {
			return clientInfo
		}
	}
	return Implementation{}
}

func (s *InProcessSession) SetClientInfo(clientInfo Implementation) {
	s.clientInfo.Store(clientInfo)
}

func (s *InProcessSession) GetClientCapabilities() ClientCapabilities {
	if value := s.clientCapabilities.Load(); value != nil {
		if clientCapabilities, ok := value.(ClientCapabilities); ok {
			return clientCapabilities
		}
	}
	return ClientCapabilities{}
}

func (s *InProcessSession) SetClientCapabilities(clientCapabilities ClientCapabilities) {
	s.clientCapabilities.Store(clientCapabilities)
}

func (s *InProcessSession) SetLogLevel(level LoggingLevel) {
	s.loggingLevel.Store(level)
}

func (s *InProcessSession) GetLogLevel() LoggingLevel {
	level := s.loggingLevel.Load()
	if level == nil {
		return LoggingLevelError
	}
	return level.(LoggingLevel)
}

func (s *InProcessSession) RequestSampling(ctx context.Context, request CreateMessageRequest) (*CreateMessageResult, error) {
	s.mu.RLock()
	handler := s.samplingHandler
	s.mu.RUnlock()

	if handler == nil {
		return nil, fmt.Errorf("no sampling handler available")
	}

	return handler.CreateMessage(ctx, request)
}

func (s *InProcessSession) RequestElicitation(ctx context.Context, request ElicitationRequest) (*ElicitationResult, error) {
	s.mu.RLock()
	handler := s.elicitationHandler
	s.mu.RUnlock()

	if handler == nil {
		return nil, fmt.Errorf("no elicitation handler available")
	}

	return handler.Elicit(ctx, request)
}

// ListRoots sends a list roots request to the client and waits for the response.
// Returns an error if no roots handler is available.
func (s *InProcessSession) ListRoots(ctx context.Context, request ListRootsRequest) (*ListRootsResult, error) {
	s.mu.RLock()
	handler := s.rootsHandler
	s.mu.RUnlock()

	if handler == nil {
		return nil, fmt.Errorf("no roots handler available")
	}

	return handler.ListRoots(ctx, request)
}

// GenerateInProcessSessionID generates a unique session ID for inprocess clients
func GenerateInProcessSessionID() string {
	return fmt.Sprintf("inprocess-%d", time.Now().UnixNano())
}

// Ensure interface compliance
var (
	_ ClientSession          = (*InProcessSession)(nil)
	_ SessionWithLogging     = (*InProcessSession)(nil)
	_ SessionWithClientInfo  = (*InProcessSession)(nil)
	_ SessionWithSampling    = (*InProcessSession)(nil)
	_ SessionWithElicitation = (*InProcessSession)(nil)
	_ SessionWithRoots       = (*InProcessSession)(nil)
)
