package mcp_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rediwo/redi-orm/database"
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
	"github.com/rediwo/redi-orm/logger"
	"github.com/rediwo/redi-orm/mcp"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/schema/generator"
)

// Test helper structures
type testServer struct {
	server      *mcp.SDKServer
	db          database.Database
	persistence *generator.SchemaPersistence
}

// setupTestServer creates a test server with SQLite in-memory database
func setupTestServer(t *testing.T) *testServer {
	// Create in-memory SQLite database
	db, err := database.NewFromURI("sqlite://:memory:")
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)

	// Create logger
	l := logger.NewDefaultLogger("TEST")
	l.SetLevel(logger.LogLevelError) // Reduce noise in tests

	// Create schema persistence in memory
	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	persistence := generator.NewSchemaPersistence("", l, migrator)

	// Create test server manually to inject our test database
	server := &mcp.SDKServer{}

	// Use reflection to set private fields (for testing only)
	// In real implementation, we'd need to add a test constructor or make fields accessible
	// For now, we'll create a minimal server setup

	return &testServer{
		server:      server,
		db:          db,
		persistence: persistence,
	}
}

// cleanupTestServer cleans up test resources
func (ts *testServer) cleanup() {
	if ts.db != nil {
		ts.db.Close()
	}
}

// Helper function to create SchemaCreateParams
func createUserSchema() mcp.SchemaCreateParams {
	return mcp.SchemaCreateParams{
		Model: "User",
		Fields: []mcp.SchemaFieldDefinition{
			{Name: "id", Type: "String", PrimaryKey: true, Default: "cuid()"},
			{Name: "name", Type: "String"},
			{Name: "email", Type: "String", Unique: true},
			{Name: "createdAt", Type: "DateTime", Default: "now()"},
		},
	}
}

func createCategorySchema() mcp.SchemaCreateParams {
	return mcp.SchemaCreateParams{
		Model: "Category",
		Fields: []mcp.SchemaFieldDefinition{
			{Name: "id", Type: "String", PrimaryKey: true, Default: "cuid()"},
			{Name: "name", Type: "String", Unique: true},
			{Name: "description", Type: "String", Optional: true},
		},
	}
}

func createPostSchema() mcp.SchemaCreateParams {
	return mcp.SchemaCreateParams{
		Model: "Post",
		Fields: []mcp.SchemaFieldDefinition{
			{Name: "id", Type: "String", PrimaryKey: true, Default: "cuid()"},
			{Name: "title", Type: "String"},
			{Name: "content", Type: "String"},
			{Name: "authorId", Type: "String"},
			{Name: "categoryId", Type: "String"},
			{Name: "createdAt", Type: "DateTime", Default: "now()"},
		},
		Relations: []mcp.SchemaRelationDefinition{
			{
				Name:       "author",
				Type:       "manyToOne",
				Model:      "User",
				ForeignKey: "authorId",
				References: "id",
			},
			{
				Name:       "category",
				Type:       "manyToOne",
				Model:      "Category",
				ForeignKey: "categoryId",
				References: "id",
			},
		},
	}
}

func createCommentSchema() mcp.SchemaCreateParams {
	return mcp.SchemaCreateParams{
		Model: "Comment",
		Fields: []mcp.SchemaFieldDefinition{
			{Name: "id", Type: "String", PrimaryKey: true, Default: "cuid()"},
			{Name: "content", Type: "String"},
			{Name: "postId", Type: "String"},
			{Name: "authorId", Type: "String"},
			{Name: "createdAt", Type: "DateTime", Default: "now()"},
		},
		Relations: []mcp.SchemaRelationDefinition{
			{
				Name:       "post",
				Type:       "manyToOne",
				Model:      "Post",
				ForeignKey: "postId",
				References: "id",
			},
			{
				Name:       "author",
				Type:       "manyToOne",
				Model:      "User",
				ForeignKey: "authorId",
				References: "id",
			},
		},
	}
}

func createTagSchema() mcp.SchemaCreateParams {
	return mcp.SchemaCreateParams{
		Model: "Tag",
		Fields: []mcp.SchemaFieldDefinition{
			{Name: "id", Type: "String", PrimaryKey: true, Default: "cuid()"},
			{Name: "name", Type: "String", Unique: true},
		},
	}
}

