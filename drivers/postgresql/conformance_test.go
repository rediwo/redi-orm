package postgresql

import (
	"context"
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestPostgreSQLConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conformance tests in short mode")
	}

	// Skip if PostgreSQL is not available
	uri := test.GetTestDatabaseUri("postgresql")
	db, err := database.NewFromURI(uri)
	if err != nil {
		t.Skip("PostgreSQL not available for testing")
	}
	if err := db.Connect(context.Background()); err != nil {
		t.Skipf("Cannot connect to PostgreSQL: %v", err)
	}
	db.Close()

	suite := &test.DriverConformanceTests{
		DriverName: "PostgreSQL",
		NewDriver: func(uri string) (types.Database, error) {
			return database.NewFromURI(uri)
		},
		URI: uri,
		SkipTests: map[string]bool{
			// PostgreSQL-specific skips
			"TestTransactionErrorHandling": true, // PostgreSQL aborts transaction on error
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			pgDB, ok := db.(*PostgreSQLDB)
			if ok {
				cleanupTables(t, pgDB)
			}
		},
		Characteristics: test.DriverCharacteristics{
			ReturnsZeroRowsAffectedForUnchanged: false,
			SupportsLastInsertID:                false,
			SupportsReturningClause:             true,
			MigrationTableName:                  "_migrations",
			SystemIndexPatterns:                 []string{"_pkey", "_key", "_fkey", "pg_*"},
			AutoIncrementIntegerType:            "SERIAL",
		},
	}

	suite.RunAll(t)
}

