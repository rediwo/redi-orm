package drivers

import (
	"testing"

	"github.com/rediwo/redi-orm/types"
)

func TestSQLiteURIParser_ParseURI(t *testing.T) {
	parser := &SQLiteURIParser{}

	tests := []struct {
		name        string
		uri         string
		expected    types.Config
		expectError bool
	}{
		{
			name: "SQLite memory database",
			uri:  "sqlite://:memory:",
			expected: types.Config{
				Type:     "sqlite",
				FilePath: ":memory:",
			},
			expectError: false,
		},
		{
			name: "SQLite memory database path format",
			uri:  "sqlite:///:memory:",
			expected: types.Config{
				Type:     "sqlite",
				FilePath: ":memory:",
			},
			expectError: false,
		},
		{
			name: "SQLite absolute file path",
			uri:  "sqlite:///path/to/database.db",
			expected: types.Config{
				Type:     "sqlite",
				FilePath: "/path/to/database.db",
			},
			expectError: false,
		},
		{
			name: "SQLite relative file path",
			uri:  "sqlite://database.db",
			expected: types.Config{
				Type:     "sqlite",
				FilePath: "database.db",
			},
			expectError: false,
		},
		{
			name: "SQLite host with path",
			uri:  "sqlite://localhost/path/to/db.sqlite",
			expected: types.Config{
				Type:     "sqlite",
				FilePath: "localhost/path/to/db.sqlite",
			},
			expectError: false,
		},
		{
			name:        "Invalid scheme",
			uri:         "mysql://test.db",
			expectError: true,
		},
		{
			name:        "Empty file path",
			uri:         "sqlite://",
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "sqlite://[invalid-uri",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parser.ParseURI(tt.uri)

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

			if config.Type != tt.expected.Type {
				t.Errorf("Expected Type %s, got %s", tt.expected.Type, config.Type)
			}

			if config.FilePath != tt.expected.FilePath {
				t.Errorf("Expected FilePath %s, got %s", tt.expected.FilePath, config.FilePath)
			}
		})
	}
}

func TestSQLiteURIParser_GetSupportedSchemes(t *testing.T) {
	parser := &SQLiteURIParser{}
	schemes := parser.GetSupportedSchemes()

	expected := []string{"sqlite"}
	if len(schemes) != len(expected) {
		t.Errorf("Expected %d schemes, got %d", len(expected), len(schemes))
	}

	for i, scheme := range schemes {
		if scheme != expected[i] {
			t.Errorf("Expected scheme %s, got %s", expected[i], scheme)
		}
	}
}

func TestSQLiteURIParser_GetDriverType(t *testing.T) {
	parser := &SQLiteURIParser{}
	driverType := parser.GetDriverType()

	expected := "sqlite"
	if driverType != expected {
		t.Errorf("Expected driver type %s, got %s", expected, driverType)
	}
}
