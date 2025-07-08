package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rediwo/redi-orm/database"
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
	"github.com/rediwo/redi-orm/rest"
	"github.com/rediwo/redi-orm/rest/types"
)

// TestBasicRESTOperations tests basic CRUD operations via REST API
func TestBasicRESTOperations(t *testing.T) {
	// Create test database
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
	err = db.LoadSchema(ctx, testSchema)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Sync schemas
	err = db.SyncSchemas(ctx)
	if err != nil {
		t.Fatalf("Failed to sync schemas: %v", err)
	}

	// Check registered models
	models := db.GetModels()
	t.Logf("Registered models: %v", models)

	// Create REST server
	config := rest.ServerConfig{
		Database: db,
		LogLevel: "error", // Reduce logging for tests
	}

	server, err := rest.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create REST server: %v", err)
	}
	defer server.Stop()

	// Create test server
	ts := httptest.NewServer(server.Router)
	defer ts.Close()

	// Test 1: Create a user
	t.Run("CreateUser", func(t *testing.T) {
		reqBody := map[string]any{
			"data": map[string]any{
				"name":  "John Doe",
				"email": "john@example.com",
			},
		}

		resp := makeRequest(t, ts, "POST", "/api/User", reqBody)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("Expected data to be a map, got %T", resp.Data)
		}

		if data["name"] != "John Doe" {
			t.Errorf("Expected name to be 'John Doe', got %v", data["name"])
		}

		if data["email"] != "john@example.com" {
			t.Errorf("Expected email to be 'john@example.com', got %v", data["email"])
		}
	})

	// Test 2: Get all users
	t.Run("GetAllUsers", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/User", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		users, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		if len(users) != 1 {
			t.Errorf("Expected 1 user, got %d", len(users))
		}
	})

	// Test 3: Get user by ID
	t.Run("GetUserByID", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/User/1", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("Expected data to be a map, got %T", resp.Data)
		}

		if data["id"] != float64(1) {
			t.Errorf("Expected id to be 1, got %v", data["id"])
		}
	})

	// Test 4: Update user
	t.Run("UpdateUser", func(t *testing.T) {
		reqBody := map[string]any{
			"data": map[string]any{
				"name": "Jane Doe",
			},
		}

		resp := makeRequest(t, ts, "PUT", "/api/User/1", reqBody)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("Expected data to be a map, got %T", resp.Data)
		}

		if data["name"] != "Jane Doe" {
			t.Errorf("Expected name to be 'Jane Doe', got %v", data["name"])
		}
	})

	// Test 5: Delete user
	t.Run("DeleteUser", func(t *testing.T) {
		resp := makeRequest(t, ts, "DELETE", "/api/User/1", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("Expected data to be a map, got %T", resp.Data)
		}

		if data["deleted"] != true {
			t.Errorf("Expected deleted to be true, got %v", data["deleted"])
		}
	})

	// Test 6: Verify user is deleted
	t.Run("VerifyUserDeleted", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/User/1", nil)

		if resp.Success {
			t.Fatalf("Expected error, got success")
		}

		if resp.Error.Code != "NOT_FOUND" {
			t.Errorf("Expected NOT_FOUND error, got %s", resp.Error.Code)
		}
	})
}

// TestComplexQueries tests complex filtering, sorting, and pagination
func TestComplexQueries(t *testing.T) {
	// Create test database
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
	err = db.LoadSchema(ctx, testSchema)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Sync schemas
	err = db.SyncSchemas(ctx)
	if err != nil {
		t.Fatalf("Failed to sync schemas: %v", err)
	}

	// Create test data
	createTestData(t, db)

	// Create REST server
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

	// Test 1: Filtering
	t.Run("Filtering", func(t *testing.T) {
		// Test age > 25
		resp := makeRequest(t, ts, "GET", `/api/User?where={"age":{"gt":25}}`, nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		users, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		if len(users) != 2 {
			t.Errorf("Expected 2 users with age > 25, got %d", len(users))
		}
	})

	// Test 2: Sorting
	t.Run("Sorting", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/User?sort=-age,name", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		users, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		// Check if sorted by age DESC
		firstUser := users[0].(map[string]any)
		if firstUser["age"].(float64) != 35 {
			t.Errorf("Expected first user to have age 35, got %v", firstUser["age"])
		}
	})

	// Test 3: Pagination
	t.Run("Pagination", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/User?page=1&limit=2", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		users, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		if len(users) != 2 {
			t.Errorf("Expected 2 users per page, got %d", len(users))
		}

		if resp.Pagination == nil {
			t.Fatal("Expected pagination metadata")
		}

		if resp.Pagination.Total != 3 {
			t.Errorf("Expected total 3 users, got %d", resp.Pagination.Total)
		}

		if resp.Pagination.Pages != 2 {
			t.Errorf("Expected 2 pages, got %d", resp.Pagination.Pages)
		}
	})

	// Test 4: Field selection
	t.Run("FieldSelection", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/User?select=id,name", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		users, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		firstUser := users[0].(map[string]any)
		if _, exists := firstUser["email"]; exists {
			t.Error("Expected email field to be excluded")
		}

		if _, exists := firstUser["name"]; !exists {
			t.Error("Expected name field to be included")
		}
	})
}

