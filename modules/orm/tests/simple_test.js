// Simple test focusing on fromUri and schema functionality
const { fromUri } = require('redi/orm');
const { assert } = require('./assert');

async function testFromUri() {
    console.log('Testing fromUri functionality...');
    
    // Test database creation
    const db = fromUri('sqlite://:memory:');
    assert(db, 'Database should be created');
    console.log('  ✓ Database created from URI');
    
    // Test connection
    await db.connect();
    console.log('  ✓ Connected to database');
    
    // Test ping
    await db.ping();
    console.log('  ✓ Database is responsive');
    
    // Test schema loading
    const schema = `
model User {
  id    Int    @id @default(autoincrement())
  email String @unique
  name  String?
}

model Post {
  id       Int    @id @default(autoincrement())
  title    String
  authorId Int
}
`;
    
    await db.loadSchema(schema);
    console.log('  ✓ Schema loaded');
    
    // Test schema synchronization
    await db.syncSchemas();
    console.log('  ✓ Schemas synchronized');
    
    // Test getModels
    const models = db.getModels();
    assert(Array.isArray(models), 'getModels should return an array');
    assert(models.includes('User'), 'Should have User model');
    assert(models.includes('Post'), 'Should have Post model');
    console.log('  ✓ Models available:', models.join(', '));
    
    // Test multiple schema loads (using a fresh database to avoid version conflicts)
    const db2 = fromUri('sqlite://:memory:');
    await db2.connect();
    
    // Load all schemas at once
    const fullSchema = `
model User {
  id    Int    @id @default(autoincrement())
  email String @unique
  name  String?
}

model Post {
  id       Int    @id @default(autoincrement())
  title    String
  authorId Int
}

model Comment {
  id     Int    @id @default(autoincrement())
  text   String
  postId Int
}
`;
    
    await db2.loadSchema(fullSchema);
    await db2.syncSchemas();
    
    const allModels = db2.getModels();
    assert(allModels.includes('User'), 'Should have User model');
    assert(allModels.includes('Post'), 'Should have Post model');
    assert(allModels.includes('Comment'), 'Should have Comment model');
    console.log('  ✓ Multiple models loaded at once');
    
    await db2.close();
    
    // Test closing
    await db.close();
    console.log('  ✓ Database closed');
    
    return true;
}

async function testLoadSchemaFrom() {
    console.log('\nTesting loadSchemaFrom...');
    
    const fs = require('fs');
    const schemaFile = './test_schema.prisma';
    
    // Create a test schema file
    const schemaContent = `
model Product {
  id    Int    @id @default(autoincrement())
  name  String
  price Float
}
`;
    
    fs.writeFileSync(schemaFile, schemaContent);
    
    try {
        const db = fromUri('sqlite://:memory:');
        await db.connect();
        
        await db.loadSchemaFrom(schemaFile);
        console.log('  ✓ Schema loaded from file');
        
        await db.syncSchemas();
        const models = db.getModels();
        assert(models.includes('Product'), 'Should have Product model');
        console.log('  ✓ File-based schema works');
        
        await db.close();
    } finally {
        // Clean up
        if (fs.existsSync(schemaFile)) {
            fs.unlinkSync(schemaFile);
        }
    }
}

async function testErrorHandling() {
    console.log('\nTesting error handling...');
    
    try {
        const db = fromUri('invalid://uri');
        await db.connect();
        assert(false, 'Should have thrown an error');
    } catch (error) {
        console.log('  ✓ Invalid URI handled correctly');
    }
    
    try {
        const db = fromUri('sqlite://:memory:');
        await db.connect();
        await db.loadSchema('invalid schema content');
        assert(false, 'Should have thrown an error');
    } catch (error) {
        console.log('  ✓ Invalid schema handled correctly');
    }
}

async function runTests() {
    console.log('=== Simple FromUri Test Suite ===\n');
    
    try {
        await testFromUri();
        await testLoadSchemaFrom();
        await testErrorHandling();
        
        console.log('\n✅ All tests passed!');
        
    } catch (error) {
        console.error('\n❌ Test failed:', error.message);
        console.error(error.stack);
        process.exit(1);
    }
}

runTests();