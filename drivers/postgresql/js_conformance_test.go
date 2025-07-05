package postgresql

import (
	"testing"

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
		CleanupTables: cleanupTablesJS,
	}

	suite.RunAll(t)
}

// cleanupTablesJS removes all non-system tables via JavaScript
func cleanupTablesJS(t *testing.T, runner *orm.JSTestRunner) {
	cleanupScript := `
		const db = fromUri(process.env.TEST_DATABASE_URI);
		await db.connect();
		
		// Get all tables in public schema
		const tables = await db.queryRaw(
			"SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename"
		);
		
		// Drop each table with CASCADE to handle foreign key constraints
		for (const table of tables) {
			try {
				await db.executeRaw('DROP TABLE IF EXISTS ' + table.tablename + ' CASCADE');
				console.log('Dropped table:', table.tablename);
			} catch (err) {
				console.error('Failed to drop table', table.tablename, ':', err.message);
			}
		}
	`
	
	err := runner.RunCleanupScript(cleanupScript)
	if err != nil {
		t.Logf("Failed to cleanup tables: %v", err)
	}
}