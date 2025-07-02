package utils

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanRows_WithMaps(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec(`CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		name TEXT,
		value INTEGER,
		data BLOB
	)`)
	require.NoError(t, err)

	// Insert test data
	_, err = db.Exec(`INSERT INTO test_table (id, name, value, data) VALUES 
		(1, 'test1', 100, 'binary1'),
		(2, 'test2', 200, 'binary2'),
		(3, 'test3', 300, 'binary3')`)
	require.NoError(t, err)

	// Test scanning into []map[string]any
	rows, err := db.Query("SELECT * FROM test_table ORDER BY id")
	require.NoError(t, err)
	defer rows.Close()

	var results []map[string]any
	err = ScanRows(rows, &results)
	require.NoError(t, err)

	// Verify results
	assert.Len(t, results, 3)

	// Check first row
	assert.Equal(t, int64(1), results[0]["id"])
	assert.Equal(t, "test1", results[0]["name"])
	assert.Equal(t, int64(100), results[0]["value"])
	assert.Equal(t, "binary1", results[0]["data"]) // Should be converted from []byte to string

	// Check second row
	assert.Equal(t, int64(2), results[1]["id"])
	assert.Equal(t, "test2", results[1]["name"])
	assert.Equal(t, int64(200), results[1]["value"])
	assert.Equal(t, "binary2", results[1]["data"])

	// Check third row
	assert.Equal(t, int64(3), results[2]["id"])
	assert.Equal(t, "test3", results[2]["name"])
	assert.Equal(t, int64(300), results[2]["value"])
	assert.Equal(t, "binary3", results[2]["data"])
}

func TestScanRows_WithStructs(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		name TEXT,
		email TEXT
	)`)
	require.NoError(t, err)

	// Insert test data
	_, err = db.Exec(`INSERT INTO users (id, name, email) VALUES 
		(1, 'John Doe', 'john@example.com'),
		(2, 'Jane Smith', 'jane@example.com')`)
	require.NoError(t, err)

	// Define struct
	type User struct {
		ID    int
		Name  string
		Email string
	}

	// Test scanning into []User
	rows, err := db.Query("SELECT * FROM users ORDER BY id")
	require.NoError(t, err)
	defer rows.Close()

	var users []User
	err = ScanRows(rows, &users)
	require.NoError(t, err)

	// Verify results
	assert.Len(t, users, 2)
	assert.Equal(t, 1, users[0].ID)
	assert.Equal(t, "John Doe", users[0].Name)
	assert.Equal(t, "john@example.com", users[0].Email)
	assert.Equal(t, 2, users[1].ID)
	assert.Equal(t, "Jane Smith", users[1].Name)
	assert.Equal(t, "jane@example.com", users[1].Email)
}

func TestScanRowsToMaps_Direct(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create test table
	_, err = db.Exec(`CREATE TABLE test_table (
		id INTEGER PRIMARY KEY,
		name TEXT,
		nullable_value INTEGER
	)`)
	require.NoError(t, err)

	// Insert test data with NULL value
	_, err = db.Exec(`INSERT INTO test_table (id, name, nullable_value) VALUES 
		(1, 'test1', 100),
		(2, 'test2', NULL)`)
	require.NoError(t, err)

	// Test ScanRowsToMaps directly
	rows, err := db.Query("SELECT * FROM test_table ORDER BY id")
	require.NoError(t, err)
	defer rows.Close()

	results, err := ScanRowsToMaps(rows)
	require.NoError(t, err)

	// Verify results
	assert.Len(t, results, 2)

	// Check handling of non-NULL value
	assert.Equal(t, int64(1), results[0]["id"])
	assert.Equal(t, "test1", results[0]["name"])
	assert.Equal(t, int64(100), results[0]["nullable_value"])

	// Check handling of NULL value
	assert.Equal(t, int64(2), results[1]["id"])
	assert.Equal(t, "test2", results[1]["name"])
	assert.Nil(t, results[1]["nullable_value"])
}
