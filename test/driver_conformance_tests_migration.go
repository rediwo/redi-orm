package test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Migration Tests =====

func (dct *DriverConformanceTests) TestGetMigrator(t *testing.T) {
	if dct.shouldSkip("TestGetMigrator") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// Get migrator
	migrator := td.DB.GetMigrator()
	assert.NotNil(t, migrator)
	
	// Verify database type
	assert.Equal(t, strings.ToLower(dct.DriverName), migrator.GetDatabaseType())
}

func (dct *DriverConformanceTests) TestGetTables(t *testing.T) {
	if dct.shouldSkip("TestGetTables") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()
	
	// Initially should have no tables (or only system tables)
	migrator := td.DB.GetMigrator()
	tables, err := migrator.GetTables()
	assert.NoError(t, err)
	initialTableCount := len(tables)

	// Create a test table
	err = td.CreateStandardSchemas()
	require.NoError(t, err)

	// Get tables again
	tables, err = migrator.GetTables()
	assert.NoError(t, err)
	assert.Greater(t, len(tables), initialTableCount)
	
	// Should include our tables
	tableNames := make(map[string]bool)
	for _, table := range tables {
		tableNames[table] = true
	}
	assert.True(t, tableNames["users"])
	assert.True(t, tableNames["posts"])
}

func (dct *DriverConformanceTests) TestGetTableInfo(t *testing.T) {
	if dct.shouldSkip("TestGetTableInfo") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()
	
	// Create standard schemas
	err := td.CreateStandardSchemas()
	require.NoError(t, err)

	// Get table info
	migrator := td.DB.GetMigrator()
	tableInfo, err := migrator.GetTableInfo("users")
	assert.NoError(t, err)
	assert.NotNil(t, tableInfo)
	
	// Verify columns
	assert.True(t, len(tableInfo.Columns) > 0)
	
	// Check for expected columns
	columnNames := make(map[string]bool)
	for _, col := range tableInfo.Columns {
		columnNames[col.Name] = true
	}
	assert.True(t, columnNames["id"])
	assert.True(t, columnNames["name"])
	assert.True(t, columnNames["email"])
	
	// Check primary key
	for _, col := range tableInfo.Columns {
		if col.Name == "id" {
			assert.True(t, col.PrimaryKey)
			assert.True(t, col.AutoIncrement)
		}
	}
	
	// Check indexes
	assert.True(t, len(tableInfo.Indexes) > 0)
	
	// Email should have unique index
	hasEmailIndex := false
	for _, idx := range tableInfo.Indexes {
		if len(idx.Columns) == 1 && idx.Columns[0] == "email" {
			hasEmailIndex = true
			assert.True(t, idx.Unique)
		}
	}
	assert.True(t, hasEmailIndex, "Should have unique index on email")
}

func (dct *DriverConformanceTests) TestGenerateCreateTableSQL(t *testing.T) {
	if dct.shouldSkip("TestGenerateCreateTableSQL") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	// Create a test schema
	testSchema := schema.New("TestTable").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt64, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "email", Type: schema.FieldTypeString, Unique: true}).
		AddField(schema.Field{Name: "age", Type: schema.FieldTypeInt, Nullable: true}).
		AddField(schema.Field{Name: "active", Type: schema.FieldTypeBool, Default: true}).
		AddField(schema.Field{Name: "createdAt", Type: schema.FieldTypeDateTime})

	migrator := td.DB.GetMigrator()
	sql, err := migrator.GenerateCreateTableSQL(testSchema)
	assert.NoError(t, err)
	assert.NotEmpty(t, sql)
	
	// Verify SQL contains expected elements
	sqlLower := strings.ToLower(sql)
	assert.Contains(t, sqlLower, "create table")
	assert.Contains(t, sqlLower, "test_tables") // Pluralized
	assert.Contains(t, sqlLower, "id")
	assert.Contains(t, sqlLower, "name")
	assert.Contains(t, sqlLower, "email")
	assert.Contains(t, sqlLower, "primary key")
	
	// Test execution
	ctx := context.Background()
	err = td.DB.RegisterSchema("TestTable", testSchema)
	require.NoError(t, err)
	
	_, err = td.DB.Exec(sql)
	assert.NoError(t, err)
	
	// Clean up
	err = td.DB.DropModel(ctx, "TestTable")
	assert.NoError(t, err)
}

func (dct *DriverConformanceTests) TestGenerateDropTableSQL(t *testing.T) {
	if dct.shouldSkip("TestGenerateDropTableSQL") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	migrator := td.DB.GetMigrator()
	sql := migrator.GenerateDropTableSQL("test_table")
	assert.NotEmpty(t, sql)
	assert.Contains(t, strings.ToLower(sql), "drop table")
	assert.Contains(t, strings.ToLower(sql), "test_table")
}

func (dct *DriverConformanceTests) TestGenerateAddColumnSQL(t *testing.T) {
	if dct.shouldSkip("TestGenerateAddColumnSQL") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	ctx := context.Background()
	
	// Create initial table
	initialSchema := schema.New("AddColTest").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString})
	
	err := td.DB.RegisterSchema("AddColTest", initialSchema)
	require.NoError(t, err)
	err = td.DB.CreateModel(ctx, "AddColTest")
	require.NoError(t, err)

	// Generate add column SQL
	migrator := td.DB.GetMigrator()
	newField := schema.Field{Name: "email", Type: schema.FieldTypeString, Nullable: true}
	sql, err := migrator.GenerateAddColumnSQL("add_col_tests", newField)
	assert.NoError(t, err)
	assert.NotEmpty(t, sql)
	
	// Execute the SQL
	_, err = td.DB.Exec(sql)
	assert.NoError(t, err)
	
	// Verify column was added
	tableInfo, err := migrator.GetTableInfo("add_col_tests")
	assert.NoError(t, err)
	
	hasEmail := false
	for _, col := range tableInfo.Columns {
		if col.Name == "email" {
			hasEmail = true
			break
		}
	}
	assert.True(t, hasEmail, "Email column should exist")

	// Clean up
	err = td.DB.DropModel(ctx, "AddColTest")
	assert.NoError(t, err)
}

