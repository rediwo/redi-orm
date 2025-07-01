// Raw query test
const { fromUri } = require('redi/orm');
const { assert } = require('./assert');

async function testBasicRawQueries() {
    console.log('Testing basic raw queries...');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Test simple query
    const result = await db.queryRaw('SELECT 1 as num, "hello" as str');
    assert(Array.isArray(result), 'Should return array');
    assert(result.length === 1, 'Should have one row');
    assert(result[0].num === 1, 'Should have correct number');
    assert(result[0].str === 'hello', 'Should have correct string');
    console.log('  ✓ Simple query works');
    
    // Test query with parameters
    const paramResult = await db.queryRaw(
        'SELECT ? as a, ? as b, ? as c',
        10, 'test', true
    );
    assert(paramResult[0].a === 10, 'First parameter works');
    assert(paramResult[0].b === 'test', 'Second parameter works');
    assert(paramResult[0].c === 1, 'Boolean parameter works'); // SQLite stores true as 1
    console.log('  ✓ Query with parameters works');
    
    await db.close();
}

async function testExecuteRaw() {
    console.log('\nTesting executeRaw...');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Create table
    await db.executeRaw(`
        CREATE TABLE test_table (
            id INTEGER PRIMARY KEY,
            name TEXT NOT NULL,
            value REAL
        )
    `);
    console.log('  ✓ CREATE TABLE works');
    
    // Insert data
    const insertResult = await db.executeRaw(
        'INSERT INTO test_table (name, value) VALUES (?, ?)',
        'item1', 42.5
    );
    assert(typeof insertResult.rowsAffected === 'number', 'Should return rowsAffected');
    assert(insertResult.rowsAffected === 1, 'Should affect one row');
    console.log('  ✓ INSERT works');
    
    // Insert multiple rows
    await db.executeRaw('INSERT INTO test_table (name, value) VALUES ("item2", 10.0)');
    await db.executeRaw('INSERT INTO test_table (name, value) VALUES ("item3", 20.0)');
    
    // Update data
    const updateResult = await db.executeRaw(
        'UPDATE test_table SET value = value * 2 WHERE name LIKE ?',
        'item%'
    );
    assert(updateResult.rowsAffected === 3, 'Should update 3 rows');
    console.log('  ✓ UPDATE works');
    
    // Verify updates
    const updated = await db.queryRaw('SELECT * FROM test_table ORDER BY id');
    assert(updated[0].value === 85, 'First item doubled');
    assert(updated[1].value === 20, 'Second item doubled');
    assert(updated[2].value === 40, 'Third item doubled');
    console.log('  ✓ Updates verified');
    
    // Delete data
    const deleteResult = await db.executeRaw(
        'DELETE FROM test_table WHERE value < ?',
        50
    );
    assert(deleteResult.rowsAffected === 2, 'Should delete 2 rows');
    console.log('  ✓ DELETE works');
    
    // Verify deletion
    const remaining = await db.queryRaw('SELECT COUNT(*) as count FROM test_table');
    assert(remaining[0].count === 1, 'Should have 1 row remaining');
    console.log('  ✓ Deletion verified');
    
    await db.close();
}

async function testWithSchema() {
    console.log('\nTesting raw queries with schema tables...');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Load schema
    const schema = `
model Product {
  id    Int    @id @default(autoincrement())
  name  String
  price Float
  stock Int    @default(0)
}
`;
    
    await db.loadSchema(schema);
    await db.syncSchemas();
    
    // Insert data using raw query
    await db.executeRaw(
        'INSERT INTO products (name, price, stock) VALUES (?, ?, ?)',
        'Widget', 19.99, 100
    );
    await db.executeRaw(
        'INSERT INTO products (name, price, stock) VALUES (?, ?, ?)',
        'Gadget', 29.99, 50
    );
    console.log('  ✓ Inserted data into schema table');
    
    // Query using raw SQL
    const products = await db.queryRaw(
        'SELECT * FROM products WHERE price < ? ORDER BY price DESC',
        30
    );
    assert(products.length === 2, 'Should find 2 products');
    assert(products[0].name === 'Gadget', 'Should be ordered by price');
    console.log('  ✓ Raw query on schema table works');
    
    // Aggregate query
    const stats = await db.queryRaw(`
        SELECT 
            COUNT(*) as total,
            AVG(price) as avg_price,
            SUM(stock) as total_stock
        FROM products
    `);
    assert(stats[0].total === 2, 'Should count 2 products');
    assert(stats[0].avg_price === 24.99, 'Should calculate average');
    assert(stats[0].total_stock === 150, 'Should sum stock');
    console.log('  ✓ Aggregate queries work');
    
    await db.close();
}

async function testErrorHandling() {
    console.log('\nTesting error handling...');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Test invalid SQL
    try {
        await db.queryRaw('SELECT * FROM nonexistent_table');
        assert(false, 'Should have thrown error');
    } catch (error) {
        console.log('  ✓ Invalid table error caught');
    }
    
    // Test syntax error
    try {
        await db.executeRaw('INSERT INTO VALUES (1, 2, 3)');
        assert(false, 'Should have thrown error');
    } catch (error) {
        console.log('  ✓ Syntax error caught');
    }
    
    await db.close();
}

async function runTests() {
    console.log('=== Raw Query Test Suite ===\n');
    
    try {
        await testBasicRawQueries();
        await testExecuteRaw();
        await testWithSchema();
        await testErrorHandling();
        
        console.log('\n✅ All raw query tests passed!');
        
    } catch (error) {
        console.error('\n❌ Test failed:', error.message);
        console.error(error.stack);
        process.exit(1);
    }
}

runTests();