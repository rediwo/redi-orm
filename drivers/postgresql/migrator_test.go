package postgresql

import (
	"context"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgreSQLMigrator_NewPostgreSQLMigrator(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Get migrator
	migrator := db.GetMigrator()
	require.NotNil(t, migrator)

	// Test that it's a wrapper
	_, ok := migrator.(*PostgreSQLMigratorWrapper)
	require.True(t, ok)
}

func TestPostgreSQLMigrator_GetDatabaseType(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	assert.Equal(t, "postgresql", migrator.GetDatabaseType())
}

func TestPostgreSQLMigrator_GetTables(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()

	// Create test tables
	_, _ = db.Exec("DROP TABLE IF EXISTS test_table1, test_table2")
	_, err = db.Exec("CREATE TABLE test_table1 (id INT PRIMARY KEY)")
	require.NoError(t, err)
	_, err = db.Exec("CREATE TABLE test_table2 (id INT PRIMARY KEY)")
	require.NoError(t, err)
	defer db.Exec("DROP TABLE IF EXISTS test_table1, test_table2")

	// Get tables
	tables, err := migrator.GetTables()
	assert.NoError(t, err)

	// Check that our test tables are in the list
	var foundTable1, foundTable2 bool
	for _, table := range tables {
		if table == "test_table1" {
			foundTable1 = true
		}
		if table == "test_table2" {
			foundTable2 = true
		}
	}
	assert.True(t, foundTable1)
	assert.True(t, foundTable2)
}

func TestPostgreSQLMigrator_GetTableInfo(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create a test table with various column types
	_, _ = db.Exec("DROP TABLE IF EXISTS test_table")
	_, err = db.Exec(`
		CREATE TABLE test_table (
			id BIGSERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			email VARCHAR(255) UNIQUE,
			age INTEGER,
			active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)
	defer db.Exec("DROP TABLE IF EXISTS test_table")

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
	assert.False(t, idCol.Nullable)
	assert.Equal(t, "BIGINT", idCol.Type)

	// Check name column
	nameCol := findColumn(tableInfo.Columns, "name")
	require.NotNil(t, nameCol)
	assert.False(t, nameCol.Nullable)
	assert.Equal(t, "VARCHAR(100)", nameCol.Type)

	// Check email column
	emailCol := findColumn(tableInfo.Columns, "email")
	require.NotNil(t, emailCol)
	assert.True(t, emailCol.Unique)
	assert.Equal(t, "VARCHAR(255)", emailCol.Type)

	// Check active column
	activeCol := findColumn(tableInfo.Columns, "active")
	require.NotNil(t, activeCol)
	assert.Equal(t, "TRUE", activeCol.Default)

	// Check indexes (should include our unique index)
	foundEmailIndex := false
	for _, idx := range tableInfo.Indexes {
		if idx.Name == "idx_email" {
			foundEmailIndex = true
			assert.True(t, idx.Unique)
			assert.Equal(t, []string{"email"}, idx.Columns)
		}
	}
	assert.True(t, foundEmailIndex)
}

func TestPostgreSQLMigrator_GenerateCreateTableSQL(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
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
	assert.Contains(t, sql, `CREATE TABLE IF NOT EXISTS "users"`)
	assert.Contains(t, sql, `"id" BIGSERIAL`)
	assert.Contains(t, sql, `"name" VARCHAR(255) NOT NULL`)
	assert.Contains(t, sql, `"email" VARCHAR(255) NOT NULL UNIQUE`)
	assert.Contains(t, sql, `"age" INTEGER`)
	assert.Contains(t, sql, `"active" BOOLEAN NOT NULL DEFAULT TRUE`)
	assert.Contains(t, sql, `PRIMARY KEY ("id")`)
}

func TestPostgreSQLMigrator_GenerateModifyColumnSQL(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()

	t.Run("Add UNIQUE constraint to column", func(t *testing.T) {
		change := types.ColumnChange{
			TableName:  "users",
			ColumnName: "email",
			OldColumn: &types.ColumnInfo{
				Name: "email",
				Type: "VARCHAR(255)",
			},
			NewColumn: &types.ColumnInfo{
				Name:   "email",
				Type:   "VARCHAR(255)",
				Unique: true,
			},
		}

		sqls, err := migrator.GenerateModifyColumnSQL(change)
		assert.NoError(t, err)
		assert.Greater(t, len(sqls), 0)
		// Should contain ALTER TABLE for type, NULL/NOT NULL, default, and UNIQUE constraint
		hasAlterTable := false
		hasUniqueConstraint := false
		for _, sql := range sqls {
			if strings.Contains(sql, "ALTER TABLE") {
				hasAlterTable = true
			}
			if strings.Contains(sql, "UNIQUE") || strings.Contains(sql, "ADD CONSTRAINT") {
				hasUniqueConstraint = true
			}
		}
		assert.True(t, hasAlterTable)
		assert.True(t, hasUniqueConstraint)
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
				Type: "BIGINT",
			},
		}

		sqls, err := migrator.GenerateModifyColumnSQL(change)
		assert.NoError(t, err)
		assert.Greater(t, len(sqls), 0)
		hasTypeChange := false
		for _, sql := range sqls {
			if strings.Contains(sql, "ALTER COLUMN") && strings.Contains(sql, "TYPE BIGINT") {
				hasTypeChange = true
			}
		}
		assert.True(t, hasTypeChange)
	})

	t.Run("Rename column", func(t *testing.T) {
		change := types.ColumnChange{
			TableName:  "users",
			ColumnName: "name",
			OldColumn: &types.ColumnInfo{
				Name: "name",
				Type: "VARCHAR(100)",
			},
			NewColumn: &types.ColumnInfo{
				Name: "full_name",
				Type: "VARCHAR(100)",
			},
		}

		sqls, err := migrator.GenerateModifyColumnSQL(change)
		assert.NoError(t, err)
		assert.Greater(t, len(sqls), 0)
		hasRename := false
		for _, sql := range sqls {
			if strings.Contains(sql, "RENAME COLUMN") {
				hasRename = true
			}
		}
		assert.True(t, hasRename)
	})
}

func TestPostgreSQLMigrator_GenerateModifyColumnSQL_Integration(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create a test table with data
	_, _ = db.Exec("DROP TABLE IF EXISTS test_products")
	_, err = db.Exec(`
		CREATE TABLE test_products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			price INTEGER,
			description TEXT
		)
	`)
	require.NoError(t, err)
	defer db.Exec("DROP TABLE IF EXISTS test_products")

	// Insert some test data
	_, err = db.Exec("INSERT INTO test_products (name, price, description) VALUES ($1, $2, $3)", "Widget", 100, "A nice widget")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO test_products (name, price, description) VALUES ($1, $2, $3)", "Gadget", 200, "A cool gadget")
	require.NoError(t, err)

	migrator := db.GetMigrator()

	// Test actual execution of column modification
	change := types.ColumnChange{
		TableName:  "test_products",
		ColumnName: "price",
		OldColumn: &types.ColumnInfo{
			Name: "price",
			Type: "INTEGER",
		},
		NewColumn: &types.ColumnInfo{
			Name:     "price",
			Type:     "DECIMAL(10,2)",
			Nullable: false,
			Default:  "0.00",
		},
	}

	sqls, err := migrator.GenerateModifyColumnSQL(change)
	require.NoError(t, err)

	// Execute the migration SQL
	for _, sql := range sqls {
		_, err = db.Exec(sql)
		require.NoError(t, err, "Failed to execute: %s", sql)
	}

	// Verify the column type changed
	tableInfo, err := migrator.GetTableInfo("test_products")
	require.NoError(t, err)

	priceCol := findColumn(tableInfo.Columns, "price")
	require.NotNil(t, priceCol)
	assert.Equal(t, "DECIMAL(10,2)", priceCol.Type)
	assert.Equal(t, "0.00", priceCol.Default)

	// Verify data integrity
	rows, err := db.Query("SELECT name, price FROM test_products ORDER BY name")
	require.NoError(t, err)
	defer rows.Close()

	var results []struct {
		name  string
		price float64
	}

	for rows.Next() {
		var r struct {
			name  string
			price float64
		}
		err = rows.Scan(&r.name, &r.price)
		require.NoError(t, err)
		results = append(results, r)
	}

	assert.Len(t, results, 2)
	assert.Equal(t, "Gadget", results[0].name)
	assert.Equal(t, 200.0, results[0].price)
	assert.Equal(t, "Widget", results[1].name)
	assert.Equal(t, 100.0, results[1].price)
}

func TestPostgreSQLMigrator_GenerateDropColumnSQL(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()
	sqls, err := migrator.GenerateDropColumnSQL("users", "old_column")
	assert.NoError(t, err)
	assert.Len(t, sqls, 1)
	assert.Equal(t, `ALTER TABLE "users" DROP COLUMN "old_column"`, sqls[0])
}

func TestPostgreSQLMigrator_GenerateCreateIndexSQL(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()

	// Regular index
	sql := migrator.GenerateCreateIndexSQL("users", "idx_name", []string{"name"}, false)
	assert.Equal(t, `CREATE INDEX "idx_name" ON "users" ("name")`, sql)

	// Unique index
	sql2 := migrator.GenerateCreateIndexSQL("users", "idx_email", []string{"email"}, true)
	assert.Equal(t, `CREATE UNIQUE INDEX "idx_email" ON "users" ("email")`, sql2)

	// Composite index
	sql3 := migrator.GenerateCreateIndexSQL("users", "idx_name_email", []string{"name", "email"}, false)
	assert.Equal(t, `CREATE INDEX "idx_name_email" ON "users" ("name", "email")`, sql3)
}

func TestPostgreSQLMigrator_ApplyMigration(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()

	// Apply a CREATE TABLE migration
	sql := "CREATE TABLE IF NOT EXISTS test_migration (id INT PRIMARY KEY)"
	err = migrator.ApplyMigration(sql)
	assert.NoError(t, err)

	// Verify table was created
	tables, err := migrator.GetTables()
	assert.NoError(t, err)
	assert.Contains(t, tables, "test_migration")

	// Clean up
	err = migrator.ApplyMigration("DROP TABLE IF EXISTS test_migration")
	assert.NoError(t, err)
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

func TestPostgreSQLMigrator_IndexComparison(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	migrator := db.GetMigrator()

	tests := []struct {
		name          string
		setupTable    func()
		desiredSchema *schema.Schema
		expectedAdds  int
		expectedDrops int
	}{
		{
			name: "add new index",
			setupTable: func() {
				// Create a simple table without indexes
				sql := `CREATE TABLE users (
					id BIGSERIAL PRIMARY KEY,
					email VARCHAR(255),
					user_name VARCHAR(255)
				)`
				err := migrator.ApplyMigration(sql)
				require.NoError(t, err)
			},
			desiredSchema: func() *schema.Schema {
				s := schema.New("User").WithTableName("users")
				s.AddField(schema.Field{
					Name:          "id",
					Type:          schema.FieldTypeInt64,
					PrimaryKey:    true,
					AutoIncrement: true,
				})
				s.AddField(schema.Field{
					Name: "email",
					Type: schema.FieldTypeString,
				})
				s.AddField(schema.Field{
					Name: "userName",
					Type: schema.FieldTypeString,
					Map:  "user_name",
				})
				s.AddIndex(schema.Index{
					Fields: []string{"email"},
				})
				return s
			}(),
			expectedAdds:  1,
			expectedDrops: 0,
		},
		{
			name: "drop existing index",
			setupTable: func() {
				// Create table with an index
				sqls := []string{
					`CREATE TABLE users (
						id BIGSERIAL PRIMARY KEY,
						email VARCHAR(255),
						user_name VARCHAR(255)
					)`,
					`CREATE INDEX idx_users_email ON users(email)`,
				}
				for _, sql := range sqls {
					err := migrator.ApplyMigration(sql)
					require.NoError(t, err)
				}
			},
			desiredSchema: func() *schema.Schema {
				s := schema.New("User").WithTableName("users")
				s.AddField(schema.Field{
					Name:          "id",
					Type:          schema.FieldTypeInt64,
					PrimaryKey:    true,
					AutoIncrement: true,
				})
				s.AddField(schema.Field{
					Name: "email",
					Type: schema.FieldTypeString,
				})
				s.AddField(schema.Field{
					Name: "userName",
					Type: schema.FieldTypeString,
					Map:  "user_name",
				})
				// No indexes
				return s
			}(),
			expectedAdds:  0,
			expectedDrops: 1,
		},
		{
			name: "composite index",
			setupTable: func() {
				// Create a simple table without indexes
				sql := `CREATE TABLE users (
					id BIGSERIAL PRIMARY KEY,
					email VARCHAR(255),
					user_name VARCHAR(255),
					status VARCHAR(255)
				)`
				err := migrator.ApplyMigration(sql)
				require.NoError(t, err)
			},
			desiredSchema: func() *schema.Schema {
				s := schema.New("User").WithTableName("users")
				s.AddField(schema.Field{
					Name:          "id",
					Type:          schema.FieldTypeInt64,
					PrimaryKey:    true,
					AutoIncrement: true,
				})
				s.AddField(schema.Field{
					Name: "email",
					Type: schema.FieldTypeString,
				})
				s.AddField(schema.Field{
					Name: "userName",
					Type: schema.FieldTypeString,
					Map:  "user_name",
				})
				s.AddField(schema.Field{
					Name: "status",
					Type: schema.FieldTypeString,
				})
				s.AddIndex(schema.Index{
					Fields: []string{"email", "status"},
				})
				return s
			}(),
			expectedAdds:  1,
			expectedDrops: 0,
		},
		{
			name: "field with column mapping in index",
			setupTable: func() {
				// Create a simple table without indexes
				sql := `CREATE TABLE users (
					id BIGSERIAL PRIMARY KEY,
					email VARCHAR(255),
					user_name VARCHAR(255),
					created_at TIMESTAMP
				)`
				err := migrator.ApplyMigration(sql)
				require.NoError(t, err)
			},
			desiredSchema: func() *schema.Schema {
				s := schema.New("User").WithTableName("users")
				s.AddField(schema.Field{
					Name:          "id",
					Type:          schema.FieldTypeInt64,
					PrimaryKey:    true,
					AutoIncrement: true,
				})
				s.AddField(schema.Field{
					Name: "email",
					Type: schema.FieldTypeString,
				})
				s.AddField(schema.Field{
					Name: "userName",
					Type: schema.FieldTypeString,
					Map:  "user_name",
				})
				s.AddField(schema.Field{
					Name: "createdAt",
					Type: schema.FieldTypeDateTime,
					Map:  "created_at",
				})
				s.AddIndex(schema.Index{
					Fields: []string{"createdAt"}, // Should use created_at column
				})
				return s
			}(),
			expectedAdds:  1,
			expectedDrops: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up from previous test
			migrator.ApplyMigration("DROP TABLE IF EXISTS users")

			// Setup table
			tt.setupTable()

			// Register schema
			err = db.RegisterSchema("User", tt.desiredSchema)
			require.NoError(t, err)

			// Get existing table info
			existingTable, err := migrator.GetTableInfo("users")
			require.NoError(t, err)

			// Compare schemas
			plan, err := migrator.CompareSchema(existingTable, tt.desiredSchema)
			require.NoError(t, err)

			// Check index changes
			assert.Equal(t, tt.expectedAdds, len(plan.AddIndexes), "Added indexes count mismatch")
			assert.Equal(t, tt.expectedDrops, len(plan.DropIndexes), "Dropped indexes count mismatch")

			// Cleanup
			migrator.ApplyMigration("DROP TABLE IF EXISTS users")
		})
	}
}

func TestPostgreSQLMigrator_SystemIndexDetection(t *testing.T) {
	skipIfPostgreSQLNotAvailable(t)

	config := getTestConfig()
	db, err := NewPostgreSQLDB(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.Connect(ctx)
	require.NoError(t, err)
	defer db.Close()

	// Create PostgreSQL-specific migrator
	postgresDB := db
	migrator := &PostgreSQLMigrator{
		db:           postgresDB.GetDB(),
		postgresqlDB: postgresDB,
	}

	// Test system index patterns
	tests := []struct {
		indexName     string
		isSystemIndex bool
	}{
		{"users_pkey", true},
		{"posts_pkey", true},
		{"email_key", true},
		{"users_email_key", true},
		{"users_posts_fkey", true},
		{"pg_internal_index", true},
		{"idx_users_email", false},
		{"custom_index", false},
		{"users_email_idx", false},
		{"unique_email", false},
	}

	for _, tt := range tests {
		t.Run(tt.indexName, func(t *testing.T) {
			result := migrator.IsSystemIndex(tt.indexName)
			assert.Equal(t, tt.isSystemIndex, result, "IsSystemIndex result mismatch for %s", tt.indexName)
		})
	}
}
