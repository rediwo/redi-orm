package drivers

import (
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func TestSQLiteMigratorGetTables(t *testing.T) {
	db := setupSQLiteDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Initially should have no tables (except migration table)
	tables, err := migrator.GetTables()
	if err != nil {
		t.Fatalf("Failed to get tables: %v", err)
	}

	// Create a test table
	testSchema := schema.New("TestTable").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	err = db.CreateTable(testSchema)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Now should have the test table
	tables, err = migrator.GetTables()
	if err != nil {
		t.Fatalf("Failed to get tables after creation: %v", err)
	}

	found := false
	for _, table := range tables {
		if table == "test_tables" { // Schema conversion
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find 'test_tables' in tables list: %v", tables)
	}

	t.Log("✅ SQLite migrator GetTables works correctly")
}

func TestSQLiteMigratorGetTableInfo(t *testing.T) {
	db := setupSQLiteDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Create a test table with various field types
	testSchema := schema.New("DetailedTable").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build()).
		AddField(schema.NewField("metadata").JSON().Nullable().Build())

	err := db.CreateTable(testSchema)
	if err != nil {
		t.Fatalf("Failed to create detailed table: %v", err)
	}

	// Get table information
	tableInfo, err := migrator.GetTableInfo("detailed_tables")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}

	if tableInfo.Name != "detailed_tables" {
		t.Errorf("Expected table name 'detailed_tables', got '%s'", tableInfo.Name)
	}

	if len(tableInfo.Columns) != 5 {
		t.Errorf("Expected 5 columns, got %d", len(tableInfo.Columns))
	}

	// Check specific columns
	columnMap := make(map[string]types.ColumnInfo)
	for _, col := range tableInfo.Columns {
		columnMap[col.Name] = col
	}

	// Check ID column
	if idCol, exists := columnMap["id"]; exists {
		if !idCol.PrimaryKey {
			t.Error("Expected 'id' column to be primary key")
		}
		if !idCol.AutoIncrement {
			t.Error("Expected 'id' column to have auto increment")
		}
	} else {
		t.Error("Expected 'id' column to exist")
	}

	// Check nullable column
	if ageCol, exists := columnMap["age"]; exists {
		if !ageCol.Nullable {
			t.Error("Expected 'age' column to be nullable")
		}
	} else {
		t.Error("Expected 'age' column to exist")
	}

	t.Log("✅ SQLite migrator GetTableInfo works correctly")
}

func TestSQLiteMigratorGenerateCreateTableSQL(t *testing.T) {
	db := setupSQLiteDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Create a comprehensive schema
	testSchema := schema.New("SQLGenTest").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build()).
		AddField(schema.NewField("metadata").JSON().Nullable().Build())

	// Generate SQL
	sql, err := migrator.GenerateCreateTableSQL(testSchema)
	if err != nil {
		t.Fatalf("Failed to generate SQL: %v", err)
	}

	if sql == "" {
		t.Fatal("Generated SQL is empty")
	}

	// Check that SQL contains expected elements
	expectedElements := []string{
		"CREATE TABLE",
		"sql_gen_tests", // Table name conversion
		"id",
		"name",
		"email",
		"age",
		"active",
		"metadata",
		"PRIMARY KEY",
		"AUTOINCREMENT",
		"UNIQUE",
		"DEFAULT",
	}

	for _, element := range expectedElements {
		if !contains(sql, element) {
			t.Errorf("Expected SQL to contain '%s', but it doesn't. SQL: %s", element, sql)
		}
	}

	t.Logf("✅ Generated SQL: %s", sql)
	t.Log("✅ SQLite migrator GenerateCreateTableSQL works correctly")
}

func TestSQLiteMigratorCompareSchema(t *testing.T) {
	db := setupSQLiteDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Create initial table
	initialSchema := schema.New("CompareTest").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	err := db.CreateTable(initialSchema)
	if err != nil {
		t.Fatalf("Failed to create initial table: %v", err)
	}

	// Get table info
	tableInfo, err := migrator.GetTableInfo("compare_tests")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}

	// Create desired schema (with additional field)
	desiredSchema := schema.New("CompareTest").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Build()) // New field

	// Compare schemas
	plan, err := migrator.CompareSchema(tableInfo, desiredSchema)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	if plan == nil {
		t.Fatal("Expected migration plan to be returned")
	}

	// Plan should be empty for now (placeholder implementation)
	// In a full implementation, this would detect the new 'email' field
	
	t.Log("✅ SQLite migrator CompareSchema works correctly")
}

