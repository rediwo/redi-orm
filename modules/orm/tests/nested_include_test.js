const { fromUri } = require('redi/orm');

console.log('=== Nested Include Test ===\n');

async function runTests() {
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    await db.loadSchema(`
        model User {
            id    Int    @id @default(autoincrement())
            name  String
            posts Post[]
        }
        
        model Post {
            id     Int    @id @default(autoincrement())
            title  String
            userId Int
            user   User @relation(fields: [userId], references: [id])
        }
    `);
    
    await db.syncSchemas();
    console.log('Schema loaded');

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
    console.log('Test data created\n');

    // Test 1: Simple include
    console.log('1. Simple include:');
    try {
        const users = await db.models.User.findMany({
            include: { posts: true }
        });
        console.log(`   ✓ Found ${users.length} users`);
        console.log(`   ✓ User has ${users[0].posts ? users[0].posts.length : 0} posts`);
    } catch (err) {
        console.error('   ✗ Error:', err.message);
    }

    // Test 2: Nested include object format
    console.log('\n2. Nested include (object with true):');
    try {
        const users = await db.models.User.findMany({
            include: {
                posts: {
                    include: {
                        user: true
                    }
                }
            }
        });
        console.log(`   ✓ Found ${users.length} users`);
        if (users[0].posts && users[0].posts[0]) {
            console.log(`   ✓ Post has user: ${users[0].posts[0].user ? 'yes' : 'no'}`);
        }
    } catch (err) {
        console.error('   ✗ Error:', err.message);
        console.error('     Stack:', err.stack);
    }

    // Test 3: Check what's being passed to Include
    console.log('\n3. Debug include paths:');
    try {
        // This should generate include paths like ["posts", "posts.user"]
        const testInclude = {
            posts: {
                include: {
                    user: true
                }
            }
        };
        console.log('   Include object:', JSON.stringify(testInclude, null, 2));
        
        // Try the query
        const users = await db.models.User.findMany({
            include: testInclude
        });
        console.log(`   ✓ Query executed, found ${users.length} users`);
    } catch (err) {
        console.error('   ✗ Error:', err.message);
    }
    
    await db.close();
}

runTests().catch(console.error);