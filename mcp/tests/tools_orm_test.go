package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/mcp"
	"github.com/rediwo/redi-orm/schema"
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
)

func setupORMTestServer(t *testing.T) (*mcp.Server, func()) {
	// Create in-memory database
	db, err := database.NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Connect to database
	ctx := context.Background()
	if err := db.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Create test schemas
	userSchema := schema.New("User").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "email", Type: schema.FieldTypeString, Unique: true}).
		AddField(schema.Field{Name: "age", Type: schema.FieldTypeInt, Nullable: true}).
		AddField(schema.Field{Name: "active", Type: schema.FieldTypeBool, Default: true})

	postSchema := schema.New("Post").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "title", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "content", Type: schema.FieldTypeString, Nullable: true}).
		AddField(schema.Field{Name: "authorId", Type: schema.FieldTypeInt}).
		AddField(schema.Field{Name: "published", Type: schema.FieldTypeBool, Default: false})

	// Add relations
	userSchema.AddRelation("posts", schema.Relation{
		Type:       schema.RelationOneToMany,
		Model:      "Post",
		ForeignKey: "authorId",
		References: "id",
	})

	postSchema.AddRelation("author", schema.Relation{
		Type:       schema.RelationManyToOne,
		Model:      "User",
		ForeignKey: "authorId",
		References: "id",
	})

	// Register schemas
	if err := db.RegisterSchema("User", userSchema); err != nil {
		t.Fatalf("Failed to register User schema: %v", err)
	}
	if err := db.RegisterSchema("Post", postSchema); err != nil {
		t.Fatalf("Failed to register Post schema: %v", err)
	}

	// Sync schemas to create tables
	if err := db.SyncSchemas(ctx); err != nil {
		t.Fatalf("Failed to sync schemas: %v", err)
	}

	// Create MCP server
	config := mcp.ServerConfig{
		DatabaseURI:  "sqlite://:memory:",
		ReadOnly:     false,
		MaxQueryRows: 1000,
		LogLevel:     "debug",
	}

	// Create server (it will create its own logger)
	server, err := mcp.NewServer(config)
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}
	
	// Set the database we created
	server.SetDatabase(db)

	// Register schemas with server
	server.RegisterSchema("User", userSchema)
	server.RegisterSchema("Post", postSchema)

	cleanup := func() {
		db.Close()
	}

	return server, cleanup
}

func TestORMFindMany(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// First, create some test data
	createData := []struct {
		name  string
		email string
		age   int
	}{
		{"Alice", "alice@example.com", 25},
		{"Bob", "bob@example.com", 30},
		{"Charlie", "charlie@example.com", 35},
	}

	for _, user := range createData {
		args := map[string]interface{}{
			"model": "User",
			"data": map[string]interface{}{
				"name":  user.name,
				"email": user.email,
				"age":   user.age,
			},
		}
		argsJSON, _ := json.Marshal(args)
		_, err := server.CallTool(ctx, "data.create", argsJSON)
		if err != nil {
			t.Fatalf("Failed to create user %s: %v", user.name, err)
		}
	}

	// Test findMany with no filters
	t.Run("FindAll", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.findMany", argsJSON)
		if err != nil {
			t.Fatalf("Failed to find users: %v", err)
		}

		// Parse result
		var users []map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &users); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if len(users) != 3 {
			t.Errorf("Expected 3 users, got %d", len(users))
		}
	})

	// Test findMany with where clause
	t.Run("FindWithWhere", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"where": map[string]interface{}{
				"age": map[string]interface{}{"gt": 25},
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.findMany", argsJSON)
		if err != nil {
			t.Fatalf("Failed to find users: %v", err)
		}

		var users []map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &users); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if len(users) != 2 {
			t.Errorf("Expected 2 users with age > 25, got %d", len(users))
		}
	})

	// Test findMany with orderBy
	t.Run("FindWithOrderBy", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"orderBy": map[string]interface{}{
				"age": "desc",
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.findMany", argsJSON)
		if err != nil {
			t.Fatalf("Failed to find users: %v", err)
		}

		var users []map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &users); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if len(users) > 0 {
			firstAge := int(users[0]["age"].(float64))
			if firstAge != 35 {
				t.Errorf("Expected first user to have age 35, got %d", firstAge)
			}
		}
	})

	// Test findMany with take and skip
	t.Run("FindWithPagination", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"take":  2,
			"skip":  1,
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.findMany", argsJSON)
		if err != nil {
			t.Fatalf("Failed to find users: %v", err)
		}

		var users []map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &users); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if len(users) != 2 {
			t.Errorf("Expected 2 users with pagination, got %d", len(users))
		}
	})
}

