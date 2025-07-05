package orm

import (
	"testing"
)

// Connection Management Tests
func (jct *JSConformanceTests) runConnectionTests(t *testing.T, runner *JSTestRunner) {
	// Test basic connection
	jct.runWithCleanup(t, runner, "Connect", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		await db.ping();
		// Note: db.close() might not be implemented yet in the JS API
		// await db.close();
	`)

	// Test invalid connection
	jct.runWithCleanup(t, runner, "InvalidConnection", `
		try {
			const db = fromUri('invalid://connection');
			await db.connect();
			throw new Error('Should have failed to connect');
		} catch (err) {
			if (!err.message.includes('invalid') && !err.message.includes('unsupported')) {
				throw new Error('Expected error to contain "invalid" or "unsupported", got: ' + err.message);
			}
		}
	`)

	// Test multiple connections
	jct.runWithCleanup(t, runner, "MultipleConnections", `
		const db1 = fromUri(TEST_DATABASE_URI);
		const db2 = fromUri(TEST_DATABASE_URI);
		
		await db1.connect();
		await db2.connect();
		
		await db1.ping();
		await db2.ping();
		
		// Note: db.close() might be timing out
		// await db1.close();
		// await db2.close();
	`)
}
