package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rediwo/redi-orm/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServer creates a test MCP server with in-memory SQLite database
func setupTestServer(t *testing.T) *mcp.Server {
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Transport:   "stdio",
		LogLevel:    "error",
		Security: mcp.SecurityConfig{
			ReadOnlyMode:  true,
			MaxQueryRows:  1000,
		},
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)
	require.NotNil(t, server)

	return server
}

func TestBatchQueryTool(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()

	// Create test data
	_, err := server.GetDatabase().Raw("CREATE TABLE IF NOT EXISTS batch_test (id INTEGER, name TEXT, value INTEGER)").Exec(ctx)
	require.NoError(t, err)
	_, err = server.GetDatabase().Raw("INSERT INTO batch_test (id, name, value) VALUES (1, 'test1', 100), (2, 'test2', 200), (3, 'test3', 300)").Exec(ctx)
	require.NoError(t, err)

	t.Run("Successful Batch Execution", func(t *testing.T) {
		request := map[string]interface{}{
			"queries": []map[string]interface{}{
				{
					"sql":   "SELECT COUNT(*) as count FROM batch_test",
					"label": "count_all",
				},
				{
					"sql":        "SELECT * FROM batch_test WHERE value > ?",
					"parameters": []interface{}{150},
					"label":      "high_values",
				},
			},
			"fail_fast": false,
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "batch_query", requestJSON)
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].Text), &response)
		require.NoError(t, err)

		assert.Equal(t, 2, int(response["batch_size"].(float64)))
		assert.Equal(t, 2, int(response["executed"].(float64)))

		results := response["results"].([]interface{})
		assert.Len(t, results, 2)

		// Check first query result
		firstResult := results[0].(map[string]interface{})
		assert.Equal(t, "count_all", firstResult["label"])
		assert.True(t, firstResult["success"].(bool))
	})

	t.Run("Fail Fast Mode", func(t *testing.T) {
		request := map[string]interface{}{
			"queries": []map[string]interface{}{
				{
					"sql":   "SELECT * FROM nonexistent_table", // This will fail
					"label": "bad_query",
				},
				{
					"sql":   "SELECT COUNT(*) FROM batch_test",
					"label": "good_query",
				},
			},
			"fail_fast": true,
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "batch_query", requestJSON)
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].Text), &response)
		require.NoError(t, err)

		results := response["results"].([]interface{})
		assert.Len(t, results, 1) // Should stop after first failure

		firstResult := results[0].(map[string]interface{})
		assert.False(t, firstResult["success"].(bool))
		assert.NotEmpty(t, firstResult["error"])
	})
}

func TestStreamQueryTool(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()

	// Create test data with more rows
	_, err := server.GetDatabase().Raw("CREATE TABLE IF NOT EXISTS stream_test (id INTEGER, data TEXT)").Exec(ctx)
	require.NoError(t, err)

	// Insert multiple rows
	for i := 1; i <= 250; i++ {
		_, err = server.GetDatabase().Raw("INSERT INTO stream_test (id, data) VALUES (?, ?)", i, "data"+string(rune(i))).Exec(ctx)
		require.NoError(t, err)
	}

	t.Run("Stream Large Result Set", func(t *testing.T) {
		request := map[string]interface{}{
			"sql":        "SELECT * FROM stream_test ORDER BY id",
			"batch_size": 50,
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "stream_query", requestJSON)
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].Text), &response)
		require.NoError(t, err)

		assert.Equal(t, 250, int(response["total_rows"].(float64)))
		assert.Equal(t, 50, int(response["batch_size"].(float64)))
		assert.Equal(t, 5, int(response["batch_count"].(float64))) // 250/50 = 5 batches
		assert.True(t, response["streaming"].(bool))

		batches := response["batches"].([]interface{})
		assert.Len(t, batches, 5)

		// Check first batch has 50 rows
		firstBatch := batches[0].([]interface{})
		assert.Len(t, firstBatch, 50)
	})

	t.Run("Stream with Security Validation", func(t *testing.T) {
		request := map[string]interface{}{
			"sql": "DELETE FROM stream_test", // Should be blocked by security
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "stream_query", requestJSON)
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, "Security error")
	})
}

func TestAnalyzeTableTool(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()

	// Create test data with various data types
	_, err := server.GetDatabase().Raw("CREATE TABLE IF NOT EXISTS analyze_test (id INTEGER, name TEXT, score REAL, active BOOLEAN, created_at TEXT)").Exec(ctx)
	require.NoError(t, err)

	// Insert test data with different values
	_, err = server.GetDatabase().Raw(`
		INSERT INTO analyze_test (id, name, score, active, created_at) VALUES 
		(1, 'Alice', 95.5, 1, '2023-01-01'),
		(2, 'Bob', 87.2, 0, '2023-01-02'),
		(3, 'Charlie', 92.8, 1, '2023-01-03'),
		(4, 'Diana', 88.1, 1, '2023-01-04'),
		(5, NULL, 90.0, 0, '2023-01-05')
	`).Exec(ctx)
	require.NoError(t, err)

	t.Run("Full Table Analysis", func(t *testing.T) {
		request := map[string]interface{}{
			"table":       "analyze_test",
			"sample_size": 100,
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "analyze_table", requestJSON)
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].Text), &response)
		require.NoError(t, err)

		assert.Equal(t, "analyze_test", response["table"])
		assert.Equal(t, int64(5), int64(response["total_rows"].(float64)))
		assert.Equal(t, 5, int(response["sample_size"].(float64)))

		statistics := response["statistics"].(map[string]interface{})
		assert.Contains(t, statistics, "id")
		assert.Contains(t, statistics, "name")
		assert.Contains(t, statistics, "score")

		// Check ID column statistics
		idStats := statistics["id"].(map[string]interface{})
		assert.Equal(t, "integer", idStats["data_type"])
		assert.Equal(t, 0, int(idStats["null_count"].(float64)))
		assert.Equal(t, 5, int(idStats["unique_count"].(float64)))

		// Check name column statistics (has one NULL)
		nameStats := statistics["name"].(map[string]interface{})
		assert.Equal(t, "string", nameStats["data_type"])
		assert.Equal(t, 1, int(nameStats["null_count"].(float64)))
		assert.Equal(t, 4, int(nameStats["unique_count"].(float64)))
	})

	t.Run("Specific Columns Analysis", func(t *testing.T) {
		request := map[string]interface{}{
			"table":   "analyze_test",
			"columns": []string{"score", "active"},
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "analyze_table", requestJSON)
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].Text), &response)
		require.NoError(t, err)

		statistics := response["statistics"].(map[string]interface{})
		assert.Contains(t, statistics, "score")
		assert.Contains(t, statistics, "active")
		assert.NotContains(t, statistics, "id") // Should not include other columns
		assert.NotContains(t, statistics, "name")
	})
}

