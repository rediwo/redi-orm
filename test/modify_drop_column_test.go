package test

import (
	"database/sql"
	"testing"

	mysqldriver "github.com/rediwo/redi-orm/drivers/mysql"
	postgresqldriver "github.com/rediwo/redi-orm/drivers/postgresql"
	sqlitedriver "github.com/rediwo/redi-orm/drivers/sqlite"
	"github.com/rediwo/redi-orm/types"
	
	_ "github.com/mattn/go-sqlite3"
)

// TestModifyColumnSQLGeneration tests database-specific SQL generation for column modifications
func TestModifyColumnSQLGeneration(t *testing.T) {
	// Create a dummy database connection for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Test cases for different databases
	testCases := []struct {
		name     string
		migrator types.DatabaseMigrator
		change   types.ColumnChange
		expected []string
	}{
		{
			name:     "SQLite modify column",
			migrator: sqlitedriver.NewSQLiteMigrator(db),
			change: types.ColumnChange{
				TableName:  "users",
				ColumnName: "age",
				OldColumn: &types.ColumnInfo{
					Name:     "age",
					Type:     "INTEGER",
					Nullable: true,
				},
				NewColumn: &types.ColumnInfo{
					Name:     "age",
					Type:     "INTEGER",
					Nullable: false,
					Default:  18,
				},
			},
			expected: []string{
				"-- SQLite: MODIFY COLUMN age in table users requires table recreation",
			},
		},
		{
			name:     "MySQL modify column",
			migrator: mysqldriver.NewMySQLMigrator(db),
			change: types.ColumnChange{
				TableName:  "users",
				ColumnName: "age",
				OldColumn: &types.ColumnInfo{
					Name:     "age",
					Type:     "INT",
					Nullable: true,
				},
				NewColumn: &types.ColumnInfo{
					Name:     "age",
					Type:     "INT",
					Nullable: false,
					Default:  18,
				},
			},
			expected: []string{
				"ALTER TABLE users MODIFY COLUMN age INT NOT NULL DEFAULT 18",
			},
		},
		{
			name:     "PostgreSQL modify column - multiple changes",
			migrator: postgresqldriver.NewPostgreSQLMigrator(db),
			change: types.ColumnChange{
				TableName:  "users",
				ColumnName: "age",
				OldColumn: &types.ColumnInfo{
					Name:     "age",
					Type:     "INTEGER",
					Nullable: true,
				},
				NewColumn: &types.ColumnInfo{
					Name:     "age",
					Type:     "BIGINT",
					Nullable: false,
					Default:  18,
				},
			},
			expected: []string{
				"ALTER TABLE users ALTER COLUMN age TYPE BIGINT",
				"ALTER TABLE users ALTER COLUMN age SET NOT NULL",
				"ALTER TABLE users ALTER COLUMN age SET DEFAULT 18",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sqls, err := tc.migrator.GenerateModifyColumnSQL(tc.change)
			if err != nil {
				t.Fatalf("Failed to generate modify column SQL: %v", err)
			}

			if len(sqls) != len(tc.expected) {
				t.Errorf("Expected %d SQL statements, got %d", len(tc.expected), len(sqls))
				t.Errorf("Expected: %v", tc.expected)
				t.Errorf("Got: %v", sqls)
				return
			}

			for i, sql := range sqls {
				if sql != tc.expected[i] {
					t.Errorf("SQL statement %d mismatch:\nExpected: %s\nGot: %s", 
						i, tc.expected[i], sql)
				}
			}
		})
	}
}

// TestDropColumnSQLGeneration tests database-specific SQL generation for column drops
func TestDropColumnSQLGeneration(t *testing.T) {
	// Create a dummy database connection for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Test cases for different databases
	testCases := []struct {
		name       string
		migrator   types.DatabaseMigrator
		tableName  string
		columnName string
		expected   []string
	}{
		{
			name:       "SQLite drop column",
			migrator:   sqlitedriver.NewSQLiteMigrator(db),
			tableName:  "users",
			columnName: "old_field",
			expected: []string{
				"-- SQLite: DROP COLUMN old_field from table users requires table recreation",
			},
		},
		{
			name:       "MySQL drop column",
			migrator:   mysqldriver.NewMySQLMigrator(db),
			tableName:  "users",
			columnName: "old_field",
			expected: []string{
				"ALTER TABLE users DROP COLUMN old_field",
			},
		},
		{
			name:       "PostgreSQL drop column",
			migrator:   postgresqldriver.NewPostgreSQLMigrator(db),
			tableName:  "users",
			columnName: "old_field",
			expected: []string{
				"ALTER TABLE users DROP COLUMN old_field",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sqls, err := tc.migrator.GenerateDropColumnSQL(tc.tableName, tc.columnName)
			if err != nil {
				t.Fatalf("Failed to generate drop column SQL: %v", err)
			}

			if len(sqls) != len(tc.expected) {
				t.Errorf("Expected %d SQL statements, got %d", len(tc.expected), len(sqls))
				t.Errorf("Expected: %v", tc.expected)
				t.Errorf("Got: %v", sqls)
				return
			}

			for i, sql := range sqls {
				if sql != tc.expected[i] {
					t.Errorf("SQL statement %d mismatch:\nExpected: %s\nGot: %s", 
						i, tc.expected[i], sql)
				}
			}
		})
	}
}

// TestMigrationPlanSQLGeneration tests that the base migrator correctly delegates to database-specific implementations
func TestMigrationPlanSQLGeneration(t *testing.T) {
	// Create a dummy database connection for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create a migration plan with various changes
	plan := &types.MigrationPlan{
		ModifyColumns: []types.ColumnChange{
			{
				TableName:  "users",
				ColumnName: "age",
				OldColumn: &types.ColumnInfo{
					Name:     "age",
					Type:     "INT",
					Nullable: true,
				},
				NewColumn: &types.ColumnInfo{
					Name:     "age",
					Type:     "INT",
					Nullable: false,
					Default:  18,
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

	// Test with MySQL migrator
	mysqlMigrator := mysqldriver.NewMySQLMigrator(db)
	mysqlSqls, err := mysqlMigrator.GenerateMigrationSQL(plan)
	if err != nil {
		t.Fatalf("MySQL: Failed to generate migration SQL: %v", err)
	}

	// Verify MySQL generates proper SQL
	expectedMySQL := []string{
		"ALTER TABLE users MODIFY COLUMN age INT NOT NULL DEFAULT 18",
		"ALTER TABLE users DROP COLUMN old_field",
	}

	if len(mysqlSqls) != len(expectedMySQL) {
		t.Errorf("MySQL: Expected %d SQL statements, got %d", len(expectedMySQL), len(mysqlSqls))
	}

	// Test with SQLite migrator
	sqliteMigrator := sqlitedriver.NewSQLiteMigrator(db)
	sqliteSqls, err := sqliteMigrator.GenerateMigrationSQL(plan)
	if err != nil {
		t.Fatalf("SQLite: Failed to generate migration SQL: %v", err)
	}

	// Verify SQLite generates comments about table recreation
	for _, sql := range sqliteSqls {
		if !contains(sql, "requires table recreation") && !contains(sql, "--") {
			t.Errorf("SQLite: Expected comment about table recreation, got: %s", sql)
		}
	}

	t.Log("✅ Database-specific SQL generation works correctly")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || len(substr) < len(s) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}