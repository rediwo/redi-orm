package mcp_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rediwo/redi-orm/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPEndToEnd(t *testing.T) {
	// Create server
	config := mcp.ServerConfig{
		DatabaseURI:  "sqlite://:memory:",
		Transport:    "stdio",
		LogLevel:     "error",
		MaxQueryRows: 100,
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	handler := mcp.NewHandler(server)
	ctx := context.Background()

	// Helper to send request and get response
	sendRequest := func(method string, params interface{}) (interface{}, error) {
		paramsJSON, _ := json.Marshal(params)
		
		request := mcp.JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  method,
			Params:  paramsJSON,
			ID:      1,
		}
		
		requestJSON, _ := json.Marshal(request)
		responseJSON := handler.Handle(ctx, requestJSON)
		
		var response mcp.JSONRPCResponse
		err := json.Unmarshal(responseJSON, &response)
		if err != nil {
			return nil, err
		}
		
		if response.Error != nil {
			return nil, response.Error
		}
		
		return response.Result, nil
	}

	t.Run("Full Workflow", func(t *testing.T) {
		// 1. Initialize
		result, err := sendRequest("initialize", map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// 2. Create test data
		db := server.GetDatabase()
		require.NotNil(t, db)
		
		_, err = db.Raw("CREATE TABLE products (id INTEGER PRIMARY KEY, name TEXT, price REAL)").Exec(ctx)
		require.NoError(t, err)
		
		_, err = db.Raw("INSERT INTO products (name, price) VALUES (?, ?)", "Laptop", 999.99).Exec(ctx)
		require.NoError(t, err)
		
		_, err = db.Raw("INSERT INTO products (name, price) VALUES (?, ?)", "Mouse", 29.99).Exec(ctx)
		require.NoError(t, err)

		// 3. List resources
		result, err = sendRequest("resources/list", nil)
		assert.NoError(t, err)
		
		resultJSON, _ := json.Marshal(result)
		var listResult mcp.ListResourcesResult
		json.Unmarshal(resultJSON, &listResult)
		
		// Should have table and data resources for products
		hasTableResource := false
		hasDataResource := false
		for _, r := range listResult.Resources {
			if r.URI == "table://products" {
				hasTableResource = true
			}
			if r.URI == "data://products" {
				hasDataResource = true
			}
		}
		assert.True(t, hasTableResource)
		assert.True(t, hasDataResource)

		// 4. Query data
		result, err = sendRequest("tools/call", map[string]interface{}{
			"name": "query",
			"arguments": map[string]interface{}{
				"sql":        "SELECT * FROM products WHERE price < ?",
				"parameters": []interface{}{100},
			},
		})
		assert.NoError(t, err)
		
		resultJSON, _ = json.Marshal(result)
		var toolResult mcp.ToolResult
		json.Unmarshal(resultJSON, &toolResult)
		
		assert.False(t, toolResult.IsError)
		assert.Contains(t, toolResult.Content[0].Text, "Mouse")
		assert.NotContains(t, toolResult.Content[0].Text, "Laptop")

		// 5. Count records
		result, err = sendRequest("tools/call", map[string]interface{}{
			"name": "count_records",
			"arguments": map[string]interface{}{
				"table": "products",
			},
		})
		assert.NoError(t, err)
		
		resultJSON, _ = json.Marshal(result)
		json.Unmarshal(resultJSON, &toolResult)
		
		assert.Contains(t, toolResult.Content[0].Text, `"count": 2`)

		// 6. Inspect schema
		result, err = sendRequest("tools/call", map[string]interface{}{
			"name": "inspect_schema",
			"arguments": map[string]interface{}{
				"table": "products",
			},
		})
		assert.NoError(t, err)
		
		resultJSON, _ = json.Marshal(result)
		json.Unmarshal(resultJSON, &toolResult)
		
		assert.Contains(t, toolResult.Content[0].Text, "columns")
		assert.Contains(t, toolResult.Content[0].Text, "id")
		assert.Contains(t, toolResult.Content[0].Text, "name")
		assert.Contains(t, toolResult.Content[0].Text, "price")

		// 7. Test security - non-SELECT query
		result, err = sendRequest("tools/call", map[string]interface{}{
			"name": "query",
			"arguments": map[string]interface{}{
				"sql":        "DELETE FROM products",
				"parameters": []interface{}{},
			},
		})
		assert.NoError(t, err)
		
		resultJSON, _ = json.Marshal(result)
		json.Unmarshal(resultJSON, &toolResult)
		
		assert.True(t, toolResult.IsError)
		assert.Contains(t, toolResult.Content[0].Text, "Security error")

		// 8. Test prompts
		result, err = sendRequest("prompts/list", nil)
		assert.NoError(t, err)
		
		resultJSON, _ = json.Marshal(result)
		var promptsResult mcp.ListPromptsResult
		json.Unmarshal(resultJSON, &promptsResult)
		
		assert.True(t, len(promptsResult.Prompts) > 0)

		// 9. Get a specific prompt
		result, err = sendRequest("prompts/get", map[string]interface{}{
			"name": "optimize_query",
			"arguments": map[string]interface{}{
				"query": "SELECT * FROM products WHERE name LIKE '%top%'",
			},
		})
		assert.NoError(t, err)
		
		resultJSON, _ = json.Marshal(result)
		var promptResult mcp.GetPromptResult
		json.Unmarshal(resultJSON, &promptResult)
		
		assert.Len(t, promptResult.Messages, 1)
		assert.Equal(t, "user", promptResult.Messages[0].Role)
		assert.Contains(t, promptResult.Messages[0].Content.Text, "SELECT * FROM products WHERE name LIKE '%top%'")
	})
}

func TestMCPPerformance(t *testing.T) {
	config := mcp.ServerConfig{
		DatabaseURI:  "sqlite://:memory:",
		Transport:    "stdio",
		LogLevel:     "error",
		MaxQueryRows: 1000,
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	handler := mcp.NewHandler(server)
	ctx := context.Background()

	// Create test data
	db := server.GetDatabase()
	_, err = db.Raw("CREATE TABLE test_data (id INTEGER PRIMARY KEY, value TEXT)").Exec(ctx)
	require.NoError(t, err)

	// Insert 1000 rows
	for i := 0; i < 1000; i++ {
		_, err = db.Raw("INSERT INTO test_data (value) VALUES (?)", 
			"test_value_" + string(rune(i))).Exec(ctx)
		require.NoError(t, err)
	}

	// Test query performance
	start := time.Now()
	
	request := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: json.RawMessage(`{
			"name": "query",
			"arguments": {
				"sql": "SELECT * FROM test_data",
				"parameters": []
			}
		}`),
		ID: 1,
	}
	
	requestJSON, _ := json.Marshal(request)
	responseJSON := handler.Handle(ctx, requestJSON)
	
	elapsed := time.Since(start)
	
	var response mcp.JSONRPCResponse
	err = json.Unmarshal(responseJSON, &response)
	require.NoError(t, err)
	require.Nil(t, response.Error)
	
	// Should complete within reasonable time
	assert.Less(t, elapsed, 500*time.Millisecond, "Query took too long: %v", elapsed)
	
	t.Logf("Query 1000 rows took: %v", elapsed)
}

func TestMCPErrorHandling(t *testing.T) {
	config := mcp.ServerConfig{
		DatabaseURI:  "sqlite://:memory:",
		Transport:    "stdio",
		LogLevel:     "error",
		MaxQueryRows: 10,
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	handler := mcp.NewHandler(server)
	ctx := context.Background()

	tests := []struct {
		name        string
		request     mcp.JSONRPCRequest
		wantErrCode int
		wantErrMsg  string
	}{
		{
			name: "invalid params type",
			request: mcp.JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  "tools/call",
				Params:  json.RawMessage(`"not an object"`),
				ID:      1,
			},
			wantErrCode: mcp.ErrorCodeInvalidParams,
		},
		{
			name: "unknown tool",
			request: mcp.JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  "tools/call",
				Params: json.RawMessage(`{
					"name": "unknown_tool",
					"arguments": {}
				}`),
				ID: 1,
			},
			wantErrCode: mcp.ErrorCodeInternalError,
		},
		{
			name: "malformed SQL",
			request: mcp.JSONRPCRequest{
				JSONRPC: "2.0",
				Method:  "tools/call",
				Params: json.RawMessage(`{
					"name": "query",
					"arguments": {
						"sql": "SELECT * FROM",
						"parameters": []
					}
				}`),
				ID: 1,
			},
			wantErrCode: -1, // Special marker to check for tool error instead of JSON-RPC error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestJSON, _ := json.Marshal(tt.request)
			responseJSON := handler.Handle(ctx, requestJSON)
			
			var response mcp.JSONRPCResponse
			err := json.Unmarshal(responseJSON, &response)
			require.NoError(t, err)
			
			// Special handling for tool errors (not JSON-RPC errors)
			if tt.wantErrCode == -1 {
				assert.Nil(t, response.Error)
				assert.NotNil(t, response.Result)
				
				// Check if it's a tool error
				resultJSON, _ := json.Marshal(response.Result)
				var toolResult mcp.ToolResult
				json.Unmarshal(resultJSON, &toolResult)
				assert.True(t, toolResult.IsError)
				assert.Contains(t, toolResult.Content[0].Text, "error")
			} else {
				assert.NotNil(t, response.Error)
				assert.Equal(t, tt.wantErrCode, response.Error.Code)
				if tt.wantErrMsg != "" {
					assert.Contains(t, response.Error.Message, tt.wantErrMsg)
				}
			}
		})
	}
}