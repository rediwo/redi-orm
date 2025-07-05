package postgresql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgreSQLURIParser_ParseURI(t *testing.T) {
	parser := NewPostgreSQLURIParser()

	testCases := []struct {
		name        string
		uri         string
		expectedDSN string
		expectError bool
	}{
		{
			name:        "Full URI with all components",
			uri:         "postgresql://user:password@localhost:5432/mydb",
			expectedDSN: "host=localhost user=user password=password dbname=mydb",
			expectError: false,
		},
		{
			name:        "URI with postgres scheme",
			uri:         "postgres://user:password@localhost:5432/mydb",
			expectedDSN: "host=localhost user=user password=password dbname=mydb",
			expectError: false,
		},
		{
			name:        "URI without port (default to 5432)",
			uri:         "postgresql://user:password@localhost/mydb",
			expectedDSN: "host=localhost user=user password=password dbname=mydb",
			expectError: false,
		},
		{
			name:        "URI without password",
			uri:         "postgresql://user@localhost/mydb",
			expectedDSN: "host=localhost user=user dbname=mydb",
			expectError: false,
		},
		{
			name:        "URI without authentication",
			uri:         "postgresql://localhost/mydb",
			expectedDSN: "host=localhost dbname=mydb",
			expectError: false,
		},
		{
			name:        "URI with custom port",
			uri:         "postgresql://user:pass@db.example.com:5433/testdb",
			expectedDSN: "host=db.example.com port=5433 user=user password=pass dbname=testdb",
			expectError: false,
		},
		{
			name:        "URI with query parameters",
			uri:         "postgresql://user:pass@localhost/mydb?sslmode=require&connect_timeout=10",
			expectedDSN: "", // Will check components individually due to map iteration order
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
			dsn, err := parser.ParseURI(tc.uri)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tc.expectedDSN == "" {
					// Special case for query parameters - check individual components
					if tc.name == "URI with query parameters" {
						assert.Contains(t, dsn, "host=localhost")
						assert.Contains(t, dsn, "user=user")
						assert.Contains(t, dsn, "password=pass")
						assert.Contains(t, dsn, "dbname=mydb")
						assert.Contains(t, dsn, "sslmode=require")
						assert.Contains(t, dsn, "connect_timeout=10")
					}
				} else {
					assert.Equal(t, tc.expectedDSN, dsn)
				}
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
		name        string
		uri         string
		expectedDSN string
		expectError bool
	}{
		{
			name:        "with SSL mode",
			uri:         "postgresql://user:pass@localhost:5432/testdb?sslmode=require",
			expectedDSN: "host=localhost user=user password=pass dbname=testdb sslmode=require",
			expectError: false,
		},
		{
			name:        "with multiple options",
			uri:         "postgresql://user:pass@localhost:5432/testdb?sslmode=require&application_name=myapp&connect_timeout=10",
			expectedDSN: "host=localhost user=user password=pass dbname=testdb sslmode=require application_name=myapp connect_timeout=10",
			expectError: false,
		},
		{
			name:        "with timezone",
			uri:         "postgresql://user:pass@localhost:5432/testdb?timezone=UTC",
			expectedDSN: "host=localhost user=user password=pass dbname=testdb timezone=UTC",
			expectError: false,
		},
		{
			name:        "with search_path",
			uri:         "postgresql://user:pass@localhost:5432/testdb?search_path=myschema",
			expectedDSN: "host=localhost user=user password=pass dbname=testdb search_path=myschema",
			expectError: false,
		},
		{
			name:        "with client_encoding",
			uri:         "postgresql://user:pass@localhost:5432/testdb?client_encoding=UTF8",
			expectedDSN: "host=localhost user=user password=pass dbname=testdb client_encoding=UTF8",
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
			// Check that the DSN contains all expected components
			assert.Contains(t, dsn, "host=localhost")
			assert.Contains(t, dsn, "user=user")
			assert.Contains(t, dsn, "password=pass")
			assert.Contains(t, dsn, "dbname=testdb")
		})
	}
}
