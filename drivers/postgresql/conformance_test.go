package postgresql

import (
	"context"
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestPostgreSQLConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conformance tests in short mode")
	}

	// Skip if PostgreSQL is not available
	config := test.GetTestConfig("postgresql")
	db, err := NewPostgreSQLDB(config)
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	if err := db.Connect(context.Background()); err != nil {
		t.Skipf("Cannot connect to PostgreSQL: %v", err)
	}
	db.Close()

	suite := &test.DriverConformanceTests{
		DriverName: "PostgreSQL",
		NewDriver: func(cfg types.Config) (types.Database, error) {
			return NewPostgreSQLDB(cfg)
		},
		Config:    config,
		SkipTests: map[string]bool{
			// PostgreSQL-specific skips
			"TestTransactionErrorHandling": true, // PostgreSQL aborts transaction on error
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			pgDB, ok := db.(*PostgreSQLDB)
			if ok {
				cleanupTables(t, pgDB)
			}
		},
		Characteristics: test.DriverCharacteristics{
			ReturnsZeroRowsAffectedForUnchanged: false,
			SupportsLastInsertID: false,
			SupportsReturningClause: true,
			MigrationTableName: "_migrations",
			SystemIndexPatterns: []string{"_pkey", "_key", "_fkey", "pg_*"},
			AutoIncrementIntegerType: "SERIAL",
		},
	}

	suite.RunAll(t)
}

// cleanupTables removes all non-system tables from the database
func cleanupTables(t *testing.T, db *PostgreSQLDB) {
	ctx := context.Background()

	// Get all tables in public schema
	rows, err := db.DB.QueryContext(ctx, `
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public'
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

	// Drop all tables with CASCADE
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS "%s" CASCADE`, table))
		if err != nil {
			t.Logf("Failed to drop table %s: %v", table, err)
		}
	}
}