func (dct *DriverConformanceTests) TestGenerateDropColumnSQL(t *testing.T) {
	if dct.shouldSkip("TestGenerateDropColumnSQL") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	migrator := td.DB.GetMigrator()
	sqls, err := migrator.GenerateDropColumnSQL("test_table", "column_to_drop")
	assert.NoError(t, err)
	assert.NotEmpty(t, sqls)
	
	// At least one SQL statement should contain DROP
	foundDrop := false
	for _, sql := range sqls {
		sqlLower := strings.ToLower(sql)
		if strings.Contains(sqlLower, "drop") {
			foundDrop = true
			break
		}
	}
	assert.True(t, foundDrop, "Should have DROP in SQL")
}

func (dct *DriverConformanceTests) TestGenerateCreateIndexSQL(t *testing.T) {
	if dct.shouldSkip("TestGenerateCreateIndexSQL") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	migrator := td.DB.GetMigrator()
	
	// Test single column index
	sql := migrator.GenerateCreateIndexSQL("users", "idx_users_email", []string{"email"}, true)
	assert.NotEmpty(t, sql)
	assert.Contains(t, strings.ToLower(sql), "create")
	assert.Contains(t, strings.ToLower(sql), "index")
	assert.Contains(t, strings.ToLower(sql), "unique")
	
	// Test composite index
	sql = migrator.GenerateCreateIndexSQL("posts", "idx_posts_user_created", []string{"user_id", "created_at"}, false)
	assert.NotEmpty(t, sql)
	assert.Contains(t, strings.ToLower(sql), "user_id")
	assert.Contains(t, strings.ToLower(sql), "created_at")
}

func (dct *DriverConformanceTests) TestGenerateDropIndexSQL(t *testing.T) {
	if dct.shouldSkip("TestGenerateDropIndexSQL") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	migrator := td.DB.GetMigrator()
	sql := migrator.GenerateDropIndexSQL("idx_users_email")
	
	// Some databases return empty string for certain operations
	if sql != "" {
		assert.Contains(t, strings.ToLower(sql), "drop")
		assert.Contains(t, strings.ToLower(sql), "index")
	}
}

func (dct *DriverConformanceTests) TestApplyMigration(t *testing.T) {
	if dct.shouldSkip("TestApplyMigration") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()
	
	// Create a simple migration SQL
	integerType := dct.Characteristics.AutoIncrementIntegerType
	if integerType == "" {
		integerType = "INTEGER" // default fallback
	}
	createSQL := fmt.Sprintf("CREATE TABLE test_migration (id %s PRIMARY KEY, name VARCHAR(255))",
		integerType)

	migrator := td.DB.GetMigrator()
	
	// Apply migration
	err := migrator.ApplyMigration(createSQL)
	assert.NoError(t, err)
	
	// Verify table exists
	tables, err := migrator.GetTables()
	assert.NoError(t, err)
	hasTable := slices.Contains(tables, "test_migration")
	assert.True(t, hasTable, "Table should exist after migration")
	
	// Clean up
	_, _ = td.DB.Exec("DROP TABLE test_migration")
}

func (dct *DriverConformanceTests) TestMigrationWorkflow(t *testing.T) {
	if dct.shouldSkip("TestMigrationWorkflow") {
		t.Skip("Test skipped by driver")
	}

	td := dct.createTestDB(t)
	defer td.Cleanup()

	migrator := td.DB.GetMigrator()

	// Step 1: Create initial schema
	initialSchema := schema.New("Product").
		AddField(schema.Field{Name: "id", Type: schema.FieldTypeInt, PrimaryKey: true, AutoIncrement: true}).
		AddField(schema.Field{Name: "name", Type: schema.FieldTypeString}).
		AddField(schema.Field{Name: "price", Type: schema.FieldTypeDecimal})
	
	err := td.DB.RegisterSchema("Product", initialSchema)
	require.NoError(t, err)

	// Generate and apply initial migration
	createSQL, err := migrator.GenerateCreateTableSQL(initialSchema)
	assert.NoError(t, err)
	
	err = migrator.ApplyMigration(createSQL)
	assert.NoError(t, err)

	// Step 2: Add new column
	newField := schema.Field{Name: "description", Type: schema.FieldTypeString, Nullable: true}
	addColumnSQL, err := migrator.GenerateAddColumnSQL("products", newField)
	assert.NoError(t, err)
	
	err = migrator.ApplyMigration(addColumnSQL)
	assert.NoError(t, err)

	// Step 3: Add index
	indexSQL := migrator.GenerateCreateIndexSQL("products", "idx_products_name", []string{"name"}, false)
	
	if indexSQL != "" { // Some operations might return empty SQL
		err = migrator.ApplyMigration(indexSQL)
		assert.NoError(t, err)
	}

	// Verify final state
	tableInfo, err := migrator.GetTableInfo("products")
	assert.NoError(t, err)
	
	// Check columns
	columnNames := make(map[string]bool)
	for _, col := range tableInfo.Columns {
		columnNames[col.Name] = true
	}
	assert.True(t, columnNames["id"])
	assert.True(t, columnNames["name"])
	assert.True(t, columnNames["price"])
	assert.True(t, columnNames["description"])

	// Clean up
	_, _ = td.DB.Exec("DROP TABLE products")
}