const { fromUri } = require('redi/orm');

async function test() {
    console.log('Testing groupBy feature...\n');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Simple schema
    await db.loadSchema(`
        model User {
            id    Int     @id @default(autoincrement())
            name  String
            age   Int
            city  String
        }
    `);
    
    await db.syncSchemas();
    console.log('Schema synced');
    
    // Create test data
    await db.models.User.createMany({
        data: [
            { name: 'Alice', age: 25, city: 'NYC' },
            { name: 'Bob', age: 30, city: 'NYC' },
            { name: 'Charlie', age: 25, city: 'LA' },
            { name: 'David', age: 30, city: 'LA' },
            { name: 'Eve', age: 35, city: 'NYC' }
        ]
    });
    console.log('Test data created');
    
    // Test simple groupBy with count
    console.log('\nTest 1: Group by age with count');
    try {
        const result = await db.models.User.groupBy({
            by: ['age'],
            _count: true,
            orderBy: { age: 'asc' }
        });
        console.log('Result:', JSON.stringify(result, null, 2));
    } catch (err) {
        console.error('Error:', err.message);
        console.error('Stack:', err.stack);
    }
    
    // Test groupBy with multiple fields
    console.log('\nTest 2: Group by city with count');
    try {
        const result = await db.models.User.groupBy({
            by: ['city'],
            _count: true,
            orderBy: { city: 'asc' }
        });
        console.log('Result:', JSON.stringify(result, null, 2));
    } catch (err) {
        console.error('Error:', err.message);
        console.error('Stack:', err.stack);
    }
    
    // Test groupBy with multiple aggregations
    console.log('\nTest 3: Group by city with multiple aggregations');
    try {
        const result = await db.models.User.groupBy({
            by: ['city'],
            _count: true,
            _avg: { age: true },
            _min: { age: true },
            _max: { age: true }
        });
        console.log('Result:', JSON.stringify(result, null, 2));
    } catch (err) {
        console.error('Error:', err.message);
        console.error('Stack:', err.stack);
    }
    
    await db.close();
}

test().catch(err => {
    console.error('Error:', err);
    process.exit(1);
});