package orm

import (
	"testing"
)

// Raw Query Tests
func (jct *JSConformanceTests) runRawQueryTests(t *testing.T, runner *JSTestRunner) {
	// Test raw query
	jct.runWithCleanup(t, runner, "RawQuery", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model User {
	id   Int    @id @default(autoincrement())
	name String
	age  Int
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		await db.models.User.create({ data: { name: 'Alice', age: 25 } });
		await db.models.User.create({ data: { name: 'Bob', age: 30 } });
		await db.models.User.create({ data: { name: 'Charlie', age: 35 } });
		
		// Test raw query with parameters
		const results = await db.queryRaw('SELECT * FROM users WHERE age > ?', 28);
		assert(results.length === 2);
		
		// Results should contain Bob and Charlie
		const names = results.map(r => r.name).sort();
		assert.lengthOf(names, 2);
		assert.strictEqual(names[0], 'Bob');
		assert.strictEqual(names[1], 'Charlie');
		
		// await db.close();
	`)

	// Test raw execute
	jct.runWithCleanup(t, runner, "RawExecute", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model User {
	id   Int    @id @default(autoincrement())
	name String
	age  Int
}
` + "`" + `);
		await db.syncSchemas();
		
		// Test raw execute
		const result = await db.executeRaw('INSERT INTO users (name, age) VALUES (?, ?)', 'Charlie', 35);
		assert(result.rowsAffected > 0);
		
		// Verify insert
		const users = await db.models.User.findMany({ where: { name: 'Charlie' } });
		assert(users.length === 1);
		assert.strictEqual(users[0].age, 35);
		
		// await db.close();
	`)

	// Test raw query with complex types
	jct.runWithCleanup(t, runner, "RawQueryComplexTypes", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model TestData {
	id        Int      @id @default(autoincrement())
	name      String
	price     Float
	active    Boolean
	createdAt DateTime @default(now())
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		await db.models.TestData.create({
			data: { name: 'Test Item', price: 99.99, active: true }
		});
		
		// Test raw query returning various types
		const results = await db.queryRaw('SELECT * FROM test_datas WHERE active = ?', true);
		assert(results.length === 1);
		
		const item = results[0];
		assert.strictEqual(item.name, 'Test Item');
		assert.strictEqual(item.price, 99.99);
		// SQLite returns 1/0 for booleans
		assert(item.active === true || item.active === 1);
		// Check date field exists and has a value
		assert(item.created_at || item.createdAt); // May be snake_case in raw query
		
		// await db.close();
	`)
}