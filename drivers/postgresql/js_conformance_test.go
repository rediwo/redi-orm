package postgresql

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/modules/orm"
	"github.com/rediwo/redi-orm/test"
)

func TestPostgreSQLJSConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping JS conformance tests in short mode")
	}

	// Get test database URI
	uri := test.GetTestDatabaseUri("postgresql")

	suite := &orm.JSConformanceTests{
		DriverName:  "PostgreSQL",
		DatabaseURI: uri,
		SkipTests: map[string]bool{
			// PostgreSQL aborts transaction on error
			"TestTransactionErrorHandling": true,
		},
		Characteristics: orm.JSDriverCharacteristics{
			SupportsArrayTypes:         true,
			SupportsJSONTypes:          true, // JSONB support
			SupportsEnumTypes:          true,
			MaxConnectionPoolSize:      20,
			SupportsNestedTransactions: true,
		},
		CleanupTables: func(t *testing.T, runner *orm.JSTestRunner) {
			// Use the shared cleanupTables function by creating a database connection
			db, err := database.NewFromURI(uri)
			if err != nil {
				t.Logf("Failed to create database for cleanup: %v", err)
				return
			}
			defer db.Close()

			// Connect to the database
			ctx := context.Background()
			if err := db.Connect(ctx); err != nil {
				t.Logf("Failed to connect to database for cleanup: %v", err)
				return
			}

			if pgDB, ok := db.(*PostgreSQLDB); ok {
				cleanupTables(t, pgDB)
			}
		},
	}

	suite.RunAll(t)
}