// TestRelations tests relation loading
func TestRelations(t *testing.T) {
	// Create test database
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
	err = db.LoadSchema(ctx, testSchema)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Sync schemas
	err = db.SyncSchemas(ctx)
	if err != nil {
		t.Fatalf("Failed to sync schemas: %v", err)
	}

	// Create test data with relations
	createTestDataWithRelations(t, db)

	// Create REST server
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

	// Test 1: Include posts
	t.Run("IncludePosts", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/User/1?include=posts", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		user, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("Expected data to be a map, got %T", resp.Data)
		}

		posts, ok := user["posts"].([]any)
		if !ok {
			t.Fatalf("Expected posts to be an array, got %T", user["posts"])
		}

		if len(posts) != 2 {
			t.Errorf("Expected 2 posts, got %d", len(posts))
		}
	})

	// Test 2: Include author
	t.Run("IncludeAuthor", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/Post?include=author", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		posts, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		firstPost := posts[0].(map[string]any)
		author, ok := firstPost["author"].(map[string]any)
		if !ok {
			t.Fatalf("Expected author to be a map, got %T", firstPost["author"])
		}

		if author["name"] != "Alice" {
			t.Errorf("Expected author name to be 'Alice', got %v", author["name"])
		}
	})
}

// TestBatchOperations tests batch create operations
func TestBatchOperations(t *testing.T) {
	// Create test database
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
	err = db.LoadSchema(ctx, testSchema)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Sync schemas
	err = db.SyncSchemas(ctx)
	if err != nil {
		t.Fatalf("Failed to sync schemas: %v", err)
	}

	// Create REST server
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

	// Test batch create
	t.Run("BatchCreate", func(t *testing.T) {
		reqBody := map[string]any{
			"data": []any{
				map[string]any{
					"name":  "User 1",
					"email": "user1@example.com",
					"age":   25,
				},
				map[string]any{
					"name":  "User 2",
					"email": "user2@example.com",
					"age":   30,
				},
				map[string]any{
					"name":  "User 3",
					"email": "user3@example.com",
					"age":   35,
				},
			},
		}

		resp := makeRequest(t, ts, "POST", "/api/User/batch", reqBody)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		data, ok := resp.Data.(map[string]any)
		if !ok {
			t.Fatalf("Expected data to be a map, got %T", resp.Data)
		}

		if data["created"].(float64) != 3 {
			t.Errorf("Expected 3 created records, got %v", data["created"])
		}
	})

	// Verify batch create
	t.Run("VerifyBatchCreate", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/User", nil)

		if !resp.Success {
			t.Fatalf("Expected success, got error: %s", resp.Error.Message)
		}

		users, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("Expected data to be an array, got %T", resp.Data)
		}

		if len(users) != 3 {
			t.Errorf("Expected 3 users, got %d", len(users))
		}
	})
}