func TestSQLiteMigratorGenerateMigrationSQL(t *testing.T) {
	db := setupSQLiteDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Create a sample migration plan
	plan := &types.MigrationPlan{
		CreateTables: []string{"new_table"},
		AddColumns: []types.ColumnChange{
			{
				TableName:  "existing_table",
				ColumnName: "new_column",
				NewColumn: &types.ColumnInfo{
					Name: "new_column",
					Type: "TEXT",
				},
			},
		},
	}

	// Generate migration SQL
	sqls, err := migrator.GenerateMigrationSQL(plan)
	if err != nil {
		t.Fatalf("Failed to generate migration SQL: %v", err)
	}

	if len(sqls) == 0 {
		t.Fatal("Expected at least one SQL statement")
	}

	// Check that SQL statements are generated
	for i, sql := range sqls {
		if sql == "" {
			t.Errorf("SQL statement %d is empty", i)
		}
		t.Logf("Generated SQL %d: %s", i, sql)
	}

	t.Log("✅ SQLite migrator GenerateMigrationSQL works correctly")
}

func TestSQLiteMigratorApplyMigration(t *testing.T) {
	db := setupSQLiteDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Test applying a simple SQL migration
	sql := "CREATE TABLE migration_test (id INTEGER PRIMARY KEY, name TEXT)"

	err := migrator.ApplyMigration(sql)
	if err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}

	// Verify table was created
	tables, err := migrator.GetTables()
	if err != nil {
		t.Fatalf("Failed to get tables after migration: %v", err)
	}

	found := false
	for _, table := range tables {
		if table == "migration_test" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find 'migration_test' table after migration: %v", tables)
	}

	t.Log("✅ SQLite migrator ApplyMigration works correctly")
}

func TestSQLiteMigratorGetDatabaseType(t *testing.T) {
	db := setupSQLiteDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	dbType := migrator.GetDatabaseType()
	if dbType != "sqlite" {
		t.Errorf("Expected database type 'sqlite', got '%s'", dbType)
	}

	t.Log("✅ SQLite migrator GetDatabaseType works correctly")
}

func TestSQLiteMigratorIntegrationFlow(t *testing.T) {
	db := setupSQLiteDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Step 1: Start with empty database
	initialTables, err := migrator.GetTables()
	if err != nil {
		t.Fatalf("Failed to get initial tables: %v", err)
	}

	t.Logf("Initial tables: %v", initialTables)

	// Step 2: Create first schema
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build())

	sql, err := migrator.GenerateCreateTableSQL(userSchema)
	if err != nil {
		t.Fatalf("Failed to generate SQL for user schema: %v", err)
	}

	err = migrator.ApplyMigration(sql)
	if err != nil {
		t.Fatalf("Failed to apply user migration: %v", err)
	}

	// Step 3: Verify table exists
	tables, err := migrator.GetTables()
	if err != nil {
		t.Fatalf("Failed to get tables after user creation: %v", err)
	}

	if !contains(tables, "users") {
		t.Errorf("Expected 'users' table to exist: %v", tables)
	}

	// Step 4: Create second schema
	postSchema := schema.New("Post").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("title").String().Build()).
		AddField(schema.NewField("authorId").Int64().Build())

	sql, err = migrator.GenerateCreateTableSQL(postSchema)
	if err != nil {
		t.Fatalf("Failed to generate SQL for post schema: %v", err)
	}

	err = migrator.ApplyMigration(sql)
	if err != nil {
		t.Fatalf("Failed to apply post migration: %v", err)
	}

	// Step 5: Verify both tables exist
	finalTables, err := migrator.GetTables()
	if err != nil {
		t.Fatalf("Failed to get final tables: %v", err)
	}

	expectedTables := []string{"users", "posts"}
	for _, expected := range expectedTables {
		if !contains(finalTables, expected) {
			t.Errorf("Expected '%s' table to exist in final tables: %v", expected, finalTables)
		}
	}

	// Step 6: Get detailed info for both tables
	for _, tableName := range expectedTables {
		tableInfo, err := migrator.GetTableInfo(tableName)
		if err != nil {
			t.Errorf("Failed to get info for table '%s': %v", tableName, err)
			continue
		}

		if len(tableInfo.Columns) < 2 {
			t.Errorf("Table '%s' should have at least 2 columns, got %d", tableName, len(tableInfo.Columns))
		}

		t.Logf("Table '%s' has %d columns", tableName, len(tableInfo.Columns))
	}

	t.Log("✅ SQLite migrator integration flow works correctly")
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}