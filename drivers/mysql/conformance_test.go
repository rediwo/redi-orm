package mysql

import (
	"context"
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestMySQLConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conformance tests in short mode")
	}

	// Skip if MySQL is not available
	config := test.GetTestConfig("mysql")
	db, err := NewMySQLDB(config)
	if err != nil {
		t.Skip("MySQL not available for testing")
	}
	if err := db.Connect(context.Background()); err != nil {
		t.Skipf("Cannot connect to MySQL: %v", err)
	}
	db.Close()

	suite := &test.DriverConformanceTests{
		DriverName: "MySQL",
		NewDriver: func(cfg types.Config) (types.Database, error) {
			return NewMySQLDB(cfg)
		},
		Config: config,
		SkipTests: map[string]bool{
			// MySQL-specific skips
			"TestAggregations": true, // MySQL returns aggregation results as strings
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			mysqlDB, ok := db.(*MySQLDB)
			if ok {
				cleanupTables(t, mysqlDB)
			}
		},
		Characteristics: test.DriverCharacteristics{
			ReturnsZeroRowsAffectedForUnchanged: true,
			SupportsLastInsertID: true,
			SupportsReturningClause: false,
			MigrationTableName: "_migrations",
			SystemIndexPatterns: []string{"PRIMARY", "fk_*", "mysql_*"},
			AutoIncrementIntegerType: "INT AUTO_INCREMENT",
		},
	}

	suite.RunAll(t)
}

// cleanupTables removes all non-system tables from the database
func cleanupTables(t *testing.T, db *MySQLDB) {
	ctx := context.Background()

	// Disable foreign key checks
	_, err := db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	if err != nil {
		t.Logf("Failed to disable foreign key checks: %v", err)
		return
	}
	defer db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Get all tables
	rows, err := db.DB.QueryContext(ctx, "SHOW TABLES")
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