func TestORMFindUnique(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	createArgs := map[string]interface{}{
		"model": "User",
		"data": map[string]interface{}{
			"name":  "TestUser",
			"email": "test@example.com",
			"age":   25,
		},
	}
	createArgsJSON, _ := json.Marshal(createArgs)
	createResult, err := server.CallTool(ctx, "data.create", createArgsJSON)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Parse created user to get ID
	var createdUser map[string]interface{}
	if err := json.Unmarshal([]byte(createResult.Content[0].Text), &createdUser); err != nil {
		t.Fatalf("Failed to parse created user: %v", err)
	}

	// Test findUnique by email
	t.Run("FindByEmail", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"where": map[string]interface{}{
				"email": "test@example.com",
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.findUnique", argsJSON)
		if err != nil {
			t.Fatalf("Failed to find user: %v", err)
		}

		var user map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &user); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if user["email"] != "test@example.com" {
			t.Errorf("Expected email test@example.com, got %v", user["email"])
		}
	})

	// Test findUnique with non-existent record
	t.Run("FindNonExistent", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"where": map[string]interface{}{
				"email": "nonexistent@example.com",
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.findUnique", argsJSON)
		if err != nil {
			t.Fatalf("Failed to find user: %v", err)
		}

		if result.Content[0].Text != "null" {
			t.Errorf("Expected null for non-existent record, got %s", result.Content[0].Text)
		}
	})
}

func TestORMCreate(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Test create with all fields
	t.Run("CreateWithAllFields", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"data": map[string]interface{}{
				"name":   "John Doe",
				"email":  "john@example.com",
				"age":    30,
				"active": false,
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.create", argsJSON)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		var user map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &user); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if user["name"] != "John Doe" {
			t.Errorf("Expected name 'John Doe', got %v", user["name"])
		}
		if user["email"] != "john@example.com" {
			t.Errorf("Expected email 'john@example.com', got %v", user["email"])
		}
	})

	// Test create with default values
	t.Run("CreateWithDefaults", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"data": map[string]interface{}{
				"name":  "Jane Doe",
				"email": "jane@example.com",
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.create", argsJSON)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		var user map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &user); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		// Check default value for active field
		if user["active"] != true && user["active"] != 1.0 {
			t.Errorf("Expected active to be true (default), got %v", user["active"])
		}
	})
}

func TestORMUpdate(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	createArgs := map[string]interface{}{
		"model": "User",
		"data": map[string]interface{}{
			"name":  "UpdateTest",
			"email": "update@example.com",
			"age":   25,
		},
	}
	createArgsJSON, _ := json.Marshal(createArgs)
	_, err := server.CallTool(ctx, "data.create", createArgsJSON)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test update
	t.Run("UpdateUser", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"where": map[string]interface{}{
				"email": "update@example.com",
			},
			"data": map[string]interface{}{
				"age": 30,
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.update", argsJSON)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		var updateResult map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &updateResult); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if updateResult["count"].(float64) < 1 {
			t.Errorf("Expected at least 1 row updated, got %v", updateResult["count"])
		}

		// Verify the update
		findArgs := map[string]interface{}{
			"model": "User",
			"where": map[string]interface{}{
				"email": "update@example.com",
			},
		}
		findArgsJSON, _ := json.Marshal(findArgs)
		findResult, _ := server.CallTool(ctx, "data.findUnique", findArgsJSON)

		var user map[string]interface{}
		json.Unmarshal([]byte(findResult.Content[0].Text), &user)

		if int(user["age"].(float64)) != 30 {
			t.Errorf("Expected age to be updated to 30, got %v", user["age"])
		}
	})
}

func TestORMDelete(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test user
	createArgs := map[string]interface{}{
		"model": "User",
		"data": map[string]interface{}{
			"name":  "DeleteTest",
			"email": "delete@example.com",
		},
	}
	createArgsJSON, _ := json.Marshal(createArgs)
	_, err := server.CallTool(ctx, "data.create", createArgsJSON)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Test delete
	t.Run("DeleteUser", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"where": map[string]interface{}{
				"email": "delete@example.com",
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.delete", argsJSON)
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		var deleteResult map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &deleteResult); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		// Verify deletion
		findArgs := map[string]interface{}{
			"model": "User",
			"where": map[string]interface{}{
				"email": "delete@example.com",
			},
		}
		findArgsJSON, _ := json.Marshal(findArgs)
		findResult, _ := server.CallTool(ctx, "data.findUnique", findArgsJSON)

		if findResult.Content[0].Text != "null" {
			t.Errorf("Expected user to be deleted, but still found")
		}
	})
}

