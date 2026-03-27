package mcp

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/json"
)

const (
	CtxKeyUserID    = "mcp.user_id"
	CtxKeySessionID = "mcp.session_id"
	CtxKeyAuthToken = "mcp.auth_token"
)

func (s *Server) HandleMessage(ctx *context.Context, message []byte) JSONRPCMessage {
	id := extractJSONString(message, "id")
	method := extractJSONString(message, "method")
	jsonrpc := extractJSONString(message, "jsonrpc")

	if jsonrpc != JSONRPC_VERSION {
		return createErrorResponse(id, INVALID_REQUEST, "Invalid JSON-RPC version")
	}

	if id == "" {
		var notification JSONRPCNotification
		notification.JSONRPC = JSONRPC_VERSION
		notification.Method = method
		s.handleNotification(ctx, notification)
		return nil
	}

	if s.auth != nil {
		token := ctx.Value(CtxKeyAuthToken)
		userID, err := s.auth.Authorize(token)
		if err != nil {
			return createErrorResponse(id, -32001, "Unauthorized")
		}
		ctx.Set(CtxKeyUserID, userID)
	}

	switch MCPMethod(method) {
	case MethodInitialize:
		var p initializeParams
		json.Decode(message, &p)
		result, reqErr := s.handleInitialize(ctx, id, p)
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(id, result)

	case MethodPing:
		result, reqErr := s.handlePing(ctx, id)
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(id, result)

	case MethodToolsList:
		var p PaginatedParams
		json.Decode(message, &p)
		result, reqErr := s.handleListTools(ctx, id, p)
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(id, result)

	case MethodToolsCall:
		var p callToolParams
		json.Decode(message, &p)
		result, reqErr := s.handleToolCall(ctx, id, p)
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(id, result)

	default:
		if method == "" {
			return createErrorResponse(id, PARSE_ERROR, "Failed to parse message")
		}
		return createErrorResponse(id, METHOD_NOT_FOUND, "Method not found: "+method)
	}
}

func extractJSONString(data []byte, key string) string {
	search := "\"" + key + "\":"
	start := -1
	for i := 0; i <= len(data)-len(search); i++ {
		match := true
		for j := 0; j < len(search); j++ {
			if data[i+j] != search[j] {
				match = false
				break
			}
		}
		if match {
			start = i + len(search)
			break
		}
	}
	if start == -1 {
		return ""
	}
	for start < len(data) && (data[start] == ' ' || data[start] == '\t' || data[start] == '\n' || data[start] == '\r') {
		start++
	}
	if start >= len(data) {
		return ""
	}
	if data[start] == '"' {
		start++
		end := start
		for end < len(data) && data[end] != '"' {
			if data[end] == '\\' && end+1 < len(data) {
				end += 2
				continue
			}
			end++
		}
		return string(data[start:end])
	} else {
		end := start
		for end < len(data) && data[end] != ',' && data[end] != '}' && data[end] != ' ' && data[end] != '\t' && data[end] != '\n' && data[end] != '\r' {
			end++
		}
		return string(data[start:end])
	}
}
