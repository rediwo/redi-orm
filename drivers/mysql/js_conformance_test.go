package mysql

import (
	"testing"

	"github.com/rediwo/redi-orm/modules/orm"
	"github.com/rediwo/redi-orm/test"
)

func TestMySQLJSConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping JS conformance tests in short mode")
	}

	// Get test database URI
	uri := test.GetTestDatabaseUri("mysql")

	suite := &orm.JSConformanceTests{
		DriverName:  "MySQL",
		DatabaseURI: uri,
		SkipTests:   map[string]bool{
			// MySQL-specific skips if needed
		},
		Characteristics: orm.JSDriverCharacteristics{
			SupportsArrayTypes:         false,
			SupportsJSONTypes:          true, // MySQL 5.7+ supports JSON
			SupportsEnumTypes:          true,
			MaxConnectionPoolSize:      10,
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
		
		// Get database name from URI
		const dbName = process.env.TEST_DATABASE_URI.split('/').pop().split('?')[0];
		
		// Get all tables
		const tables = await db.queryRaw(
			"SELECT table_name FROM information_schema.tables WHERE table_schema = ? AND table_type = 'BASE TABLE'",
			dbName
		);
		
		// Disable foreign key checks temporarily
		await db.executeRaw('SET FOREIGN_KEY_CHECKS = 0');
		
		// Drop each table
		for (const table of tables) {
			const tableName = table.table_name || table.TABLE_NAME;
			if (tableName) {
				try {
					await db.executeRaw('DROP TABLE IF EXISTS ' + tableName);
					console.log('Dropped table:', tableName);
				} catch (err) {
					console.error('Failed to drop table', tableName, ':', err.message);
				}
			}
		}
		
		// Re-enable foreign key checks
		await db.executeRaw('SET FOREIGN_KEY_CHECKS = 1');
	`

	err := runner.RunCleanupScript(cleanupScript)
	if err != nil {
		t.Logf("Failed to cleanup tables: %v", err)
	}
}
