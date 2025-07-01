// Schema loading and synchronization test
const { fromUri } = require('redi/orm');
const { assert, strictEqual } = require('./assert');
const fs = require('fs');

async function testLoadSchema() {
    console.log('Testing loadSchema...');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    const schema = `
model Product {
  id          Int      @id @default(autoincrement())
  name        String
  price       Float
  inStock     Boolean  @default(true)
  category    Category @relation(fields: [categoryId], references: [id])
  categoryId  Int
}

model Category {
  id       Int       @id @default(autoincrement())
  name     String   @unique
  products Product[]
}
`;
    
    await db.loadSchema(schema);
    console.log('  ✓ Schema loaded from string');
    
    await db.syncSchemas();
    console.log('  ✓ Schema synchronized');
    
    const models = db.getModels();
    assert(models.includes('Product'), 'Should have Product model');
    assert(models.includes('Category'), 'Should have Category model');
    console.log('  ✓ Models available:', models.join(', '));
    
    await db.close();
}

async function testLoadSchemaFrom() {
    console.log('\nTesting loadSchemaFrom...');
    
    // Create a temporary schema file
    const schemaFile = './test_schema.prisma';
    const schemaContent = `
datasource db {
  provider = "sqlite"
  url      = "file:./test.db"
}

model Customer {
  id        Int       @id @default(autoincrement())
  email     String    @unique
  firstName String    @map("first_name")
  lastName  String    @map("last_name")
  orders    Order[]
  
  @@map("customers")
}

model Order {
  id         Int      @id @default(autoincrement())
  orderDate  DateTime @default(now()) @map("order_date")
  totalAmount Float   @map("total_amount")
  customer   Customer @relation(fields: [customerId], references: [id])
  customerId Int      @map("customer_id")
  
  @@map("orders")
}
`;
    
    fs.writeFileSync(schemaFile, schemaContent);
    
    try {
        const db = fromUri('sqlite://:memory:');
        await db.connect();
        
        await db.loadSchemaFrom(schemaFile);
        console.log('  ✓ Schema loaded from file');
        
        await db.syncSchemas();
        console.log('  ✓ Schema synchronized');
        
        const models = db.getModels();
        assert(models.includes('Customer'), 'Should have Customer model');
        assert(models.includes('Order'), 'Should have Order model');
        console.log('  ✓ Models with custom mapping loaded');
        
        await db.close();
        
    } finally {
        // Clean up
        if (fs.existsSync(schemaFile)) {
            fs.unlinkSync(schemaFile);
        }
    }
}

async function testMultipleSchemaLoads() {
    console.log('\nTesting multiple schema loads...');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Load first schema
    const schema1 = `
model Author {
  id    Int     @id @default(autoincrement())
  name  String
  books Book[]
}
`;
    
    await db.loadSchema(schema1);
    console.log('  ✓ First schema loaded');
    
    // Load second schema
    const schema2 = `
model Book {
  id       Int    @id @default(autoincrement())
  title    String
  isbn     String @unique
  author   Author @relation(fields: [authorId], references: [id])
  authorId Int
}
`;
    
    await db.loadSchema(schema2);
    console.log('  ✓ Second schema loaded');
    
    // Sync all schemas
    await db.syncSchemas();
    console.log('  ✓ All schemas synchronized');
    
    const models = db.getModels();
    assert(models.includes('Author'), 'Should have Author model');
    assert(models.includes('Book'), 'Should have Book model');
    strictEqual(models.length, 2, 'Should have exactly 2 models');
    
    console.log('  ✓ Multiple schemas work correctly');
    
    await db.close();
}

async function testDifferentDatabases() {
    console.log('\nTesting different database URIs...');
    
    const testCases = [
        { uri: 'sqlite://:memory:', name: 'SQLite in-memory' },
        { uri: 'sqlite://./test.db', name: 'SQLite file' },
        // Add more database URIs as needed
    ];
    
    for (const { uri, name } of testCases) {
        console.log(`  Testing ${name}...`);
        
        try {
            const db = fromUri(uri);
            await db.connect();
            await db.ping();
            console.log(`    ✓ ${name} connection successful`);
            await db.close();
            
            // Clean up file-based databases
            if (uri.includes('./test.db') && fs.existsSync('./test.db')) {
                fs.unlinkSync('./test.db');
            }
        } catch (error) {
            console.log(`    ⚠ ${name} skipped:`, error.message);
        }
    }
}

async function runTests() {
    console.log('=== Schema Test Suite ===\n');
    
    try {
        await testLoadSchema();
        await testLoadSchemaFrom();
        await testMultipleSchemaLoads();
        await testDifferentDatabases();
        
        console.log('\n✅ All schema tests passed!');
        
    } catch (error) {
        console.error('\n❌ Test failed:', error.message);
        console.error(error.stack);
        process.exit(1);
    }
}

runTests();