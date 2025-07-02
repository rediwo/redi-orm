// Simple Migration Demo
// This example shows basic migration workflow with RediORM

const { fromUri } = require('redi/orm');

async function main() {
    console.log('=== Simple Migration Demo ===\n');
    
    // Clean up any existing database from previous runs
    const fs = require('fs');
    const path = require('path');
    const dbFile = 'simple-demo.db';
    const dbPath = path.join(process.cwd(), dbFile);
    
    if (fs.existsSync(dbPath)) {
        fs.unlinkSync(dbPath);
        console.log('Cleaned up existing database\n');
    }
    
    // Use absolute path for database
    const db = fromUri(`sqlite://${dbPath}`);
    await db.connect();
    
    // Initial schema
    console.log('1. Loading initial schema...');
    await db.loadSchema(`
        model User {
            id        Int      @id @default(autoincrement())
            email     String   @unique
            name      String
            createdAt DateTime @default(now())
        }
    `);
    
    await db.syncSchemas();
    console.log('✓ Initial schema applied\n');
    
    // Add some data
    console.log('2. Adding initial data...');
    const users = await Promise.all([
        db.models.User.create({ data: { email: 'alice@example.com', name: 'Alice' } }),
        db.models.User.create({ data: { email: 'bob@example.com', name: 'Bob' } })
    ]);
    console.log(`✓ Created ${users.length} users\n`);
    
    // Evolve schema (simulating a migration)
    console.log('3. Evolving schema (adding Post model)...');
    await db.loadSchema(`
        model User {
            id        Int      @id @default(autoincrement())
            email     String   @unique
            name      String
            posts     Post[]   // New relation
            createdAt DateTime @default(now())
        }
        
        model Post {
            id        Int      @id @default(autoincrement())
            title     String
            content   String?
            authorId  Int
            author    User     @relation(fields: [authorId], references: [id])
            createdAt DateTime @default(now())
        }
    `);
    
    await db.syncSchemas();
    console.log('✓ Schema evolution applied\n');
    
    // Use new model
    console.log('4. Using new Post model...');
    const post = await db.models.Post.create({
        data: {
            title: 'Hello World',
            content: 'My first post',
            authorId: users[0].id
        }
    });
    console.log(`✓ Created post: "${post.title}"\n`);
    
    // Query with relations
    console.log('5. Querying data...');
    const userCount = await db.models.User.count();
    const postCount = await db.models.Post.count();
    console.log(`✓ Users: ${userCount}, Posts: ${postCount}\n`);
    
    // Show migration workflow notes
    console.log('=== Migration Workflow Notes ===');
    console.log('In production, you would:');
    console.log('1. Generate migration files: redi-orm migrate:generate');
    console.log('2. Review the generated SQL');
    console.log('3. Apply to staging first');
    console.log('4. Apply to production: redi-orm migrate:apply');
    console.log('5. Rollback if needed: redi-orm migrate:rollback');
    console.log('\nFor detailed examples, see:');
    console.log('- test-production-migrations.sh');
    console.log('- production-migration-workflow.js');
    console.log('- PRODUCTION_MIGRATIONS.md');
    
    await db.close();
    
    // Note: Database file will be cleaned up on next run or can be manually deleted
    console.log('\nDemo completed successfully!');
}

main().catch(err => {
    console.error('Error:', err.message || err.toString());
    if (err.stack) {
        console.error('Stack:', err.stack);
    }
    
    // Clean up on error
    const fs = require('fs');
    const path = require('path');
    const dbFile = 'simple-demo.db';
    const dbPath = path.join(process.cwd(), dbFile);
    
    if (fs.existsSync(dbPath)) {
        fs.unlinkSync(dbPath);
        console.log('Cleaned up database file after error');
    }
    
    process.exit(1);
});