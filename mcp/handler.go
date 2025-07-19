package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// Handler processes JSON-RPC requests
type Handler struct {
	server  *Server
	methods map[string]HandlerFunc
}

// NewHandler creates a new JSON-RPC handler
func NewHandler(server *Server) *Handler {
	h := &Handler{
		server:  server,
		methods: make(map[string]HandlerFunc),
	}

	// Register core MCP methods
	h.methods[MethodInitialize] = h.Initialize
	h.methods[MethodResourcesList] = h.ResourcesList
	h.methods[MethodResourcesRead] = h.ResourcesRead
	h.methods[MethodToolsList] = h.ToolsList
	h.methods[MethodToolsCall] = h.ToolsCall
	h.methods[MethodPromptsList] = h.PromptsList
	h.methods[MethodPromptsGet] = h.PromptsGet

	return h
}

// Handle processes a JSON-RPC request and returns a response
func (h *Handler) Handle(ctx context.Context, message json.RawMessage) json.RawMessage {
	var request JSONRPCRequest
	if err := json.Unmarshal(message, &request); err != nil {
		return h.errorResponse(nil, ErrorCodeParseError, "Parse error", err)
	}

	// Validate JSON-RPC version
	if request.JSONRPC != "2.0" {
		return h.errorResponse(request.ID, ErrorCodeInvalidRequest, "Invalid request", "JSON-RPC version must be 2.0")
	}

	// Find and execute method
	handler, exists := h.methods[request.Method]
	if !exists {
		return h.errorResponse(request.ID, ErrorCodeMethodNotFound, "Method not found", fmt.Sprintf("Method '%s' not found", request.Method))
	}

	// Execute method
	result, err := handler(ctx, request.Params)
	if err != nil {
		// Check if it's already a JSON-RPC error
		if jsonErr, ok := err.(*JSONRPCError); ok {
			return h.errorResponse(request.ID, jsonErr.Code, jsonErr.Message, jsonErr.Data)
		}
		return h.errorResponse(request.ID, ErrorCodeInternalError, "Internal error", err.Error())
	}

	// Build success response
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      request.ID,
	}

	responseData, _ := json.Marshal(response)
	return responseData
}

// errorResponse creates a JSON-RPC error response
func (h *Handler) errorResponse(id interface{}, code int, message string, data interface{}) json.RawMessage {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}

	responseData, _ := json.Marshal(response)
	return responseData
}

// Initialize handles the initialize method
func (h *Handler) Initialize(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var initParams InitializeParams
	if params != nil {
		if err := json.Unmarshal(params, &initParams); err != nil {
			return nil, &JSONRPCError{Code: ErrorCodeInvalidParams, Message: "Invalid params", Data: err.Error()}
		}
	}

	// Log initialization
	if h.server.logger != nil {
		h.server.logger.Info("MCP server initializing with protocol version: %s", initParams.ProtocolVersion)
		if initParams.ClientInfo != nil {
			h.server.logger.Debug("Client info: %s v%s", initParams.ClientInfo.Name, initParams.ClientInfo.Version)
		}
	}

	return InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: Capabilities{
			Resources: &ResourcesCapability{},
			Tools:     &ToolsCapability{},
			Prompts:   &PromptsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    "redi-orm-mcp",
			Version: "1.0.0",
		},
	}, nil
}

// ResourcesList handles the resources/list method
func (h *Handler) ResourcesList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	resources, err := h.server.ListResources(ctx)
	if err != nil {
		return nil, err
	}

	return ListResourcesResult{Resources: resources}, nil
}

// ResourcesRead handles the resources/read method
func (h *Handler) ResourcesRead(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var readParams ReadResourceParams
	if err := json.Unmarshal(params, &readParams); err != nil {
		return nil, &JSONRPCError{Code: ErrorCodeInvalidParams, Message: "Invalid params", Data: err.Error()}
	}

	content, err := h.server.ReadResource(ctx, readParams.URI)
	if err != nil {
		return nil, err
	}

	return ReadResourceResult{Contents: []ResourceContent{*content}}, nil
}

// ToolsList handles the tools/list method
func (h *Handler) ToolsList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	tools := h.server.ListTools()
	return ListToolsResult{Tools: tools}, nil
}

// ToolsCall handles the tools/call method
func (h *Handler) ToolsCall(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var callParams CallToolParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, &JSONRPCError{Code: ErrorCodeInvalidParams, Message: "Invalid params", Data: err.Error()}
	}

	result, err := h.server.CallTool(ctx, callParams.Name, callParams.Arguments)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// PromptsList handles the prompts/list method
func (h *Handler) PromptsList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	prompts := h.server.ListPrompts()
	return ListPromptsResult{Prompts: prompts}, nil
}

// PromptsGet handles the prompts/get method
func (h *Handler) PromptsGet(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var getParams GetPromptParams
	if err := json.Unmarshal(params, &getParams); err != nil {
		return nil, &JSONRPCError{Code: ErrorCodeInvalidParams, Message: "Invalid params", Data: err.Error()}
	}

	result, err := h.server.GetPrompt(ctx, getParams.Name, getParams.Arguments)
	if err != nil {
		return nil, err
	}

	return result, nil
}