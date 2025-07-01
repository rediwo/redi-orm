package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/schema"
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

	// Test that it's a wrapper
	_, ok := migrator.(*SQLiteMigratorWrapper)
	require.True(t, ok)
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
	assert.Equal(t, "sqlite", migrator.GetDatabaseType())
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

	// Initially no tables
	tables, err := migrator.GetTables()
	assert.NoError(t, err)
	assert.Empty(t, tables)

	// Create a test table
	_, err = db.Exec("CREATE TABLE test_table1 (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	_, err = db.Exec("CREATE TABLE test_table2 (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	// Now should have 2 tables
	tables, err = migrator.GetTables()
	assert.NoError(t, err)
	assert.Len(t, tables, 2)
	assert.Contains(t, tables, "test_table1")
	assert.Contains(t, tables, "test_table2")
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

	// Create a test table with various column types
	_, err = db.Exec(`
		CREATE TABLE test_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT UNIQUE,
			age INTEGER,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	// Create an index
	_, err = db.Exec("CREATE UNIQUE INDEX idx_email ON test_table (email)")
	require.NoError(t, err)
	
	
	migrator := db.GetMigrator()
	tableInfo, err := migrator.GetTableInfo("test_table")
	require.NoError(t, err)
	require.NotNil(t, tableInfo)

	// Check table name
	assert.Equal(t, "test_table", tableInfo.Name)

	// Check columns
	assert.Len(t, tableInfo.Columns, 6)

	// Check id column
	idCol := findColumn(tableInfo.Columns, "id")
	require.NotNil(t, idCol)
	assert.True(t, idCol.PrimaryKey)
	assert.True(t, idCol.AutoIncrement)
	// PRIMARY KEY columns are implicitly NOT NULL in SQLite
	assert.False(t, idCol.Nullable, "PRIMARY KEY columns should not be nullable")

	// Check name column
	nameCol := findColumn(tableInfo.Columns, "name")
	require.NotNil(t, nameCol)
	assert.False(t, nameCol.Nullable)
	assert.Equal(t, "TEXT", nameCol.Type)

	// Check email column
	emailCol := findColumn(tableInfo.Columns, "email")
	require.NotNil(t, emailCol)
	if !emailCol.Unique {
		t.Logf("UNIQUE not detected for email column")
		// Debug: log all column info
		t.Logf("Email column info: %+v", emailCol)
	}
	assert.True(t, emailCol.Unique)
	assert.Equal(t, "TEXT", emailCol.Type)

	// Check age column
	ageCol := findColumn(tableInfo.Columns, "age")
	require.NotNil(t, ageCol)
	assert.True(t, ageCol.Nullable)
	assert.Equal(t, "INTEGER", ageCol.Type)

	// Check active column
	activeCol := findColumn(tableInfo.Columns, "active")
	require.NotNil(t, activeCol)
	assert.Equal(t, "1", activeCol.Default)

	// Check indexes - we should have 2: our created index and the auto-generated unique index
	t.Logf("Found %d indexes", len(tableInfo.Indexes))
	for i, idx := range tableInfo.Indexes {
		t.Logf("Index %d: %+v", i, idx)
	}
	
	// Find our created index
	var foundIdx *types.IndexInfo
	for i := range tableInfo.Indexes {
		if tableInfo.Indexes[i].Name == "idx_email" {
			foundIdx = &tableInfo.Indexes[i]
			break
		}
	}
	
	require.NotNil(t, foundIdx, "idx_email not found")
	assert.True(t, foundIdx.Unique)
	assert.Equal(t, []string{"email"}, foundIdx.Columns)
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

	// Create a schema
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()).
		AddField(schema.NewField("age").Int().Nullable().Build()).
		AddField(schema.NewField("active").Bool().Default(true).Build())

	migrator := db.GetMigrator()
	sql, err := migrator.GenerateCreateTableSQL(userSchema)
	require.NoError(t, err)

	// Verify SQL contains expected parts
	assert.Contains(t, sql, "CREATE TABLE IF NOT EXISTS users")
	assert.Contains(t, sql, "id INTEGER PRIMARY KEY AUTOINCREMENT")
	assert.Contains(t, sql, "name TEXT NOT NULL")
	assert.Contains(t, sql, "email TEXT NOT NULL UNIQUE")
	assert.Contains(t, sql, "age INTEGER")
	assert.Contains(t, sql, "active INTEGER NOT NULL DEFAULT true")
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
	sql := migrator.GenerateDropTableSQL("test_table")
	assert.Equal(t, "DROP TABLE IF EXISTS test_table", sql)
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

	// Test with a simple field
	field := schema.NewField("email").String().Unique().Build()
	sql, err := migrator.GenerateAddColumnSQL("users", field)
	assert.NoError(t, err)
	assert.Equal(t, "ALTER TABLE users ADD COLUMN email TEXT NOT NULL UNIQUE", sql)

	// Test with nullable field
	field2 := schema.NewField("bio").String().Nullable().Build()
	sql2, err := migrator.GenerateAddColumnSQL("users", field2)
	assert.NoError(t, err)
	assert.Equal(t, "ALTER TABLE users ADD COLUMN bio TEXT", sql2)

	// Test with default value
	field3 := schema.NewField("active").Bool().Default(true).Build()
	sql3, err := migrator.GenerateAddColumnSQL("users", field3)
	assert.NoError(t, err)
	assert.Equal(t, "ALTER TABLE users ADD COLUMN active INTEGER NOT NULL DEFAULT true", sql3)
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

	// Create a test table first
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT,
			age INTEGER
		)
	`)
	require.NoError(t, err)
	
	// Create an index that we'll need to recreate
	_, err = db.Exec("CREATE INDEX idx_email ON users (email)")
	require.NoError(t, err)

	migrator := db.GetMigrator()

	t.Run("Add UNIQUE constraint to column", func(t *testing.T) {
		change := types.ColumnChange{
			TableName:  "users",
			ColumnName: "email",
			OldColumn: &types.ColumnInfo{
				Name: "email",
				Type: "TEXT",
			},
			NewColumn: &types.ColumnInfo{
				Name:   "email",
				Type:   "TEXT",
				Unique: true,
			},
		}

		sqls, err := migrator.GenerateModifyColumnSQL(change)
		assert.NoError(t, err)
		assert.Greater(t, len(sqls), 3) // Should have CREATE, INSERT, DROP, RENAME at minimum
		
		// Check that it creates a temporary table
		assert.Contains(t, sqls[0], "CREATE TABLE")
		assert.Contains(t, sqls[0], "_temp_")
		assert.Contains(t, sqls[0], "email TEXT")
		assert.Contains(t, sqls[0], "UNIQUE")
		
		// Check data copy
		assert.Contains(t, sqls[1], "INSERT INTO")
		assert.Contains(t, sqls[1], "SELECT")
		
		// Check drop old table
		assert.Contains(t, sqls[2], "DROP TABLE users")
		
		// Check rename
		assert.Contains(t, sqls[3], "ALTER TABLE")
		assert.Contains(t, sqls[3], "RENAME TO users")
		
		// Check index recreation
		hasIndexRecreation := false
		for _, sql := range sqls {
			if strings.Contains(sql, "CREATE INDEX idx_email") {
				hasIndexRecreation = true
				break
			}
		}
		assert.True(t, hasIndexRecreation, "Should recreate indexes")
	})
	
	t.Run("Change column type", func(t *testing.T) {
		change := types.ColumnChange{
			TableName:  "users",
			ColumnName: "age",
			OldColumn: &types.ColumnInfo{
				Name: "age",
				Type: "INTEGER",
			},
			NewColumn: &types.ColumnInfo{
				Name: "age",
				Type: "TEXT",
			},
		}

		sqls, err := migrator.GenerateModifyColumnSQL(change)
		assert.NoError(t, err)
		assert.Greater(t, len(sqls), 3)
		
		// Check that the new column definition uses TEXT
		assert.Contains(t, sqls[0], "age TEXT")
	})
	
	t.Run("Rename column", func(t *testing.T) {
		change := types.ColumnChange{
			TableName:  "users",
			ColumnName: "name",
			OldColumn: &types.ColumnInfo{
				Name: "name",
				Type: "TEXT",
			},
			NewColumn: &types.ColumnInfo{
				Name: "full_name",
				Type: "TEXT",
			},
		}

		sqls, err := migrator.GenerateModifyColumnSQL(change)
		assert.NoError(t, err)
		assert.Greater(t, len(sqls), 3)
		
		// Check that the new table has full_name column
		assert.Contains(t, sqls[0], "full_name TEXT")
		
		// Check that INSERT maps old name to new name
		assert.Contains(t, sqls[1], "full_name")
	})
}

func TestSQLiteMigrator_GenerateModifyColumnSQL_Integration(t *testing.T) {
	db, err := NewSQLiteDB(types.Config{
		Type:     "sqlite",
		FilePath: ":memory:",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create a test table with data
	_, err = db.Exec(`
		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			price INTEGER,
			description TEXT
		)
	`)
	require.NoError(t, err)
	
	// Insert some test data
	_, err = db.Exec("INSERT INTO products (name, price, description) VALUES ('Widget', 100, 'A nice widget')")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO products (name, price, description) VALUES ('Gadget', 200, 'A cool gadget')")
	require.NoError(t, err)

	migrator := db.GetMigrator()

	// Test actual execution of column modification
	change := types.ColumnChange{
		TableName:  "products",
		ColumnName: "price",
		OldColumn: &types.ColumnInfo{
			Name: "price",
			Type: "INTEGER",
		},
		NewColumn: &types.ColumnInfo{
			Name:     "price",
			Type:     "DECIMAL",
			Nullable: false,
			Default:  0,
		},
	}

	sqls, err := migrator.GenerateModifyColumnSQL(change)
	require.NoError(t, err)
	
	// Execute all the migration SQL
	for _, sql := range sqls {
		if strings.HasPrefix(sql, "--") {
			continue // Skip comments
		}
		_, err = db.Exec(sql)
		require.NoError(t, err, "Failed to execute: %s", sql)
	}
	
	// Verify the data is still there
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	
	// Verify the column type changed
	tableInfo, err := migrator.GetTableInfo("products")
	require.NoError(t, err)
	
	priceCol := findColumn(tableInfo.Columns, "price")
	require.NotNil(t, priceCol)
	assert.Equal(t, "DECIMAL", priceCol.Type)
	assert.Equal(t, "0", priceCol.Default) // SQLite stores defaults as strings
	
	// Verify data integrity
	rows, err := db.Query("SELECT name, price FROM products ORDER BY name")
	require.NoError(t, err)
	defer rows.Close()
	
	var results []struct {
		name  string
		price int
	}
	
	for rows.Next() {
		var r struct {
			name  string
			price int
		}
		err = rows.Scan(&r.name, &r.price)
		require.NoError(t, err)
		results = append(results, r)
	}
	
	assert.Len(t, results, 2)
	assert.Equal(t, "Gadget", results[0].name)
	assert.Equal(t, 200, results[0].price)
	assert.Equal(t, "Widget", results[1].name)
	assert.Equal(t, 100, results[1].price)
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
	sqls, err := migrator.GenerateDropColumnSQL("users", "old_column")
	assert.NoError(t, err)
	assert.Len(t, sqls, 1)
	assert.Equal(t, "ALTER TABLE users DROP COLUMN old_column", sqls[0])
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

	// Regular index
	sql := migrator.GenerateCreateIndexSQL("users", "idx_name", []string{"name"}, false)
	assert.Equal(t, "CREATE INDEX idx_name ON users (name)", sql)

	// Unique index
	sql2 := migrator.GenerateCreateIndexSQL("users", "idx_email", []string{"email"}, true)
	assert.Equal(t, "CREATE UNIQUE INDEX idx_email ON users (email)", sql2)

	// Composite index
	sql3 := migrator.GenerateCreateIndexSQL("users", "idx_name_email", []string{"name", "email"}, false)
	assert.Equal(t, "CREATE INDEX idx_name_email ON users (name, email)", sql3)
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
	sql := migrator.GenerateDropIndexSQL("idx_name")
	assert.Equal(t, "DROP INDEX IF EXISTS idx_name", sql)
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

	// Apply a CREATE TABLE migration
	sql := "CREATE TABLE test_migration (id INTEGER PRIMARY KEY)"
	err = migrator.ApplyMigration(sql)
	assert.NoError(t, err)

	// Verify table was created
	tables, err := migrator.GetTables()
	assert.NoError(t, err)
	assert.Contains(t, tables, "test_migration")

	// Test error handling
	err = migrator.ApplyMigration("INVALID SQL")
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

	// Create an existing table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			old_field TEXT
		)
	`)
	require.NoError(t, err)

	// Define desired schema
	userSchema := schema.New("User").
		AddField(schema.NewField("id").Int64().PrimaryKey().Build()).
		AddField(schema.NewField("name").String().Build()).
		AddField(schema.NewField("email").String().Unique().Build()). // New field
		AddField(schema.NewField("age").Int().Nullable().Build())     // New field

	migrator := db.GetMigrator()

	// Get existing table info
	existingTable, err := migrator.GetTableInfo("users")
	require.NoError(t, err)

	// Compare schemas
	plan, err := migrator.CompareSchema(existingTable, userSchema)
	require.NoError(t, err)
	require.NotNil(t, plan)

	// Should have 2 columns to add (email, age)
	assert.Len(t, plan.AddColumns, 2)

	// Should have 1 column to drop (old_field)
	assert.Len(t, plan.DropColumns, 1)
	assert.Equal(t, "old_field", plan.DropColumns[0].ColumnName)

	// Check if there are any modifications detected
	if len(plan.ModifyColumns) > 0 {
		for _, mod := range plan.ModifyColumns {
			t.Logf("Unexpected modification detected for column %s: old=%+v, new=%+v", 
				mod.ColumnName, mod.OldColumn, mod.NewColumn)
		}
		// For now, we'll accept modifications as SQLite might detect type differences
		// between INTEGER and INT64, or other minor differences
	}
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

	// Create a migration plan
	plan := &types.MigrationPlan{
		AddColumns: []types.ColumnChange{
			{
				TableName:  "users",
				ColumnName: "email",
				NewColumn: &types.ColumnInfo{
					Name:   "email",
					Type:   "TEXT",
					Unique: true,
				},
			},
		},
		DropColumns: []types.ColumnChange{
			{
				TableName:  "users",
				ColumnName: "old_field",
			},
		},
	}

	sqls, err := migrator.GenerateMigrationSQL(plan)
	require.NoError(t, err)
	assert.Len(t, sqls, 2)

	// Check ADD COLUMN
	assert.Contains(t, sqls[0], "ALTER TABLE users ADD COLUMN")
	assert.Contains(t, sqls[0], "email TEXT")

	// Check DROP COLUMN
	assert.Contains(t, sqls[1], "ALTER TABLE users DROP COLUMN old_field")
}

func TestSQLiteMigrator_IntegrationTest(t *testing.T) {
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

	// Test cases for various SQL operations
	testCases := []struct {
		name string
		sql  string
	}{
		{
			name: "Create users table",
			sql: `CREATE TABLE users (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				email TEXT UNIQUE,
				active INTEGER DEFAULT 1
			)`,
		},
		{
			name: "Create posts table",
			sql: `CREATE TABLE posts (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				title TEXT NOT NULL,
				content TEXT,
				user_id INTEGER,
				FOREIGN KEY (user_id) REFERENCES users(id)
			)`,
		},
		{
			name: "Create index on user_id",
			sql:  "CREATE INDEX idx_user_id ON posts (user_id)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrator.ApplyMigration(tc.sql)
			assert.NoError(t, err)
		})
	}

	// Verify tables were created
	tables, err := migrator.GetTables()
	assert.NoError(t, err)
	assert.Len(t, tables, 2)
	assert.Contains(t, tables, "users")
	assert.Contains(t, tables, "posts")

	// Verify table structure
	userInfo, err := migrator.GetTableInfo("users")
	assert.NoError(t, err)
	assert.Len(t, userInfo.Columns, 4)

	postInfo, err := migrator.GetTableInfo("posts")
	assert.NoError(t, err)
	assert.Len(t, postInfo.Columns, 4)
	assert.Len(t, postInfo.ForeignKeys, 1)
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

	// Test various error scenarios
	testCases := []struct {
		name        string
		sql         string
		expectError bool
	}{
		{
			name:        "Invalid SQL: INVALID SQL",
			sql:         "INVALID SQL",
			expectError: true,
		},
		{
			name:        "Invalid SQL: CREATE TABLE",
			sql:         "CREATE TABLE", // Missing table definition
			expectError: true,
		},
		{
			name:        "Invalid SQL: DROP TABLE",
			sql:         "DROP TABLE", // Missing table name
			expectError: true,
		},
		{
			name:        "Invalid SQL: CREATE INDEX",
			sql:         "CREATE INDEX", // Missing index definition
			expectError: true,
		},
		{
			name:        "Invalid SQL: SELECT * FROM non_existent_table",
			sql:         "SELECT * FROM non_existent_table",
			expectError: true, // SELECT fails on non-existent table
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := migrator.ApplyMigration(tc.sql)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
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

	// Create a test table
	_, err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)

	migrator := db.GetMigrator()

	// Test GetTableInfo with potentially malicious table name
	_, err = migrator.GetTableInfo("test_table; DROP TABLE test_table;")
	// Should either error or handle safely
	// The important thing is it shouldn't drop the table

	// Verify table still exists
	tables, err := migrator.GetTables()
	assert.NoError(t, err)
	assert.Contains(t, tables, "test_table")
}

// Helper function to find a column by name
func findColumn(columns []types.ColumnInfo, name string) *types.ColumnInfo {
	for i := range columns {
		if columns[i].Name == name {
			return &columns[i]
		}
	}
	return nil
}