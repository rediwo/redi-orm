package drivers

import (
	"testing"

	"github.com/rediwo/redi-orm/types"
)

func TestPostgreSQLURIParser_ParseURI(t *testing.T) {
	parser := &PostgreSQLURIParser{}

	tests := []struct {
		name        string
		uri         string
		expected    types.Config
		expectError bool
	}{
		{
			name: "PostgreSQL with all parameters",
			uri:  "postgresql://user:password@host:5432/database",
			expected: types.Config{
				Type:     "postgresql",
				Host:     "host",
				Port:     5432,
				User:     "user",
				Password: "password",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "PostgreSQL with postgres scheme",
			uri:  "postgres://user:password@host:5432/database",
			expected: types.Config{
				Type:     "postgresql",
				Host:     "host",
				Port:     5432,
				User:     "user",
				Password: "password",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "PostgreSQL with default port",
			uri:  "postgresql://user:password@host/database",
			expected: types.Config{
				Type:     "postgresql",
				Host:     "host",
				Port:     5432,
				User:     "user",
				Password: "password",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "PostgreSQL with default host",
			uri:  "postgresql://user:password@/database",
			expected: types.Config{
				Type:     "postgresql",
				Host:     "localhost",
				Port:     5432,
				User:     "user",
				Password: "password",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "PostgreSQL without password",
			uri:  "postgresql://user@host/database",
			expected: types.Config{
				Type:     "postgresql",
				Host:     "host",
				Port:     5432,
				User:     "user",
				Password: "",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "PostgreSQL without user info",
			uri:  "postgresql://host/database",
			expected: types.Config{
				Type:     "postgresql",
				Host:     "host",
				Port:     5432,
				User:     "",
				Password: "",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "PostgreSQL without database",
			uri:  "postgresql://user:password@host:5432",
			expected: types.Config{
				Type:     "postgresql",
				Host:     "host",
				Port:     5432,
				User:     "user",
				Password: "password",
				Database: "",
			},
			expectError: false,
		},
		{
			name:        "Invalid scheme",
			uri:         "mysql://user@host/db",
			expectError: true,
		},
		{
			name:        "Invalid port",
			uri:         "postgresql://user@host:invalid/db",
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "postgresql://[invalid-uri",
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
			if config.Host != tt.expected.Host {
				t.Errorf("Expected Host %s, got %s", tt.expected.Host, config.Host)
			}
			if config.Port != tt.expected.Port {
				t.Errorf("Expected Port %d, got %d", tt.expected.Port, config.Port)
			}
			if config.User != tt.expected.User {
				t.Errorf("Expected User %s, got %s", tt.expected.User, config.User)
			}
			if config.Password != tt.expected.Password {
				t.Errorf("Expected Password %s, got %s", tt.expected.Password, config.Password)
			}
			if config.Database != tt.expected.Database {
				t.Errorf("Expected Database %s, got %s", tt.expected.Database, config.Database)
			}
		})
	}
}

func TestPostgreSQLURIParser_GetSupportedSchemes(t *testing.T) {
	parser := &PostgreSQLURIParser{}
	schemes := parser.GetSupportedSchemes()

	expected := []string{"postgresql", "postgres"}
	if len(schemes) != len(expected) {
		t.Errorf("Expected %d schemes, got %d", len(expected), len(schemes))
	}

	for i, scheme := range schemes {
		if scheme != expected[i] {
			t.Errorf("Expected scheme %s, got %s", expected[i], scheme)
		}
	}
}

func TestPostgreSQLURIParser_GetDriverType(t *testing.T) {
	parser := &PostgreSQLURIParser{}
	driverType := parser.GetDriverType()

	expected := "postgresql"
	if driverType != expected {
		t.Errorf("Expected driver type %s, got %s", expected, driverType)
	}
}
