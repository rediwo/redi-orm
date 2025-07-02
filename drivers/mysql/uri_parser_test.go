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

func TestMySQLURIParser_ParseURI_WithOptions(t *testing.T) {
	parser := NewMySQLURIParser()

	tests := []struct {
		name           string
		uri            string
		expectedConfig map[string]string
		expectError    bool
	}{
		{
			name: "with charset option",
			uri:  "mysql://user:pass@localhost:3306/testdb?charset=utf8mb4",
			expectedConfig: map[string]string{
				"charset":   "utf8mb4",
				"parseTime": "true", // Default
			},
			expectError: false,
		},
		{
			name: "with parseTime disabled",
			uri:  "mysql://user:pass@localhost:3306/testdb?parseTime=false",
			expectedConfig: map[string]string{
				"parseTime": "false",
				"charset":   "utf8mb4", // Default
			},
			expectError: false,
		},
		{
			name: "with multiple options",
			uri:  "mysql://user:pass@localhost:3306/testdb?charset=utf8&parseTime=true&timeout=10s&readTimeout=30s",
			expectedConfig: map[string]string{
				"charset":     "utf8",
				"parseTime":   "true",
				"timeout":     "10s",
				"readTimeout": "30s",
			},
			expectError: false,
		},
		{
			name: "with TLS option",
			uri:  "mysql://user:pass@localhost:3306/testdb?tls=true",
			expectedConfig: map[string]string{
				"tls":       "true",
				"charset":   "utf8mb4", // Default
				"parseTime": "true",    // Default
			},
			expectError: false,
		},
		{
			name: "with interpolateParams and multiStatements",
			uri:  "mysql://user:pass@localhost:3306/testdb?interpolateParams=true&multiStatements=true",
			expectedConfig: map[string]string{
				"interpolateParams": "true",
				"multiStatements":   "true",
				"charset":           "utf8mb4", // Default
				"parseTime":         "true",    // Default
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parser.ParseURI(tt.uri)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "mysql", config.Type)
			assert.Equal(t, "localhost", config.Host)
			assert.Equal(t, 3306, config.Port)
			assert.Equal(t, "user", config.User)
			assert.Equal(t, "pass", config.Password)
			assert.Equal(t, "testdb", config.Database)

			// Check options
			require.NotNil(t, config.Options)
			for key, expectedValue := range tt.expectedConfig {
				assert.Equal(t, expectedValue, config.Options[key], "Option %s should match", key)
			}
		})
	}
}
