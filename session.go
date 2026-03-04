package mcp

import (
	"context"
	"maps"
)

// ClientSession represents an active session that can be used by MCPServer to interact with
type ClientSession interface {
	// Initialize marks session as fully initialized and ready for notifications
	Initialize()
	// Initialized returns if session is ready to accept notifications
	Initialized() bool
	// NotificationChannel provides a channel suitable for sending notifications to
	NotificationChannel() chan<- JSONRPCNotification
	// SessionID is a unique identifier used to track user session.
	SessionID() string
}

// SessionWithTools is an extension of ClientSession that can store session-specific tool data
type SessionWithTools interface {
	ClientSession
	// GetSessionTools returns the tools specific to this session, if any
	// This method must be thread-safe for concurrent access
	GetSessionTools() map[string]ServerTool
	// SetSessionTools sets tools specific to this session
	// This method must be thread-safe for concurrent access
	SetSessionTools(tools map[string]ServerTool)
}

// SessionWithClientInfo is an extension of ClientSession that can store client info
type SessionWithClientInfo interface {
	ClientSession
	// GetClientInfo returns the client information for this session
	GetClientInfo() Implementation
	// SetClientInfo sets the client information for this session
	SetClientInfo(clientInfo Implementation)
	// GetClientCapabilities returns the client capabilities for this session
	GetClientCapabilities() ClientCapabilities
	// SetClientCapabilities sets the client capabilities for this session
	SetClientCapabilities(clientCapabilities ClientCapabilities)
}

// SessionWithStreamableHTTPConfig extends ClientSession to support streamable HTTP transport configurations
type SessionWithStreamableHTTPConfig interface {
	ClientSession
	// UpgradeToSSEWhenReceiveNotification upgrades the client-server communication to SSE stream when the server
	// sends notifications to the client
	UpgradeToSSEWhenReceiveNotification()
}

// clientSessionKey is the context key for storing current client notification channel.
type clientSessionKey struct{}

// ClientSessionFromContext retrieves current client notification context from context.
func ClientSessionFromContext(ctx context.Context) ClientSession {
	if session, ok := ctx.Value(clientSessionKey{}).(ClientSession); ok {
		return session
	}
	return nil
}

// WithContext sets the current client session and returns the provided context
func (s *MCPServer) WithContext(
	ctx context.Context,
	session ClientSession,
) context.Context {
	return context.WithValue(ctx, clientSessionKey{}, session)
}

// RegisterSession saves session that should be notified in case if some server attributes changed.
func (s *MCPServer) RegisterSession(
	ctx context.Context,
	session ClientSession,
) error {
	sessionID := session.SessionID()
	if _, exists := s.sessions.LoadOrStore(sessionID, session); exists {
		return ErrSessionExists
	}
	// hooks logic removed for tools-only lean mcp
	return nil
}

func (s *MCPServer) sendNotificationToAllClients(notification JSONRPCNotification) {
	s.sessions.Range(func(k, v any) bool {
		if session, ok := v.(ClientSession); ok && session.Initialized() {
			if sessionWithStreamableHTTPConfig, ok := session.(SessionWithStreamableHTTPConfig); ok {
				sessionWithStreamableHTTPConfig.UpgradeToSSEWhenReceiveNotification()
			}
			select {
			case session.NotificationChannel() <- notification:
				// Successfully sent notification
			default:
				// Channel is blocked
			}
		}
		return true
	})
}

func (s *MCPServer) sendNotificationToSpecificClient(session ClientSession, notification JSONRPCNotification) error {
	// upgrades the client-server communication to SSE stream when the server sends notifications to the client
	if sessionWithStreamableHTTPConfig, ok := session.(SessionWithStreamableHTTPConfig); ok {
		sessionWithStreamableHTTPConfig.UpgradeToSSEWhenReceiveNotification()
	}
	select {
	case session.NotificationChannel() <- notification:
		return nil
	default:
		return ErrNotificationChannelBlocked
	}
}

// UnregisterSession removes from storage session that is shut down.
func (s *MCPServer) UnregisterSession(
	ctx context.Context,
	sessionID string,
) {
	s.sessions.LoadAndDelete(sessionID)
}

// SendNotificationToAllClients sends a notification to all the currently active clients.
func (s *MCPServer) SendNotificationToAllClients(
	method string,
	params map[string]any,
) {
	notification := JSONRPCNotification{
		JSONRPC: JSONRPC_VERSION,
		Notification: Notification{
			Method: method,
			Params: NotificationParams{
				AdditionalFields: params,
			},
		},
	}
	s.sendNotificationToAllClients(notification)
}

