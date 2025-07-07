package postgresql

import (
	"testing"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/orm"
	"github.com/rediwo/redi-orm/test"
	"github.com/rediwo/redi-orm/types"
)

func TestPostgreSQLOrmConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping orm conformance tests in short mode")
	}

	// Get test database URI and parse it
	uri := test.GetTestDatabaseUri("postgresql")

	suite := &orm.OrmConformanceTests{
		DriverName:  "PostgreSQL",
		DatabaseURI: uri,
		NewDatabase: func(uri string) (types.Database, error) {
			return database.NewFromURI(uri)
		},
		SkipTests: map[string]bool{
			// PostgreSQL-specific skips (if any)
		},
		CleanupTables: func(t *testing.T, db types.Database) {
			// PostgreSQL-specific cleanup
			if pgDB, ok := db.(*PostgreSQLDB); ok {
				cleanupTables(t, pgDB)
			}
		},
		Characteristics: orm.OrmDriverCharacteristics{
			SupportsReturning:          true,
			MaxConnectionPoolSize:      10,
			SupportsNestedTransactions: true,
			ReturnsStringForNumbers:    false,
		},
	}

	suite.RunAll(t)
}
