package drivers

import (
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func TestSQLiteMigratorCompareSchemaDetectNewColumns(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Create existing table info with basic columns
	existingTable := &types.TableInfo{
		Name: "users",
		Columns: []types.ColumnInfo{
			{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: "TEXT", Nullable: false},
		},
	}

	// Create desired schema with additional columns
	desiredSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Nullable().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build())

	// Compare schemas
	plan, err := migrator.CompareSchema(existingTable, desiredSchema)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	// Should detect 2 new columns
	if len(plan.AddColumns) != 2 {
		t.Errorf("Expected 2 columns to add, got %d", len(plan.AddColumns))
	}

	// Check specific columns
	addedColumns := make(map[string]bool)
	for _, change := range plan.AddColumns {
		addedColumns[change.ColumnName] = true
	}

	if !addedColumns["email"] {
		t.Error("Expected 'email' column to be added")
	}

	if !addedColumns["age"] {
		t.Error("Expected 'age' column to be added")
	}

	// Should not detect modifications or deletions
	if len(plan.ModifyColumns) != 0 {
		t.Errorf("Expected 0 columns to modify, got %d", len(plan.ModifyColumns))
	}

	if len(plan.DropColumns) != 0 {
		t.Errorf("Expected 0 columns to drop, got %d", len(plan.DropColumns))
	}

	t.Log("✅ CompareSchema correctly detects new columns")
}

func TestSQLiteMigratorCompareSchemaDetectRemovedColumns(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Create existing table info with extra columns
	existingTable := &types.TableInfo{
		Name: "users",
		Columns: []types.ColumnInfo{
			{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: "TEXT", Nullable: false},
			{Name: "email", Type: "TEXT", Nullable: true},
			{Name: "old_field", Type: "TEXT", Nullable: true},
		},
	}

	// Create desired schema without some columns
	desiredSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Nullable().Build())

	// Compare schemas
	plan, err := migrator.CompareSchema(existingTable, desiredSchema)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	// Should detect 1 removed column
	if len(plan.DropColumns) != 1 {
		t.Errorf("Expected 1 column to drop, got %d", len(plan.DropColumns))
	}

	if plan.DropColumns[0].ColumnName != "old_field" {
		t.Errorf("Expected 'old_field' to be dropped, got %s", plan.DropColumns[0].ColumnName)
	}

	// Should not detect additions or modifications
	if len(plan.AddColumns) != 0 {
		t.Errorf("Expected 0 columns to add, got %d", len(plan.AddColumns))
	}

	if len(plan.ModifyColumns) != 0 {
		t.Errorf("Expected 0 columns to modify, got %d", len(plan.ModifyColumns))
	}

	t.Log("✅ CompareSchema correctly detects removed columns")
}

func TestSQLiteMigratorCompareSchemaDetectModifiedColumns(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Create existing table info
	existingTable := &types.TableInfo{
		Name: "users",
		Columns: []types.ColumnInfo{
			{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: "TEXT", Nullable: false},
			{Name: "age", Type: "TEXT", Nullable: true}, // Wrong type
			{Name: "active", Type: "INTEGER", Nullable: false}, // Wrong nullable
		},
	}

	// Create desired schema with correct types
	desiredSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Nullable().Build())

	// Compare schemas
	plan, err := migrator.CompareSchema(existingTable, desiredSchema)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	// Should detect 2 modified columns
	if len(plan.ModifyColumns) != 2 {
		t.Errorf("Expected 2 columns to modify, got %d", len(plan.ModifyColumns))
	}

	// Check specific modifications
	modifiedColumns := make(map[string]bool)
	for _, change := range plan.ModifyColumns {
		modifiedColumns[change.ColumnName] = true
	}

	if !modifiedColumns["age"] {
		t.Error("Expected 'age' column to be modified (type change)")
	}

	if !modifiedColumns["active"] {
		t.Error("Expected 'active' column to be modified (nullable change)")
	}

	// Should not detect additions or deletions
	if len(plan.AddColumns) != 0 {
		t.Errorf("Expected 0 columns to add, got %d", len(plan.AddColumns))
	}

	if len(plan.DropColumns) != 0 {
		t.Errorf("Expected 0 columns to drop, got %d", len(plan.DropColumns))
	}

	t.Log("✅ CompareSchema correctly detects modified columns")
}

func TestSQLiteMigratorCompareSchemaNoChanges(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Create existing table info
	existingTable := &types.TableInfo{
		Name: "users",
		Columns: []types.ColumnInfo{
			{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: "TEXT", Nullable: false},
			{Name: "email", Type: "TEXT", Nullable: true},
		},
	}

	// Create identical desired schema
	desiredSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Nullable().Build())

	// Compare schemas
	plan, err := migrator.CompareSchema(existingTable, desiredSchema)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	// Should detect no changes
	if len(plan.AddColumns) != 0 {
		t.Errorf("Expected 0 columns to add, got %d", len(plan.AddColumns))
	}

	if len(plan.ModifyColumns) != 0 {
		t.Errorf("Expected 0 columns to modify, got %d", len(plan.ModifyColumns))
	}

	if len(plan.DropColumns) != 0 {
		t.Errorf("Expected 0 columns to drop, got %d", len(plan.DropColumns))
	}

	t.Log("✅ CompareSchema correctly detects no changes")
}

