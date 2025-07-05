package orm

import (
	"testing"
)

// CRUD Tests
func (jct *JSConformanceTests) runCRUDTests(t *testing.T, runner *JSTestRunner) {
	// Test create
	jct.runWithCleanup(t, runner, "Create", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model User {
	id    Int    @id @default(autoincrement())
	name  String
	email String @unique
}
` + "`" + `);
		await db.syncSchemas();
		
		const user = await db.models.User.create({
			data: {
				name: 'John Doe',
				email: 'john@example.com'
			}
		});
		
		assert(user.id > 0);
		assert.strictEqual(user.name, 'John Doe');
		assert.strictEqual(user.email, 'john@example.com');
		
		// await db.close();
	`)

	// Test findMany
	jct.runWithCleanup(t, runner, "FindMany", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model User {
	id    Int    @id @default(autoincrement())
	name  String
	email String @unique
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		await db.models.User.create({ data: { name: 'User 1', email: 'user1@example.com' } });
		await db.models.User.create({ data: { name: 'User 2', email: 'user2@example.com' } });
		
		const users = await db.models.User.findMany();
		assert(users.length >= 2);
		
		// await db.close();
	`)

	// Test update
	jct.runWithCleanup(t, runner, "Update", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model User {
	id    Int    @id @default(autoincrement())
	name  String
	email String @unique
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create user
		const user = await db.models.User.create({
			data: { name: 'Original', email: 'original@example.com' }
		});
		
		// Update user
		const updated = await db.models.User.update({
			where: { id: user.id },
			data: { name: 'Updated' }
		});
		
		assert.strictEqual(updated.id, user.id);
		assert.strictEqual(updated.name, 'Updated');
		assert.strictEqual(updated.email, 'original@example.com');
		
		// await db.close();
	`)

	// Test delete
	jct.runWithCleanup(t, runner, "Delete", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model User {
	id    Int    @id @default(autoincrement())
	name  String
	email String @unique
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create user
		const user = await db.models.User.create({
			data: { name: 'To Delete', email: 'delete@example.com' }
		});
		
		// Delete user
		const deleted = await db.models.User.delete({
			where: { id: user.id }
		});
		
		assert.strictEqual(deleted.id, user.id);
		
		// Verify deletion
		try {
			await db.models.User.findUnique({ where: { id: user.id } });
			throw new Error('Should have failed to find deleted user');
		} catch (err) {
			assert(err.message.includes('no rows found') || err.message.includes('not found'));
		}
		
		// await db.close();
	`)

	// Test findUnique
	jct.runWithCleanup(t, runner, "FindUnique", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model User {
	id    Int    @id @default(autoincrement())
	name  String
	email String @unique
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create user
		const created = await db.models.User.create({
			data: { name: 'Unique User', email: 'unique@example.com' }
		});
		
		// Find by id
		const byId = await db.models.User.findUnique({
			where: { id: created.id }
		});
		assert.strictEqual(byId.name, 'Unique User');
		
		// Find by unique field
		const byEmail = await db.models.User.findUnique({
			where: { email: 'unique@example.com' }
		});
		assert.strictEqual(byEmail.id, created.id);
		
		// await db.close();
	`)

	// Test findFirst
	jct.runWithCleanup(t, runner, "FindFirst", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model User {
	id    Int    @id @default(autoincrement())
	name  String
	email String @unique
	age   Int?
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		await db.models.User.create({ data: { name: 'Alice', email: 'alice@example.com', age: 25 } });
		await db.models.User.create({ data: { name: 'Bob', email: 'bob@example.com', age: 30 } });
		await db.models.User.create({ data: { name: 'Charlie', email: 'charlie@example.com', age: 25 } });
		
		// Find first with condition
		const firstAge25 = await db.models.User.findFirst({
			where: { age: 25 },
			orderBy: { name: 'asc' }
		});
		assert.strictEqual(firstAge25.name, 'Alice');
		
		// await db.close();
	`)

	// Test count
	jct.runWithCleanup(t, runner, "Count", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model Product {
	id       Int     @id @default(autoincrement())
	name     String
	category String
	active   Boolean @default(true)
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		await db.models.Product.create({ data: { name: 'Product 1', category: 'Electronics' } });
		await db.models.Product.create({ data: { name: 'Product 2', category: 'Electronics' } });
		await db.models.Product.create({ data: { name: 'Product 3', category: 'Books', active: false } });
		
		// Count all
		const total = await db.models.Product.count();
		assert.strictEqual(total, 3);
		
		// Count with filter
		const electronics = await db.models.Product.count({
			where: { category: 'Electronics' }
		});
		assert.strictEqual(electronics, 2);
		
		// Count active
		const active = await db.models.Product.count({
			where: { active: true }
		});
		assert.strictEqual(active, 2);
		
		// await db.close();
	`)

	// Test upsert
	jct.runWithCleanup(t, runner, "Upsert", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model Config {
	id    Int    @id @default(autoincrement())
	key   String @unique
	value String
}
` + "`" + `);
		await db.syncSchemas();
		
		// First upsert - should create
		const created = await db.models.Config.upsert({
			where: { key: 'theme' },
			create: { key: 'theme', value: 'light' },
			update: { value: 'dark' }
		});
		assert.strictEqual(created.key, 'theme');
		assert.strictEqual(created.value, 'light');
		
		// Second upsert - should update
		const updated = await db.models.Config.upsert({
			where: { key: 'theme' },
			create: { key: 'theme', value: 'light' },
			update: { value: 'dark' }
		});
		assert.strictEqual(updated.key, 'theme');
		assert.strictEqual(updated.value, 'dark');
		
		// await db.close();
	`)

	// Test createMany
	jct.runWithCleanup(t, runner, "CreateMany", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model Task {
	id     Int    @id @default(autoincrement())
	title  String
	status String @default("pending")
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create multiple records
		const result = await db.models.Task.createMany({
			data: [
				{ title: 'Task 1' },
				{ title: 'Task 2' },
				{ title: 'Task 3', status: 'completed' }
			]
		});
		
		// Different databases return different results
		// Some return count, some return array of created records
		if (typeof result === 'object' && result.count !== undefined) {
			assert.strictEqual(result.count, 3);
		} else if (Array.isArray(result)) {
			assert.strictEqual(result.length, 3);
		} else if (typeof result === 'number') {
			assert.strictEqual(result, 3);
		}
		
		// Verify records
		const tasks = await db.models.Task.findMany({ orderBy: { title: 'asc' } });
		assert.strictEqual(tasks.length, 3);
		assert.strictEqual(tasks[0].title, 'Task 1');
		assert.strictEqual(tasks[0].status, 'pending');
		assert.strictEqual(tasks[2].status, 'completed');
		
		// await db.close();
	`)

	// Test updateMany
	jct.runWithCleanup(t, runner, "UpdateMany", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model Task {
	id     Int    @id @default(autoincrement())
	title  String
	status String @default("pending")
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		await db.models.Task.create({ data: { title: 'Task 1' } });
		await db.models.Task.create({ data: { title: 'Task 2' } });
		await db.models.Task.create({ data: { title: 'Task 3', status: 'completed' } });
		
		// Update multiple records
		const result = await db.models.Task.updateMany({
			where: { status: 'pending' },
			data: { status: 'in_progress' }
		});
		
		// Verify count
		if (typeof result === 'object' && result.count !== undefined) {
			assert.strictEqual(result.count, 2);
		} else if (typeof result === 'number') {
			assert.strictEqual(result, 2);
		}
		
		// Verify updates
		const pending = await db.models.Task.count({ where: { status: 'pending' } });
		const inProgress = await db.models.Task.count({ where: { status: 'in_progress' } });
		assert.strictEqual(pending, 0);
		assert.strictEqual(inProgress, 2);
		
		// await db.close();
	`)

	// Test deleteMany
	jct.runWithCleanup(t, runner, "DeleteMany", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model TempData {
	id        Int      @id @default(autoincrement())
	data      String
	createdAt DateTime @default(now())
	temp      Boolean  @default(true)
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		await db.models.TempData.create({ data: { data: 'Keep 1', temp: false } });
		await db.models.TempData.create({ data: { data: 'Delete 1' } });
		await db.models.TempData.create({ data: { data: 'Delete 2' } });
		await db.models.TempData.create({ data: { data: 'Keep 2', temp: false } });
		
		// Delete multiple records
		const result = await db.models.TempData.deleteMany({
			where: { temp: true }
		});
		
		// Verify count
		if (typeof result === 'object' && result.count !== undefined) {
			assert.strictEqual(result.count, 2);
		} else if (typeof result === 'number') {
			assert.strictEqual(result, 2);
		}
		
		// Verify remaining records
		const remaining = await db.models.TempData.findMany();
		assert.strictEqual(remaining.length, 2);
		// SQLite returns 0/1 for booleans
		assert(remaining.every(r => r.temp === false || r.temp === 0));
		
		// await db.close();
	`)
}