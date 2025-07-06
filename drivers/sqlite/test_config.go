package sqlite

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/rediwo/redi-orm/test"
)

var (
	tempFileOnce sync.Once
	tempFileURI  string
)

func init() {
	// Create a temporary file URI once and reuse it
	tempFileOnce.Do(func() {
		tempFile, err := os.CreateTemp("", "sqlite-test-*.db")
		if err != nil {
			panic("failed to create temp file for SQLite: " + err.Error())
		}
		tempFile.Close()
		tempFileURI = "sqlite://" + tempFile.Name()
	})

	test.RegisterTestDatabaseUri("sqlite", tempFileURI)
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
