package sqlite

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestSQLiteConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping conformance tests in short mode")
	}

	// Get test database URI
	uri := test.GetTestDatabaseUri("sqlite")

	suite := &test.DriverConformanceTests{
		DriverName: "SQLite",
		NewDriver: func(uri string) (types.Database, error) {
			return database.NewFromURI(uri)
		},
		URI: uri,
		SkipTests: map[string]bool{
			// SQLite-specific skips
			"TestTransactionIsolation":        true, // SQLite doesn't support concurrent write transactions
			"TestTransactionConcurrentAccess": true, // SQLite uses database-level locking preventing concurrent writes
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			// SQLite with file-based database might benefit from cleanup
			sqliteDB, ok := db.(*SQLiteDB)
			if ok {
				cleanupTables(t, sqliteDB)
			}
		},
		Characteristics: test.DriverCharacteristics{
			ReturnsZeroRowsAffectedForUnchanged: false,
			SupportsLastInsertID:                true,
			SupportsReturningClause:             false,
			MigrationTableName:                  "_migrations",
			SystemIndexPatterns:                 []string{"sqlite_*", "pk_*"},
			AutoIncrementIntegerType:            "INTEGER",
		},
	}

	suite.RunAll(t)
}
