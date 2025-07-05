package orm

import (
	"testing"
)

// Advanced Query Tests
func (jct *JSConformanceTests) runAdvancedQueryTests(t *testing.T, runner *JSTestRunner) {
	// Test complex where conditions
	jct.runWithCleanup(t, runner, "ComplexWhereConditions", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Product {
	id       Int    @id @default(autoincrement())
	name     String
	price    Float
	category String
	inStock  Boolean @default(true)
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		await db.models.Product.create({ data: { name: 'Laptop', price: 999, category: 'Electronics', inStock: true } });
		await db.models.Product.create({ data: { name: 'Mouse', price: 29, category: 'Electronics', inStock: true } });
		await db.models.Product.create({ data: { name: 'Book', price: 15, category: 'Books', inStock: false } });
		await db.models.Product.create({ data: { name: 'Pen', price: 2, category: 'Stationery', inStock: true } });
		
		// Test OR conditions
		const results = await db.models.Product.findMany({
			where: {
				OR: [
					{ category: 'Electronics' },
					{ price: { lt: 10 } }
				]
			},
			orderBy: { price: 'asc' }
		});
		assert.lengthOf(results, 3); // Electronics (2) + Pen (1)
		
		// Test NOT conditions
		const notElectronics = await db.models.Product.findMany({
			where: {
				NOT: { category: 'Electronics' }
			}
		});
		assert.lengthOf(notElectronics, 2);
		
		// Test complex nested conditions
		const complex = await db.models.Product.findMany({
			where: {
				AND: [
					{ inStock: true },
					{
						OR: [
							{ category: 'Electronics' },
							{ price: { lt: 5 } }
						]
					}
				]
			}
		});
		assert.lengthOf(complex, 3); // Laptop, Mouse, Pen
		
		// await db.close();
	`)

	// Test string operators
	jct.runWithCleanup(t, runner, "StringOperators", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Article {
	id      Int    @id @default(autoincrement())
	title   String
	content String
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		await db.models.Article.create({ data: { title: 'JavaScript Basics', content: 'Learn JS fundamentals' } });
		await db.models.Article.create({ data: { title: 'Advanced TypeScript', content: 'Deep dive into TS' } });
		await db.models.Article.create({ data: { title: 'Node.js Guide', content: 'Server-side JavaScript' } });
		
		// Test startsWith
		const jsArticles = await db.models.Article.findMany({
			where: { title: { startsWith: 'Java' } }
		});
		assert.lengthOf(jsArticles, 1);
		assert.strictEqual(jsArticles[0].title, 'JavaScript Basics');
		
		// Test endsWith
		const scriptArticles = await db.models.Article.findMany({
			where: { title: { endsWith: 'Script' } }
		});
		assert.lengthOf(scriptArticles, 1);
		assert.strictEqual(scriptArticles[0].title, 'Advanced TypeScript');
		
		// Test contains
		const containsGuide = await db.models.Article.findMany({
			where: { title: { contains: 'Guide' } }
		});
		assert.lengthOf(containsGuide, 1);
		assert.strictEqual(containsGuide[0].title, 'Node.js Guide');
		
		// await db.close();
	`)

	// Test numeric operators
	jct.runWithCleanup(t, runner, "NumericOperators", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Score {
	id     Int    @id @default(autoincrement())
	player String
	points Int
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		await db.models.Score.create({ data: { player: 'Alice', points: 100 } });
		await db.models.Score.create({ data: { player: 'Bob', points: 85 } });
		await db.models.Score.create({ data: { player: 'Charlie', points: 95 } });
		await db.models.Score.create({ data: { player: 'David', points: 100 } });
		
		// Test gt (greater than)
		const highScores = await db.models.Score.findMany({
			where: { points: { gt: 90 } }
		});
		assert.lengthOf(highScores, 3);
		
		// Test gte (greater than or equal)
		const topScores = await db.models.Score.findMany({
			where: { points: { gte: 100 } }
		});
		assert.lengthOf(topScores, 2);
		
		// Test lt and lte
		const lowScores = await db.models.Score.findMany({
			where: { points: { lte: 90 } }
		});
		assert.lengthOf(lowScores, 1);
		assert.strictEqual(lowScores[0].player, 'Bob');
		
		// await db.close();
	`)

	// Test between operator
	jct.runWithCleanup(t, runner, "BetweenOperator", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Event {
	id    Int      @id @default(autoincrement())
	name  String
	date  DateTime
	price Float
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		const now = new Date();
		const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
		const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000);
		const nextWeek = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);
		
		await db.models.Event.create({ data: { name: 'Past Event', date: yesterday, price: 50 } });
		await db.models.Event.create({ data: { name: 'Today Event', date: now, price: 75 } });
		await db.models.Event.create({ data: { name: 'Tomorrow Event', date: tomorrow, price: 100 } });
		await db.models.Event.create({ data: { name: 'Future Event', date: nextWeek, price: 150 } });
		
		// Test price between
		const midPrice = await db.models.Event.findMany({
			where: {
				price: {
					gte: 60,
					lte: 120
				}
			},
			orderBy: { price: 'asc' }
		});
		assert.lengthOf(midPrice, 2);
		assert.strictEqual(midPrice[0].name, 'Today Event');
		assert.strictEqual(midPrice[1].name, 'Tomorrow Event');
		
		// await db.close();
	`)

	// Test in operator
	jct.runWithCleanup(t, runner, "InOperator", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model User {
	id      Int    @id @default(autoincrement())
	name    String
	role    String
	country String
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		await db.models.User.create({ data: { name: 'Alice', role: 'admin', country: 'USA' } });
		await db.models.User.create({ data: { name: 'Bob', role: 'user', country: 'UK' } });
		await db.models.User.create({ data: { name: 'Charlie', role: 'moderator', country: 'Canada' } });
		await db.models.User.create({ data: { name: 'David', role: 'user', country: 'USA' } });
		
		// Test in operator
		const privileged = await db.models.User.findMany({
			where: {
				role: { in: ['admin', 'moderator'] }
			}
		});
		assert.lengthOf(privileged, 2);
		
		// Test notIn operator
		const regularUsers = await db.models.User.findMany({
			where: {
				role: { notIn: ['admin', 'moderator'] }
			}
		});
		assert.lengthOf(regularUsers, 2);
		assert(regularUsers.every(u => u.role === 'user'));
		
		// await db.close();
	`)

	// Test null checks
	jct.runWithCleanup(t, runner, "NullChecks", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Task {
	id          Int       @id @default(autoincrement())
	title       String
	description String?
	completedAt DateTime?
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		await db.models.Task.create({ data: { title: 'Task 1', description: 'Do something' } });
		await db.models.Task.create({ data: { title: 'Task 2' } });
		await db.models.Task.create({ data: { title: 'Task 3', description: 'Done', completedAt: new Date() } });
		
		// Test isNull
		const noDescription = await db.models.Task.findMany({
			where: { description: null }
		});
		assert.lengthOf(noDescription, 1);
		assert.strictEqual(noDescription[0].title, 'Task 2');
		
		// Test isNotNull
		const withDescription = await db.models.Task.findMany({
			where: { description: { not: null } }
		});
		assert.lengthOf(withDescription, 2);
		
		// Test completed tasks
		const completed = await db.models.Task.findMany({
			where: { completedAt: { not: null } }
		});
		assert.lengthOf(completed, 1);
		assert.strictEqual(completed[0].title, 'Task 3');
		
		// await db.close();
	`)

	// Test aggregations
	jct.runWithCleanup(t, runner, "Aggregations", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Sale {
	id       Int    @id @default(autoincrement())
	product  String
	amount   Float
	quantity Int
	region   String
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		await db.models.Sale.create({ data: { product: 'Widget', amount: 100, quantity: 2, region: 'North' } });
		await db.models.Sale.create({ data: { product: 'Gadget', amount: 150, quantity: 1, region: 'North' } });
		await db.models.Sale.create({ data: { product: 'Widget', amount: 200, quantity: 4, region: 'South' } });
		await db.models.Sale.create({ data: { product: 'Gadget', amount: 300, quantity: 2, region: 'South' } });
		
		// Test aggregate functions
		const stats = await db.models.Sale.aggregate({
			_sum: { amount: true, quantity: true },
			_avg: { amount: true },
			_min: { amount: true },
			_max: { amount: true },
			_count: true
		});
		
		assert.strictEqual(stats._count, 4);
		assert.strictEqual(stats._sum.amount, 750);
		assert.strictEqual(stats._sum.quantity, 9);
		assert.strictEqual(stats._avg.amount, 187.5);
		assert.strictEqual(stats._min.amount, 100);
		assert.strictEqual(stats._max.amount, 300);
		
		// Test groupBy
		const byRegion = await db.models.Sale.groupBy({
			by: ['region'],
			_sum: { amount: true },
			_count: true
		});
		
		assert.lengthOf(byRegion, 2);
		const north = byRegion.find(r => r.region === 'North');
		const south = byRegion.find(r => r.region === 'South');
		assert.strictEqual(north._sum.amount, 250);
		assert.strictEqual(south._sum.amount, 500);
		
		// await db.close();
	`)
}
