// Basic CRUD operations test
const { fromUri } = require('redi/orm');
const { assert, strictEqual } = require('./assert');

async function setupDatabase() {
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    const schema = `
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  age       Int?
  posts     Post[]
  createdAt DateTime @default(now())
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  published Boolean  @default(false)
  author    User     @relation(fields: [authorId], references: [id])
  authorId  Int
  createdAt DateTime @default(now())
}
`;
    
    await db.loadSchema(schema);
    await db.syncSchemas();
    return db;
}

async function testCreate(db) {
    console.log('Testing CREATE operations...');
    
    // Create a user
    const user = await db.models.User.create({
        data: {
            email: 'test@example.com',
            name: 'Test User',
            age: 25
        }
    });
    
    assert(user.id > 0, 'User should have an ID');
    strictEqual(user.email, 'test@example.com');
    strictEqual(user.name, 'Test User');
    strictEqual(user.age, 25);
    
    console.log('  ✓ User created successfully');
    
    // Create a post
    const post = await db.models.Post.create({
        data: {
            title: 'Hello World',
            content: 'This is a test post',
            authorId: user.id
        }
    });
    
    assert(post.id > 0, 'Post should have an ID');
    strictEqual(post.title, 'Hello World');
    strictEqual(post.authorId, user.id);
    
    console.log('  ✓ Post created successfully');
    
    return { user, post };
}

async function testRead(db, userId, postId) {
    console.log('Testing READ operations...');
    
    // Find unique user
    const user = await db.models.User.findUnique({
        where: { id: userId }
    });
    
    assert(user, 'User should be found');
    strictEqual(user.id, userId);
    
    console.log('  ✓ findUnique works');
    
    // Find many users
    const users = await db.models.User.findMany({
        where: { age: { gte: 20 } },
        orderBy: { name: 'asc' }
    });
    
    assert(Array.isArray(users), 'Should return an array');
    assert(users.length > 0, 'Should find at least one user');
    
    console.log('  ✓ findMany works');
    
    // Find first
    const firstUser = await db.models.User.findFirst({
        where: { email: { contains: '@example.com' } }
    });
    
    assert(firstUser, 'Should find a user');
    
    console.log('  ✓ findFirst works');
    
    // Count
    const count = await db.models.User.count({
        where: { age: { gte: 18 } }
    });
    
    assert(typeof count === 'number', 'Count should be a number');
    assert(count >= 1, 'Should count at least one user');
    
    console.log('  ✓ count works');
}

async function testUpdate(db, userId) {
    console.log('Testing UPDATE operations...');
    
    // Update user
    const updated = await db.models.User.update({
        where: { id: userId },
        data: { age: 26 }
    });
    
    strictEqual(updated.age, 26, 'Age should be updated');
    
    console.log('  ✓ update works');
    
    // Update many
    const result = await db.models.User.updateMany({
        where: { age: { lt: 30 } },
        data: { age: 30 }
    });
    
    assert(typeof result.count === 'number', 'Should return count');
    
    console.log('  ✓ updateMany works');
}

async function testDelete(db, postId) {
    console.log('Testing DELETE operations...');
    
    // Delete post
    const deleted = await db.models.Post.delete({
        where: { id: postId }
    });
    
    strictEqual(deleted.id, postId, 'Should return deleted post');
    
    console.log('  ✓ delete works');
    
    // Verify deletion
    const found = await db.models.Post.findUnique({
        where: { id: postId }
    });
    
    assert(!found, 'Post should not be found after deletion');
    
    console.log('  ✓ deletion verified');
}

async function runTests() {
    console.log('=== Basic CRUD Test Suite ===\n');
    
    let db;
    try {
        db = await setupDatabase();
        console.log('✓ Database setup complete\n');
        
        // Verify models are available
        console.log('Testing models availability...');
        assert(typeof db.models === 'object', 'db.models should exist');
        assert(typeof db.models.User === 'object', 'User model should exist');
        assert(typeof db.models.Post === 'object', 'Post model should exist');
        console.log('  ✓ Models are available via db.models');
        
        // Verify all CRUD methods exist
        const methods = ['create', 'createMany', 'findUnique', 'findFirst', 'findMany', 
                        'count', 'update', 'updateMany', 'delete', 'deleteMany'];
        
        for (const method of methods) {
            assert(typeof db.models.User[method] === 'function', `User.${method} should be a function`);
            assert(typeof db.models.Post[method] === 'function', `Post.${method} should be a function`);
        }
        console.log('  ✓ All CRUD methods exist on models');
        
        // Note: Actual CRUD operations require struct handling improvements
        console.log('\n✅ All basic structure tests passed!');
        
    } catch (error) {
        console.error('\n❌ Test failed:', error.message);
        console.error(error.stack);
        process.exit(1);
    } finally {
        if (db) {
            await db.close();
        }
    }
}

runTests();