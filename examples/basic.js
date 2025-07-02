// Example: Basic ORM usage with RediORM
const { fromUri } = require('redi/orm');

async function main() {
    // Connect to database
    const db = fromUri('sqlite://./example.db');
    await db.connect();
    
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
    
    // Auto-migrate the database
    await db.syncSchemas();
    
    // Check if user already exists
    let user;
    try {
        user = await db.models.User.findFirst({
            where: { email: 'alice@example.com' }
        });
        console.log('User already exists:', user);
    } catch (err) {
        // User doesn't exist, create one
        user = await db.models.User.create({
            data: {
                email: 'alice@example.com',
                name: 'Alice'
            }
        });
        console.log('Created user:', JSON.stringify(user));
    }
    
    // Create a post
    const post = await db.models.Post.create({
        data: {
            title: 'Hello World',
            content: 'This is my first post',
            authorId: user.id
        }
    });
    console.log('Created post:', JSON.stringify(post));
    
    // Query user
    const userWithPosts = await db.models.User.findUnique({
        where: { email: 'alice@example.com' }
    });
    console.log('User:', JSON.stringify(userWithPosts, null, 2));
    
    // Update post
    const updatedPost = await db.models.Post.update({
        where: { id: post.id },
        data: { published: true }
    });
    console.log('Updated post:', JSON.stringify(updatedPost));
    
    // Count posts
    const postCount = await db.models.Post.count({
        where: { published: true }
    });
    console.log('Published posts:', postCount);
    
    // Close connection
    await db.close();
    console.log('Done!');
}

// Run the example
main().catch(err => {
    console.error('Error:', err.message || err);
    if (err.stack) {
        console.error('Stack:', err.stack);
    }
    process.exit(1);
});