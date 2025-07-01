package drivers

import (
	"testing"

	"github.com/rediwo/redi-orm/types"
)

func TestMySQLURIParser_ParseURI(t *testing.T) {
	parser := &MySQLURIParser{}

	tests := []struct {
		name        string
		uri         string
		expected    types.Config
		expectError bool
	}{
		{
			name: "MySQL with all parameters",
			uri:  "mysql://user:password@host:3306/database",
			expected: types.Config{
				Type:     "mysql",
				Host:     "host",
				Port:     3306,
				User:     "user",
				Password: "password",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "MySQL with default port",
			uri:  "mysql://user:password@host/database",
			expected: types.Config{
				Type:     "mysql",
				Host:     "host",
				Port:     3306,
				User:     "user",
				Password: "password",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "MySQL with default host",
			uri:  "mysql://user:password@/database",
			expected: types.Config{
				Type:     "mysql",
				Host:     "localhost",
				Port:     3306,
				User:     "user",
				Password: "password",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "MySQL without password",
			uri:  "mysql://user@host/database",
			expected: types.Config{
				Type:     "mysql",
				Host:     "host",
				Port:     3306,
				User:     "user",
				Password: "",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "MySQL without user info",
			uri:  "mysql://host/database",
			expected: types.Config{
				Type:     "mysql",
				Host:     "host",
				Port:     3306,
				User:     "",
				Password: "",
				Database: "database",
			},
			expectError: false,
		},
		{
			name: "MySQL without database",
			uri:  "mysql://user:password@host:3306",
			expected: types.Config{
				Type:     "mysql",
				Host:     "host",
				Port:     3306,
				User:     "user",
				Password: "password",
				Database: "",
			},
			expectError: false,
		},
		{
			name:        "Invalid scheme",
			uri:         "postgresql://user@host/db",
			expectError: true,
		},
		{
			name:        "Invalid port",
			uri:         "mysql://user@host:invalid/db",
			expectError: true,
		},
		{
			name:        "Invalid URI",
			uri:         "mysql://[invalid-uri",
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

func TestMySQLURIParser_GetSupportedSchemes(t *testing.T) {
	parser := &MySQLURIParser{}
	schemes := parser.GetSupportedSchemes()

	expected := []string{"mysql"}
	if len(schemes) != len(expected) {
		t.Errorf("Expected %d schemes, got %d", len(expected), len(schemes))
	}

	for i, scheme := range schemes {
		if scheme != expected[i] {
			t.Errorf("Expected scheme %s, got %s", expected[i], scheme)
		}
	}
}

func TestMySQLURIParser_GetDriverType(t *testing.T) {
	parser := &MySQLURIParser{}
	driverType := parser.GetDriverType()

	expected := "mysql"
	if driverType != expected {
		t.Errorf("Expected driver type %s, got %s", expected, driverType)
	}
}