// SendNotificationToClient sends a notification to the current client
func (s *MCPServer) sendNotificationCore(
	ctx context.Context,
	session ClientSession,
	notification JSONRPCNotification,
) error {
	// upgrades the client-server communication to SSE stream when the server sends notifications to the client
	if sessionWithStreamableHTTPConfig, ok := session.(SessionWithStreamableHTTPConfig); ok {
		sessionWithStreamableHTTPConfig.UpgradeToSSEWhenReceiveNotification()
	}
	select {
	case session.NotificationChannel() <- notification:
		return nil
	default:
		return ErrNotificationChannelBlocked
	}
}

// SendNotificationToClient sends a notification to the current client
func (s *MCPServer) SendNotificationToClient(
	ctx context.Context,
	method string,
	params map[string]any,
) error {
	session := ClientSessionFromContext(ctx)
	if session == nil || !session.Initialized() {
		return ErrNotificationNotInitialized
	}
	notification := JSONRPCNotification{
		JSONRPC: JSONRPC_VERSION,
		Notification: Notification{
			Method: method,
			Params: NotificationParams{
				AdditionalFields: params,
			},
		},
	}
	return s.sendNotificationCore(ctx, session, notification)
}

// SendNotificationToSpecificClient sends a notification to a specific client by session ID
func (s *MCPServer) SendNotificationToSpecificClient(
	sessionID string,
	method string,
	params map[string]any,
) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}
	session, ok := sessionValue.(ClientSession)
	if !ok || !session.Initialized() {
		return ErrSessionNotInitialized
	}
	notification := JSONRPCNotification{
		JSONRPC: JSONRPC_VERSION,
		Notification: Notification{
			Method: method,
			Params: NotificationParams{
				AdditionalFields: params,
			},
		},
	}
	return s.sendNotificationToSpecificClient(session, notification)
}

// AddSessionTool adds a tool for a specific session
func (s *MCPServer) AddSessionTool(sessionID string, tool Tool, handler ToolHandlerFunc) error {
	return s.AddSessionTools(sessionID, ServerTool{Tool: tool, Handler: handler})
}

// AddSessionTools adds tools for a specific session
func (s *MCPServer) AddSessionTools(sessionID string, tools ...ServerTool) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	session, ok := sessionValue.(SessionWithTools)
	if !ok {
		return ErrSessionDoesNotSupportTools
	}

	s.implicitlyRegisterToolCapabilities()

	// Get existing tools (this should return a thread-safe copy)
	sessionTools := session.GetSessionTools()

	// Create a new map to avoid concurrent modification issues
	newSessionTools := make(map[string]ServerTool, len(sessionTools)+len(tools))

	// Copy existing tools
	maps.Copy(newSessionTools, sessionTools)

	// Add new tools
	for _, tool := range tools {
		newSessionTools[tool.Tool.Name] = tool
	}

	// Set the tools (this should be thread-safe)
	session.SetSessionTools(newSessionTools)

	if session.Initialized() && s.capabilities.tools != nil && s.capabilities.tools.listChanged {
		// Send notification only to this session
		if err := s.SendNotificationToSpecificClient(sessionID, "notifications/tools/list_changed", nil); err != nil {
			// Log the error but don't fail the operation
			// The tools were successfully added, but notification failed
		}
	}

	return nil
}

// DeleteSessionTools removes tools from a specific session
func (s *MCPServer) DeleteSessionTools(sessionID string, names ...string) error {
	sessionValue, ok := s.sessions.Load(sessionID)
	if !ok {
		return ErrSessionNotFound
	}

	session, ok := sessionValue.(SessionWithTools)
	if !ok {
		return ErrSessionDoesNotSupportTools
	}

	// Get existing tools (this should return a thread-safe copy)
	sessionTools := session.GetSessionTools()
	if sessionTools == nil {
		return nil
	}

	// Create a new map to avoid concurrent modification issues
	newSessionTools := make(map[string]ServerTool, len(sessionTools))

	// Copy existing tools except those being deleted
	maps.Copy(newSessionTools, sessionTools)

	// Remove specified tools
	for _, name := range names {
		delete(newSessionTools, name)
	}

	// Set the tools (this should be thread-safe)
	session.SetSessionTools(newSessionTools)

	if session.Initialized() && s.capabilities.tools != nil && s.capabilities.tools.listChanged {
		// Send notification only to this session
		if err := s.SendNotificationToSpecificClient(sessionID, "notifications/tools/list_changed", nil); err != nil {
			// Log the error but don't fail the operation
			// The tools were successfully deleted, but notification failed
		}
	}

	return nil
}
