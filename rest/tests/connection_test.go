package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rediwo/redi-orm/database"
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
	"github.com/rediwo/redi-orm/rest"
	"github.com/rediwo/redi-orm/rest/types"
)

// TestConnectionManagement tests database connection management via REST API
func TestConnectionManagement(t *testing.T) {
	// Create REST server without pre-connected database
	config := rest.ServerConfig{
		LogLevel: "error",
	}

	server, err := rest.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create REST server: %v", err)
	}
	defer server.Stop()

	// Create test server
	ts := httptest.NewServer(server.Router)
	defer ts.Close()

	// Test 1: List connections (should be empty initially)
	t.Run("ListConnectionsEmpty", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/connections", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		connections, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		if len(connections) != 0 {
			t.Errorf("Expected 0 connections, got %d", len(connections))
		}
	})

	// Test 2: Connect to database
	t.Run("ConnectDatabase", func(t *testing.T) {
		reqBody := map[string]any{
			"uri":    "sqlite://:memory:",
			"name":   "test-db",
			"schema": testConnectionSchema,
		}

		resp := makeRequest(t, ts, "POST", "/api/connections/connect", reqBody)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("Expected data to be a map, got %T", resp.Data)
		}

		if data["name"] != "test-db" {
			t.Errorf("Expected name to be 'test-db', got %v", data["name"])
		}

		if data["status"] != "connected" {
			t.Errorf("Expected status to be 'connected', got %v", data["status"])
		}
	})

	// Test 3: List connections (should have one connection)
	t.Run("ListConnectionsWithOne", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/connections", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		connections, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		if len(connections) != 1 {
			t.Errorf("Expected 1 connection, got %d", len(connections))
		}

		conn := connections[0].(map[string]any)
		if conn["name"] != "test-db" {
			t.Errorf("Expected connection name to be 'test-db', got %v", conn["name"])
		}
	})

	// Test 4: Use the connected database
	t.Run("UseConnectedDatabase", func(t *testing.T) {
		// Create a product using the test-db connection
		reqBody := map[string]any{
			"data": map[string]any{
				"name":  "Test Product",
				"price": 29.99,
			},
		}

		req, _ := createRequestWithHeaders(t, ts.URL+"/api/Product", "POST", reqBody)
		req.Header.Set("X-Connection-Name", "test-db")

		resp := executeRequest(t, req)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("Expected data to be a map, got %T", resp.Data)
		}

		if data["name"] != "Test Product" {
			t.Errorf("Expected name to be 'Test Product', got %v", data["name"])
		}
	})

	// Test 5: Connect with duplicate name (should fail)
	t.Run("ConnectDuplicateName", func(t *testing.T) {
		reqBody := map[string]any{
			"uri":  "sqlite://:memory:",
			"name": "test-db",
		}

		resp := makeRequest(t, ts, "POST", "/api/connections/connect", reqBody)

		if resp.Success {
			t.Fatal("Expected error, got success")
		}

		if resp.Error.Code != "CONNECTION_EXISTS" {
			t.Errorf("Expected CONNECTION_EXISTS error, got %s", resp.Error.Code)
		}
	})

	// Test 6: Disconnect database
	t.Run("DisconnectDatabase", func(t *testing.T) {
		resp := makeRequest(t, ts, "DELETE", "/api/connections/disconnect?name=test-db", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("Expected data to be a map, got %T", resp.Data)
		}

		if data["status"] != "disconnected" {
			t.Errorf("Expected status to be 'disconnected', got %v", data["status"])
		}
	})

	// Test 7: Try to use disconnected database
	t.Run("UseDisconnectedDatabase", func(t *testing.T) {
		req, _ := createRequestWithHeaders(t, ts.URL+"/api/Product", "GET", nil)
		req.Header.Set("X-Connection-Name", "test-db")

		resp := executeRequest(t, req)

		if resp.Success {
			t.Fatal("Expected error, got success")
		}

		if resp.Error.Code != "NO_CONNECTION" {
			t.Errorf("Expected NO_CONNECTION error, got %s", resp.Error.Code)
		}
	})
}

