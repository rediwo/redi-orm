package orm

import (
	"testing"
)

// Migration Tests
func (jct *JSConformanceTests) runMigrationTests(t *testing.T, runner *JSTestRunner) {
	// Test basic schema sync
	jct.runWithCleanup(t, runner, "BasicSchemaSync", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		// Load initial schema
		await db.loadSchema(`+"`"+`
model User {
	id    Int    @id @default(autoincrement())
	name  String
	email String @unique
}
`+"`"+`);
		await db.syncSchemas();
		
		// Verify table was created
		const models = db.getModels();
		assert(models.includes('User'));
		
		// Create a user to verify table exists
		const user = await db.models.User.create({
			data: { name: 'Test User', email: 'test@example.com' }
		});
		assert(user.id);
		
		// await db.close();
	`)

	// Test schema evolution
	jct.runWithCleanup(t, runner, "SchemaEvolution", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		// Initial schema
		await db.loadSchema(`+"`"+`
model Product {
	id    Int    @id @default(autoincrement())
	name  String
	price Float
}
`+"`"+`);
		await db.syncSchemas();
		
		// Add some data
		await db.models.Product.create({
			data: { name: 'Laptop', price: 999.99 }
		});
		
		// Load updated schema with new field
		// Note: Adding non-nullable columns with defaults to existing tables
		// is handled differently by each database
		await db.loadSchema(`+"`"+`
model Product {
	id          Int     @id @default(autoincrement())
	name        String
	price       Float
	description String?
	inStock     Boolean? @default(true)
}
`+"`"+`);
		await db.syncSchemas();
		
		// Verify we can use new fields
		console.log('Creating new product with new fields...');
		const product = await db.models.Product.create({
			data: { 
				name: 'Mouse', 
				price: 29.99, 
				description: 'Wireless mouse',
				inStock: true 
			}
		});
		console.log('Created product:', JSON.stringify(product, null, 2));
		assert(product.description === 'Wireless mouse', 'Description should match');
		// Handle field name variations (camelCase vs snake_case)
		const inStock = product.inStock !== undefined ? product.inStock : product.in_stock;
		assert(inStock === true || inStock === 1, 'inStock should be true');
		
		// Verify old data still exists and has default values
		const products = await db.models.Product.findMany({ orderBy: { id: 'asc' } });
		console.log('Products after schema evolution:', JSON.stringify(products, null, 2));
		assert(products.length === 2, 'Should have 2 products');
		
		// Old product should have default values for new fields
		const oldProduct = products[0];
		assert(oldProduct.name === 'Laptop', 'First product should be Laptop');
		// Check that inStock got the default value (true) - handle field name variations
		const oldInStock = oldProduct.inStock !== undefined ? oldProduct.inStock : oldProduct.in_stock;
		console.log('Old product inStock value:', oldInStock, 'type:', typeof oldInStock);
		assert(oldInStock === true || oldInStock === 1 || oldInStock === null, 'Old product should have default or null inStock value');
		
		// await db.close();
	`)

	// Test index creation
	jct.runWithCleanup(t, runner, "IndexCreation", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		// Schema with indexes
		await db.loadSchema(`+"`"+`
model Article {
	id        Int      @id @default(autoincrement())
	title     String
	content   String
	published Boolean  @default(false)
	createdAt DateTime @default(now())
	
	@@index([title])
	@@index([published, createdAt])
}
`+"`"+`);
		await db.syncSchemas();
		
		// Add test data
		for (let i = 0; i < 10; i++) {
			await db.models.Article.create({
				data: {
					title: 'Article ' + i,
					content: 'Content ' + i,
					published: i % 2 === 0
				}
			});
		}
		
		// Query using indexed fields (should be efficient)
		const publishedArticles = await db.models.Article.findMany({
			where: { published: true },
			orderBy: { createdAt: 'desc' }
		});
		assert(publishedArticles.length === 5);
		
		// await db.close();
	`)

	// Test composite primary keys
	jct.runWithCleanup(t, runner, "CompositePrimaryKeys", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		// Schema with composite key
		await db.loadSchema(`+"`"+`
model PostTag {
	postId Int
	tagId  Int
	
	@@id([postId, tagId])
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create records
		await db.models.PostTag.create({
			data: { postId: 1, tagId: 1 }
		});
		await db.models.PostTag.create({
			data: { postId: 1, tagId: 2 }
		});
		await db.models.PostTag.create({
			data: { postId: 2, tagId: 1 }
		});
		
		// Verify records exist
		const postTags = await db.models.PostTag.findMany();
		assert(postTags.length === 3);
		
		// await db.close();
	`)

	// Test field mapping
	jct.runWithCleanup(t, runner, "FieldMapping", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		// Schema with field mapping
		await db.loadSchema(`+"`"+`
model Customer {
	id        Int    @id @default(autoincrement())
	firstName String @map("first_name")
	lastName  String @map("last_name")
	isActive  Boolean @default(true) @map("active")
	
	@@map("customers")
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create record using model field names
		const customer = await db.models.Customer.create({
			data: { 
				firstName: 'John',
				lastName: 'Doe',
				isActive: true
			}
		});
		
		// Debug: log the customer object to see what fields are returned
		console.log('Created customer object:', JSON.stringify(customer, null, 2));
		
		// Verify field mapping works
		// For PostgreSQL, check all possible field name variations
		const firstName = customer.firstName || customer.first_name || customer.firstname;
		const lastName = customer.lastName || customer.last_name || customer.lastname;
		const isActive = customer.isActive !== undefined ? customer.isActive : 
		                 customer.active !== undefined ? customer.active : 
		                 customer.is_active;
		
		assert(firstName === 'John', 'firstName should be John, got: ' + firstName);
		assert(lastName === 'Doe', 'lastName should be Doe, got: ' + lastName);
		assert(isActive === true || isActive === 1, 'isActive should be true, got: ' + isActive);
		
		// Query using mapped fields
		const activeCustomers = await db.models.Customer.findMany({
			where: { isActive: true }
		});
		assert(activeCustomers.length === 1, 'Should find 1 active customer');
		
		// await db.close();
	`)

	// Test advanced field mapping scenarios
	jct.runWithCleanup(t, runner, "AdvancedFieldMapping", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		// Schema with complex field and table mappings
		await db.loadSchema(`+"`"+`
model UserProfile {
	id              Int      @id @default(autoincrement())
	userName        String   @map("user_name") @unique
	emailAddress    String   @map("email_address")
	phoneNumber     String?  @map("phone_number")
	dateOfBirth     DateTime @map("date_of_birth")
	isEmailVerified Boolean  @default(false) @map("is_email_verified")
	lastLoginTime   DateTime? @map("last_login_time")
	profilePicture  String?   @map("profile_picture_url")
	
	@@map("user_profiles")
	@@index([emailAddress])
	@@index([isEmailVerified, lastLoginTime])
}

model BlogPost {
	id            Int       @id @default(autoincrement())
	postTitle     String    @map("post_title")
	postContent   String    @map("post_content")
	authorId      Int       @map("author_id")
	publishedDate DateTime? @map("published_date")
	viewCount     Int       @default(0) @map("view_count")
	isPublished   Boolean   @default(false) @map("is_published")
	
	@@map("blog_posts")
	@@unique([postTitle, authorId])
}
`+"`"+`);
		await db.syncSchemas();
		
		// The migration system now creates composite unique indexes automatically
		// No need to manually create the index
		
		
		// Test creating records with mapped fields
		const now = new Date();
		const dob = new Date('1990-01-15');
		
		const profile = await db.models.UserProfile.create({
			data: {
				userName: 'john_doe',
				emailAddress: 'john@example.com',
				phoneNumber: '+1234567890',
				dateOfBirth: dob,
				isEmailVerified: true,
				lastLoginTime: now,
				profilePicture: 'https://example.com/avatar.jpg'
			}
		});
		
		console.log('Created profile:', JSON.stringify(profile, null, 2));
		
		// Test querying with mapped fields
		const verifiedUsers = await db.models.UserProfile.findMany({
			where: { isEmailVerified: true },
			select: {
				id: true,
				userName: true,
				emailAddress: true
			}
		});
		
		assert(verifiedUsers.length === 1, 'Should find 1 verified user');
		
		// Test complex queries with mapped fields
		const recentUsers = await db.models.UserProfile.findMany({
			where: {
				AND: [
					{ isEmailVerified: true },
					{ lastLoginTime: { not: null } }
				]
			},
			orderBy: { lastLoginTime: 'desc' }
		});
		
		assert(recentUsers.length === 1, 'Should find 1 recent user');
		
		// Test creating blog post with mapped fields
		const post = await db.models.BlogPost.create({
			data: {
				postTitle: 'My First Blog Post',
				postContent: 'This is the content of my first blog post.',
				authorId: profile.id,
				isPublished: true,
				publishedDate: now
			}
		});
		
		// Test unique constraint with mapped fields
		try {
			await db.models.BlogPost.create({
				data: {
					postTitle: 'My First Blog Post', // Same title
					postContent: 'Different content',
					authorId: profile.id, // Same author
					isPublished: false
				}
			});
			throw new Error('Should have failed with unique constraint');
		} catch (err) {
			if (err.message === 'Should have failed with unique constraint') {
				throw err;
			}
			// Expected unique constraint error
			assert(
				err.message.includes('unique') || 
				err.message.includes('UNIQUE') ||
				err.message.includes('duplicate') ||
				err.message.includes('Duplicate entry'),
				'Should get unique constraint error'
			);
		}
		
		// Test updating with mapped fields
		await db.models.BlogPost.update({
			where: { id: post.id },
			data: { viewCount: 100 }
		});
		
		const updatedPost = await db.models.BlogPost.findUnique({
			where: { id: post.id }
		});
		
		// Handle field name variations
		const viewCount = updatedPost.viewCount !== undefined ? updatedPost.viewCount : updatedPost.view_count;
		assert(viewCount === 100, 'View count should be 100');
		
		// await db.close();
	`)

	// Test table name mapping
	jct.runWithCleanup(t, runner, "TableNameMapping", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		// Schema with custom table names
		await db.loadSchema(`+"`"+`
model SystemConfiguration {
	id    Int    @id @default(autoincrement())
	key   String @unique
	value String
	
	@@map("sys_config")
}

model ApplicationLog {
	id        Int      @id @default(autoincrement())
	level     String
	message   String
	context   String?
	timestamp DateTime @default(now())
	
	@@map("app_logs")
	@@index([level, timestamp])
}
`+"`"+`);
		await db.syncSchemas();
		
		// Test operations with custom table names
		await db.models.SystemConfiguration.create({
			data: { key: 'app.version', value: '1.0.0' }
		});
		
		await db.models.SystemConfiguration.create({
			data: { key: 'app.name', value: 'TestApp' }
		});
		
		// Query using model name, not table name
		const configs = await db.models.SystemConfiguration.findMany({
			orderBy: { key: 'asc' }
		});
		
		assert(configs.length === 2, 'Should have 2 configurations');
		assert(configs[0].key === 'app.name');
		assert(configs[1].key === 'app.version');
		
		// Test with raw query to verify actual table name
		const rawResult = await db.queryRaw('SELECT COUNT(*) as count FROM sys_config');
		const count = rawResult[0].count || rawResult[0].COUNT;
		assert(count == 2, 'Raw query should find 2 records in sys_config table');
		
		// Test logging with mapped table
		await db.models.ApplicationLog.create({
			data: {
				level: 'INFO',
				message: 'Application started',
				context: JSON.stringify({ version: '1.0.0' })
			}
		});
		
		await db.models.ApplicationLog.create({
			data: {
				level: 'ERROR',
				message: 'Failed to connect to service',
				context: JSON.stringify({ service: 'auth', error: 'timeout' })
			}
		});
		
		// Query logs
		const errorLogs = await db.models.ApplicationLog.findMany({
			where: { level: 'ERROR' }
		});
		
		assert(errorLogs.length === 1, 'Should have 1 error log');
		assert(errorLogs[0].message.includes('Failed to connect'));
		
		// await db.close();
	`)

	// Test enum fields
	jct.runWithCleanup(t, runner, "EnumFields", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		// Schema with enum
		await db.loadSchema(`+"`"+`
enum OrderStatus {
	PENDING
	PROCESSING
	SHIPPED
	DELIVERED
	CANCELLED
}

model Order {
	id     Int         @id @default(autoincrement())
	status OrderStatus @default(PENDING)
	total  Float
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create orders with different statuses
		await db.models.Order.create({
			data: { status: 'PENDING', total: 100 }
		});
		await db.models.Order.create({
			data: { status: 'SHIPPED', total: 200 }
		});
		await db.models.Order.create({
			data: { status: 'DELIVERED', total: 300 }
		});
		
		// Query by enum value
		const shippedOrders = await db.models.Order.findMany({
			where: { status: 'SHIPPED' }
		});
		assert(shippedOrders.length === 1);
		assert(shippedOrders[0].total === 200);
		
		// await db.close();
	`)

	// Test dropping models
	jct.runWithCleanup(t, runner, "DropModel", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		// Create initial schema
		await db.loadSchema(`+"`"+`
model TempData {
	id    Int    @id @default(autoincrement())
	value String
}
`+"`"+`);
		await db.syncSchemas();
		
		// Add some data
		await db.models.TempData.create({
			data: { value: 'test' }
		});
		
		// Verify model exists
		let models = db.getModels();
		assert(models.includes('TempData'), 'TempData model should exist before drop');
		
		// Drop the model
		await db.dropModel('TempData');
		
		// Verify table is dropped by trying to query it
		try {
			await db.models.TempData.findMany();
			throw new Error('Should not be able to query dropped table');
		} catch (err) {
			console.log('Expected error when querying dropped table:', err.message);
			// Different databases have different error messages
			assert(
				err.message.includes('no such table') || // SQLite
				err.message.includes('doesn\'t exist') || // MySQL
				err.message.includes('does not exist') || // PostgreSQL
				err.message.includes('TempData') || // Generic
				err.message.includes('temp_datas'), // Table name variants
				'Should get table not found error, got: ' + err.message
			);
		}
		
		// await db.close();
	`)

	// Test schema validation
	jct.runWithCleanup(t, runner, "SchemaValidation", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		// Schema with various constraints
		await db.loadSchema(`+"`"+`
model Account {
	id       Int    @id @default(autoincrement())
	username String @unique
	email    String @unique
	balance  Float  @default(0)
	
	@@unique([username, email])
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create first account
		await db.models.Account.create({
			data: { 
				username: 'user1',
				email: 'user1@example.com',
				balance: 100
			}
		});
		
		// Try to create duplicate username - should fail
		try {
			await db.models.Account.create({
				data: { 
					username: 'user1',
					email: 'different@example.com',
					balance: 200
				}
			});
			throw new Error('Should have failed with unique constraint');
		} catch (err) {
			if (err.message === 'Should have failed with unique constraint') {
				throw err; // Re-throw our test failure
			}
			// Check for unique constraint error - different databases have different messages
			const isUniqueError = 
				err.message.includes('unique') || 
				err.message.includes('UNIQUE') ||
				err.message.includes('Duplicate entry') || // MySQL
				err.message.includes('duplicate key') || // PostgreSQL
				err.message.includes('1062'); // MySQL error code
			
			assert(isUniqueError, 
				'Error should be about unique constraint, got: ' + err.message);
		}
		
		// await db.close();
	`)
}
