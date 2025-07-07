package tests

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/graphql"
	"github.com/rediwo/redi-orm/prisma"
	"github.com/stretchr/testify/require"

	// Import all drivers
	_ "github.com/rediwo/redi-orm/drivers/mongodb"
	_ "github.com/rediwo/redi-orm/drivers/mysql"
	_ "github.com/rediwo/redi-orm/drivers/postgresql"
	_ "github.com/rediwo/redi-orm/drivers/sqlite"
)

// TestGraphQLMultiDatabase tests GraphQL with different database drivers
func TestGraphQLMultiDatabase(t *testing.T) {
	// Test schema
	schemaContent := `
		model User {
			id    Int     @id @default(autoincrement())
			name  String
			email String  @unique
		}
	`

	testCases := []struct {
		name       string
		uri        string
		skip       bool
		skipReason string
	}{
		{
			name: "SQLite",
			uri:  "sqlite://:memory:",
			skip: false,
		},
		{
			name:       "MySQL",
			uri:        "mysql://testuser:testpass@localhost:3306/testdb",
			skip:       true,
			skipReason: "Requires MySQL server",
		},
		{
			name:       "PostgreSQL",
			uri:        "postgresql://testuser:testpass@localhost:5432/testdb",
			skip:       true,
			skipReason: "Requires PostgreSQL server",
		},
		{
			name:       "MongoDB",
			uri:        "mongodb://localhost:27017/testdb",
			skip:       true,
			skipReason: "Requires MongoDB server",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skip {
				t.Skipf("Skipping %s test: %s", tc.name, tc.skipReason)
			}

			// Create database connection
			db, err := database.NewFromURI(tc.uri)
			require.NoError(t, err)

			ctx := context.Background()
			err = db.Connect(ctx)
			require.NoError(t, err)
			defer db.Close()

			// Parse and register schema
			schemas, err := prisma.ParseSchema(schemaContent)
			require.NoError(t, err)

			for modelName, schema := range schemas {
				err = db.RegisterSchema(modelName, schema)
				require.NoError(t, err)
			}

			// Sync schemas
			err = db.SyncSchemas(ctx)
			require.NoError(t, err)

			// Generate GraphQL schema
			generator := graphql.NewSchemaGenerator(db, schemas)
			graphqlSchema, err := generator.Generate()
			require.NoError(t, err)

			// Create handler
			handler := graphql.NewHandler(graphqlSchema)
			require.NotNil(t, handler)

			// Test that we can create the schema and handler successfully
			t.Logf("Successfully created GraphQL handler for %s", tc.name)
		})
	}
}

// TestGraphQLWithDocker tests GraphQL with real databases using Docker
func TestGraphQLWithDocker(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker tests in short mode")
	}

	// This test requires running: make docker-up
	t.Skip("Enable this test when Docker databases are running")
}