// TestMultipleConnections tests handling multiple database connections
func TestMultipleConnections(t *testing.T) {
	// Create REST server
	config := rest.ServerConfig{
		LogLevel: "error",
	}

	server, err := rest.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create REST server: %v", err)
	}
	defer server.Stop()

	// Create test server
	ts := httptest.NewServer(server.Router)
	defer ts.Close()

	// Connect to multiple databases
	databases := []struct {
		name   string
		uri    string
		schema string
	}{
		{
			name:   "db1",
			uri:    "sqlite://:memory:",
			schema: testConnectionSchema,
		},
		{
			name:   "db2",
			uri:    "sqlite://:memory:",
			schema: testConnectionSchema,
		},
		{
			name:   "db3",
			uri:    "sqlite://:memory:",
			schema: testConnectionSchema,
		},
	}

	// Connect all databases
	for _, db := range databases {
		t.Run("Connect_"+db.name, func(t *testing.T) {
			reqBody := map[string]any{
				"uri":    db.uri,
				"name":   db.name,
				"schema": db.schema,
			}

			resp := makeRequest(t, ts, "POST", "/api/connections/connect", reqBody)

			if !resp.Success {
				t.Fatalf("Failed to connect %s: %s", db.name, resp.Error.Message)
			}
		})
	}

	// Create different data in each database
	for i, db := range databases {
		t.Run("CreateData_"+db.name, func(t *testing.T) {
			reqBody := map[string]any{
				"data": map[string]any{
					"name":  fmt.Sprintf("Product from %s", db.name),
					"price": float64((i + 1) * 10),
				},
			}

			req, _ := createRequestWithHeaders(t, ts.URL+"/api/Product", "POST", reqBody)
			req.Header.Set("X-Connection-Name", db.name)

			resp := executeRequest(t, req)

			if !resp.Success {
				t.Fatalf("Failed to create product in %s: %s", db.name, resp.Error.Message)
			}
		})
	}

	// Verify data isolation between databases
	for _, db := range databases {
		t.Run("VerifyData_"+db.name, func(t *testing.T) {
			req, _ := createRequestWithHeaders(t, ts.URL+"/api/Product", "GET", nil)
			req.Header.Set("X-Connection-Name", db.name)

			resp := executeRequest(t, req)

			if !resp.Success {
				t.Fatalf("Failed to get products from %s: %s", db.name, resp.Error.Message)
			}

			products, ok := resp.Data.([]any)
			if !ok {
				t.Fatalf("Expected data to be an array, got %T", resp.Data)
			}

			if len(products) != 1 {
				t.Errorf("Expected 1 product in %s, got %d", db.name, len(products))
			}

			product := products[0].(map[string]any)
			expectedName := fmt.Sprintf("Product from %s", db.name)
			if product["name"] != expectedName {
				t.Errorf("Expected product name to be '%s', got %v", expectedName, product["name"])
			}
		})
	}

	// List all connections
	t.Run("ListAllConnections", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/connections", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		connections, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		if len(connections) != 3 {
			t.Errorf("Expected 3 connections, got %d", len(connections))
		}
	})
}

// TestDefaultConnection tests using the default connection
func TestDefaultConnection(t *testing.T) {
	// Create database
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Connect to database
	ctx := context.Background()
	err = db.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Load schema
	err = db.LoadSchema(ctx, testConnectionSchema)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Sync schemas
	err = db.SyncSchemas(ctx)
	if err != nil {
		t.Fatalf("Failed to sync schemas: %v", err)
	}

	// Create REST server with default database
	config := rest.ServerConfig{
		Database: db,
		LogLevel: "error",
	}

	server, err := rest.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create REST server: %v", err)
	}
	defer server.Stop()

	// Create test server
	ts := httptest.NewServer(server.Router)
	defer ts.Close()

	// Test using default connection (no X-Connection-Name header)
	t.Run("UseDefaultConnection", func(t *testing.T) {
		reqBody := map[string]any{
			"data": map[string]any{
				"name":  "Default Product",
				"price": 19.99,
			},
		}

		// Don't set X-Connection-Name header
		resp := makeRequest(t, ts, "POST", "/api/Product", reqBody)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("Expected data to be a map, got %T", resp.Data)
		}

		if data["name"] != "Default Product" {
			t.Errorf("Expected name to be 'Default Product', got %v", data["name"])
		}
	})

	// Test listing connections (should show default)
	t.Run("ListConnectionsWithDefault", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/connections", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		connections, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		if len(connections) != 1 {
			t.Errorf("Expected 1 connection, got %d", len(connections))
		}

		conn := connections[0].(map[string]any)
		if conn["name"] != "default" {
			t.Errorf("Expected connection name to be 'default', got %v", conn["name"])
		}
	})
}

const testConnectionSchema = `
model Product {
  id    Int    @id @default(autoincrement())
  name  String
  price Float
}
`

// Helper functions for connection tests
func createRequestWithHeaders(t *testing.T, url, method string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func executeRequest(t *testing.T, req *http.Request) *types.Response {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	var response types.Response
	err = json.Unmarshal(respBody, &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	return &response
}
