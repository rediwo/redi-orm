# RediORM Examples

This directory contains example JavaScript files demonstrating various features of RediORM.

## Running Examples

You can run any example using the `redi-orm run` command:

```bash
# Run basic example
redi-orm run examples/basic.js

# Run transaction example
redi-orm run examples/transaction.js

# Run simplified queries example
redi-orm run examples/queries-simple.js

# Run with timeout for long-running scripts
redi-orm run --timeout=30000 examples/long-running.js
redi-orm run --timeout=20000 examples/batch-process.js
```

## Examples

### basic.js
Demonstrates basic CRUD operations:
- Database connection using `fromUri()`
- Schema definition and auto-migration
- Creating records
- Querying with relations
- Updating records
- Counting records

### transaction.js
Shows how to use database transactions:
- Transaction rollback on error
- Multiple operations in a single transaction
- Maintaining data consistency
- Recording transaction history

### queries.js
Advanced query operations (demonstrates features, some not yet implemented):
- Complex conditions with operators
- AND/OR logical operations
- Aggregations (avg, max, min, count)
- Group by operations
- Nested creates with relations
- Include relations in queries

### queries-simple.js
Simplified query operations using currently supported features:
- Basic filtering
- Finding records
- Counting
- Updates (single and batch)
- Raw SQL queries
- Delete operations

### long-running.js
Demonstrates timeout functionality:
- Long-running batch operations
- Asynchronous task processing
- Proper timeout handling
- Run with: `redi-orm run --timeout=30000 examples/long-running.js`

### batch-process.js
Batch processing example:
- Processing records in batches
- Progress tracking
- Timeout management for long operations
- Run with: `redi-orm run --timeout=20000 examples/batch-process.js`

### Additional Examples

- **simple-demo.js**: Simple todo list management
- **run-demo.js**: Demonstrates the run command features
- **test-run.js**: Basic test of the run command
- **working-demo.js**: Product and order management with transactions

## Database URIs

The examples use SQLite for simplicity, but RediORM supports multiple databases:

```javascript
// SQLite
const db = fromUri('sqlite://./myapp.db');
const db = fromUri('sqlite://:memory:');

// MySQL
const db = fromUri('mysql://user:pass@localhost:3306/dbname');

// PostgreSQL
const db = fromUri('postgresql://user:pass@localhost:5432/dbname');

// MongoDB
const db = fromUri('mongodb://user:pass@localhost:27017/dbname');
const db = fromUri('mongodb+srv://cluster.mongodb.net/dbname');
```

## Schema Definition

RediORM uses Prisma-compatible schema syntax:

```javascript
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
    }
`);
```

## Accessing Models

After loading schemas and syncing, models are accessed through the `db.models` object:

```javascript
// Create a user
const user = await db.models.User.create({
    data: { email: 'alice@example.com', name: 'Alice' }
});

// Query with relations
const userWithPosts = await db.models.User.findUnique({
    where: { email: 'alice@example.com' },
    include: { posts: true }
});

// In transactions, models are also accessed through tx.models
await db.transaction(async (tx) => {
    await tx.models.User.update({
        where: { id: userId },
        data: { name: 'Updated Name' }
    });
});
```

## Key Features

- **Type-safe queries**: All queries are validated against your schema
- **Relations**: Automatic handling of foreign keys and joins
- **Transactions**: ACID-compliant transactions with rollback
- **Migrations**: Automatic schema synchronization in development
- **Multiple databases**: Support for SQLite, MySQL, PostgreSQL, and MongoDB
- **Raw SQL**: Execute raw SQL when needed with `queryRaw` and `executeRaw`

## Migration Examples

For comprehensive migration examples and production workflows, see the [migration](./migration/) subdirectory:
- Production migration guide
- Test scripts for complete workflow
- Real-world deployment examples
- Best practices and safety guidelines