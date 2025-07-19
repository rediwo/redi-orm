# Advanced Features

Comprehensive guide to RediORM's advanced features including relations, transactions, aggregations, and performance optimization techniques.

## Table of Contents

- [Relations](#relations)
- [Transactions](#transactions)
- [Batch Operations](#batch-operations)
- [Complex Queries](#complex-queries)
- [Eager Loading](#eager-loading)
- [Aggregations](#aggregations)
- [Raw Queries](#raw-queries)
- [Performance Optimization](#performance-optimization)

## Relations

### Defining Relations

```prisma
// One-to-Many: User has many Posts
model User {
    id    Int    @id @default(autoincrement())
    name  String
    posts Post[]
}

model Post {
    id       Int    @id @default(autoincrement())
    title    String
    authorId Int
    author   User   @relation(fields: [authorId], references: [id])
}

// Many-to-Many: Posts have many Tags
model Post {
    id   Int   @id @default(autoincrement())
    tags Tag[]
}

model Tag {
    id    Int    @id @default(autoincrement())
    name  String
    posts Post[]
}

// One-to-One: User has one Profile
model User {
    id      Int      @id @default(autoincrement())
    profile Profile?
}

model Profile {
    id     Int  @id @default(autoincrement())
    bio    String
    userId Int  @unique
    user   User @relation(fields: [userId], references: [id])
}
```

### Creating Related Data

```javascript
// Create user with posts (nested create)
const user = await db.models.User.create({
    data: {
        name: 'Alice',
        email: 'alice@example.com',
        posts: {
            create: [
                { title: 'First Post', content: 'Hello World!' },
                { title: 'Second Post', content: 'Learning RediORM' }
            ]
        }
    }
});

// Create post and connect to existing user
const post = await db.models.Post.create({
    data: {
        title: 'New Post',
        content: 'Content here',
        author: {
            connect: { id: 1 }
        }
    }
});

// Many-to-many: Create post with tags
const post = await db.models.Post.create({
    data: {
        title: 'Tagged Post',
        content: 'Content with tags',
        tags: {
            create: [
                { name: 'javascript' },
                { name: 'database' }
            ],
            connect: [
                { id: 1 }, // Connect to existing tag
                { id: 2 }
            ]
        }
    }
});
```

### Updating Relations

```javascript
// Add posts to existing user
await db.models.User.update({
    where: { id: 1 },
    data: {
        posts: {
            create: [
                { title: 'Another Post', content: 'More content' }
            ]
        }
    }
});

// Update post and change author
await db.models.Post.update({
    where: { id: 1 },
    data: {
        title: 'Updated Title',
        author: {
            connect: { id: 2 } // Change to different author
        }
    }
});

// Many-to-many updates
await db.models.Post.update({
    where: { id: 1 },
    data: {
        tags: {
            connect: [{ id: 3 }],    // Add new tags
            disconnect: [{ id: 1 }]  // Remove existing tags
        }
    }
});
```

## Transactions

### Basic Transactions

```javascript
// Simple transaction
await db.transaction(async (tx) => {
    const user = await tx.models.User.create({
        data: { name: 'Alice', balance: 1000 }
    });
    
    await tx.models.Account.create({
        data: { userId: user.id, type: 'checking' }
    });
    
    // If any operation fails, entire transaction is rolled back
});
```

### Complex Business Logic

```javascript
// Transfer money between accounts
async function transferMoney(fromAccountId, toAccountId, amount) {
    return await db.transaction(async (tx) => {
        // Lock accounts to prevent concurrent modifications
        const fromAccount = await tx.models.Account.findUnique({
            where: { id: fromAccountId }
        });
        
        const toAccount = await tx.models.Account.findUnique({
            where: { id: toAccountId }
        });
        
        if (fromAccount.balance < amount) {
            throw new Error('Insufficient funds');
        }
        
        // Update balances
        await tx.models.Account.update({
            where: { id: fromAccountId },
            data: { balance: fromAccount.balance - amount }
        });
        
        await tx.models.Account.update({
            where: { id: toAccountId },
            data: { balance: toAccount.balance + amount }
        });
        
        // Log transaction
        await tx.models.Transaction.create({
            data: {
                fromAccountId,
                toAccountId,
                amount,
                type: 'transfer',
                timestamp: new Date()
            }
        });
        
        return { success: true, transactionId: transaction.id };
    });
}
```

### Nested Transactions

```javascript
// Nested transaction example
await db.transaction(async (tx1) => {
    const user = await tx1.models.User.create({
        data: { name: 'Alice' }
    });
    
    // Inner transaction (savepoint)
    try {
        await tx1.transaction(async (tx2) => {
            await tx2.models.Post.create({
                data: { title: 'Post 1', authorId: user.id }
            });
            
            // This might fail
            await tx2.models.Post.create({
                data: { title: 'Post 2', authorId: user.id, invalidField: 'value' }
            });
        });
    } catch (error) {
        console.log('Inner transaction failed, but user creation is preserved');
    }
    
    // This will still succeed
    await tx1.models.Profile.create({
        data: { userId: user.id, bio: 'User bio' }
    });
});
```

## Batch Operations

### CreateMany

```javascript
// Bulk insert users
const result = await db.models.User.createMany({
    data: [
        { name: 'Alice', email: 'alice@example.com' },
        { name: 'Bob', email: 'bob@example.com' },
        { name: 'Charlie', email: 'charlie@example.com' }
    ]
});

console.log(`Created ${result.count} users`);

// With skipDuplicates (if supported by database)
const result = await db.models.User.createMany({
    data: users,
    skipDuplicates: true
});
```

### UpdateMany

```javascript
// Update multiple records
const result = await db.models.User.updateMany({
    where: {
        email: { endsWith: '@company.com' }
    },
    data: {
        role: 'employee',
        updatedAt: new Date()
    }
});

console.log(`Updated ${result.count} users`);

// Conditional updates
await db.models.Post.updateMany({
    where: {
        AND: [
            { published: false },
            { createdAt: { lt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000) } }
        ]
    },
    data: { status: 'draft_expired' }
});
```

### DeleteMany

```javascript
// Delete multiple records
const result = await db.models.User.deleteMany({
    where: {
        AND: [
            { active: false },
            { lastLoginAt: { lt: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000) } }
        ]
    }
});

console.log(`Deleted ${result.count} inactive users`);

// Delete all records (use with caution)
await db.models.TempData.deleteMany({});
```

### Batch Operations in Transactions

```javascript
// Process large datasets efficiently
async function processUsers(userUpdates) {
    const batchSize = 1000;
    
    for (let i = 0; i < userUpdates.length; i += batchSize) {
        await db.transaction(async (tx) => {
            const batch = userUpdates.slice(i, i + batchSize);
            
            // Process batch
            for (const update of batch) {
                await tx.models.User.update({
                    where: { id: update.id },
                    data: update.data
                });
            }
        });
        
        console.log(`Processed ${Math.min(i + batchSize, userUpdates.length)} of ${userUpdates.length} users`);
    }
}
```

## Complex Queries

### Advanced Where Conditions

```javascript
// Complex filtering
const users = await db.models.User.findMany({
    where: {
        AND: [
            { age: { gte: 18 } },
            {
                OR: [
                    { email: { endsWith: '@company.com' } },
                    { role: { in: ['admin', 'moderator'] } }
                ]
            },
            {
                NOT: {
                    status: 'banned'
                }
            }
        ]
    }
});

// String operations
const posts = await db.models.Post.findMany({
    where: {
        OR: [
            { title: { contains: 'javascript' } },
            { title: { startsWith: 'How to' } },
            { content: { endsWith: 'tutorial' } }
        ]
    }
});

// Numeric operations
const products = await db.models.Product.findMany({
    where: {
        price: { gt: 100, lt: 1000 },
        rating: { gte: 4.0 },
        categoryId: { in: [1, 2, 3] }
    }
});

// Date operations
const recentPosts = await db.models.Post.findMany({
    where: {
        createdAt: {
            gte: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000), // Last 7 days
            lt: new Date()
        }
    }
});
```

### Sorting and Pagination

```javascript
// Multiple sorting criteria
const users = await db.models.User.findMany({
    orderBy: [
        { role: 'desc' },      // Admin, Moderator, User
        { createdAt: 'desc' }, // Newest first
        { name: 'asc' }        // Then alphabetically
    ]
});

// Pagination
const page = 1;
const pageSize = 20;

const users = await db.models.User.findMany({
    orderBy: { id: 'asc' },
    skip: (page - 1) * pageSize,
    take: pageSize
});

// Cursor-based pagination (more efficient for large datasets)
const users = await db.models.User.findMany({
    cursor: { id: lastUserId },
    take: 20,
    orderBy: { id: 'asc' }
});
```

## Eager Loading

### Basic Includes

```javascript
// Include related data
const users = await db.models.User.findMany({
    include: {
        posts: true,
        profile: true
    }
});

// Selective field loading
const users = await db.models.User.findMany({
    include: {
        posts: {
            select: {
                id: true,
                title: true,
                createdAt: true
            }
        }
    }
});
```

### Nested Includes

```javascript
// Deep nesting
const users = await db.models.User.findMany({
    include: {
        posts: {
            include: {
                comments: {
                    include: {
                        author: {
                            select: {
                                id: true,
                                name: true
                            }
                        }
                    }
                }
            }
        }
    }
});

// Include with filtering
const users = await db.models.User.findMany({
    include: {
        posts: {
            where: { published: true },
            orderBy: { createdAt: 'desc' },
            take: 5
        }
    }
});
```

### Conditional Includes

```javascript
// Include based on conditions
const users = await db.models.User.findMany({
    include: {
        posts: {
            where: {
                AND: [
                    { published: true },
                    { createdAt: { gte: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000) } }
                ]
            },
            orderBy: { views: 'desc' },
            take: 10
        },
        profile: {
            where: { isPublic: true }
        }
    }
});
```

## Aggregations

### Count Operations

```javascript
// Count records
const userCount = await db.models.User.count({
    where: { active: true }
});

// Count with grouping (using raw queries)
const departmentCounts = await db.queryRaw(`
    SELECT department, COUNT(*) as count
    FROM users 
    WHERE active = true
    GROUP BY department
    ORDER BY count DESC
`);
```

### Sum, Average, Min, Max

```javascript
// Financial aggregations
const orderStats = await db.queryRaw(`
    SELECT 
        COUNT(*) as totalOrders,
        SUM(amount) as totalRevenue,
        AVG(amount) as averageOrderValue,
        MIN(amount) as smallestOrder,
        MAX(amount) as largestOrder
    FROM orders 
    WHERE status = 'completed'
    AND created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
`);

// MongoDB aggregations
const stats = await db.queryRaw(`{
    "aggregate": "orders",
    "pipeline": [
        {"$match": {"status": "completed"}},
        {"$group": {
            "_id": null,
            "totalOrders": {"$sum": 1},
            "totalRevenue": {"$sum": "$amount"},
            "averageOrderValue": {"$avg": "$amount"},
            "smallestOrder": {"$min": "$amount"},
            "largestOrder": {"$max": "$amount"}
        }}
    ]
}`);
```

### Group By Operations

```javascript
// Group by with having clause
const popularCategories = await db.queryRaw(`
    SELECT 
        category_id,
        COUNT(*) as post_count,
        AVG(views) as avg_views
    FROM posts 
    WHERE published = true
    GROUP BY category_id
    HAVING post_count >= 10
    ORDER BY avg_views DESC
    LIMIT 5
`);

// Time-based grouping
const dailyStats = await db.queryRaw(`
    SELECT 
        DATE(created_at) as date,
        COUNT(*) as registrations,
        COUNT(CASE WHEN email_verified = true THEN 1 END) as verified
    FROM users
    WHERE created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
    GROUP BY DATE(created_at)
    ORDER BY date DESC
`);
```

## Raw Queries

### SQL Queries

```javascript
// Parameterized queries (prevents SQL injection)
const users = await db.queryRaw(
    'SELECT * FROM users WHERE age > ? AND department = ?',
    18, 'engineering'
);

// Complex joins
const userPostCounts = await db.queryRaw(`
    SELECT 
        u.id,
        u.name,
        u.email,
        COUNT(p.id) as post_count,
        MAX(p.created_at) as last_post_date
    FROM users u
    LEFT JOIN posts p ON u.id = p.author_id
    WHERE u.active = true
    GROUP BY u.id, u.name, u.email
    HAVING post_count > 0
    ORDER BY post_count DESC
    LIMIT 20
`);

// Subqueries
const topAuthors = await db.queryRaw(`
    SELECT * FROM users
    WHERE id IN (
        SELECT author_id 
        FROM posts 
        WHERE views > 1000
        GROUP BY author_id
        HAVING COUNT(*) >= 5
    )
`);
```

### Database-Specific Features

```javascript
// PostgreSQL: JSON operations
const users = await db.queryRaw(`
    SELECT * FROM users 
    WHERE metadata->>'department' = ?
    AND (metadata->'preferences'->>'theme') = ?
`, 'engineering', 'dark');

// PostgreSQL: Array operations
const posts = await db.queryRaw(`
    SELECT * FROM posts 
    WHERE ? = ANY(tags)
    AND array_length(tags, 1) > 2
`, 'javascript');

// MySQL: Full-text search
const posts = await db.queryRaw(`
    SELECT *, MATCH(title, content) AGAINST(? IN NATURAL LANGUAGE MODE) as relevance
    FROM posts 
    WHERE MATCH(title, content) AGAINST(? IN NATURAL LANGUAGE MODE)
    ORDER BY relevance DESC
`, 'javascript tutorial', 'javascript tutorial');

// MongoDB: Complex aggregations
const userEngagement = await db.queryRaw(`{
    "aggregate": "users",
    "pipeline": [
        {"$lookup": {
            "from": "posts",
            "localField": "_id",
            "foreignField": "authorId",
            "as": "posts"
        }},
        {"$lookup": {
            "from": "comments",
            "localField": "_id", 
            "foreignField": "authorId",
            "as": "comments"
        }},
        {"$addFields": {
            "postCount": {"$size": "$posts"},
            "commentCount": {"$size": "$comments"},
            "engagementScore": {"$add": [
                {"$multiply": [{"$size": "$posts"}, 3]},
                {"$size": "$comments"}
            ]}
        }},
        {"$match": {"engagementScore": {"$gt": 10}}},
        {"$sort": {"engagementScore": -1}},
        {"$limit": 50}
    ]
}`);
```

### Raw Modifications

```javascript
// Bulk updates with complex logic
const result = await db.executeRaw(`
    UPDATE posts 
    SET featured = true,
        updated_at = NOW()
    WHERE views > 10000 
    AND created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
    AND featured = false
`);

console.log(`Featured ${result.rowsAffected} posts`);

// Database maintenance
await db.executeRaw('VACUUM'); // SQLite
await db.executeRaw('OPTIMIZE TABLE users'); // MySQL
await db.executeRaw('REINDEX DATABASE'); // PostgreSQL

// MongoDB: Create indexes
await db.executeRaw(`{
    "createIndex": {
        "collection": "posts",
        "index": {"authorId": 1, "createdAt": -1},
        "options": {"background": true}
    }
}`);
```

## Performance Optimization

### Query Optimization

```javascript
// Use specific field selection (when available)
const users = await db.models.User.findMany({
    select: {
        id: true,
        name: true,
        email: true
    }
});

// Optimize includes with specific fields
const users = await db.models.User.findMany({
    include: {
        posts: {
            select: { id: true, title: true },
            where: { published: true },
            take: 5 // Limit related records
        }
    }
});

// Use indexes effectively
const users = await db.models.User.findMany({
    where: {
        email: 'specific@email.com' // Use unique index
    }
});
```

### Batch Processing

```javascript
// Process large datasets in chunks
async function processAllUsers(processor) {
    const batchSize = 1000;
    let processed = 0;
    
    while (true) {
        const batch = await db.models.User.findMany({
            skip: processed,
            take: batchSize,
            orderBy: { id: 'asc' }
        });
        
        if (batch.length === 0) break;
        
        await processor(batch);
        processed += batch.length;
        
        console.log(`Processed ${processed} users`);
        
        // Optional: Add delay to prevent overwhelming the database
        await new Promise(resolve => setTimeout(resolve, 100));
    }
}

// Usage
await processAllUsers(async (users) => {
    await db.models.UserStatistics.createMany({
        data: users.map(user => ({
            userId: user.id,
            calculatedAt: new Date(),
            score: calculateUserScore(user)
        }))
    });
});
```

### Connection Optimization

```javascript
// Use connection pooling
const db = fromUri('postgresql://user:pass@host/db?pool_max_conns=20&pool_min_conns=5');

// Monitor query performance
const logger = createLogger('Performance');
logger.setLevel(logger.levels.DEBUG);
db.setLogger(logger);

// Queries will show execution time:
// [Performance] DEBUG: SQL (245ms): SELECT * FROM users WHERE ...
```

### Caching Strategies

```javascript
// Simple in-memory cache
const cache = new Map();

async function getCachedUser(id) {
    const cacheKey = `user:${id}`;
    
    if (cache.has(cacheKey)) {
        return cache.get(cacheKey);
    }
    
    const user = await db.models.User.findUnique({
        where: { id },
        include: { profile: true }
    });
    
    // Cache for 5 minutes
    cache.set(cacheKey, user);
    setTimeout(() => cache.delete(cacheKey), 5 * 60 * 1000);
    
    return user;
}

// Use Redis for distributed caching
const Redis = require('redis');
const redis = Redis.createClient();

async function getCachedUserWithRedis(id) {
    const cacheKey = `user:${id}`;
    const cached = await redis.get(cacheKey);
    
    if (cached) {
        return JSON.parse(cached);
    }
    
    const user = await db.models.User.findUnique({
        where: { id },
        include: { profile: true }
    });
    
    await redis.setex(cacheKey, 300, JSON.stringify(user)); // 5 minutes
    return user;
}
```

---

For more information on specific topics, see:
- [Database Guide](./database-guide.md) for database-specific optimizations
- [APIs & Servers](./apis-and-servers.md) for server-side optimizations
- [Getting Started](./getting-started.md) for basic setup and configuration