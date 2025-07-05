package orm

import (
	"testing"
)

// Query Building Tests
func (jct *JSConformanceTests) runQueryTests(t *testing.T, runner *JSTestRunner) {
	// Test where conditions
	jct.runWithCleanup(t, runner, "WhereConditions", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model User {
	id    Int    @id @default(autoincrement())
	name  String
	email String @unique
	age   Int
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		await db.models.User.create({ data: { name: 'Alice', email: 'alice@example.com', age: 25 } });
		await db.models.User.create({ data: { name: 'Bob', email: 'bob@example.com', age: 30 } });
		await db.models.User.create({ data: { name: 'Charlie', email: 'charlie@example.com', age: 35 } });
		
		// Test equals
		const alice = await db.models.User.findMany({ where: { name: 'Alice' } });
		assert.lengthOf(alice, 1);
		assert.strictEqual(alice[0].name, 'Alice');
		
		// Test multiple conditions (AND)
		const young = await db.models.User.findMany({ 
			where: { 
				age: { lt: 30 },
				name: { not: 'Bob' }
			} 
		});
		assert.lengthOf(young, 1);
		assert.strictEqual(young[0].name, 'Alice');
		
		// await db.close();
	`)

	// Test orderBy
	jct.runWithCleanup(t, runner, "OrderBy", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model Product {
	id    Int    @id @default(autoincrement())
	name  String
	price Float
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		await db.models.Product.create({ data: { name: 'Laptop', price: 999.99 } });
		await db.models.Product.create({ data: { name: 'Mouse', price: 29.99 } });
		await db.models.Product.create({ data: { name: 'Keyboard', price: 79.99 } });
		
		// Test orderBy ascending
		const asc = await db.models.Product.findMany({ 
			orderBy: { price: 'asc' } 
		});
		assert.strictEqual(asc[0].name, 'Mouse');
		assert.strictEqual(asc[2].name, 'Laptop');
		
		// Test orderBy descending
		const desc = await db.models.Product.findMany({ 
			orderBy: { price: 'desc' } 
		});
		assert.strictEqual(desc[0].name, 'Laptop');
		assert.strictEqual(desc[2].name, 'Mouse');
		
		// await db.close();
	`)

	// Test pagination
	jct.runWithCleanup(t, runner, "Pagination", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model Item {
	id   Int    @id @default(autoincrement())
	name String
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		for (let i = 1; i <= 10; i++) {
			await db.models.Item.create({ data: { name: 'Item ' + i } });
		}
		
		// Test take
		const firstThree = await db.models.Item.findMany({ 
			take: 3,
			orderBy: { id: 'asc' }
		});
		assert.lengthOf(firstThree, 3);
		assert.strictEqual(firstThree[0].name, 'Item 1');
		assert.strictEqual(firstThree[2].name, 'Item 3');
		
		// Test skip and take
		const page2 = await db.models.Item.findMany({ 
			skip: 3,
			take: 3,
			orderBy: { id: 'asc' }
		});
		assert.lengthOf(page2, 3);
		assert.strictEqual(page2[0].name, 'Item 4');
		assert.strictEqual(page2[2].name, 'Item 6');
		
		// await db.close();
	`)

	// Test select
	jct.runWithCleanup(t, runner, "Select", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model User {
	id        Int    @id @default(autoincrement())
	email     String @unique
	password  String
	firstName String
	lastName  String
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data
		await db.models.User.create({ 
			data: { 
				email: 'user@example.com',
				password: 'secret123',
				firstName: 'John',
				lastName: 'Doe'
			} 
		});
		
		// Test select specific fields
		const partial = await db.models.User.findMany({ 
			select: { 
				id: true,
				email: true,
				firstName: true,
				lastName: true
			} 
		});
		
		assert(partial[0].id);
		assert(partial[0].email);
		assert(partial[0].firstName);
		assert(partial[0].lastName);
		// Password should not be included
		assert.strictEqual(partial[0].password, undefined);
		
		// await db.close();
	`)

	// Test distinct
	jct.runWithCleanup(t, runner, "Distinct", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(` + "`" + `
model Event {
	id       Int    @id @default(autoincrement())
	category String
	name     String
}
` + "`" + `);
		await db.syncSchemas();
		
		// Create test data with duplicate categories
		await db.models.Event.create({ data: { category: 'Sports', name: 'Football' } });
		await db.models.Event.create({ data: { category: 'Sports', name: 'Basketball' } });
		await db.models.Event.create({ data: { category: 'Music', name: 'Concert' } });
		await db.models.Event.create({ data: { category: 'Sports', name: 'Tennis' } });
		await db.models.Event.create({ data: { category: 'Music', name: 'Festival' } });
		
		// Test distinct on category
		const categories = await db.models.Event.findMany({ 
			distinct: ['category'],
			orderBy: { category: 'asc' }
		});
		
		// Should only have 2 distinct categories
		assert.lengthOf(categories, 2);
		assert.strictEqual(categories[0].category, 'Music');
		assert.strictEqual(categories[1].category, 'Sports');
		
		// await db.close();
	`)
}