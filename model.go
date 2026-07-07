package mcp

import "github.com/tinywasm/model"

var rpcRequestModel = model.Definition{
	Name: "rpc_request",
	Fields: model.Fields{
		{Name: "jsonrpc", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
		{Name: "method", Type: model.FieldText},
		// JSON-RPC 2.0 requires params to be a structured value (object/array),
		// never a JSON string — FieldRaw embeds it inline instead of
		// double-encoding it as a quoted string the server can't decode.
		{Name: "params", Type: model.FieldRaw},
	},
}

var rpcResponseModel = model.Definition{
	Name: "rpc_response",
	Fields: model.Fields{
		{Name: "jsonrpc", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
		{Name: "result", Type: model.FieldRaw, OmitEmpty: true},
		// The server's error is a nested {code,message,data} object
		// (JSONRPCErrorModel), never a plain string — FieldRaw so the
		// client can decode it instead of silently failing to match a string.
		{Name: "error", Type: model.FieldRaw, OmitEmpty: true},
	},
}

var jsonRPCErrorModel = model.Definition{
	Name: "json_rpcerror",
	Fields: model.Fields{
		{Name: "code", Type: model.FieldInt},
		{Name: "message", Type: model.FieldText},
		{Name: "data", Type: model.FieldText, OmitEmpty: true},
	},
}

var initializeParamsModel = model.Definition{
	Name: "initialize_params",
	Fields: model.Fields{
		{Name: "protocolVersion", Type: model.FieldText},
		{Name: "clientInfo", Type: model.FieldStruct, Ref: &implementationInfoModel},
	},
}

var implementationInfoModel = model.Definition{
	Name: "implementation_info",
	Fields: model.Fields{
		{Name: "name", Type: model.FieldText},
		{Name: "version", Type: model.FieldText},
	},
}

var initializeResultModel = model.Definition{
	Name: "initialize_result",
	Fields: model.Fields{
		{Name: "protocolVersion", Type: model.FieldText},
		{Name: "serverInfo", Type: model.FieldStruct, Ref: &implementationInfoModel},
		{Name: "capabilities", Type: model.FieldRaw, OmitEmpty: true},
	},
}

var CallToolParamsModel = model.Definition{
	Name: "call_tool_params",
	Fields: model.Fields{
		{Name: "name", Type: model.FieldText},
		{Name: "arguments", Type: model.FieldRaw},
	},
}

var ResultModel = model.Definition{
	Name: "result",
	Fields: model.Fields{
		{Name: "isError", Type: model.FieldBool, OmitEmpty: true},
		{Name: "content", Type: model.FieldRaw, OmitEmpty: true},
	},
}

var TextContentModel = model.Definition{
	Name: "text_content",
	Fields: model.Fields{
		{Name: "type", Type: model.FieldText},
		{Name: "text", Type: model.FieldText},
	},
}

var toolEntryModel = model.Definition{
	Name: "tool_entry",
	Fields: model.Fields{
		{Name: "name", Type: model.FieldText},
		{Name: "description", Type: model.FieldText, OmitEmpty: true},
		{Name: "inputSchema", Type: model.FieldRaw},
	},
}

var listToolsResultModel = model.Definition{
	Name: "list_tools_result",
	Fields: model.Fields{
		{Name: "tools", Type: model.FieldRaw},
		{Name: "nextCursor", Type: model.FieldText, OmitEmpty: true},
	},
}

var errorResponseModel = model.Definition{
	Name: "error_response",
	Fields: model.Fields{
		{Name: "jsonrpc", Type: model.FieldText},
		{Name: "id", Type: model.FieldText, OmitEmpty: true},
		{Name: "error", Type: model.FieldStruct, Ref: &jsonRPCErrorModel},
	},
}

var MetaModel = model.Definition{
	Name: "meta",
	Fields: model.Fields{
		{Name: "progressToken", Type: model.FieldText, OmitEmpty: true},
	},
}

var NotificationParamsModel = model.Definition{
	Name: "notification_params",
	Fields: model.Fields{
		{Name: "meta", Type: model.FieldText, OmitEmpty: true},
	},
}

var EmptyResultModel = model.Definition{
	Name: "empty_result",
	Fields: model.Fields{
		{Name: "result", Type: model.FieldText, OmitEmpty: true},
	},
}

var JSONRPCRequestModel = model.Definition{
	Name: "jsonrpcrequest",
	Fields: model.Fields{
		{Name: "jsonrpc", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
		{Name: "method", Type: model.FieldText},
		{Name: "params", Type: model.FieldRaw, OmitEmpty: true},
	},
}

var JSONRPCNotificationModel = model.Definition{
	Name: "jsonrpcnotification",
	Fields: model.Fields{
		{Name: "jsonrpc", Type: model.FieldText},
		{Name: "method", Type: model.FieldText},
		{Name: "params", Type: model.FieldRaw, OmitEmpty: true},
	},
}

var JSONRPCResponseStructModel = model.Definition{
	Name: "jsonrpcresponse_struct",
	Fields: model.Fields{
		{Name: "jsonrpc", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
		{Name: "result", Type: model.FieldRaw, OmitEmpty: true},
		{Name: "error", Type: model.FieldRaw, OmitEmpty: true},
	},
}

// jsonrpcMessage marks the response variants HandleMessage can return.
// The type itself is declared in model_orm.go (generated from
// JSONRPCResponseStructModel above); the marker method lives here since it
// isn't part of the generated codec.
func (r *JSONRPCResponseStruct) jsonrpcMessage() {}

var JSONRPCErrorModel = model.Definition{
	Name: "jsonrpcerror",
	Fields: model.Fields{
		{Name: "jsonrpc", Type: model.FieldText},
		{Name: "id", Type: model.FieldText},
		{Name: "error", Type: model.FieldRaw},
	},
}

func (e *JSONRPCError) jsonrpcMessage() {}

var JSONRPCErrorDetailsModel = model.Definition{
	Name: "jsonrpcerror_details",
	Fields: model.Fields{
		{Name: "code", Type: model.FieldInt},
		{Name: "message", Type: model.FieldText},
		{Name: "data", Type: model.FieldText, OmitEmpty: true},
	},
}