func TestGenerateSampleTool(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()

	// Create test data
	_, err := server.GetDatabase().Raw("CREATE TABLE IF NOT EXISTS sample_test (id INTEGER, category TEXT, value INTEGER)").Exec(ctx)
	require.NoError(t, err)

	for i := 1; i <= 20; i++ {
		category := "A"
		if i > 10 {
			category = "B"
		}
		_, err = server.GetDatabase().Raw("INSERT INTO sample_test (id, category, value) VALUES (?, ?, ?)", i, category, i*10).Exec(ctx)
		require.NoError(t, err)
	}

	t.Run("Basic Sampling", func(t *testing.T) {
		request := map[string]interface{}{
			"table": "sample_test",
			"count": 5,
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "generate_sample", requestJSON)
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].Text), &response)
		require.NoError(t, err)

		assert.Equal(t, "sample_test", response["table"])
		assert.Equal(t, 5, int(response["requested"].(float64)))
		assert.Equal(t, 5, int(response["sample_size"].(float64)))

		data := response["data"].([]interface{})
		assert.Len(t, data, 5)
	})

	t.Run("Filtered Sampling", func(t *testing.T) {
		request := map[string]interface{}{
			"table": "sample_test",
			"count": 3,
			"where": map[string]interface{}{
				"category": "B",
			},
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "generate_sample", requestJSON)
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].Text), &response)
		require.NoError(t, err)

		data := response["data"].([]interface{})
		assert.Len(t, data, 3)

		// All samples should have category = "B"
		for _, row := range data {
			rowMap := row.(map[string]interface{})
			assert.Equal(t, "B", rowMap["category"])
		}
	})

	t.Run("Random Sampling", func(t *testing.T) {
		request := map[string]interface{}{
			"table":  "sample_test",
			"count":  5,
			"random": true,
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "generate_sample", requestJSON)
		require.NoError(t, err)
		assert.False(t, result.IsError)

		// Parse response
		var response map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].Text), &response)
		require.NoError(t, err)

		assert.True(t, response["random"].(bool))
		data := response["data"].([]interface{})
		assert.Len(t, data, 5)
	})
}

func TestAdvancedToolsSecurity(t *testing.T) {
	// Create server with restricted table access
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Transport:   "stdio",
		LogLevel:    "error",
		Security: mcp.SecurityConfig{
			EnableAuth:     false,
			ReadOnlyMode:   true,
			AllowedTables:  []string{"allowed_table"},
			MaxQueryRows:   100,
		},
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Create tables
	_, err = server.GetDatabase().Raw("CREATE TABLE allowed_table (id INTEGER, data TEXT)").Exec(ctx)
	require.NoError(t, err)
	_, err = server.GetDatabase().Raw("CREATE TABLE forbidden_table (id INTEGER, secret TEXT)").Exec(ctx)
	require.NoError(t, err)

	t.Run("Batch Query Security", func(t *testing.T) {
		request := map[string]interface{}{
			"queries": []map[string]interface{}{
				{
					"sql": "INSERT INTO allowed_table (id, data) VALUES (1, 'test')", // Should be blocked by read-only mode
				},
			},
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "batch_query", requestJSON)
		require.NoError(t, err)
		
		// The tool result itself shouldn't be an error, but the individual query should fail
		assert.False(t, result.IsError)

		var response map[string]interface{}
		err = json.Unmarshal([]byte(result.Content[0].Text), &response)
		require.NoError(t, err)

		results := response["results"].([]interface{})
		require.Len(t, results, 1)
		
		firstResult := results[0].(map[string]interface{})
		assert.False(t, firstResult["success"].(bool))
		assert.Contains(t, firstResult["error"], "Security error")
		assert.Contains(t, firstResult["error"], "read-only mode")
	})

	t.Run("Analyze Table Security", func(t *testing.T) {
		request := map[string]interface{}{
			"table": "forbidden_table",
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "analyze_table", requestJSON)
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, "Security error")
	})

	t.Run("Generate Sample Security", func(t *testing.T) {
		request := map[string]interface{}{
			"table": "forbidden_table",
		}

		requestJSON, _ := json.Marshal(request)
		result, err := server.CallTool(ctx, "generate_sample", requestJSON)
		require.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, "Security error")
	})
}