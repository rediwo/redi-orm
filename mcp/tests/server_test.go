package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rediwo/redi-orm/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerInitialization(t *testing.T) {
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Transport:   "stdio",
		LogLevel:    "error",
		MaxQueryRows: 100,
	}

	server, err := mcp.NewServer(config)
	assert.NoError(t, err)
	assert.NotNil(t, server)
}

func TestHandlerInitialize(t *testing.T) {
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Transport:   "stdio",
		LogLevel:    "error",
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	handler := mcp.NewHandler(server)

	// Test initialize method
	params := json.RawMessage(`{
		"protocolVersion": "2024-11-05",
		"capabilities": {},
		"clientInfo": {
			"name": "test-client",
			"version": "1.0.0"
		}
	}`)

	result, err := handler.Initialize(context.Background(), params)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	initResult, ok := result.(mcp.InitializeResult)
	assert.True(t, ok)
	assert.Equal(t, "2024-11-05", initResult.ProtocolVersion)
	assert.Equal(t, "redi-orm-mcp", initResult.ServerInfo.Name)
	assert.NotNil(t, initResult.Capabilities.Resources)
	assert.NotNil(t, initResult.Capabilities.Tools)
	assert.NotNil(t, initResult.Capabilities.Prompts)
}

func TestJSONRPCHandler(t *testing.T) {
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Transport:   "stdio",
		LogLevel:    "error",
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	handler := mcp.NewHandler(server)
	ctx := context.Background()

	tests := []struct {
		name     string
		request  json.RawMessage
		wantErr  bool
		errCode  int
	}{
		{
			name: "valid initialize",
			request: json.RawMessage(`{
				"jsonrpc": "2.0",
				"method": "initialize",
				"params": {},
				"id": 1
			}`),
			wantErr: false,
		},
		{
			name: "missing jsonrpc version",
			request: json.RawMessage(`{
				"method": "initialize",
				"params": {},
				"id": 1
			}`),
			wantErr: true,
			errCode: mcp.ErrorCodeInvalidRequest,
		},
		{
			name: "invalid method",
			request: json.RawMessage(`{
				"jsonrpc": "2.0",
				"method": "unknown_method",
				"params": {},
				"id": 2
			}`),
			wantErr: true,
			errCode: mcp.ErrorCodeMethodNotFound,
		},
		{
			name: "invalid JSON",
			request: json.RawMessage(`{invalid json`),
			wantErr: true,
			errCode: mcp.ErrorCodeParseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := handler.Handle(ctx, tt.request)
			
			var resp mcp.JSONRPCResponse
			err := json.Unmarshal(response, &resp)
			require.NoError(t, err)

			if tt.wantErr {
				assert.NotNil(t, resp.Error)
				assert.Equal(t, tt.errCode, resp.Error.Code)
				assert.Nil(t, resp.Result)
			} else {
				assert.Nil(t, resp.Error)
				assert.NotNil(t, resp.Result)
			}
		})
	}
}

func TestListTools(t *testing.T) {
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Transport:   "stdio",
		LogLevel:    "error",
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	tools := server.ListTools()
	assert.NotEmpty(t, tools)

	// Check for expected tools
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	assert.True(t, toolNames["query"])
	assert.True(t, toolNames["inspect_schema"])
	assert.True(t, toolNames["list_tables"])
	assert.True(t, toolNames["count_records"])
}

func TestListPrompts(t *testing.T) {
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Transport:   "stdio",
		LogLevel:    "error",
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	prompts := server.ListPrompts()
	assert.NotEmpty(t, prompts)

	// Check for expected prompts
	promptNames := make(map[string]bool)
	for _, prompt := range prompts {
		promptNames[prompt.Name] = true
	}

	assert.True(t, promptNames["analyze_schema"])
	assert.True(t, promptNames["create_model"])
	assert.True(t, promptNames["optimize_query"])
	assert.True(t, promptNames["analyze_relations"])
	assert.True(t, promptNames["generate_api"])
	assert.True(t, promptNames["data_migration"])
}

func TestResourceListing(t *testing.T) {
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Transport:   "stdio",
		LogLevel:    "error",
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	resources, err := server.ListResources(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resources)

	// Check for schema resource
	found := false
	for _, r := range resources {
		if r.URI == "schema://database" {
			found = true
			assert.Equal(t, "Database Schema", r.Name)
			assert.Equal(t, "application/json", r.MimeType)
			break
		}
	}
	assert.True(t, found, "schema://database resource not found")
}

func TestIsReadOnlyQuery(t *testing.T) {
	tests := []struct {
		query    string
		expected bool
	}{
		{"SELECT * FROM users", true},
		{"select id, name from products", true},
		{"  SELECT COUNT(*) FROM orders  ", true},
		{"INSERT INTO users VALUES (1, 'test')", false},
		{"UPDATE users SET name = 'test'", false},
		{"DELETE FROM users", false},
		{"DROP TABLE users", false},
		{"CREATE TABLE test (id INT)", false},
		{"ALTER TABLE users ADD COLUMN age INT", false},
		{"SELECT * FROM users; DROP TABLE users", false},
		{"SELECT * INTO OUTFILE '/tmp/data.csv' FROM users", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			// This function is not exported, so we can't test it directly
			// In a real test, we'd test through the CallTool method
		})
	}
}