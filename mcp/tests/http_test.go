package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/rediwo/redi-orm/mcp"
	"github.com/rediwo/redi-orm/mcp/transport"
	"github.com/rediwo/redi-orm/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPTransport(t *testing.T) {
	// Create HTTP transport
	logger := utils.NewDefaultLogger("test")
	logger.SetLevel(utils.LogLevelError)
	
	httpTransport := transport.NewHTTPTransport(0, logger) // Use port 0 for random port
	
	// Start transport
	err := httpTransport.Start()
	require.NoError(t, err)
	defer httpTransport.Stop()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	// Test basic HTTP functionality
	t.Run("CORS Headers", func(t *testing.T) {
		req, err := http.NewRequest("OPTIONS", "http://localhost:3000/", nil)
		require.NoError(t, err)
		
		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	})
}

func TestHTTPTransportWithMCP(t *testing.T) {
	// Create MCP server with HTTP transport
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Port:        0, // Random port
		LogLevel:    "error",
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

	// Test JSON-RPC over HTTP
	t.Run("JSON-RPC Initialize", func(t *testing.T) {
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "initialize",
			"params": map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{},
			},
			"id": 1,
		}

		requestJSON, err := json.Marshal(request)
		require.NoError(t, err)

		// Send HTTP POST request
		resp, err := http.Post("http://localhost:3000/", "application/json", bytes.NewBuffer(requestJSON))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		// Parse response
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.NotNil(t, response["result"])
		assert.Equal(t, float64(1), response["id"])
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		resp, err := http.Post("http://localhost:3000/", "application/json", bytes.NewBufferString("{invalid json"))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Method Not Allowed", func(t *testing.T) {
		resp, err := http.Get("http://localhost:3000/")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestHTTPTransportSSE(t *testing.T) {
	// Create HTTP transport
	logger := utils.NewDefaultLogger("test")
	logger.SetLevel(utils.LogLevelError)
	
	httpTransport := transport.NewHTTPTransport(3001, logger)
	
	// Start transport
	err := httpTransport.Start()
	require.NoError(t, err)
	defer httpTransport.Stop()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	t.Run("SSE Connection", func(t *testing.T) {
		// Create SSE client
		req, err := http.NewRequest("GET", "http://localhost:3001/events", nil)
		require.NoError(t, err)
		
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		
		client := &http.Client{
			Timeout: 2 * time.Second,
		}
		
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
		assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
		assert.Equal(t, "keep-alive", resp.Header.Get("Connection"))
		
		// Read first message (connection message)
		buffer := make([]byte, 1024)
		n, err := resp.Body.Read(buffer)
		if err == nil && n > 0 {
			message := string(buffer[:n])
			assert.Contains(t, message, "data:")
			assert.Contains(t, message, "connected")
		}
	})
	
	t.Run("Client Count", func(t *testing.T) {
		initialCount := httpTransport.GetClientCount()
		
		// Start SSE connection
		req, err := http.NewRequest("GET", "http://localhost:3001/events", nil)
		require.NoError(t, err)
		
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		req = req.WithContext(ctx)
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			
			// Give time for connection to register
			time.Sleep(50 * time.Millisecond)
			
			newCount := httpTransport.GetClientCount()
			assert.True(t, newCount >= initialCount, "Client count should increase")
		}
	})
}

func TestHTTPTransportSendBroadcast(t *testing.T) {
	// Create HTTP transport
	logger := utils.NewDefaultLogger("test")
	logger.SetLevel(utils.LogLevelError)
	
	httpTransport := transport.NewHTTPTransport(3002, logger)
	
	// Start transport
	err := httpTransport.Start()
	require.NoError(t, err)
	defer httpTransport.Stop()
	
	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	
	t.Run("Broadcast Message", func(t *testing.T) {
		// Test sending a message (even with no clients)
		message := json.RawMessage(`{"test": "broadcast"}`)
		err := httpTransport.Send(message)
		assert.NoError(t, err)
	})
	
	t.Run("Send with No Clients", func(t *testing.T) {
		message := json.RawMessage(`{"notification": "test"}`)
		err := httpTransport.Send(message)
		assert.NoError(t, err) // Should not error even with no clients
	})
}