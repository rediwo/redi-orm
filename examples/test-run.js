// Test script for redi-orm run command
const { fromUri } = require('redi/orm');

async function main() {
    // Use in-memory database for testing
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    console.log('✓ Connected to in-memory database');
    
    // Load schema
    await db.loadSchema(`
        model User {
            id        Int      @id @default(autoincrement())
            email     String   @unique
            name      String?
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
    `);
    console.log('✓ Schema loaded');
    
    // Auto-migrate the database
    await db.syncSchemas();
    console.log('✓ Database synced');
    
    // Create a user
    const user = await db.models.User.create({
        data: {
            email: 'test@example.com',
            name: 'Test User'
        }
    });
    console.log('✓ Created user:', JSON.stringify(user));
    
    // Create a post
    const post = await db.models.Post.create({
        data: {
            title: 'Hello from redi-orm run!',
            content: 'This is a test post',
            authorId: user.id
        }
    });
    console.log('✓ Created post:', JSON.stringify(post));
    
    // Query all users
    const users = await db.models.User.findMany();
    console.log(`✓ Found ${users.length} user(s)`);
    
    // Query all posts
    const posts = await db.models.Post.findMany();
    console.log(`✓ Found ${posts.length} post(s)`);
    
    // Count posts
    const postCount = await db.models.Post.count();
    console.log(`✓ Total posts: ${postCount}`);
    
    // Close connection
    await db.close();
    console.log('✓ Database closed');
    
    console.log('\n🎉 All tests passed! The run command works correctly.');
}

// Run the example
main().catch(err => {
    console.error('❌ Error:', err.message || err);
    process.exit(1);
});