func TestSQLiteMigratorCompareSchemaComplexChanges(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Create existing table info
	existingTable := &types.TableInfo{
		Name: "products",
		Columns: []types.ColumnInfo{
			{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: "TEXT", Nullable: false},
			{Name: "price", Type: "TEXT", Nullable: true}, // Wrong type - should be REAL
			{Name: "old_category", Type: "TEXT", Nullable: true}, // To be removed
			{Name: "stock", Type: "INTEGER", Nullable: false},
		},
	}

	// Create desired schema with mixed changes
	desiredSchema := schema.New("Product").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("price").Float().Nullable().Build()). // Type change
		AddField(schema.NewField("stock").Int().Build()).              // No change
		AddField(schema.NewField("category").String().Build()).        // New column
		AddField(schema.NewField("active").Bool().Default(true).Build()) // New column with default

	// Compare schemas
	plan, err := migrator.CompareSchema(existingTable, desiredSchema)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	// Should detect 2 new columns
	if len(plan.AddColumns) != 2 {
		t.Errorf("Expected 2 columns to add, got %d", len(plan.AddColumns))
	}

	// Should detect 1 modified column
	if len(plan.ModifyColumns) != 1 {
		t.Errorf("Expected 1 column to modify, got %d", len(plan.ModifyColumns))
	}

	// Should detect 1 removed column
	if len(plan.DropColumns) != 1 {
		t.Errorf("Expected 1 column to drop, got %d", len(plan.DropColumns))
	}

	// Verify specific changes
	addedColumns := make(map[string]bool)
	for _, change := range plan.AddColumns {
		addedColumns[change.ColumnName] = true
	}

	if !addedColumns["category"] || !addedColumns["active"] {
		t.Error("Expected 'category' and 'active' columns to be added")
	}

	if plan.ModifyColumns[0].ColumnName != "price" {
		t.Errorf("Expected 'price' column to be modified, got %s", plan.ModifyColumns[0].ColumnName)
	}

	if plan.DropColumns[0].ColumnName != "old_category" {
		t.Errorf("Expected 'old_category' column to be dropped, got %s", plan.DropColumns[0].ColumnName)
	}

	t.Log("✅ CompareSchema correctly handles complex mixed changes")
}

func TestSQLiteMigratorGenerateAddColumnSQL(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Test various column types
	testCases := []struct {
		name     string
		field    schema.Field
		expected []string // strings that should be in the SQL
	}{
		{
			name:     "Simple nullable string",
			field:    schema.NewField("email").String().Nullable().Build(),
			expected: []string{"ALTER TABLE", "ADD COLUMN", "email", "TEXT"},
		},
		{
			name:     "Integer with default",
			field:    schema.NewField("count").Int().Default(0).Build(),
			expected: []string{"ALTER TABLE", "ADD COLUMN", "count", "INTEGER", "NOT NULL", "DEFAULT 0"},
		},
		{
			name:     "Boolean with default true",
			field:    schema.NewField("active").Bool().Default(true).Build(),
			expected: []string{"ALTER TABLE", "ADD COLUMN", "active", "INTEGER", "NOT NULL", "DEFAULT 1"},
		},
		{
			name:     "Unique string field",
			field:    schema.NewField("username").String().Unique().Build(),
			expected: []string{"ALTER TABLE", "ADD COLUMN", "username", "TEXT", "NOT NULL", "UNIQUE"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sql, err := migrator.GenerateAddColumnSQL("test_table", tc.field)
			if err != nil {
				t.Fatalf("Failed to generate ADD COLUMN SQL: %v", err)
			}

			for _, expected := range tc.expected {
				if !containsString(sql, expected) {
					t.Errorf("Expected SQL to contain '%s', but it doesn't. SQL: %s", expected, sql)
				}
			}

			t.Logf("Generated SQL: %s", sql)
		})
	}

	t.Log("✅ GenerateAddColumnSQL works correctly for various column types")
}

func TestSQLiteMigratorColumnComparison(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()

	migrator := db.GetMigrator()
	if migrator == nil {
		t.Fatal("Expected migrator to be available")
	}

	// Test column comparison logic indirectly through CompareSchema
	existingTable := &types.TableInfo{
		Name: "test_table",
		Columns: []types.ColumnInfo{
			{Name: "id", Type: "INTEGER", PrimaryKey: true, AutoIncrement: true},
			{Name: "name", Type: "TEXT", Nullable: false},
			{Name: "old_field", Type: "TEXT", Nullable: true},
		},
	}

	// Create schema with modifications
	desiredSchema := schema.New("TestTable").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("new_field").String().Nullable().Build())

	// Compare schemas
	plan, err := migrator.CompareSchema(existingTable, desiredSchema)
	if err != nil {
		t.Fatalf("Failed to compare schemas: %v", err)
	}

	// Should detect 1 new column and 1 removed column
	if len(plan.AddColumns) != 1 {
		t.Errorf("Expected 1 column to add, got %d", len(plan.AddColumns))
	}

	if len(plan.DropColumns) != 1 {
		t.Errorf("Expected 1 column to drop, got %d", len(plan.DropColumns))
	}

	if plan.AddColumns[0].ColumnName != "new_field" {
		t.Errorf("Expected 'new_field' to be added, got %s", plan.AddColumns[0].ColumnName)
	}

	if plan.DropColumns[0].ColumnName != "old_field" {
		t.Errorf("Expected 'old_field' to be dropped, got %s", plan.DropColumns[0].ColumnName)
	}

	t.Log("✅ Column comparison logic works correctly")
}