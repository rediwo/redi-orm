package orm

import (
	"testing"
)

// Schema Management Tests
func (jct *JSConformanceTests) runSchemaTests(t *testing.T, runner *JSTestRunner) {
	// Test schema loading from string
	jct.runWithCleanup(t, runner, "LoadSchemaFromString", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		const schema = ` + "`" + `
model User {
	id    Int    @id @default(autoincrement())
	name  String
	email String @unique
}
` + "`" + `;
		
		await db.loadSchema(schema);
		await db.syncSchemas();
		
		// Verify model is accessible
		if (!db.models || !db.models.User) {
			throw new Error('db.models.User should be defined after schema sync');
		}
		
		// await db.close();
	`)

	// Test invalid schema
	jct.runWithCleanup(t, runner, "InvalidSchema", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		const invalidSchema = 'model User { id @id }';
		
		try {
			await db.loadSchema(invalidSchema);
			throw new Error('Should have failed to load invalid schema');
		} catch (err) {
			if (!err.message.includes('type') && !err.message.includes('invalid') && !err.message.includes('parse') && !err.message.includes('expected')) {
				throw new Error('Expected error to indicate parse/type error, got: ' + err.message);
			}
		}
		
		// await db.close();
	`)
}