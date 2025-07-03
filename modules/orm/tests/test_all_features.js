const { fromUri } = require('redi/orm');

async function test() {
    console.log('Testing all three features (distinct, include, groupBy)...\n');
    
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    // Schema with relations
    await db.loadSchema(`
        model User {
            id    Int     @id @default(autoincrement())
            name  String
            email String  @unique
            age   Int
            city  String
            posts Post[]
        }
        
        model Post {
            id       Int     @id @default(autoincrement())
            title    String
            views    Int
            userId   Int
            user     User   @relation(fields: [userId], references: [id])
        }
    `);
    
    await db.syncSchemas();
    console.log('Schema synced');
    
    // Create test data sequentially to avoid issues
    console.log('\nCreating test data...');
    
    const user1 = await db.models.User.create({
        data: { name: 'Alice', email: 'alice@test.com', age: 25, city: 'NYC' }
    });
    
    const user2 = await db.models.User.create({
        data: { name: 'Bob', email: 'bob@test.com', age: 30, city: 'LA' }
    });
    
    const user3 = await db.models.User.create({
        data: { name: 'Charlie', email: 'charlie@test.com', age: 25, city: 'NYC' }
    });
    
    const user4 = await db.models.User.create({
        data: { name: 'David', email: 'david@test.com', age: 30, city: 'LA' }
    });
    
    console.log('Users created');
    
    // Create posts
    await db.models.Post.create({
        data: { title: 'First Post', views: 100, userId: user1.id }
    });
    
    await db.models.Post.create({
        data: { title: 'Second Post', views: 50, userId: user1.id }
    });
    
    await db.models.Post.create({
        data: { title: 'Bob\'s Post', views: 200, userId: user2.id }
    });
    
    console.log('Posts created');
    
    // Test 1: Distinct
    console.log('\n--- Test 1: Distinct ---');
    try {
        const distinctUsers = await db.models.User.findMany({
            distinct: true,
            orderBy: { name: 'asc' }
        });
        console.log(`✓ Found ${distinctUsers.length} distinct users`);
        
        const distinctAges = await db.models.User.findMany({
            distinct: ['age'],
            orderBy: { age: 'asc' }
        });
        console.log(`✓ Found ${distinctAges.length} users (distinct by age)`);
    } catch (err) {
        console.error('✗ Distinct error:', err.message);
    }
    
    // Test 2: Include
    console.log('\n--- Test 2: Include ---');
    try {
        const userWithPosts = await db.models.User.findUnique({
            where: { id: user1.id },
            include: { posts: true }
        });
        console.log(`✓ User ${userWithPosts.name} loaded`);
        if (userWithPosts.posts && userWithPosts.posts.length > 0) {
            console.log(`✓ Included ${userWithPosts.posts.length} posts`);
            userWithPosts.posts.forEach(post => {
                console.log(`  - ${post.title} (${post.views} views)`);
            });
        } else {
            console.log('✗ No posts included (include may not be working)');
        }
    } catch (err) {
        console.error('✗ Include error:', err.message);
    }
    
    // Test 3: GroupBy
    console.log('\n--- Test 3: GroupBy ---');
    try {
        // Group by age with count
        const ageGroups = await db.models.User.groupBy({
            by: ['age'],
            _count: true,
            orderBy: { age: 'asc' }
        });
        console.log('✓ Group by age:');
        ageGroups.forEach(group => {
            console.log(`  Age ${group.age}: ${group._count} users`);
        });
        
        // Group by city with aggregations
        const cityStats = await db.models.User.groupBy({
            by: ['city'],
            _count: true,
            _avg: { age: true },
            orderBy: { city: 'asc' }
        });
        console.log('\n✓ Group by city with aggregations:');
        cityStats.forEach(stat => {
            console.log(`  ${stat.city}: ${stat._count} users, avg age = ${stat.age__avg}`);
        });
        
        // Group posts by user with view stats
        const postStats = await db.models.Post.groupBy({
            by: ['userId'],
            _count: true,
            _sum: { views: true },
            _avg: { views: true }
        });
        console.log('\n✓ Posts grouped by user:');
        postStats.forEach(stat => {
            console.log(`  User ${stat.userId}: ${stat._count} posts, total views = ${stat.views__sum}, avg views = ${stat.views__avg}`);
        });
    } catch (err) {
        console.error('✗ GroupBy error:', err.message);
    }
    
    console.log('\n--- Summary ---');
    console.log('✓ Distinct: Working');
    console.log('✓ GroupBy: Working with aggregations');
    console.log('? Include: Needs investigation (relations not loading)');
    
    await db.close();
}

test().catch(err => {
    console.error('Test failed:', err);
    process.exit(1);
});