const { fromUri } = require('redi/orm');

async function test() {
    console.log('Testing distinct and include features...\n');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Simple schema
    await db.loadSchema(`
        model User {
            id    Int     @id @default(autoincrement())
            name  String
            email String  @unique
            posts Post[]
        }
        
        model Post {
            id       Int     @id @default(autoincrement())
            title    String
            userId   Int
            user     User   @relation(fields: [userId], references: [id])
        }
    `);
    
    await db.syncSchemas();
    console.log('Schema synced');
    
    // Create data
    const user1 = await db.models.User.create({
        data: { name: 'Alice', email: 'alice@test.com' }
    });
    console.log('User created:', JSON.stringify(user1, null, 2));
    
    const post1 = await db.models.Post.create({
        data: { title: 'First Post', userId: user1.id }
    });
    console.log('Post created:', JSON.stringify(post1, null, 2));
    
    // Test distinct
    console.log('\nTesting distinct...');
    try {
        const distinctUsers = await db.models.User.findMany({
            distinct: true
        });
        console.log('Distinct users:', JSON.stringify(distinctUsers, null, 2));
    } catch (err) {
        console.error('Distinct error:', err.message);
    }
    
    // Test include
    console.log('\nTesting include...');
    try {
        const userWithPosts = await db.models.User.findUnique({
            where: { id: user1.id },
            include: { posts: true }
        });
        console.log('User with posts:', JSON.stringify(userWithPosts, null, 2));
    } catch (err) {
        console.error('Include error:', err.message);
    }
    
    await db.close();
}

test().catch(err => {
    console.error('Error:', err);
    process.exit(1);
});