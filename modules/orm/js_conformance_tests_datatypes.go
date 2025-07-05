package orm

import (
	"testing"
)

// Data Type Tests
func (jct *JSConformanceTests) runDataTypeTests(t *testing.T, runner *JSTestRunner) {
	// Test various data types
	jct.runWithCleanup(t, runner, "DataTypes", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model TestModel {
	id       Int       @id @default(autoincrement())
	text     String
	number   Int
	decimal  Float
	bool     Boolean
	date     DateTime  @default(now())
	optional String?
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create with various types
		const created = await db.models.TestModel.create({
			data: {
				text: 'Hello',
				number: 42,
				decimal: 3.14,
				bool: true,
				optional: null
			}
		});
		
		assert.strictEqual(created.text, 'Hello');
		assert.strictEqual(created.number, 42);
		assert.strictEqual(created.decimal, 3.14);
		// SQLite returns 1/0 for booleans, not true/false
		assert(created.bool === true || created.bool === 1, 'Expected bool to be true or 1, got: ' + created.bool);
		assert.strictEqual(created.optional, null);
		// Check date - SQLite may return as string or time.Time object
		if (created.date) {
			if (typeof created.date === 'string') {
				// Try to parse it as a date
				const parsed = new Date(created.date);
				assert(!isNaN(parsed.getTime()), 'Expected valid date string, got: ' + created.date);
			} else if (created.date instanceof Date) {
				// It's already a Date object, that's good
				assert(true);
			} else if (typeof created.date === 'object' && created.date.toString) {
				// It might be a Go time.Time object converted to JS
				const dateStr = created.date.toString();
				const parsed = new Date(dateStr);
				assert(!isNaN(parsed.getTime()), 'Expected valid date from object, got: ' + dateStr);
			} else {
				throw new Error('Expected created.date to be a Date instance or valid date string, got: ' + typeof created.date + ' value: ' + JSON.stringify(created.date));
			}
		} else {
			throw new Error('Expected created.date to be defined');
		}
		
		// await db.close();
	`)
}
