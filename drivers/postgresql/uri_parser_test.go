package postgresql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgreSQLURIParser_ParseURI(t *testing.T) {
	parser := NewPostgreSQLURIParser()

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
			uri:  "postgresql://user:password@localhost:5432/mydb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     5432,
				user:     "user",
				password: "password",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name: "URI with postgres scheme",
			uri:  "postgres://user:password@localhost:5432/mydb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     5432,
				user:     "user",
				password: "password",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name: "URI without port (default to 5432)",
			uri:  "postgresql://user:password@localhost/mydb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     5432,
				user:     "user",
				password: "password",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name: "URI without password",
			uri:  "postgresql://user@localhost/mydb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     5432,
				user:     "user",
				password: "",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name: "URI without authentication",
			uri:  "postgresql://localhost/mydb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     5432,
				user:     "",
				password: "",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name: "URI with custom port",
			uri:  "postgresql://user:pass@db.example.com:5433/testdb",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "db.example.com",
				port:     5433,
				user:     "user",
				password: "pass",
				database: "testdb",
			},
			expectError: false,
		},
		{
			name: "URI with query parameters",
			uri:  "postgresql://user:pass@localhost/mydb?sslmode=require&connect_timeout=10",
			expected: struct {
				host     string
				port     int
				user     string
				password string
				database string
			}{
				host:     "localhost",
				port:     5432,
				user:     "user",
				password: "pass",
				database: "mydb",
			},
			expectError: false,
		},
		{
			name:        "Invalid scheme",
			uri:         "mysql://localhost/mydb",
			expectError: true,
		},
		{
			name:        "Missing host",
			uri:         "postgresql:///mydb",
			expectError: true,
		},
		{
			name:        "Missing database",
			uri:         "postgresql://localhost",
			expectError: true,
		},
		{
			name:        "Invalid port",
			uri:         "postgresql://localhost:abc/mydb",
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
				assert.Equal(t, "postgresql", config.Type)
				assert.Equal(t, tc.expected.host, config.Host)
				assert.Equal(t, tc.expected.port, config.Port)
				assert.Equal(t, tc.expected.user, config.User)
				assert.Equal(t, tc.expected.password, config.Password)
				assert.Equal(t, tc.expected.database, config.Database)
			}
		})
	}
}

func TestPostgreSQLURIParser_GetSupportedSchemes(t *testing.T) {
	parser := NewPostgreSQLURIParser()
	schemes := parser.GetSupportedSchemes()

	assert.Contains(t, schemes, "postgresql")
	assert.Contains(t, schemes, "postgres")
}

func TestPostgreSQLURIParser_GetDriverType(t *testing.T) {
	parser := NewPostgreSQLURIParser()
	driverType := parser.GetDriverType()

	assert.Equal(t, "postgresql", driverType)
}

func TestPostgreSQLURIParser_ParseURI_WithOptions(t *testing.T) {
	parser := NewPostgreSQLURIParser()

	tests := []struct {
		name           string
		uri            string
		expectedConfig map[string]string
		expectError    bool
	}{
		{
			name: "with SSL mode",
			uri:  "postgresql://user:pass@localhost:5432/testdb?sslmode=require",
			expectedConfig: map[string]string{
				"sslmode": "require",
			},
			expectError: false,
		},
		{
			name: "with multiple options",
			uri:  "postgresql://user:pass@localhost:5432/testdb?sslmode=require&application_name=myapp&connect_timeout=10",
			expectedConfig: map[string]string{
				"sslmode":          "require",
				"application_name": "myapp",
				"connect_timeout":  "10",
			},
			expectError: false,
		},
		{
			name: "with timezone",
			uri:  "postgresql://user:pass@localhost:5432/testdb?timezone=UTC",
			expectedConfig: map[string]string{
				"timezone": "UTC",
			},
			expectError: false,
		},
		{
			name: "with search_path",
			uri:  "postgresql://user:pass@localhost:5432/testdb?search_path=myschema",
			expectedConfig: map[string]string{
				"search_path": "myschema",
			},
			expectError: false,
		},
		{
			name: "with client_encoding",
			uri:  "postgresql://user:pass@localhost:5432/testdb?client_encoding=UTF8",
			expectedConfig: map[string]string{
				"client_encoding": "UTF8",
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
			assert.Equal(t, "postgresql", config.Type)
			assert.Equal(t, "localhost", config.Host)
			assert.Equal(t, 5432, config.Port)
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