func createPostTagSchema() mcp.SchemaCreateParams {
	return mcp.SchemaCreateParams{
		Model: "PostTag",
		Fields: []mcp.SchemaFieldDefinition{
			{Name: "postId", Type: "String"},
			{Name: "tagId", Type: "String"},
		},
		Relations: []mcp.SchemaRelationDefinition{
			{
				Name:       "post",
				Type:       "manyToOne",
				Model:      "Post",
				ForeignKey: "postId",
				References: "id",
			},
			{
				Name:       "tag",
				Type:       "manyToOne",
				Model:      "Tag",
				ForeignKey: "tagId",
				References: "id",
			},
		},
	}
}

// Circular dependency schemas
func createEmployeeSchema() mcp.SchemaCreateParams {
	return mcp.SchemaCreateParams{
		Model: "Employee",
		Fields: []mcp.SchemaFieldDefinition{
			{Name: "id", Type: "String", PrimaryKey: true, Default: "cuid()"},
			{Name: "name", Type: "String"},
			{Name: "departmentId", Type: "String"},
		},
		Relations: []mcp.SchemaRelationDefinition{
			{
				Name:       "department",
				Type:       "manyToOne",
				Model:      "Department",
				ForeignKey: "departmentId",
				References: "id",
			},
		},
	}
}

func createDepartmentSchema() mcp.SchemaCreateParams {
	return mcp.SchemaCreateParams{
		Model: "Department",
		Fields: []mcp.SchemaFieldDefinition{
			{Name: "id", Type: "String", PrimaryKey: true, Default: "cuid()"},
			{Name: "name", Type: "String"},
			{Name: "managerId", Type: "String", Optional: true},
		},
		Relations: []mcp.SchemaRelationDefinition{
			{
				Name:       "manager",
				Type:       "manyToOne",
				Model:      "Employee",
				ForeignKey: "managerId",
				References: "id",
			},
		},
	}
}

// verifyTableExists checks if a table exists in the database
func verifyTableExists(t *testing.T, db database.Database, tableName string) bool {
	// Query sqlite_master to check if table exists
	rows, err := db.Query(
		"SELECT name FROM sqlite_master WHERE type='table' AND name=?",
		tableName)
	require.NoError(t, err)
	defer rows.Close()

	// If we can scan at least one row, the table exists
	return rows.Next()
}

// Integration test that creates a full MCP server
func TestSchemaCreate_Integration(t *testing.T) {
	// Skip this test for now as it requires a full server setup
	t.Skip("Integration test requires full server setup")
}

// Unit tests for PendingSchemaManager
func TestPendingSchemaManager_SimpleSchema(t *testing.T) {
	// Create in-memory SQLite database
	db, err := database.NewFromURI("sqlite://:memory:")
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create logger
	l := logger.NewDefaultLogger("TEST")
	l.SetLevel(logger.LogLevelError)

	// Create PendingSchemaManager
	manager := mcp.NewPendingSchemaManager(l)

	// Create a simple schema
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:       "id",
			Type:       schema.FieldTypeString,
			PrimaryKey: true,
			Default:    "cuid()",
		}).
		AddField(schema.Field{
			Name: "name",
			Type: schema.FieldTypeString,
		}).
		AddField(schema.Field{
			Name:   "email",
			Type:   schema.FieldTypeString,
			Unique: true,
		})

	// Add schema to pending
	manager.AddSchema(userSchema)

	// Process pending schemas
	result, err := manager.ProcessPendingSchemas(ctx, db)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify results
	assert.Contains(t, result.TablesCreated, "users")
	assert.Empty(t, result.PendingSchemas)
	assert.False(t, result.CircularDeps)

	// Verify table exists in database
	exists := verifyTableExists(t, db, "users")
	assert.True(t, exists, "Table 'users' should exist in database")
}

