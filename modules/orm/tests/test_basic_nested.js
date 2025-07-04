const { fromUri } = require('redi/orm');

console.log('=== Basic Nested Test ===\n');

async function runTest() {
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    await db.loadSchema(`
        model User {
            id    Int    @id @default(autoincrement())
            name  String
            posts Post[]
        }
        
        model Post {
            id      Int    @id @default(autoincrement())
            title   String
            userId  Int
            user    User @relation(fields: [userId], references: [id])
        }
    `);
    
    await db.syncSchemas();
    
    // Create test data
    const user = await db.models.User.create({
        data: { name: 'Alice' }
    });
    
    await db.models.Post.create({
        data: { title: 'Post 1', userId: user.id }
    });
    await db.models.Post.create({
        data: { title: 'Post 2', userId: user.id }
    });
    
    // Test simple include first
    console.log('Test 1: Simple include');
    try {
        const result1 = await db.models.User.findMany({
            include: { posts: true }
        });
        console.log('Result:', JSON.stringify(result1, null, 2));
    } catch (error) {
        console.error('Error:', error.message);
    }
    
    // Test nested include
    console.log('\nTest 2: Nested include');
    try {
        const result2 = await db.models.Post.findMany({
            include: {
                user: {
                    include: {
                        posts: true
                    }
                }
            }
        });
        console.log('Result:', JSON.stringify(result2, null, 2));
    } catch (error) {
        console.error('Error:', error.message);
    }
    
    await db.close();
}

runTest().catch(console.error);