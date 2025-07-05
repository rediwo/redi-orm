package sqlite

import (
	"context"
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestSQLiteConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conformance tests in short mode")
	}

	// Get test database URI and parse it
	uri := test.GetTestDatabaseUri("sqlite")
	config, err := NewSQLiteURIParser().ParseURI(uri)
	if err != nil {
		t.Fatalf("Failed to parse SQLite URI: %v", err)
	}

	suite := &test.DriverConformanceTests{
		DriverName: "SQLite",
		NewDriver: func(cfg types.Config) (types.Database, error) {
			return NewSQLiteDB(cfg)
		},
		Config: config,
		SkipTests: map[string]bool{
			// SQLite-specific skips
			"TestTransactionIsolation":        true, // SQLite doesn't support concurrent write transactions
			"TestTransactionConcurrentAccess": true, // SQLite uses database-level locking preventing concurrent writes
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			// SQLite with file-based database might benefit from cleanup
			sqliteDB, ok := db.(*SQLiteDB)
			if ok {
				cleanupTables(t, sqliteDB)
			}
		},
		Characteristics: test.DriverCharacteristics{
			ReturnsZeroRowsAffectedForUnchanged: false,
			SupportsLastInsertID:                true,
			SupportsReturningClause:             false,
			MigrationTableName:                  "_migrations",
			SystemIndexPatterns:                 []string{"sqlite_*", "pk_*"},
			AutoIncrementIntegerType:            "INTEGER",
		},
	}

	suite.RunAll(t)
}

// cleanupTables removes all non-system tables from the database
func cleanupTables(t *testing.T, db *SQLiteDB) {
	ctx := context.Background()

	// Get all tables
	rows, err := db.DB.QueryContext(ctx, `
		SELECT name FROM sqlite_master 
		WHERE type='table' 
		AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		t.Logf("Failed to get tables: %v", err)
		return
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			t.Logf("Failed to scan table name: %v", err)
			continue
		}
		tables = append(tables, table)
	}

	// Drop all tables
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table))
		if err != nil {
			t.Logf("Failed to drop table %s: %v", table, err)
		}
	}
}