func TestORMCount(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create test users
	users := []map[string]interface{}{
		{"name": "User1", "email": "user1@example.com", "age": 20},
		{"name": "User2", "email": "user2@example.com", "age": 30},
		{"name": "User3", "email": "user3@example.com", "age": 40},
	}

	for _, user := range users {
		args := map[string]interface{}{
			"model": "User",
			"data":  user,
		}
		argsJSON, _ := json.Marshal(args)
		_, err := server.CallTool(ctx, "data.create", argsJSON)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// Test count all
	t.Run("CountAll", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.count", argsJSON)
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}

		var count int
		if err := json.Unmarshal([]byte(result.Content[0].Text), &count); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if count != 3 {
			t.Errorf("Expected count 3, got %d", count)
		}
	})

	// Test count with filter
	t.Run("CountWithFilter", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"where": map[string]interface{}{
				"age": map[string]interface{}{"gte": 30},
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.count", argsJSON)
		if err != nil {
			t.Fatalf("Failed to count users: %v", err)
		}

		var count int
		if err := json.Unmarshal([]byte(result.Content[0].Text), &count); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if count != 2 {
			t.Errorf("Expected count 2 for age >= 30, got %d", count)
		}
	})
}

func TestORMAggregate(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create test users
	users := []map[string]interface{}{
		{"name": "User1", "email": "user1@test.com", "age": 20},
		{"name": "User2", "email": "user2@test.com", "age": 30},
		{"name": "User3", "email": "user3@test.com", "age": 40},
		{"name": "User4", "email": "user4@test.com", "age": 30},
	}

	for _, user := range users {
		args := map[string]interface{}{
			"model": "User",
			"data":  user,
		}
		argsJSON, _ := json.Marshal(args)
		_, err := server.CallTool(ctx, "data.create", argsJSON)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// Test aggregate functions
	t.Run("AggregateAge", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"count": true,
			"avg": map[string]bool{"age": true},
			"sum": map[string]bool{"age": true},
			"min": map[string]bool{"age": true},
			"max": map[string]bool{"age": true},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.aggregate", argsJSON)
		if err != nil {
			t.Fatalf("Failed to aggregate: %v", err)
		}

		var agg map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &agg); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		// Check count
		if count, ok := agg["_count"].(float64); !ok || count != 4 {
			t.Errorf("Expected count 4, got %v", agg["_count"])
		}

		// Check average (should be 30)
		if avg, ok := agg["_avg_age"].(float64); !ok || avg != 30 {
			t.Errorf("Expected average age 30, got %v", agg["_avg_age"])
		}

		// Check sum (should be 120)
		if sum, ok := agg["_sum_age"].(float64); !ok || sum != 120 {
			t.Errorf("Expected sum of ages 120, got %v", agg["_sum_age"])
		}

		// Check min (should be 20)
		if min, ok := agg["_min_age"].(float64); !ok || min != 20 {
			t.Errorf("Expected min age 20, got %v", agg["_min_age"])
		}

		// Check max (should be 40)
		if max, ok := agg["_max_age"].(float64); !ok || max != 40 {
			t.Errorf("Expected max age 40, got %v", agg["_max_age"])
		}
	})
}

func TestORMOperators(t *testing.T) {
	server, cleanup := setupORMTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Create test users
	users := []map[string]interface{}{
		{"name": "Alice", "email": "alice@example.com", "age": 25},
		{"name": "Bob", "email": "bob@test.com", "age": 30},
		{"name": "Charlie", "email": "charlie@example.com", "age": 35},
		{"name": "David", "email": "david@test.com", "age": 25},
	}

	for _, user := range users {
		args := map[string]interface{}{
			"model": "User",
			"data":  user,
		}
		argsJSON, _ := json.Marshal(args)
		_, err := server.CallTool(ctx, "data.create", argsJSON)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// Test contains operator
	t.Run("ContainsOperator", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"where": map[string]interface{}{
				"email": map[string]interface{}{"contains": "example"},
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.findMany", argsJSON)
		if err != nil {
			t.Fatalf("Failed to find users: %v", err)
		}

		var users []map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &users); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if len(users) != 2 {
			t.Errorf("Expected 2 users with 'example' in email, got %d", len(users))
		}
	})

	// Test startsWith operator
	t.Run("StartsWithOperator", func(t *testing.T) {
		args := map[string]interface{}{
			"model": "User",
			"where": map[string]interface{}{
				"name": map[string]interface{}{"startsWith": "Ch"},
			},
		}
		argsJSON, _ := json.Marshal(args)

		result, err := server.CallTool(ctx, "data.findMany", argsJSON)
		if err != nil {
			t.Fatalf("Failed to find users: %v", err)
		}

		var users []map[string]interface{}
		if err := json.Unmarshal([]byte(result.Content[0].Text), &users); err != nil {
			t.Fatalf("Failed to parse result: %v", err)
		}

		if len(users) != 1 || users[0]["name"] != "Charlie" {
			t.Errorf("Expected 1 user starting with 'Ch', got %d", len(users))
		}
	})
}