// TestErrorHandling tests error responses
func TestErrorHandling(t *testing.T) {
	// Create test database
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
	err = db.LoadSchema(ctx, testSchema)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	// Sync schemas
	err = db.SyncSchemas(ctx)
	if err != nil {
		t.Fatalf("Failed to sync schemas: %v", err)
	}

	// Create REST server
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

	// Test 1: Not found
	t.Run("NotFound", func(t *testing.T) {
		resp := makeRequest(t, ts, "GET", "/api/User/999", nil)

		if resp.Success {
			t.Fatal("Expected error, got success")
		}

		if resp.Error.Code != "NOT_FOUND" {
			t.Errorf("Expected NOT_FOUND error, got %s", resp.Error.Code)
		}
	})

	// Test 2: Invalid request body
	t.Run("InvalidRequestBody", func(t *testing.T) {
		resp := makeRequestRaw(t, ts, "POST", "/api/User", []byte("invalid json"))

		if resp.Success {
			t.Fatal("Expected error, got success")
		}

		if resp.Error.Code != "INVALID_REQUEST" {
			t.Errorf("Expected INVALID_REQUEST error, got %s", resp.Error.Code)
		}
	})

	// Test 3: Missing required fields
	t.Run("MissingRequiredFields", func(t *testing.T) {
		reqBody := map[string]any{
			"data": map[string]any{
				"name": "No Email User",
				// email is missing
			},
		}

		resp := makeRequest(t, ts, "POST", "/api/User", reqBody)

		if resp.Success {
			t.Fatal("Expected error, got success")
		}

		if resp.Error.Code != "CREATE_ERROR" {
			t.Errorf("Expected CREATE_ERROR, got %s", resp.Error.Code)
		}
	})

	// Test 4: Method not allowed
	t.Run("MethodNotAllowed", func(t *testing.T) {
		// Use GET on a POST-only endpoint
		resp := makeRequest(t, ts, "GET", "/api/User/batch", nil)

		if resp.Success {
			t.Fatal("Expected error, got success")
		}

		// The server returns 404 for unmatched routes
		if resp.Error == nil {
			t.Fatal("Expected error response")
		}
	})
}

// Helper functions

func makeRequest(t *testing.T, ts *httptest.Server, method, path string, body any) *types.Response {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	return makeRequestRaw(t, ts, method, path, bodyReader)
}

func makeRequestRaw(t *testing.T, ts *httptest.Server, method, path string, body any) *types.Response {
	var bodyReader io.Reader
	switch v := body.(type) {
	case []byte:
		bodyReader = bytes.NewReader(v)
	case io.Reader:
		bodyReader = v
	}

	req, err := http.NewRequest(method, ts.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Connection-Name", "default")

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
		// If we can't unmarshal, return an error response
		return &types.Response{
			Success: false,
			Error: &types.ErrorDetail{
				Code:    "PARSE_ERROR",
				Message: string(respBody),
			},
		}
	}

	return &response
}

func createTestData(t *testing.T, db database.Database) {
	ctx := context.Background()

	users := []map[string]any{
		{"name": "Alice", "email": "alice@example.com", "age": 25},
		{"name": "Bob", "email": "bob@example.com", "age": 30},
		{"name": "Charlie", "email": "charlie@example.com", "age": 35},
	}

	for _, user := range users {
		_, err := db.Model("User").Insert(user).Exec(ctx)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}
}

func createTestDataWithRelations(t *testing.T, db database.Database) {
	ctx := context.Background()

	// Create user
	result, err := db.Model("User").Insert(map[string]any{
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   25,
	}).Exec(ctx)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	userID := result.LastInsertID

	// Create posts
	posts := []map[string]any{
		{
			"title":     "First Post",
			"content":   "This is my first post",
			"published": true,
			"authorId":  userID,
		},
		{
			"title":     "Second Post",
			"content":   "This is my second post",
			"published": false,
			"authorId":  userID,
		},
	}

	for _, post := range posts {
		_, err := db.Model("Post").Insert(post).Exec(ctx)
		if err != nil {
			t.Fatalf("Failed to create post: %v", err)
		}
	}
}

const testSchema = `
model User {
  id    Int     @id @default(autoincrement())
  name  String
  email String  @unique
  age   Int?
  posts Post[]
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  published Boolean  @default(false)
  authorId  Int
  author    User     @relation(fields: [authorId], references: [id])
  createdAt DateTime @default(now())
}

model Tag {
  id    Int    @id @default(autoincrement())
  name  String @unique
  posts Post[]
}

model PostTag {
  postId Int
  tagId  Int
  post   Post @relation(fields: [postId], references: [id])
  tag    Tag  @relation(fields: [tagId], references: [id])
  
  @@id([postId, tagId])
}
`
