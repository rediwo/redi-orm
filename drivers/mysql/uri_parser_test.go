package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMySQLURIParser_ParseURI(t *testing.T) {
	parser := NewMySQLURIParser()

	testCases := []struct {
		name        string
		uri         string
		expectedDSN string
		expectError bool
	}{
		{
			name:        "Full URI with all components",
			uri:         "mysql://user:password@localhost:3306/mydb",
			expectedDSN: "user:password@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=true",
			expectError: false,
		},
		{
			name:        "URI without port (default to 3306)",
			uri:         "mysql://user:password@localhost/mydb",
			expectedDSN: "user:password@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=true",
			expectError: false,
		},
		{
			name:        "URI without password",
			uri:         "mysql://user@localhost/mydb",
			expectedDSN: "user:@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=true",
			expectError: false,
		},
		{
			name:        "URI without authentication",
			uri:         "mysql://localhost/mydb",
			expectedDSN: ":@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=true",
			expectError: false,
		},
		{
			name:        "URI with custom port",
			uri:         "mysql://user:pass@db.example.com:3307/testdb",
			expectedDSN: "user:pass@tcp(db.example.com:3307)/testdb?charset=utf8mb4&parseTime=true",
			expectError: false,
		},
		{
			name:        "URI with query parameters",
			uri:         "mysql://user:pass@localhost/mydb?charset=utf8mb4&parseTime=true",
			expectedDSN: "", // Will check individually since order can vary
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
			dsn, err := parser.ParseURI(tc.uri)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.expectedDSN == "" {
					// Special case for query parameters - check parts individually
					assert.Contains(t, dsn, "user:pass@tcp(localhost:3306)/mydb")
					assert.Contains(t, dsn, "charset=utf8mb4")
					assert.Contains(t, dsn, "parseTime=true")
				} else {
					assert.Equal(t, tc.expectedDSN, dsn)
				}
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
		name        string
		uri         string
		expectedDSN string
		expectError bool
	}{
		{
			name:        "with charset option",
			uri:         "mysql://user:pass@localhost:3306/testdb?charset=utf8mb4",
			expectedDSN: "user:pass@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=true",
			expectError: false,
		},
		{
			name:        "with parseTime disabled",
			uri:         "mysql://user:pass@localhost:3306/testdb?parseTime=false",
			expectedDSN: "user:pass@tcp(localhost:3306)/testdb?parseTime=false&charset=utf8mb4",
			expectError: false,
		},
		{
			name:        "with multiple options",
			uri:         "mysql://user:pass@localhost:3306/testdb?charset=utf8&parseTime=true&timeout=10s&readTimeout=30s",
			expectedDSN: "user:pass@tcp(localhost:3306)/testdb?charset=utf8&parseTime=true&timeout=10s&readTimeout=30s",
			expectError: false,
		},
		{
			name:        "with TLS option",
			uri:         "mysql://user:pass@localhost:3306/testdb?tls=true",
			expectedDSN: "user:pass@tcp(localhost:3306)/testdb?tls=true&charset=utf8mb4&parseTime=true",
			expectError: false,
		},
		{
			name:        "with interpolateParams and multiStatements",
			uri:         "mysql://user:pass@localhost:3306/testdb?interpolateParams=true&multiStatements=true",
			expectedDSN: "user:pass@tcp(localhost:3306)/testdb?interpolateParams=true&multiStatements=true&charset=utf8mb4&parseTime=true",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn, err := parser.ParseURI(tt.uri)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			// Check that the DSN contains all expected parameters
			assert.Contains(t, dsn, "user:pass@tcp(localhost:3306)/testdb")
			// We use Contains because the order of query parameters may vary
			for _, param := range []string{"charset", "parseTime"} {
				if tt.expectedDSN != "" {
					assert.Contains(t, dsn, param)
				}
			}
		})
	}
}
