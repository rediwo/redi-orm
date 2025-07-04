const { fromUri } = require('redi/orm');

console.log('=== Nested Includes Detailed Test ===\n');

async function runTest() {
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Setup schema
    await db.loadSchema(`
        model User {
            id        Int      @id @default(autoincrement())
            name      String
            posts     Post[]
            comments  Comment[]
        }
        
        model Post {
            id        Int      @id @default(autoincrement())
            title     String
            userId    Int
            user      User     @relation(fields: [userId], references: [id])
            comments  Comment[]
        }
        
        model Comment {
            id       Int      @id @default(autoincrement())
            content  String
            postId   Int
            post     Post     @relation(fields: [postId], references: [id])
            authorId Int
            author   User     @relation(fields: [authorId], references: [id])
        }
    `);
    
    await db.syncSchemas();
    
    // Create test data
    console.log('Creating test data...');
    
    // Create users
    const alice = await db.models.User.create({
        data: { name: 'Alice' }
    });
    const bob = await db.models.User.create({
        data: { name: 'Bob' }
    });
    
    // Create posts
    const post1 = await db.models.Post.create({
        data: { title: 'Alice Post 1', userId: alice.id }
    });
    const post2 = await db.models.Post.create({
        data: { title: 'Alice Post 2', userId: alice.id }
    });
    
    // Create comments
    await db.models.Comment.create({
        data: { content: 'Bob comment on post 1', postId: post1.id, authorId: bob.id }
    });
    await db.models.Comment.create({
        data: { content: 'Alice comment on post 1', postId: post1.id, authorId: alice.id }
    });
    
    console.log('  ✓ Test data created\n');
    
    // Test 1: Simple include
    console.log('Test 1: Simple include');
    const usersWithPosts = await db.models.User.findMany({
        include: { posts: true }
    });
    
    console.log('  Results:');
    for (const user of usersWithPosts) {
        console.log(`    ${user.name}: ${user.posts ? user.posts.length : 0} posts`);
        if (user.posts) {
            for (const post of user.posts) {
                console.log(`      - ${post.title}`);
            }
        }
    }
    console.log();
    
    // Test 2: Nested include - User -> Posts -> Comments
    console.log('Test 2: Nested include (User -> Posts -> Comments)');
    try {
        const usersWithNestedData = await db.models.User.findMany({
            where: { name: 'Alice' },
            include: {
                posts: {
                    include: {
                        comments: true
                    }
                }
            }
        });
        
        console.log('  Results:');
        for (const user of usersWithNestedData) {
            console.log(`    ${user.name}:`);
            if (user.posts) {
                for (const post of user.posts) {
                    console.log(`      Post: ${post.title}`);
                    if (post.comments) {
                        console.log(`        Comments: ${post.comments.length}`);
                        for (const comment of post.comments) {
                            console.log(`          - ${comment.content}`);
                        }
                    }
                }
            }
        }
    } catch (error) {
        console.log('  Error:', error.message);
        console.log('  Note: Deep nested includes are not yet fully implemented');
    }
    console.log();
    
    // Test 3: Multiple includes at same level
    console.log('Test 3: Multiple includes at same level');
    const usersWithMultiple = await db.models.User.findMany({
        where: { name: 'Alice' },
        include: {
            posts: true,
            comments: true
        }
    });
    
    console.log('  Results:');
    for (const user of usersWithMultiple) {
        console.log(`    ${user.name}:`);
        console.log(`      Posts: ${user.posts ? user.posts.length : 0}`);
        console.log(`      Comments: ${user.comments ? user.comments.length : 0}`);
    }
    console.log();
    
    // Test 4: FindUnique with include
    console.log('Test 4: FindUnique with include');
    const userUnique = await db.models.User.findUnique({
        where: { id: alice.id },
        include: { posts: true }
    });
    
    console.log('  Result:');
    console.log(`    ${userUnique.name}: ${userUnique.posts ? userUnique.posts.length : 0} posts`);
    
    await db.close();
    console.log('\n✓ All tests completed');
}

runTest().catch(console.error);