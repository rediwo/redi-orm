package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMySQLURIParser_ParseURI(t *testing.T) {
	parser := NewMySQLURIParser()

	testCases := []struct {
		name     string
		uri      string
		expected struct {
			host     string
			port     int
			user     string
			password string
			database string
		}
		expectError bool
	}{
		{
			name: "Full URI with all components",
			uri:  "mysql://user:password@localhost:3306/mydb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     3306,
				user:     "user",
				password: "password",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name: "URI without port (default to 3306)",
			uri:  "mysql://user:password@localhost/mydb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     3306,
				user:     "user",
				password: "password",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name: "URI without password",
			uri:  "mysql://user@localhost/mydb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     3306,
				user:     "user",
				password: "",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name: "URI without authentication",
			uri:  "mysql://localhost/mydb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     3306,
				user:     "",
				password: "",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name: "URI with custom port",
			uri:  "mysql://user:pass@db.example.com:3307/testdb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "db.example.com",
				port:     3307,
				user:     "user",
				password: "pass",
				database: "testdb",
			},
			expectError: false,
		},
		{
			name: "URI with query parameters",
			uri:  "mysql://user:pass@localhost/mydb?charset=utf8mb4&parseTime=true",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     3306,
				user:     "user",
				password: "pass",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name:        "Invalid scheme",
			uri:         "postgresql://localhost/mydb",
			expectError: true,
		},
		{
			name:        "Missing host",
			uri:         "mysql:///mydb",
			expectError: true,
		},
		{
			name:        "Missing database",
			uri:         "mysql://localhost",
			expectError: true,
		},
		{
			name:        "Invalid port",
			uri:         "mysql://localhost:abc/mydb",
			expectError: true,
		},
		{
			name:        "Invalid URI format",
			uri:         "not-a-uri",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := parser.ParseURI(tc.uri)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "mysql", config.Type)
				assert.Equal(t, tc.expected.host, config.Host)
				assert.Equal(t, tc.expected.port, config.Port)
				assert.Equal(t, tc.expected.user, config.User)
				assert.Equal(t, tc.expected.password, config.Password)
				assert.Equal(t, tc.expected.database, config.Database)
			}
		})
	}
}

func TestMySQLURIParser_GetSupportedSchemes(t *testing.T) {
	parser := NewMySQLURIParser()
	schemes := parser.GetSupportedSchemes()

	assert.Contains(t, schemes, "mysql")
	assert.Contains(t, schemes, "mysql2")
}

func TestMySQLURIParser_GetDriverType(t *testing.T) {
	parser := NewMySQLURIParser()
	driverType := parser.GetDriverType()

	assert.Equal(t, "mysql", driverType)
}
