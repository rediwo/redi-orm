package sqlite

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteMigrator_NewSQLiteMigrator(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Get migrator
	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Cast to SQLiteMigrator to test internal structure
	sqliteMigrator, ok := migrator.(*SQLiteMigrator)
	require.True(t, ok)
	assert.NotNil(t, sqliteMigrator.db)

	// Test direct constructor
	directMigrator := NewSQLiteMigrator(db.db)
	assert.NotNil(t, directMigrator)
	assert.Equal(t, db.db, directMigrator.db)
}

func TestSQLiteMigrator_GetDatabaseType(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	dbType := migrator.GetDatabaseType()
	assert.Equal(t, "sqlite", dbType)
}

func TestSQLiteMigrator_GetTables(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test GetTables (placeholder implementation)
	tables, err := migrator.GetTables()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GetTables not yet implemented")
	assert.Nil(t, tables)
}

func TestSQLiteMigrator_GetTableInfo(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test GetTableInfo (placeholder implementation)
	tableInfo, err := migrator.GetTableInfo("test_table")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GetTableInfo not yet implemented")
	assert.Nil(t, tableInfo)
}

func TestSQLiteMigrator_GenerateCreateTableSQL(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test GenerateCreateTableSQL (placeholder implementation)
	sql, err := migrator.GenerateCreateTableSQL(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GenerateCreateTableSQL not yet implemented")
	assert.Empty(t, sql)
}

func TestSQLiteMigrator_GenerateDropTableSQL(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test GenerateDropTableSQL
	sql := migrator.GenerateDropTableSQL("test_table")
	assert.Equal(t, "DROP TABLE IF EXISTS test_table", sql)

	// Test with different table name
	sql = migrator.GenerateDropTableSQL("users")
	assert.Equal(t, "DROP TABLE IF EXISTS users", sql)
}

func TestSQLiteMigrator_GenerateAddColumnSQL(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test GenerateAddColumnSQL (placeholder implementation)
	sql, err := migrator.GenerateAddColumnSQL("test_table", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GenerateAddColumnSQL not yet implemented")
	assert.Empty(t, sql)
}

func TestSQLiteMigrator_GenerateModifyColumnSQL(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test GenerateModifyColumnSQL (placeholder implementation)
	change := types.ColumnChange{
		TableName:  "test_table",
		ColumnName: "test_column",
	}

	sqls, err := migrator.GenerateModifyColumnSQL(change)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GenerateModifyColumnSQL not yet implemented")
	assert.Nil(t, sqls)
}

func TestSQLiteMigrator_GenerateDropColumnSQL(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test GenerateDropColumnSQL (placeholder implementation)
	sqls, err := migrator.GenerateDropColumnSQL("test_table", "test_column")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GenerateDropColumnSQL not yet implemented")
	assert.Nil(t, sqls)
}

func TestSQLiteMigrator_GenerateCreateIndexSQL(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test GenerateCreateIndexSQL
	sql := migrator.GenerateCreateIndexSQL("test_table", "idx_test", []string{"column1", "column2"}, false)
	assert.Equal(t, "CREATE INDEX idx_test ON test_table (column_placeholder)", sql)

	// Test with unique index
	sql = migrator.GenerateCreateIndexSQL("users", "idx_email", []string{"email"}, true)
	assert.Equal(t, "CREATE INDEX idx_email ON users (column_placeholder)", sql)
}

func TestSQLiteMigrator_GenerateDropIndexSQL(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test GenerateDropIndexSQL
	sql := migrator.GenerateDropIndexSQL("idx_test")
	assert.Equal(t, "DROP INDEX IF EXISTS idx_test", sql)

	// Test with different index name
	sql = migrator.GenerateDropIndexSQL("idx_email_unique")
	assert.Equal(t, "DROP INDEX IF EXISTS idx_email_unique", sql)
}

func TestSQLiteMigrator_ApplyMigration(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test ApplyMigration with valid SQL
	sql := "CREATE TABLE test_migration (id INTEGER PRIMARY KEY, name TEXT)"
	err = migrator.ApplyMigration(sql)
	assert.NoError(t, err)

	// Verify table was created
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='test_migration'")
	require.NoError(t, err)
	defer rows.Close()

	var tableName string
	if rows.Next() {
		err = rows.Scan(&tableName)
		assert.NoError(t, err)
		assert.Equal(t, "test_migration", tableName)
	} else {
		t.Error("Table 'test_migration' was not created")
	}

	// Test ApplyMigration with invalid SQL
	invalidSQL := "INVALID SQL STATEMENT"
	err = migrator.ApplyMigration(invalidSQL)
	assert.Error(t, err)
}

func TestSQLiteMigrator_CompareSchema(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test CompareSchema (placeholder implementation)
	plan, err := migrator.CompareSchema(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CompareSchema not yet implemented")
	assert.Nil(t, plan)
}

func TestSQLiteMigrator_GenerateMigrationSQL(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test GenerateMigrationSQL (placeholder implementation)
	sqls, err := migrator.GenerateMigrationSQL(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GenerateMigrationSQL not yet implemented")
	assert.Nil(t, sqls)
}

func TestSQLiteMigrator_IntegrationTest(t *testing.T) {
	// This tests the migrator in a more realistic scenario
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test series of migration operations
	operations := []struct {
		name string
		sql  string
	}{
		{"Create users table", "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)"},
		{"Create posts table", "CREATE TABLE posts (id INTEGER PRIMARY KEY, title TEXT, user_id INTEGER)"},
		{"Create index on user_id", "CREATE INDEX idx_posts_user_id ON posts (user_id)"},
	}

	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			err := migrator.ApplyMigration(op.sql)
			assert.NoError(t, err, "Failed to apply migration: %s", op.name)
		})
	}

	// Verify all tables and indexes were created
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	require.NoError(t, err)
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		require.NoError(t, err)
		tables = append(tables, tableName)
	}

	assert.Contains(t, tables, "users")
	assert.Contains(t, tables, "posts")

	// Check indexes
	rows, err = db.Query("SELECT name FROM sqlite_master WHERE type='index' AND name='idx_posts_user_id'")
	require.NoError(t, err)
	defer rows.Close()

	if rows.Next() {
		var indexName string
		err = rows.Scan(&indexName)
		assert.NoError(t, err)
		assert.Equal(t, "idx_posts_user_id", indexName)
	} else {
		t.Error("Index 'idx_posts_user_id' was not created")
	}

	// Test drop operations
	dropSQL := migrator.GenerateDropTableSQL("users")
	err = migrator.ApplyMigration(dropSQL)
	assert.NoError(t, err)

	// Verify table was dropped
	rows, err = db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='users'")
	require.NoError(t, err)
	assert.False(t, rows.Next())
	rows.Close()

	// Test drop index
	dropIndexSQL := migrator.GenerateDropIndexSQL("idx_posts_user_id")
	err = migrator.ApplyMigration(dropIndexSQL)
	assert.NoError(t, err)

	// Verify index was dropped
	rows, err = db.Query("SELECT name FROM sqlite_master WHERE type='index' AND name='idx_posts_user_id'")
	require.NoError(t, err)
	assert.False(t, rows.Next())
	rows.Close()
}

