package postgresql

import (
	"context"
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/orm"
	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestPostgreSQLOrmConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping orm conformance tests in short mode")
	}

	// Get test database URI and parse it
	uri := test.GetTestDatabaseUri("postgresql")

	suite := &orm.OrmConformanceTests{
		DriverName:  "PostgreSQL",
		DatabaseURI: uri,
		NewDatabase: func(uri string) (types.Database, error) {
			return database.NewFromURI(uri)
		},
		SkipTests: map[string]bool{
			// PostgreSQL-specific skips (if any)
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			// PostgreSQL-specific cleanup
			if pgDB, ok := db.(*PostgreSQLDB); ok {
				cleanupPostgreSQLTables(t, pgDB)
			}
		},
		Characteristics: orm.OrmDriverCharacteristics{
			SupportsReturning:          true,
			MaxConnectionPoolSize:      10,
			SupportsNestedTransactions: true,
			ReturnsStringForNumbers:    false,
		},
	}

	suite.RunAll(t)
}

// cleanupPostgreSQLTables removes all non-system tables from the database
func cleanupPostgreSQLTables(t *testing.T, db *PostgreSQLDB) {
	ctx := context.Background()

	// Get all tables in public schema
	rows, err := db.DB.QueryContext(ctx, `
		SELECT tablename 
		FROM pg_tables 
		WHERE schemaname = 'public'
		AND tablename NOT LIKE 'pg_%'
		AND tablename != '_migrations'
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

	// Drop all tables with CASCADE to handle foreign keys
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS \"%s\" CASCADE", table))
		if err != nil {
			t.Logf("Failed to drop table %s: %v", table, err)
		}
	}
}
