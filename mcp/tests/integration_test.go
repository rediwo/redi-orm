package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rediwo/redi-orm/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPWithRealDatabase(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := mcp.ServerConfig{
		DatabaseURI:  "sqlite://" + dbPath,
		Transport:    "stdio",
		LogLevel:     "error",
		MaxQueryRows: 10,
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("ListTables", func(t *testing.T) {
		// Call list_tables tool
		args := json.RawMessage(`{}`)
		result, err := server.CallTool(ctx, "list_tables", args)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Content, 1)
		assert.Equal(t, "text", result.Content[0].Type)
	})

	t.Run("QueryExecution", func(t *testing.T) {
		// First create a table
		db := server.GetDatabase()
		require.NotNil(t, db)

		_, err := db.Raw("CREATE TABLE test_users (id INTEGER PRIMARY KEY, name TEXT)").Exec(ctx)
		require.NoError(t, err)

		_, err = db.Raw("INSERT INTO test_users (name) VALUES (?)", "Alice").Exec(ctx)
		require.NoError(t, err)

		// Now query through MCP
		args := json.RawMessage(`{
			"sql": "SELECT * FROM test_users",
			"parameters": []
		}`)
		
		result, err := server.CallTool(ctx, "query", args)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		
		// Verify result contains data
		assert.Contains(t, result.Content[0].Text, "Alice")
	})

	t.Run("CountRecords", func(t *testing.T) {
		args := json.RawMessage(`{
			"table": "test_users"
		}`)
		
		result, err := server.CallTool(ctx, "count_records", args)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, `"count": 1`)
	})

	t.Run("InspectSchema", func(t *testing.T) {
		args := json.RawMessage(`{
			"table": "test_users"
		}`)
		
		result, err := server.CallTool(ctx, "inspect_schema", args)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, "columns")
	})
}

func TestMCPWithSchema(t *testing.T) {
	// Create a temporary schema file
	tmpDir := t.TempDir()
	schemaPath := filepath.Join(tmpDir, "schema.prisma")
	
	schemaContent := `
model User {
  id    Int     @id @default(autoincrement())
  email String  @unique
  name  String?
  posts Post[]
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  published Boolean  @default(false)
  authorId  Int
  author    User     @relation(fields: [authorId], references: [id])
}
`
	
	err := os.WriteFile(schemaPath, []byte(schemaContent), 0644)
	require.NoError(t, err)

	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		SchemaPath:  schemaPath,
		Transport:   "stdio",
		LogLevel:    "error",
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("ListResources", func(t *testing.T) {
		resources, err := server.ListResources(ctx)
		assert.NoError(t, err)
		
		// Check for model resources
		modelURIs := make(map[string]bool)
		for _, r := range resources {
			if r.URI == "model://User" || r.URI == "model://Post" {
				modelURIs[r.URI] = true
			}
		}
		
		assert.True(t, modelURIs["model://User"], "User model resource not found")
		assert.True(t, modelURIs["model://Post"], "Post model resource not found")
	})

	t.Run("ReadModelResource", func(t *testing.T) {
		// Test model details (JSON format)
		content, err := server.ReadResource(ctx, "model://User")
		assert.NoError(t, err)
		assert.NotNil(t, content)
		assert.Equal(t, "application/json", content.MimeType)
		
		// Parse JSON to verify structure
		var modelData map[string]interface{}
		err = json.Unmarshal([]byte(content.Text), &modelData)
		assert.NoError(t, err)
		assert.Equal(t, "User", modelData["name"])
		assert.Contains(t, modelData, "fields")
		assert.Contains(t, modelData, "relations")
		
		// Test model schema (Prisma format)
		schemaContent, err := server.ReadResource(ctx, "model://User/schema")
		assert.NoError(t, err)
		assert.NotNil(t, schemaContent)
		assert.Equal(t, "text/plain", schemaContent.MimeType)
		assert.Contains(t, schemaContent.Text, "model User")
		assert.Contains(t, schemaContent.Text, "email String @unique")
		assert.Contains(t, schemaContent.Text, "posts Post[]")
	})

	t.Run("ReadSchemaResource", func(t *testing.T) {
		content, err := server.ReadResource(ctx, "schema://database")
		assert.NoError(t, err)
		assert.NotNil(t, content)
		assert.Equal(t, "application/json", content.MimeType)
		
		// Parse JSON to verify structure
		var schemaData map[string]interface{}
		err = json.Unmarshal([]byte(content.Text), &schemaData)
		assert.NoError(t, err)
		assert.Contains(t, schemaData, "models")
		assert.Contains(t, schemaData, "database_type")
	})
}

func TestPrompts(t *testing.T) {
	config := mcp.ServerConfig{
		DatabaseURI: "sqlite://:memory:",
		Transport:   "stdio",
		LogLevel:    "error",
	}

	server, err := mcp.NewServer(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("AnalyzeSchemaPrompt", func(t *testing.T) {
		result, err := server.GetPrompt(ctx, "analyze_schema", map[string]string{
			"focus_area": "indexes",
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Messages, 1)
		assert.Equal(t, "user", result.Messages[0].Role)
		assert.Contains(t, result.Messages[0].Content.Text, "Focus area: indexes")
	})

	t.Run("OptimizeQueryPrompt", func(t *testing.T) {
		result, err := server.GetPrompt(ctx, "optimize_query", map[string]string{
			"query": "User.findMany({ include: { posts: true } })",
			"performance_goal": "reduce N+1 queries",
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Messages, 1)
		assert.Equal(t, "user", result.Messages[0].Role)
		assert.Contains(t, result.Messages[0].Content.Text, "User.findMany")
		assert.Contains(t, result.Messages[0].Content.Text, "reduce N+1 queries")
	})
	
	t.Run("CreateModelPrompt", func(t *testing.T) {
		result, err := server.GetPrompt(ctx, "create_model", map[string]string{
			"description": "A model for tracking user orders",
			"related_models": "User,Product",
		})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Messages, 1)
		assert.Equal(t, "user", result.Messages[0].Role)
		assert.Contains(t, result.Messages[0].Content.Text, "user orders")
		assert.Contains(t, result.Messages[0].Content.Text, "User,Product")
	})

	t.Run("MissingRequiredArgument", func(t *testing.T) {
		_, err := server.GetPrompt(ctx, "optimize_query", map[string]string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required argument: query")
	})
}