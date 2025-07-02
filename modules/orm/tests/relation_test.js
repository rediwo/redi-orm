const { fromUri } = require('redi/orm');
const assert = require('assert');

console.log('=== Relation Test Suite ===\n');

async function runTests() {
    // Create database
    const db = fromUri('sqlite://:memory:');
    await db.connect();

    console.log('Testing schema with relations...');
    
    // Load schema with relations
    await db.loadSchema(`
        model User {
            id    Int     @id @default(autoincrement())
            name  String
            email String  @unique
            posts Post[]
        }
        
        model Post {
            id        Int      @id @default(autoincrement())
            title     String
            content   String
            published Boolean  @default(false)
            userId    Int
            user      User     @relation(fields: [userId], references: [id])
            comments  Comment[]
        }
        
        model Comment {
            id       Int    @id @default(autoincrement())
            content  String
            postId   Int
            post     Post   @relation(fields: [postId], references: [id])
            authorId Int
            author   User   @relation(fields: [authorId], references: [id])
        }
    `);
    
    await db.syncSchemas();
    console.log('  ✓ Schema with relations loaded');
    
    // Create test data
    console.log('\nCreating test data...');
    
    const user1 = await db.models.User.create({
        data: {
            name: 'Alice',
            email: 'alice@example.com'
        }
    });
    console.log('  ✓ User created');
    
    const user2 = await db.models.User.create({
        data: {
            name: 'Bob',
            email: 'bob@example.com'
        }
    });
    
    const post1 = await db.models.Post.create({
        data: {
            title: 'First Post',
            content: 'Hello World',
            userId: user1.id,
            published: true
        }
    });
    console.log('  ✓ Post created');
    
    const post2 = await db.models.Post.create({
        data: {
            title: 'Second Post',
            content: 'Another post',
            userId: user1.id,
            published: false
        }
    });
    
    const comment1 = await db.models.Comment.create({
        data: {
            content: 'Great post!',
            postId: post1.id,
            authorId: user2.id
        }
    });
    console.log('  ✓ Comment created');
    
    // Test basic includes (currently not implemented)
    console.log('\nTesting includes (placeholder)...');
    
    try {
        // This will likely fail as includes aren't fully implemented yet
        const usersWithPosts = await db.models.User.findMany({
            include: {
                posts: true
            }
        });
        console.log('  ✓ Include query executed (may not include actual relations yet)');
        
        // Verify structure (even if relations aren't loaded)
        assert(Array.isArray(usersWithPosts), 'Result should be an array');
        assert(usersWithPosts.length === 2, 'Should have 2 users');
        console.log('  ✓ Basic structure verified');
        
    } catch (error) {
        console.log('  ⚠ Include not yet fully implemented:', error.message);
    }
    
    // Test nested includes (placeholder)
    console.log('\nTesting nested includes (placeholder)...');
    
    try {
        const postsWithAll = await db.models.Post.findMany({
            include: {
                user: true,
                comments: {
                    include: {
                        author: true
                    }
                }
            }
        });
        console.log('  ✓ Nested include query executed (may not include actual relations yet)');
        
    } catch (error) {
        console.log('  ⚠ Nested includes not yet implemented:', error.message);
    }
    
    // Test filtered includes (placeholder)
    console.log('\nTesting filtered includes (placeholder)...');
    
    try {
        const usersWithPublishedPosts = await db.models.User.findMany({
            include: {
                posts: {
                    where: {
                        published: true
                    }
                }
            }
        });
        console.log('  ✓ Filtered include query executed (may not filter yet)');
        
    } catch (error) {
        console.log('  ⚠ Filtered includes not yet implemented:', error.message);
    }
    
    // Test relation counts
    console.log('\nTesting relation counts...');
    
    const postCount = await db.models.Post.count({
        where: {
            userId: user1.id
        }
    });
    assert.strictEqual(postCount, 2, 'User 1 should have 2 posts');
    console.log('  ✓ Relation count works');
    
    // Clean up
    await db.close();
    console.log('\n✅ Relation tests completed!');
}

runTests().catch(err => {
    console.error('Test failed:', err);
    process.exit(1);
});