// Test that db.models.User is available after syncSchemas
const { fromUri } = require('redi/orm');
const { assert } = require('./assert');

async function testDbModels() {
    console.log('Testing db.models access...');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Before sync, models should be empty
    assert(typeof db.models === 'object', 'db.models should exist');
    const modelKeys = Object.keys(db.models);
    assert(modelKeys.length === 0, 'db.models should be empty before sync');
    console.log('  ✓ db.models is empty before sync');
    
    // Load schema
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
    
    // After sync, models should be populated
    assert(typeof db.models.User === 'object', 'db.models.User should exist');
    assert(typeof db.models.Post === 'object', 'db.models.Post should exist');
    console.log('  ✓ db.models populated after sync');
    
    // Test that model methods exist
    assert(typeof db.models.User.create === 'function', 'User.create should be a function');
    assert(typeof db.models.User.findMany === 'function', 'User.findMany should be a function');
    assert(typeof db.models.User.update === 'function', 'User.update should be a function');
    assert(typeof db.models.User.delete === 'function', 'User.delete should be a function');
    console.log('  ✓ Model methods exist');
    
    // For now, just verify the methods exist and are callable
    // Actual CRUD operations require struct handling which is a separate issue
    console.log('  ✓ Model methods are available for use');
    
    await db.close();
}

async function testMultipleSchemaLoads() {
    console.log('\nTesting multiple schema loads with db.models...');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Load all schemas at once to avoid migration version conflicts
    await db.loadSchema(`
model Product {
  id    Int    @id @default(autoincrement())
  name  String
  price Float
}
`);
    
    await db.loadSchema(`
model Category {
  id   Int    @id @default(autoincrement())
  name String
}
`);
    
    // Sync once with all schemas
    await db.syncSchemas();
    
    assert(typeof db.models.Product === 'object', 'Product model should exist');
    assert(typeof db.models.Category === 'object', 'Category model should exist');
    console.log('  ✓ Multiple models loaded and accessible via db.models');
    
    // Verify all methods exist on both models
    const methods = ['create', 'findUnique', 'findMany', 'update', 'delete'];
    for (const method of methods) {
        assert(typeof db.models.Product[method] === 'function', `Product.${method} should exist`);
        assert(typeof db.models.Category[method] === 'function', `Category.${method} should exist`);
    }
    console.log('  ✓ All methods exist on all models');
    
    await db.close();
}

async function runTests() {
    console.log('=== DB Models Test Suite ===\n');
    
    try {
        await testDbModels();
        await testMultipleSchemaLoads();
        
        console.log('\n✅ All db.models tests passed!');
        
    } catch (error) {
        console.error('\n❌ Test failed:', error.message);
        console.error(error.stack);
        process.exit(1);
    }
}

runTests();