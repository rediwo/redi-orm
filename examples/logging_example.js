// RediORM Logging Example
// Run with: redi-orm run logging_example.js

const { fromUri, createLogger, LogLevel } = require('redi/orm');

async function main() {
    // Create a logger (default level is INFO)
    const logger = createLogger('RediORM');
    
    // Set to DEBUG to see SQL queries (optional)
    logger.setLevel(LogLevel.DEBUG);
    
    // Create database connection
    const db = fromUri('sqlite://:memory:');
    
    // Set logger on the database instance
    db.setLogger(logger);
    
    await db.connect();
    
    // Define schema
    await db.loadSchema(`
        model User {
            id    Int     @id @default(autoincrement())
            name  String
            email String  @unique
            age   Int
            posts Post[]
        }
        
        model Post {
            id        Int     @id @default(autoincrement())
            title     String
            content   String
            published Boolean @default(false)
            authorId  Int
            author    User    @relation(fields: [authorId], references: [id])
        }
    `);
    
    // Sync schemas - logs CREATE TABLE statements
    console.log('=== Syncing schemas ===');
    await db.syncSchemas();
    
    // Insert users - logs INSERT statements
    console.log('\n=== Creating users ===');
    const user1 = await db.models.User.create({
        data: {
            name: 'John Doe',
            email: 'john@example.com',
            age: 30
        }
    });
    
    const user2 = await db.models.User.create({
        data: {
            name: 'Jane Smith',
            email: 'jane@example.com',
            age: 25
        }
    });
    
    // Create posts - logs INSERT with foreign keys
    console.log('\n=== Creating posts ===');
    await db.models.Post.create({
        data: {
            title: 'First Post',
            content: 'Hello World',
            authorId: user1.id
        }
    });
    
    // Query with joins - logs SELECT with JOIN
    console.log('\n=== Querying with relations ===');
    const users = await db.models.User.findMany({
        where: { age: { gt: 20 } },
        include: { posts: true },
        orderBy: { name: 'asc' }
    });
    
    console.log(`Found ${users.length} users`);
    
    // Update - logs UPDATE statement
    console.log('\n=== Updating user ===');
    await db.models.User.update({
        where: { id: user1.id },
        data: { age: 31 }
    });
    
    // Aggregation - logs GROUP BY query
    console.log('\n=== Aggregation query ===');
    const result = await db.models.User.aggregate({
        _count: { id: true },
        _avg: { age: true },
        _max: { age: true },
        _min: { age: true }
    });
    
    console.log('Aggregation result:', result);
    
    // Raw query - logs raw SQL
    console.log('\n=== Raw SQL query ===');
    const rawResult = await db.queryRaw('SELECT COUNT(*) as count FROM users WHERE age > ?', 24);
    console.log('Users older than 24:', rawResult[0].count);
    
    // Transaction - logs BEGIN, COMMIT
    console.log('\n=== Transaction ===');
    await db.transaction(async (tx) => {
        await tx.models.Post.create({
            data: {
                title: 'Transaction Post',
                content: 'Created in transaction',
                authorId: user2.id
            }
        });
        
        await tx.models.User.update({
            where: { id: user2.id },
            data: { age: 26 }
        });
    });
    
    console.log('\n=== Example completed! ===');
    await db.close();
}

// Run the example
main().catch(console.error);