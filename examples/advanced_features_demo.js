const { fromUri } = require('redi/orm');

console.log('=== RediORM Advanced Features Demo ===\n');

async function runDemo() {
    // Create database connection
    const db = fromUri('sqlite://:memory:');
    await db.connect();

    // Define schema with relations
    await db.loadSchema(`
        model User {
            id        Int      @id @default(autoincrement())
            name      String
            email     String   @unique
            bio       String?
            posts     Post[]
            comments  Comment[]
        }
        
        model Post {
            id        Int      @id @default(autoincrement())
            title     String
            content   String
            published Boolean  @default(false)
            views     Int      @default(0)
            userId    Int
            user      User     @relation(fields: [userId], references: [id])
            comments  Comment[]
            tags      Tag[]    @relation("PostTags")
        }
        
        model Comment {
            id       Int      @id @default(autoincrement())
            content  String
            postId   Int
            post     Post     @relation(fields: [postId], references: [id])
            authorId Int
            author   User     @relation(fields: [authorId], references: [id])
        }
        
        model Tag {
            id    Int      @id @default(autoincrement())
            name  String   @unique
            posts Post[]   @relation("PostTags")
        }
    `);
    
    await db.syncSchemas();
    console.log('✓ Schema created\n');

    // 1. Transaction Batch Operations
    console.log('1. Transaction Batch Operations');
    console.log('================================');
    
    await db.transaction(async (tx) => {
        // Batch create users
        const userResult = await tx.models.User.createMany({
            data: [
                { name: 'Alice Johnson', email: 'alice@example.com', bio: 'Software Engineer' },
                { name: 'Bob Smith', email: 'bob@example.com', bio: 'Product Manager' },
                { name: 'Charlie Brown', email: 'charlie@example.com', bio: 'Designer' },
                { name: 'Diana Prince', email: 'diana@example.com', bio: 'DevOps Engineer' }
            ]
        });
        console.log(`✓ Created ${userResult.count} users`);
        
        // Batch create posts
        const postResult = await tx.models.Post.createMany({
            data: [
                { title: 'Getting Started with RediORM', content: 'RediORM is a powerful ORM...', userId: 1, published: true },
                { title: 'Advanced Query Techniques', content: 'Learn how to use complex queries...', userId: 1, published: true },
                { title: 'Draft Post', content: 'This is a draft...', userId: 2, published: false },
                { title: 'Performance Tips', content: 'Optimize your database queries...', userId: 3, published: true }
            ]
        });
        console.log(`✓ Created ${postResult.count} posts`);
        
        // Batch update - increase views for published posts
        const updateResult = await tx.models.Post.updateMany({
            where: { published: true },
            data: { views: 100 }
        });
        console.log(`✓ Updated ${updateResult.count} published posts`);
        
        // Batch delete - remove users without posts
        const deleteResult = await tx.models.User.deleteMany({
            where: { name: 'Diana Prince' }
        });
        console.log(`✓ Deleted ${deleteResult.count} users without posts`);
    });

    // 2. Complex Where Conditions
    console.log('\n2. Complex Where Conditions');
    console.log('===========================');
    
    // Using operators
    const activeUsers = await db.models.User.findMany({
        where: {
            OR: [
                { email: { contains: 'example.com' } },
                { name: { startsWith: 'A' } }
            ]
        }
    });
    console.log(`✓ Found ${activeUsers.length} users with example.com email or name starting with A`);
    
    // Multiple conditions
    const popularPosts = await db.models.Post.findMany({
        where: {
            published: true,
            views: { gte: 50 },
            title: { contains: 'RediORM' }
        }
    });
    console.log(`✓ Found ${popularPosts.length} popular posts about RediORM`);

    // 3. Eager Loading with Includes
    console.log('\n3. Eager Loading with Includes');
    console.log('==============================');
    
    // Simple include
    const usersWithPosts = await db.models.User.findMany({
        where: { name: { not: 'Diana Prince' } },
        include: { posts: true }
    });
    console.log(`✓ Loaded ${usersWithPosts.length} users with their posts`);
    usersWithPosts.forEach(user => {
        console.log(`  - ${user.name}: ${user.posts ? user.posts.length : 0} posts`);
    });
    
    // Nested includes (paths generated, data loading pending full implementation)
    console.log('\n✓ Nested include paths:');
    const nestedQuery = await db.models.User.findMany({
        include: {
            posts: {
                include: {
                    user: true,
                    comments: {
                        include: {
                            author: true
                        }
                    }
                }
            }
        }
    });
    console.log('  - users → posts → user');
    console.log('  - users → posts → comments → author');

    // 4. Nested Writes (Structure Prepared)
    console.log('\n4. Nested Writes (Structure Prepared)');
    console.log('=====================================');
    
    // The nested write structure is processed and relation fields are filtered
    // Actual nested operations would be implemented in query builders
    const newUser = await db.models.User.create({
        data: {
            name: 'Eve Wilson',
            email: 'eve@example.com',
            posts: {
                create: [
                    { title: 'My First Post', content: 'Hello World!', published: true },
                    { title: 'Draft Ideas', content: 'Some ideas...', published: false }
                ]
            }
        }
    });
    console.log(`✓ Created user: ${newUser.name} (nested posts filtered for now)`);

    // 5. Aggregations
    console.log('\n5. Aggregations');
    console.log('===============');
    
    const postCount = await db.models.Post.count({
        where: { published: true }
    });
    console.log(`✓ Published posts: ${postCount}`);
    
    const userCount = await db.models.User.count();
    console.log(`✓ Total users: ${userCount}`);

    // 6. Distinct Queries
    console.log('\n6. Distinct Queries');
    console.log('==================');
    
    const uniqueUsers = await db.models.User.findMany({
        distinct: true,
        orderBy: { name: 'asc' }
    });
    console.log(`✓ Found ${uniqueUsers.length} distinct users`);

    // 7. Ordering and Pagination
    console.log('\n7. Ordering and Pagination');
    console.log('==========================');
    
    const paginatedPosts = await db.models.Post.findMany({
        where: { published: true },
        orderBy: { views: 'desc' },
        limit: 2,
        offset: 0
    });
    console.log(`✓ Top 2 most viewed posts:`);
    paginatedPosts.forEach(post => {
        console.log(`  - "${post.title}" (${post.views} views)`);
    });

    // Close connection
    await db.close();
    console.log('\n✓ Demo completed successfully!');
}

// Run the demo
runDemo().catch(console.error);