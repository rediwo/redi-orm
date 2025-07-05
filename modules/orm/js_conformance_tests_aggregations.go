package orm

import (
	"testing"
)

// Aggregation Tests
func (jct *JSConformanceTests) runAggregationTests(t *testing.T, runner *JSTestRunner) {
	// Test basic count
	jct.runWithCleanup(t, runner, "BasicCount", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model User {
	id      Int     @id @default(autoincrement())
	name    String
	age     Int
	active  Boolean @default(true)
	country String
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		await db.models.User.create({ data: { name: 'Alice', age: 25, country: 'USA' } });
		await db.models.User.create({ data: { name: 'Bob', age: 30, country: 'UK' } });
		await db.models.User.create({ data: { name: 'Charlie', age: 35, country: 'USA' } });
		await db.models.User.create({ data: { name: 'David', age: 28, country: 'Canada' } });
		await db.models.User.create({ data: { name: 'Eve', age: 32, country: 'UK', active: false } });
		
		// Test count all
		const totalCount = await db.models.User.count();
		assert.strictEqual(totalCount, 5);
		
		// Test count with where condition
		const activeCount = await db.models.User.count({
			where: { active: true }
		});
		assert.strictEqual(activeCount, 4);
		
		// Test count with multiple conditions
		const usaCount = await db.models.User.count({
			where: { 
				country: 'USA',
				active: true
			}
		});
		assert.strictEqual(usaCount, 2);
		
		// await db.close();
	`)

	// Test aggregations
	jct.runWithCleanup(t, runner, "AggregationFunctions", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Sale {
	id        Int    @id @default(autoincrement())
	product   String
	amount    Float
	quantity  Int
	category  String
	createdAt DateTime @default(now())
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		await db.models.Sale.create({ data: { product: 'Widget', amount: 100.50, quantity: 2, category: 'Hardware' } });
		await db.models.Sale.create({ data: { product: 'Gadget', amount: 150.75, quantity: 1, category: 'Hardware' } });
		await db.models.Sale.create({ data: { product: 'Service', amount: 200.00, quantity: 1, category: 'Software' } });
		await db.models.Sale.create({ data: { product: 'Widget', amount: 100.50, quantity: 3, category: 'Hardware' } });
		await db.models.Sale.create({ data: { product: 'License', amount: 500.00, quantity: 1, category: 'Software' } });
		
		// Test aggregate functions
		const stats = await db.models.Sale.aggregate({
			_count: true,
			_sum: {
				amount: true,
				quantity: true
			},
			_avg: {
				amount: true,
				quantity: true
			},
			_min: {
				amount: true
			},
			_max: {
				amount: true
			}
		});
		
		assert.strictEqual(stats._count, 5);
		assert.strictEqual(stats._sum.amount, 1051.75);
		assert.strictEqual(stats._sum.quantity, 8);
		assert.strictEqual(stats._avg.amount, 210.35);
		assert.strictEqual(stats._avg.quantity, 1.6);
		assert.strictEqual(stats._min.amount, 100.50);
		assert.strictEqual(stats._max.amount, 500.00);
		
		// Test aggregate with where condition
		const hardwareStats = await db.models.Sale.aggregate({
			where: { category: 'Hardware' },
			_count: true,
			_sum: {
				amount: true
			}
		});
		
		assert.strictEqual(hardwareStats._count, 3);
		assert.strictEqual(hardwareStats._sum.amount, 351.75);
		
		// await db.close();
	`)

	// Test groupBy
	jct.runWithCleanup(t, runner, "GroupBy", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Order {
	id         Int      @id @default(autoincrement())
	product    String
	category   String
	amount     Float
	quantity   Int
	status     String
	customerId Int
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		await db.models.Order.create({ data: { product: 'Laptop', category: 'Electronics', amount: 1000, quantity: 1, status: 'completed', customerId: 1 } });
		await db.models.Order.create({ data: { product: 'Mouse', category: 'Electronics', amount: 50, quantity: 2, status: 'completed', customerId: 1 } });
		await db.models.Order.create({ data: { product: 'Desk', category: 'Furniture', amount: 500, quantity: 1, status: 'completed', customerId: 2 } });
		await db.models.Order.create({ data: { product: 'Chair', category: 'Furniture', amount: 300, quantity: 2, status: 'pending', customerId: 2 } });
		await db.models.Order.create({ data: { product: 'Monitor', category: 'Electronics', amount: 400, quantity: 1, status: 'completed', customerId: 3 } });
		
		// Test groupBy single field
		const byCategory = await db.models.Order.groupBy({
			by: ['category'],
			_count: true,
			_sum: {
				amount: true,
				quantity: true
			},
			orderBy: {
				category: 'asc'
			}
		});
		
		assert.lengthOf(byCategory, 2);
		
		const electronics = byCategory.find(g => g.category === 'Electronics');
		assert(electronics);
		assert.strictEqual(Number(electronics._count), 3);
		assert.strictEqual(Number(electronics._sum.amount), 1450);
		assert.strictEqual(Number(electronics._sum.quantity), 4);
		
		const furniture = byCategory.find(g => g.category === 'Furniture');
		assert(furniture);
		assert.strictEqual(Number(furniture._count), 2);
		assert.strictEqual(Number(furniture._sum.amount), 800);
		assert.strictEqual(Number(furniture._sum.quantity), 3);
		
		// Test groupBy multiple fields
		const byCategoryAndStatus = await db.models.Order.groupBy({
			by: ['category', 'status'],
			_count: true,
			_avg: {
				amount: true
			},
			orderBy: [
				{ category: 'asc' },
				{ status: 'asc' }
			]
		});
		
		assert(byCategoryAndStatus.length >= 3);
		
		// Test groupBy with where
		const completedByCategory = await db.models.Order.groupBy({
			by: ['category'],
			where: { status: 'completed' },
			_count: true,
			_sum: {
				amount: true
			}
		});
		
		assert.lengthOf(completedByCategory, 2);
		
		// await db.close();
	`)

	// Test having clause
	jct.runWithCleanup(t, runner, "Having", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Purchase {
	id         Int      @id @default(autoincrement())
	customerId Int
	amount     Float
	createdAt  DateTime @default(now())
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data - customers with different purchase amounts
		// Customer 1: 3 purchases totaling 600
		await db.models.Purchase.create({ data: { customerId: 1, amount: 100 } });
		await db.models.Purchase.create({ data: { customerId: 1, amount: 200 } });
		await db.models.Purchase.create({ data: { customerId: 1, amount: 300 } });
		
		// Customer 2: 2 purchases totaling 150
		await db.models.Purchase.create({ data: { customerId: 2, amount: 50 } });
		await db.models.Purchase.create({ data: { customerId: 2, amount: 100 } });
		
		// Customer 3: 4 purchases totaling 1000
		await db.models.Purchase.create({ data: { customerId: 3, amount: 250 } });
		await db.models.Purchase.create({ data: { customerId: 3, amount: 250 } });
		await db.models.Purchase.create({ data: { customerId: 3, amount: 250 } });
		await db.models.Purchase.create({ data: { customerId: 3, amount: 250 } });
		
		// Test having with sum condition
		const bigSpenders = await db.models.Purchase.groupBy({
			by: ['customerId'],
			_sum: {
				amount: true
			},
			_count: true,
			having: {
				_sum: {
					amount: {
						gte: 500
					}
				}
			},
			orderBy: {
				customerId: 'asc'
			}
		});
		
		assert.lengthOf(bigSpenders, 2); // Only customers 1 and 3
		assert.strictEqual(bigSpenders[0].customerId, 1);
		assert.strictEqual(Number(bigSpenders[0]._sum.amount), 600);
		assert.strictEqual(bigSpenders[1].customerId, 3);
		assert.strictEqual(Number(bigSpenders[1]._sum.amount), 1000);
		
		// Test having with count condition
		const frequentBuyers = await db.models.Purchase.groupBy({
			by: ['customerId'],
			_count: true,
			having: {
				_count: {
					_all: {
						gte: 3
					}
				}
			}
		});
		
		assert.lengthOf(frequentBuyers, 2); // Customers 1 and 3
		
		// Test having with avg condition
		const highAvgPurchases = await db.models.Purchase.groupBy({
			by: ['customerId'],
			_avg: {
				amount: true
			},
			having: {
				_avg: {
					amount: {
						gt: 150
					}
				}
			}
		});
		
		assert.lengthOf(highAvgPurchases, 2); // Customers 1 (avg 200) and 3 (avg 250)
		
		// await db.close();
	`)

	// Test distinct
	jct.runWithCleanup(t, runner, "Distinct", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Product {
	id       Int    @id @default(autoincrement())
	name     String
	category String
	brand    String
	price    Float
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data with duplicates
		await db.models.Product.create({ data: { name: 'Laptop', category: 'Electronics', brand: 'BrandA', price: 1000 } });
		await db.models.Product.create({ data: { name: 'Mouse', category: 'Electronics', brand: 'BrandB', price: 50 } });
		await db.models.Product.create({ data: { name: 'Keyboard', category: 'Electronics', brand: 'BrandA', price: 100 } });
		await db.models.Product.create({ data: { name: 'Monitor', category: 'Electronics', brand: 'BrandC', price: 400 } });
		await db.models.Product.create({ data: { name: 'Desk', category: 'Furniture', brand: 'BrandD', price: 500 } });
		await db.models.Product.create({ data: { name: 'Chair', category: 'Furniture', brand: 'BrandD', price: 300 } });
		await db.models.Product.create({ data: { name: 'Lamp', category: 'Furniture', brand: 'BrandE', price: 80 } });
		
		// Test distinct on single field
		const categories = await db.models.Product.findMany({
			distinct: ['category'],
			select: { category: true },
			orderBy: { category: 'asc' }
		});
		
		assert.lengthOf(categories, 2);
		assert.strictEqual(categories[0].category, 'Electronics');
		assert.strictEqual(categories[1].category, 'Furniture');
		
		// Test distinct on single field (brand)
		const brands = await db.models.Product.findMany({
			distinct: ['brand'],
			select: { brand: true },
			orderBy: { brand: 'asc' }
		});
		
		assert.lengthOf(brands, 5); // BrandA through BrandE
		
		// Test distinct with where condition
		const electronicsBrands = await db.models.Product.findMany({
			distinct: ['brand'],
			where: { category: 'Electronics' },
			select: { brand: true },
			orderBy: { brand: 'asc' }
		});
		
		assert.lengthOf(electronicsBrands, 3); // BrandA, BrandB, BrandC
		
		// Test distinct with multiple fields
		const categoryBrandCombos = await db.models.Product.findMany({
			distinct: ['category', 'brand'],
			select: { category: true, brand: true },
			orderBy: [
				{ category: 'asc' },
				{ brand: 'asc' }
			]
		});
		
		assert.lengthOf(categoryBrandCombos, 5); // All unique combinations (Electronics: A,B,C; Furniture: D,E)
		
		// Test simple distinct (all columns)
		const allProducts = await db.models.Product.findMany({
			distinct: true
		});
		
		// Since we have 7 products with unique combinations, distinct should return all 7
		assert.lengthOf(allProducts, 7);
		
		// await db.close();
	`)

	// Test complex aggregation queries
	jct.runWithCleanup(t, runner, "ComplexAggregations", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Transaction {
	id         Int      @id @default(autoincrement())
	userId     Int
	type       String   // 'income' or 'expense'
	category   String
	amount     Float
	createdAt  DateTime @default(now())
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data for 3 users over multiple months
		const now = new Date();
		const lastMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1);
		const twoMonthsAgo = new Date(now.getFullYear(), now.getMonth() - 2, 1);
		
		// User 1 transactions
		await db.models.Transaction.create({ data: { userId: 1, type: 'income', category: 'Salary', amount: 5000, createdAt: twoMonthsAgo } });
		await db.models.Transaction.create({ data: { userId: 1, type: 'expense', category: 'Rent', amount: 1500, createdAt: twoMonthsAgo } });
		await db.models.Transaction.create({ data: { userId: 1, type: 'expense', category: 'Food', amount: 500, createdAt: twoMonthsAgo } });
		await db.models.Transaction.create({ data: { userId: 1, type: 'income', category: 'Salary', amount: 5000, createdAt: lastMonth } });
		await db.models.Transaction.create({ data: { userId: 1, type: 'expense', category: 'Rent', amount: 1500, createdAt: lastMonth } });
		
		// User 2 transactions
		await db.models.Transaction.create({ data: { userId: 2, type: 'income', category: 'Freelance', amount: 3000, createdAt: lastMonth } });
		await db.models.Transaction.create({ data: { userId: 2, type: 'expense', category: 'Food', amount: 600, createdAt: lastMonth } });
		await db.models.Transaction.create({ data: { userId: 2, type: 'expense', category: 'Transport', amount: 200, createdAt: lastMonth } });
		
		// Test complex groupBy with multiple aggregations
		const userStats = await db.models.Transaction.groupBy({
			by: ['userId', 'type'],
			_sum: {
				amount: true
			},
			_count: true,
			_avg: {
				amount: true
			},
			orderBy: [
				{ userId: 'asc' },
				{ type: 'asc' }
			]
		});
		
		// Verify user 1 income
		const user1Income = userStats.find(s => s.userId === 1 && s.type === 'income');
		assert(user1Income);
		assert.strictEqual(user1Income._count, 2);
		assert.strictEqual(user1Income._sum.amount, 10000);
		assert.strictEqual(user1Income._avg.amount, 5000);
		
		// Test aggregation with complex having
		const bigCategories = await db.models.Transaction.groupBy({
			by: ['category'],
			where: { type: 'expense' },
			_sum: {
				amount: true
			},
			_count: true,
			having: {
				_sum: {
					amount: {
						gt: 1000
					}
				}
			},
			orderBy: {
				_sum: {
					amount: 'desc'
				}
			}
		});
		
		assert(bigCategories.length > 0);
		assert.strictEqual(bigCategories[0].category, 'Rent'); // Highest expense category
		assert.strictEqual(Number(bigCategories[0]._sum.amount), 3000);
		
		// await db.close();
	`)
}
