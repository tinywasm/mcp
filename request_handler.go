package mcp

import (
	"context"
	"encoding/json"
	"net/http"
)

// HandleMessage processes an incoming JSON-RPC message and returns a response.
func (s *MCPServer) HandleMessage(ctx context.Context, message json.RawMessage) JSONRPCMessage {
	ctx = context.WithValue(ctx, serverKey{}, s)

	var baseMessage struct {
		JSONRPC string    `json:"jsonrpc"`
		Method  MCPMethod `json:"method"`
		ID      any       `json:"id,omitempty"`
		Result  any       `json:"result,omitempty"`
	}
	if err := json.Unmarshal(message, &baseMessage); err != nil {
		return createErrorResponse(nil, PARSE_ERROR, "Failed to parse message")
	}
	if baseMessage.JSONRPC != JSONRPC_VERSION {
		return createErrorResponse(baseMessage.ID, INVALID_REQUEST, "Invalid JSON-RPC version")
	}
	if baseMessage.ID == nil {
		var notification JSONRPCNotification
		if err := json.Unmarshal(message, &notification); err != nil {
			return createErrorResponse(nil, PARSE_ERROR, "Failed to parse notification")
		}
		s.handleNotification(ctx, notification)
		return nil
	}
	if baseMessage.Result != nil {
		return nil
	}

	h := ctx.Value(requestHeader)
	headers, ok := h.(http.Header)
	if headers == nil || !ok {
		headers = make(http.Header)
	}

	var reqErr *requestError
	switch baseMessage.Method {
	case MethodInitialize:
		var request InitializeRequest
		var result *InitializeResult
		if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			reqErr = &requestError{id: baseMessage.ID, code: INVALID_REQUEST, err: &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method}}
		} else {
			request.Header = headers
			result, reqErr = s.handleInitialize(ctx, baseMessage.ID, request)
		}
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)

	case MethodPing:
		var request PingRequest
		var result *EmptyResult
		if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			reqErr = &requestError{id: baseMessage.ID, code: INVALID_REQUEST, err: &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method}}
		} else {
			request.Header = headers
			result, reqErr = s.handlePing(ctx, baseMessage.ID, request)
		}
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)

	case MethodToolsList:
		var request ListToolsRequest
		var result *ListToolsResult
		if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			reqErr = &requestError{id: baseMessage.ID, code: INVALID_REQUEST, err: &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method}}
		} else {
			request.Header = headers
			result, reqErr = s.handleListTools(ctx, baseMessage.ID, request)
		}
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)

	case MethodToolsCall:
		var request CallToolRequest
		var result *CallToolResult
		if unmarshalErr := json.Unmarshal(message, &request); unmarshalErr != nil {
			reqErr = &requestError{id: baseMessage.ID, code: INVALID_REQUEST, err: &UnparsableMessageError{message: message, err: unmarshalErr, method: baseMessage.Method}}
		} else {
			request.Header = headers
			result, reqErr = s.handleToolCall(ctx, baseMessage.ID, request)
		}
		if reqErr != nil {
			return reqErr.ToJSONRPCError()
		}
		return createResponse(baseMessage.ID, *result)

	default:
		return createErrorResponse(baseMessage.ID, METHOD_NOT_FOUND, "Method not found: "+string(baseMessage.Method))
	}
}
