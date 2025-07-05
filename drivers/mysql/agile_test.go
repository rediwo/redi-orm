package mysql

import (
	"context"
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/agile"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestMySQLAgileConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping agile conformance tests in short mode")
	}

	// Get test database URI and parse it
	uri := test.GetTestDatabaseUri("mysql")

	suite := &agile.AgileConformanceTests{
		DriverName:  "MySQL",
		DatabaseURI: uri,
		NewDatabase: func(uri string) (types.Database, error) {
			config, err := NewMySQLURIParser().ParseURI(uri)
			if err != nil {
				return nil, err
			}
			return NewMySQLDB(config)
		},
		SkipTests: map[string]bool{
			// MySQL-specific skips (if any)
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			// MySQL-specific cleanup
			if mysqlDB, ok := db.(*MySQLDB); ok {
				cleanupMySQLTables(t, mysqlDB)
			}
		},
		Characteristics: agile.AgileDriverCharacteristics{
			SupportsReturning:          false,
			MaxConnectionPoolSize:      10,
			SupportsNestedTransactions: true,
			ReturnsStringForNumbers:    true, // MySQL returns strings for numeric aggregations
		},
	}

	suite.RunAll(t)
}

// cleanupMySQLTables removes all non-system tables from the database
func cleanupMySQLTables(t *testing.T, db *MySQLDB) {
	ctx := context.Background()

	// Get all tables
	rows, err := db.DB.QueryContext(ctx, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = DATABASE()
		AND table_name NOT LIKE 'mysql_%'
		AND table_name != '_migrations'
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

	// Disable foreign key checks
	_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	defer func() {
		_, _ = db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	}()

	// Drop all tables
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table))
		if err != nil {
			t.Logf("Failed to drop table %s: %v", table, err)
		}
	}
}
