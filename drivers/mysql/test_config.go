package mysql

import (
	"context"
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/test"
)

func init() {
	host := test.GetEnvOrDefault("MYSQL_TEST_HOST", "localhost")
	user := test.GetEnvOrDefault("MYSQL_TEST_USER", "testuser")
	password := test.GetEnvOrDefault("MYSQL_TEST_PASSWORD", "testpass")
	database := test.GetEnvOrDefault("MYSQL_TEST_DATABASE", "testdb")

	uri := fmt.Sprintf("mysql://%s:%s@%s:3306/%s?parseTime=true",
		user, password, host, database)

	test.RegisterTestDatabaseUri("mysql", uri)
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
