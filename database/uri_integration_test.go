package database

import (
	"testing"
)

func TestNewFromURI_Integration(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		expectError bool
	}{
		{
			name:        "SQLite memory database",
			uri:         "sqlite://:memory:",
			expectError: false,
		},
		{
			name:        "SQLite file database",
			uri:         "sqlite:///tmp/test.db",
			expectError: false,
		},
		{
			name:        "MySQL database",
			uri:         "mysql://user:pass@localhost:3306/test",
			expectError: false, // Driver registered, but connection may fail
		},
		{
			name:        "PostgreSQL database",
			uri:         "postgresql://user:pass@localhost:5432/test",
			expectError: false, // Driver registered, but connection may fail
		},
		{
			name:        "PostgreSQL with postgres scheme",
			uri:         "postgres://user:pass@localhost:5432/test",
			expectError: false, // Driver registered, but connection may fail
		},
		{
			name:        "Unsupported scheme",
			uri:         "mongodb://localhost:27017/test",
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "not-a-valid-uri",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := NewFromURI(tt.uri)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for URI %s, but got none", tt.uri)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for URI %s: %v", tt.uri, err)
				return
			}

			if db == nil {
				t.Errorf("Expected database instance for URI %s, but got nil", tt.uri)
			}

			// Don't test actual connection since it may require external services
		})
	}
}

func TestNewFromURI_SQLiteIntegration(t *testing.T) {
	// Test SQLite end-to-end since it doesn't require external services
	db, err := NewFromURI("sqlite://:memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite database from URI: %v", err)
	}

	err = db.Connect()
	if err != nil {
		t.Fatalf("Failed to connect to SQLite database: %v", err)
	}
	defer db.Close()

	// Test creating a simple table
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Test inserting data
	_, err = db.Exec("INSERT INTO test (name) VALUES (?)", "test-data")
	if err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Test querying data
	rows, err := db.Query("SELECT name FROM test WHERE id = ?", 1)
	if err != nil {
		t.Fatalf("Failed to query data: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("Expected at least one row")
	}

	var name string
	err = rows.Scan(&name)
	if err != nil {
		t.Fatalf("Failed to scan row: %v", err)
	}

	if name != "test-data" {
		t.Errorf("Expected name 'test-data', got '%s'", name)
	}
}