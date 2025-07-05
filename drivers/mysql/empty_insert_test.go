package mysql

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
	uri := test.GetTestDatabaseUri("mysql")
	config, err := NewMySQLURIParser().ParseURI(uri)
	if err != nil {
		t.Skipf("Failed to parse MySQL URI: %v", err)
	}
	if config.Host == "" {
		t.Skip("MySQL test connection not configured")
	}

	db, err := NewMySQLDB(config)
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

	// Test 1: Insert with empty map (MySQL requires at least one field)
	t.Run("insert with empty map", func(t *testing.T) {
		// MySQL doesn't support DEFAULT VALUES, so empty insert should fail
		_, err := db.Model("User").Insert(map[string]any{}).Exec(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot insert empty record")
	})

	// Test 2: Insert with minimal data to test defaults
	t.Run("insert with minimal data", func(t *testing.T) {
		// MySQL needs at least one field, so we'll use a field without a default
		result, err := db.Model("User").Insert(map[string]any{
			"id": nil, // Let auto-increment handle this
		}).Exec(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.RowsAffected)
		assert.Greater(t, result.LastInsertID, int64(0))
		
		// Verify data was inserted with defaults
		var users []map[string]any
		err = db.Model("User").Select().FindMany(ctx, &users)
		require.NoError(t, err)
		
		assert.Len(t, users, 1)
		user := users[0]
		assert.Equal(t, "active", user["status"]) // Should have default status
		assert.NotNil(t, user["createdAt"])        // Should have default timestamp
	})

	// Note: MySQL doesn't support RETURNING clause, so we don't test ExecAndReturn
}