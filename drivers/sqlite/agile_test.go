package sqlite

import (
	"testing"

	"github.com/rediwo/redi-orm/agile"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestSQLiteAgileConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping agile conformance tests in short mode")
	}

	// Get test database URI and parse it
	uri := test.GetTestDatabaseUri("sqlite")

	suite := &agile.AgileConformanceTests{
		DriverName:  "SQLite",
		DatabaseURI: uri,
		NewDatabase: func(uri string) (types.Database, error) {
			config, err := NewSQLiteURIParser().ParseURI(uri)
			if err != nil {
				return nil, err
			}
			return NewSQLiteDB(config)
		},
		SkipTests: map[string]bool{
			// SQLite-specific skips
			"TestTransactionConcurrentAccess": true, // SQLite uses database-level locking
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			// Use the same cleanup logic as conformance tests
			if sqliteDB, ok := db.(*SQLiteDB); ok {
				cleanupTables(t, sqliteDB)
			}
		},
		Characteristics: agile.AgileDriverCharacteristics{
			SupportsReturning:          false,
			MaxConnectionPoolSize:      1,
			SupportsNestedTransactions: false,
			ReturnsStringForNumbers:    false,
		},
	}

	suite.RunAll(t)
}