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

	if len(message) == 0 {
		return createErrorResponse("", PARSE_ERROR, "Empty message")
	}

	if jsonrpc == "" && method == "" {
		return createErrorResponse(id, PARSE_ERROR, "Invalid JSON-RPC message")
	}

	if jsonrpc != JSONRPC_VERSION {
		return createErrorResponse(id, INVALID_REQUEST, "Invalid JSON-RPC version: "+jsonrpc)
	}

	if id == "" {
		var notification JSONRPCNotification
		notification.JSONRPC = JSONRPC_VERSION
		notification.Method = method
		s.handleNotification(ctx, notification)
		return nil
	}

	token := ctx.Value(CtxKeyAuthToken)
	userID, err := s.auth.Authorize(token)
	if err != nil {
		return createErrorResponse(id, -32001, "Unauthorized")
	}
	if userID == "" {
		return createErrorResponse(id, -32001, "Unauthorized: empty user identity")
	}
	ctx.Set(CtxKeyUserID, userID)

	switch MCPMethod(method) {
	case MethodInitialize:
		raw := ExtractJSONValue(message, "params")
		var p initializeParams
		if err := json.Decode(raw, &p); err != nil {
			return createErrorResponse(id, INVALID_PARAMS, "Invalid params: "+err.Error())
		}
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
		result, reqErr := s.handleListTools(ctx, id)
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(id, result)

	case MethodToolsCall:
		raw := ExtractJSONValue(message, "params")
		var p CallToolParams
		if err := json.Decode(raw, &p); err != nil {
			return createErrorResponse(id, INVALID_PARAMS, "Invalid params: "+err.Error())
		}
		result, reqErr := s.handleToolCall(ctx, id, p)
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(id, result)

	default:
		if method == "" {
			return createErrorResponse(id, INVALID_REQUEST, "Method not found")
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

func ExtractJSONValue(data []byte, key string) []byte {
	search := "\"" + key + "\""
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
			// Found the key, now skip whitespace and look for ':'
			idx := i + len(search)
			for idx < len(data) && (data[idx] == ' ' || data[idx] == '\t' || data[idx] == '\n' || data[idx] == '\r') {
				idx++
			}
			if idx < len(data) && data[idx] == ':' {
				start = idx + 1
				break
			}
		}
	}
	if start == -1 {
		return nil
	}
	for start < len(data) && (data[start] == ' ' || data[start] == '\t' || data[start] == '\n' || data[start] == '\r') {
		start++
	}
	if start >= len(data) {
		return nil
	}
	end := start
	if data[start] == '"' {
		end++
		for end < len(data) {
			if data[end] == '"' {
				// check for escaped quotes
				escaped := false
				for j := end - 1; j >= start && data[j] == '\\'; j-- {
					escaped = !escaped
				}
				if !escaped {
					end++
					break
				}
			}
			end++
		}
	} else if data[start] == '{' || data[start] == '[' {
		depth := 0
		inString := false
		for end < len(data) {
			c := data[end]
			if c == '"' {
				// check for escaped quotes
				escaped := false
				for j := end - 1; j >= 0 && data[j] == '\\'; j-- {
					escaped = !escaped
				}
				if !escaped {
					inString = !inString
				}
			}
			if !inString {
				if c == '{' || c == '[' {
					depth++
				} else if c == '}' || c == ']' {
					depth--
					if depth == 0 {
						end++
						break
					}
				}
			}
			end++
		}
	} else {
		for end < len(data) && data[end] != ',' && data[end] != '}' && data[end] != ']' && data[end] != ' ' && data[end] != '\t' && data[end] != '\n' && data[end] != '\r' {
			end++
		}
	}
	return data[start:end]
}