func TestPendingSchemaManager_DependentSchemas(t *testing.T) {
	// Create in-memory SQLite database
	db, err := database.NewFromURI("sqlite://:memory:")
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create logger
	l := logger.NewDefaultLogger("TEST")
	l.SetLevel(logger.LogLevelError)

	// Create PendingSchemaManager
	manager := mcp.NewPendingSchemaManager(l)

	// Create User schema
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:       "id",
			Type:       schema.FieldTypeString,
			PrimaryKey: true,
		})

	// Create Post schema that depends on User
	postSchema := schema.New("Post").
		AddField(schema.Field{
			Name:       "id",
			Type:       schema.FieldTypeString,
			PrimaryKey: true,
		}).
		AddField(schema.Field{
			Name: "title",
			Type: schema.FieldTypeString,
		}).
		AddField(schema.Field{
			Name: "authorId",
			Type: schema.FieldTypeString,
		}).
		AddRelation("author", schema.Relation{
			Type:       schema.RelationManyToOne,
			Model:      "User",
			ForeignKey: "authorId",
			References: "id",
		})

	// Add both schemas
	manager.AddSchema(userSchema)
	manager.AddSchema(postSchema)

	// Process pending schemas
	result, err := manager.ProcessPendingSchemas(ctx, db)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify both tables were created
	assert.Len(t, result.TablesCreated, 2)
	assert.Contains(t, result.TablesCreated, "users")
	assert.Contains(t, result.TablesCreated, "posts")
	assert.Empty(t, result.PendingSchemas)

	// Verify tables exist in database
	assert.True(t, verifyTableExists(t, db, "users"))
	assert.True(t, verifyTableExists(t, db, "posts"))
}

func TestPendingSchemaManager_ComplexDependencies(t *testing.T) {
	// Create in-memory SQLite database
	db, err := database.NewFromURI("sqlite://:memory:")
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create logger
	l := logger.NewDefaultLogger("TEST")
	l.SetLevel(logger.LogLevelError)

	// Create PendingSchemaManager
	manager := mcp.NewPendingSchemaManager(l)

	// Create schemas for blog system
	userSchema := schema.New("User").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeString, PrimaryKey: true})

	categorySchema := schema.New("Category").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeString, PrimaryKey: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString})

	tagSchema := schema.New("Tag").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeString, PrimaryKey: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString})

	postSchema := schema.New("Post").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeString, PrimaryKey: true}).
		AddField(schema.Field{Name: "title", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "authorId", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "categoryId", Type: schema.FieldTypeString}).
		AddRelation("author", schema.Relation{
			Type: schema.RelationManyToOne, Model: "User",
			ForeignKey: "authorId", References: "id",
		}).
		AddRelation("category", schema.Relation{
			Type: schema.RelationManyToOne, Model: "Category",
			ForeignKey: "categoryId", References: "id",
		})

	commentSchema := schema.New("Comment").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeString, PrimaryKey: true}).
		AddField(schema.Field{Name: "content", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "postId", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "authorId", Type: schema.FieldTypeString}).
		AddRelation("post", schema.Relation{
			Type: schema.RelationManyToOne, Model: "Post",
			ForeignKey: "postId", References: "id",
		}).
		AddRelation("author", schema.Relation{
			Type: schema.RelationManyToOne, Model: "User",
			ForeignKey: "authorId", References: "id",
		})

	postTagSchema := schema.New("PostTag").
		AddField(schema.Field{Name: "postId", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "tagId", Type: schema.FieldTypeString}).
		AddRelation("post", schema.Relation{
			Type: schema.RelationManyToOne, Model: "Post",
			ForeignKey: "postId", References: "id",
		}).
		AddRelation("tag", schema.Relation{
			Type: schema.RelationManyToOne, Model: "Tag",
			ForeignKey: "tagId", References: "id",
		})

	// Add all schemas in random order
	manager.AddSchema(commentSchema)
	manager.AddSchema(postTagSchema)
	manager.AddSchema(postSchema)
	manager.AddSchema(userSchema)
	manager.AddSchema(tagSchema)
	manager.AddSchema(categorySchema)

	// Process pending schemas
	result, err := manager.ProcessPendingSchemas(ctx, db)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify all tables were created
	assert.Len(t, result.TablesCreated, 6)
	assert.Empty(t, result.PendingSchemas)

	// Verify tables exist in correct order
	expectedTables := []string{"users", "categories", "tags", "posts", "comments", "post_tags"}
	for _, table := range expectedTables {
		assert.True(t, verifyTableExists(t, db, table), "Table '%s' should exist", table)
	}

	// Verify dependency info
	assert.NotNil(t, result.DependencyInfo)
	assert.Contains(t, result.DependencyInfo["Post"], "User")
	assert.Contains(t, result.DependencyInfo["Post"], "Category")
	assert.Contains(t, result.DependencyInfo["Comment"], "Post")
	assert.Contains(t, result.DependencyInfo["Comment"], "User")
}