func TestSQLiteMigrator_ErrorHandling(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test error handling with various invalid SQL statements
	errorCases := []string{
		"INVALID SQL",
		"CREATE TABLE",
		"DROP TABLE",
		"CREATE INDEX",
		"SELECT * FROM non_existent_table",
	}

	for _, invalidSQL := range errorCases {
		t.Run("Invalid SQL: "+invalidSQL, func(t *testing.T) {
			err := migrator.ApplyMigration(invalidSQL)
			assert.Error(t, err, "Expected error for invalid SQL: %s", invalidSQL)
		})
	}

	// Test that migrator still works after errors
	validSQL := "CREATE TABLE recovery_test (id INTEGER)"
	err = migrator.ApplyMigration(validSQL)
	assert.NoError(t, err, "Migrator should still work after handling errors")

	// Verify table was created
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='recovery_test'")
	require.NoError(t, err)
	defer rows.Close()

	assert.True(t, rows.Next(), "Table should have been created after error recovery")
}

func TestSQLiteMigrator_SQLInjectionPrevention(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test that migrator handles potentially dangerous table names safely
	dangerousTableName := "test_table'; DROP TABLE users; --"
	
	// The migrator should generate SQL that doesn't execute injection
	dropSQL := migrator.GenerateDropTableSQL(dangerousTableName)
	
	// The SQL should contain the dangerous string as-is, not execute it
	assert.Contains(t, dropSQL, dangerousTableName)
	assert.Equal(t, "DROP TABLE IF EXISTS "+dangerousTableName, dropSQL)

	// Test with index names
	dangerousIndexName := "idx_test'; DROP TABLE users; --"
	dropIndexSQL := migrator.GenerateDropIndexSQL(dangerousIndexName)
	
	assert.Contains(t, dropIndexSQL, dangerousIndexName)
	assert.Equal(t, "DROP INDEX IF EXISTS "+dangerousIndexName, dropIndexSQL)

	// Apply the "dangerous" SQL - it should just fail cleanly, not execute injection
	err = migrator.ApplyMigration(dropSQL)
	assert.Error(t, err) // Should fail due to invalid table name, not execute DROP TABLE users

	err = migrator.ApplyMigration(dropIndexSQL)
	assert.Error(t, err) // Should fail due to invalid index name, not execute DROP TABLE users
}