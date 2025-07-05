package sqlite

import (
	"testing"

	"github.com/rediwo/redi-orm/modules/orm"
	"github.com/rediwo/redi-orm/test"
)

func TestSQLiteJSConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping JS conformance tests in short mode")
	}

	// Get test database URI
	uri := test.GetTestDatabaseUri("sqlite")

	suite := &orm.JSConformanceTests{
		DriverName:  "SQLite",
		DatabaseURI: uri,
		SkipTests: map[string]bool{
			// SQLite doesn't support concurrent write transactions
			"TestTransactionIsolation":        true,
			"TestTransactionConcurrentAccess": true,
		},
		Characteristics: orm.JSDriverCharacteristics{
			SupportsArrayTypes:         false,
			SupportsJSONTypes:          false,
			SupportsEnumTypes:          false,
			MaxConnectionPoolSize:      1, // SQLite is single-threaded
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
		
		// Get all tables
		const tables = await db.queryRaw(
			"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
		);
		
		// Disable foreign key constraints temporarily
		await db.executeRaw('PRAGMA foreign_keys = OFF');
		
		// Drop each table
		for (const table of tables) {
			try {
				await db.executeRaw('DROP TABLE IF EXISTS ' + table.name);
				console.log('Dropped table:', table.name);
			} catch (err) {
				console.error('Failed to drop table', table.name, ':', err.message);
			}
		}
		
		// Re-enable foreign key constraints
		await db.executeRaw('PRAGMA foreign_keys = ON');
	`

	err := runner.RunCleanupScript(cleanupScript)
	if err != nil {
		t.Logf("Failed to cleanup tables: %v", err)
	}
}