func TestPendingSchemaManager_MissingDependencies(t *testing.T) {
	// Create in-memory SQLite database
	db, err := database.NewFromURI("sqlite://:memory:")
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create logger
	l := logger.NewDefaultLogger("TEST")
	l.SetLevel(logger.LogLevelError)

	// Create PendingSchemaManager
	manager := mcp.NewPendingSchemaManager(l)

	// Create Comment schema that depends on non-existent Post and User
	commentSchema := schema.New("Comment").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeString, PrimaryKey: true}).
		AddField(schema.Field{Name: "content", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "postId", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "authorId", Type: schema.FieldTypeString}).
		AddRelation("post", schema.Relation{
			Type: schema.RelationManyToOne, Model: "Post",
			ForeignKey: "postId", References: "id",
		}).
		AddRelation("author", schema.Relation{
			Type: schema.RelationManyToOne, Model: "User",
			ForeignKey: "authorId", References: "id",
		})

	// Add only Comment schema
	manager.AddSchema(commentSchema)

	// Process pending schemas - should not create table due to missing dependencies
	result, err := manager.ProcessPendingSchemas(ctx, db)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify Comment is still pending
	assert.Empty(t, result.TablesCreated)
	assert.Contains(t, result.PendingSchemas, "Comment")
	assert.False(t, verifyTableExists(t, db, "comments"))

	// Now add the missing dependencies
	userSchema := schema.New("User").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeString, PrimaryKey: true})

	postSchema := schema.New("Post").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeString, PrimaryKey: true}).
		AddField(schema.Field{Name: "authorId", Type: schema.FieldTypeString}).
		AddRelation("author", schema.Relation{
			Type: schema.RelationManyToOne, Model: "User",
			ForeignKey: "authorId", References: "id",
		})

	manager.AddSchema(userSchema)
	manager.AddSchema(postSchema)

	// Process again - should create all tables now
	result2, err := manager.ProcessPendingSchemas(ctx, db)
	require.NoError(t, err)
	require.NotNil(t, result2)

	// Verify all tables were created
	assert.Len(t, result2.TablesCreated, 3)
	assert.Contains(t, result2.TablesCreated, "users")
	assert.Contains(t, result2.TablesCreated, "posts")
	assert.Contains(t, result2.TablesCreated, "comments")
	assert.Empty(t, result2.PendingSchemas)
}

func TestPendingSchemaManager_CircularDependencies(t *testing.T) {
	// Create in-memory SQLite database
	db, err := database.NewFromURI("sqlite://:memory:")
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create logger
	l := logger.NewDefaultLogger("TEST")
	l.SetLevel(logger.LogLevelError)

	// Create PendingSchemaManager
	manager := mcp.NewPendingSchemaManager(l)

	// Create circular dependency: Employee <-> Department
	employeeSchema := schema.New("Employee").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeString, PrimaryKey: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "departmentId", Type: schema.FieldTypeString}).
		AddRelation("department", schema.Relation{
			Type: schema.RelationManyToOne, Model: "Department",
			ForeignKey: "departmentId", References: "id",
		})

	departmentSchema := schema.New("Department").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeString, PrimaryKey: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "managerId", Type: schema.FieldTypeString, Nullable: true}).
		AddRelation("manager", schema.Relation{
			Type: schema.RelationManyToOne, Model: "Employee",
			ForeignKey: "managerId", References: "id",
		})

	// Add both schemas
	manager.AddSchema(employeeSchema)
	manager.AddSchema(departmentSchema)

	// Process pending schemas - should detect circular dependency
	result, err := manager.ProcessPendingSchemas(ctx, db)
	require.NoError(t, err) // Should not error, just report circular dependency
	require.NotNil(t, result)

	// Verify circular dependency was detected
	assert.True(t, result.CircularDeps)
	assert.NotEmpty(t, result.PendingSchemas)
	assert.NotEmpty(t, result.Errors)

	// For SQL databases, tables should not be created due to circular dependency
	// The error message should guide manual intervention
	assert.Contains(t, result.Errors[0], "Circular dependencies detected")
}
