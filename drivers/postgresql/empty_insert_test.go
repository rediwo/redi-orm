package postgresql

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyInsert(t *testing.T) {
	// Get test database URI and parse it
	uri := test.GetTestDatabaseUri("postgresql")
	config, err := NewPostgreSQLURIParser().ParseURI(uri)
	if err != nil {
		t.Skipf("Failed to parse PostgreSQL URI: %v", err)
	}
	if config.Host == "" {
		t.Skip("PostgreSQL test connection not configured")
	}

	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)

	// Clean up any existing tables
	cleanupTables(t, db)

	// Define a schema with default values
	userSchema := schema.New("User").
		AddField(schema.Field{
			Name:          "id",
			Type:          schema.FieldTypeInt,
			PrimaryKey:    true,
			AutoIncrement: true,
		}).
		AddField(schema.Field{
			Name:    "status",
			Type:    schema.FieldTypeString,
			Default: "active",
		}).
		AddField(schema.Field{
			Name:     "createdAt",
			Type:     schema.FieldTypeDateTime,
			Default:  "CURRENT_TIMESTAMP",
		})

	// Register schema and sync
	err = db.RegisterSchema("User", userSchema)
	require.NoError(t, err)
	
	err = db.SyncSchemas(ctx)
	require.NoError(t, err)

	// Test 1: Insert with empty map
	t.Run("insert with empty map", func(t *testing.T) {
		result, err := db.Model("User").Insert(map[string]any{}).Exec(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.RowsAffected)
		// PostgreSQL doesn't return LastInsertID without RETURNING
	})

	// Test 2: Insert with empty map and RETURNING
	t.Run("insert with empty map and returning", func(t *testing.T) {
		var user map[string]any
		err := db.Model("User").
			Insert(map[string]any{}).
			Returning("id", "status", "createdAt").
			ExecAndReturn(ctx, &user)
		
		require.NoError(t, err)

		// Log what we got back
		t.Logf("Returned user: %+v", user)

		// Verify returned values - note that column names are returned, not field names
		assert.NotNil(t, user["id"])
		assert.Equal(t, "active", user["status"])
		assert.NotNil(t, user["created_at"]) // Column name, not field name
	})

	// Test 3: Verify data was actually inserted
	t.Run("verify inserted data", func(t *testing.T) {
		var users []map[string]any
		err := db.Model("User").Select().FindMany(ctx, &users)
		require.NoError(t, err)

		// Should have 2 users from previous tests
		assert.Len(t, users, 2)

		// Both should have default status
		for i, user := range users {
			assert.Equal(t, "active", user["status"], "User %d should have default status", i)
			assert.NotNil(t, user["createdAt"], "User %d should have createdAt set", i)
		}
	})
}