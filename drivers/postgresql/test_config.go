package postgresql

import (
	"context"
	"fmt"
	"testing"

	"github.com/rediwo/redi-orm/test"
)

func init() {
	host := test.GetEnvOrDefault("POSTGRES_TEST_HOST", "localhost")
	user := test.GetEnvOrDefault("POSTGRES_TEST_USER", "testuser")
	password := test.GetEnvOrDefault("POSTGRES_TEST_PASSWORD", "testpass")
	database := test.GetEnvOrDefault("POSTGRES_TEST_DATABASE", "testdb")

	uri := fmt.Sprintf("postgresql://%s:%s@%s:5432/%s?sslmode=disable",
		user, password, host, database)

	test.RegisterTestDatabaseUri("postgresql", uri)
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
