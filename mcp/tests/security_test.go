package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/rediwo/redi-orm/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityManager(t *testing.T) {
	config := mcp.SecurityConfig{
		EnableAuth:      true,
		APIKey:         "test-api-key-123",
		EnableRateLimit: true,
		RequestsPerMin:  10,
		BurstLimit:     5,
		AllowedTables:  []string{"users", "posts"},
		ReadOnlyMode:   true,
		MaxQueryRows:   100,
	}

	sm := mcp.NewSecurityManager(config)

	t.Run("Table Access Validation", func(t *testing.T) {
		// Allowed table
		err := sm.ValidateTableAccess("users")
		assert.NoError(t, err)

		err = sm.ValidateTableAccess("posts")
		assert.NoError(t, err)

		// Not allowed table
		err = sm.ValidateTableAccess("admin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("Query Validation", func(t *testing.T) {
		// Valid SELECT query
		err := sm.ValidateQuery("SELECT * FROM users")
		assert.NoError(t, err)

		// Invalid non-SELECT query in read-only mode
		err = sm.ValidateQuery("INSERT INTO users VALUES (1, 'test')")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "read-only mode")

		// Dangerous patterns
		err = sm.ValidateQuery("SELECT * FROM users; DROP TABLE users")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dangerous")

		err = sm.ValidateQuery("SELECT * INTO OUTFILE '/tmp/data' FROM users")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "dangerous")
	})

	t.Run("HTTP Authentication", func(t *testing.T) {
		// Request with valid API key in Authorization header
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer test-api-key-123")
		err := sm.AuthenticateRequest(req)
		assert.NoError(t, err)

		// Request with valid API key in X-API-Key header
		req, _ = http.NewRequest("GET", "/", nil)
		req.Header.Set("X-API-Key", "test-api-key-123")
		err = sm.AuthenticateRequest(req)
		assert.NoError(t, err)

		// Request with invalid API key
		req, _ = http.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer wrong-key")
		err = sm.AuthenticateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid")

		// Request without API key
		req, _ = http.NewRequest("GET", "/", nil)
		err = sm.AuthenticateRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
	})

	t.Run("Rate Limiting", func(t *testing.T) {
		clientIP := "192.168.1.1"

		// First few requests should be allowed
		for i := 0; i < 5; i++ {
			err := sm.CheckRateLimit(clientIP)
			assert.NoError(t, err, "Request %d should be allowed", i+1)
		}

		// Should hit rate limit
		err := sm.CheckRateLimit(clientIP)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit")
	})

	t.Run("Host Validation", func(t *testing.T) {
		// Test with allowed hosts
		configWithHosts := config
		configWithHosts.AllowedHosts = []string{"localhost", "example.com"}
		smWithHosts := mcp.NewSecurityManager(configWithHosts)

		req, _ := http.NewRequest("GET", "/", nil)
		req.Host = "localhost"
		err := smWithHosts.ValidateHost(req)
		assert.NoError(t, err)

		req.Host = "api.example.com"
		err = smWithHosts.ValidateHost(req)
		assert.NoError(t, err) // Should allow subdomain

		req.Host = "evil.com"
		err = smWithHosts.ValidateHost(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})
}

func TestSecurityWithMCPServer(t *testing.T) {
	// Create MCP server with security enabled
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Port:        3100,
		LogLevel:    "error",
		Security: mcp.SecurityConfig{
			EnableAuth:      true,
			APIKey:         "secure-key-456",
			EnableRateLimit: true,
			RequestsPerMin:  30,
			BurstLimit:     10,
			ReadOnlyMode:   true,
			MaxQueryRows:   50,
		},
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	// Start server in background
	go func() {
		server.Start()
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)
	defer server.Stop()

	t.Run("Authentication Required", func(t *testing.T) {
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "initialize",
			"params":  map[string]interface{}{},
			"id":      1,
		}

		requestJSON, _ := json.Marshal(request)

		// Request without authentication
		resp, err := http.Post("http://localhost:3100/", "application/json", bytes.NewBuffer(requestJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Valid Authentication", func(t *testing.T) {
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "initialize",
			"params":  map[string]interface{}{},
			"id":      1,
		}

		requestJSON, _ := json.Marshal(request)

		// Create authenticated request
		req, err := http.NewRequest("POST", "http://localhost:3100/", bytes.NewBuffer(requestJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer secure-key-456")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.NotNil(t, response["result"])
	})

	t.Run("Query Security Validation", func(t *testing.T) {
		// Create database table first
		db := server.GetDatabase()
		require.NotNil(t, db)
		_, err := db.Raw("CREATE TABLE test_table (id INTEGER, name TEXT)").Exec(context.Background())
		require.NoError(t, err)

		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "tools/call",
			"params": map[string]interface{}{
				"name": "query",
				"arguments": map[string]interface{}{
					"sql":        "DELETE FROM test_table", // Should be blocked
					"parameters": []interface{}{},
				},
			},
			"id": 1,
		}

		requestJSON, _ := json.Marshal(request)

		// Create authenticated request
		req, err := http.NewRequest("POST", "http://localhost:3100/", bytes.NewBuffer(requestJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer secure-key-456")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Parse response
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Should get a tool result with security error
		result := response["result"].(map[string]interface{})
		content := result["content"].([]interface{})[0].(map[string]interface{})
		text := content["text"].(string)

		assert.Contains(t, text, "Security error")
		assert.Contains(t, text, "read-only mode")
	})
}

func TestRateLimiter(t *testing.T) {
	rl := mcp.NewRateLimiter(60, 10) // 60 requests per minute, 10 burst
	defer rl.Stop()

	clientIP := "192.168.1.100"

	t.Run("Burst Requests", func(t *testing.T) {
		// Should allow burst requests
		for i := 0; i < 10; i++ {
			allowed := rl.Allow(clientIP)
			assert.True(t, allowed, "Burst request %d should be allowed", i+1)
		}

		// Should deny the next request
		allowed := rl.Allow(clientIP)
		assert.False(t, allowed, "Request after burst should be denied")
	})

	t.Run("Different Clients", func(t *testing.T) {
		client1 := "192.168.1.101"
		client2 := "192.168.1.102"

		// Both clients should be allowed independently
		assert.True(t, rl.Allow(client1))
		assert.True(t, rl.Allow(client2))
	})
}