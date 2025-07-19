# API Reference

Complete reference for RediORM's Go API, JavaScript API, and CLI commands.

## Table of Contents

- [Go API](#go-api)
- [JavaScript API](#javascript-api)
- [CLI Commands](#cli-commands)
- [Schema Definition](#schema-definition)
- [Type Mapping](#type-mapping)

## Go API

### Database Connection

```go
import (
    "context"
    "github.com/rediwo/redi-orm/database"
    _ "github.com/rediwo/redi-orm/drivers/sqlite"
)

ctx := context.Background()

// Create database instance
db, err := database.NewFromURI("sqlite://./app.db")
db, err := database.NewFromURI("mysql://user:pass@host/db")
db, err := database.NewFromURI("postgresql://user:pass@host/db")
db, err := database.NewFromURI("mongodb://host/db")

// Connect
err = db.Connect(ctx)

// Load schema
err = db.LoadSchemaFrom(ctx, "./schema.prisma")
err = db.LoadSchemaString(ctx, schemaString)

// Sync database structure
err = db.SyncSchemas(ctx)

// Close connection
err = db.Close()
```

### ORM Client

```go
import "github.com/rediwo/redi-orm/orm"

client := orm.NewClient(db)

// Create
result, err := client.Model("User").Create(`{
    "data": {
        "name": "Alice",
        "email": "alice@example.com"
    }
}`)

// Find many
users, err := client.Model("User").FindMany(`{
    "where": { "email": { "contains": "@example.com" } },
    "include": { "posts": true }
}`)

// Find unique
user, err := client.Model("User").FindUnique(`{
    "where": { "id": 1 }
}`)

// Update
updated, err := client.Model("User").Update(`{
    "where": { "id": 1 },
    "data": { "name": "Alice Smith" }
}`)

// Delete
deleted, err := client.Model("User").Delete(`{
    "where": { "id": 1 }
}`)

// Batch operations
createResult, err := client.Model("User").CreateMany(`{
    "data": [
        { "name": "User1", "email": "user1@example.com" },
        { "name": "User2", "email": "user2@example.com" }
    ]
}`)

updateResult, err := client.Model("User").UpdateMany(`{
    "where": { "email": { "contains": "@example.com" } },
    "data": { "active": true }
}`)

deleteResult, err := client.Model("User").DeleteMany(`{
    "where": { "active": false }
}`)
```

### Raw Queries

```go
// Raw queries
results, err := db.QueryRaw(ctx, "SELECT * FROM users WHERE age > ?", 18)

// Raw execution
result, err := db.ExecuteRaw(ctx, "UPDATE users SET active = ? WHERE id = ?", true, 1)
fmt.Printf("Rows affected: %d\n", result.RowsAffected)
```

### Transactions

```go
err = db.Transaction(ctx, func(tx database.Database) error {
    // All operations use tx instead of db
    user, err := orm.NewClient(tx).Model("User").Create(`{
        "data": { "name": "Alice", "balance": 1000 }
    }`)
    if err != nil {
        return err // Transaction will be rolled back
    }
    
    _, err = orm.NewClient(tx).Model("Transaction").Create(`{
        "data": { "userId": ` + fmt.Sprintf("%d", user["id"]) + `, "amount": 100 }
    }`)
    return err
})
```

### Query Builder (Advanced)

```go
// Get query builder directly
userQuery := db.Model("User")

// Select with conditions
results, err := userQuery.Select().
    Where("age", ">", 18).
    Where("active", "=", true).
    OrderBy("name", "ASC").
    Limit(10).
    FindMany(ctx, &users)

// Insert
result, err := userQuery.Insert(map[string]interface{}{
    "name": "Alice",
    "email": "alice@example.com",
}).Exec(ctx)

// Update
result, err := userQuery.Update(map[string]interface{}{
    "active": true,
}).Where("id", "=", 1).Exec(ctx)

// Delete
result, err := userQuery.Delete().Where("id", "=", 1).Exec(ctx)
```

## JavaScript API

### Database Connection

```javascript
const { fromUri, createLogger } = require('redi/orm');

// Create database instance
const db = fromUri('sqlite://./app.db');
const db = fromUri('mysql://user:pass@host/db');
const db = fromUri('postgresql://user:pass@host/db');
const db = fromUri('mongodb://host/db');

// Setup logging
const logger = createLogger('MyApp');
logger.setLevel(logger.levels.DEBUG); // NONE, ERROR, WARN, INFO, DEBUG
db.setLogger(logger);

// Connect and setup
await db.connect();
await db.loadSchemaFrom('./schema.prisma');
await db.loadSchema(`model User { ... }`);
await db.syncSchemas();

// Disconnect
await db.disconnect();
```

### Model Operations

```javascript
// Create single record
const user = await db.models.User.create({
    data: {
        name: 'Alice',
        email: 'alice@example.com'
    }
});

// Create with relations
const user = await db.models.User.create({
    data: {
        name: 'Alice',
        email: 'alice@example.com',
        posts: {
            create: [
                { title: 'Post 1', content: 'Content 1' },
                { title: 'Post 2', content: 'Content 2' }
            ]
        }
    }
});

// Find many with filtering
const users = await db.models.User.findMany({
    where: {
        OR: [
            { email: { contains: '@example.com' } },
            { name: { startsWith: 'A' } }
        ],
        age: { gte: 18 }
    },
    orderBy: { name: 'asc' },
    take: 10,
    skip: 20
});

// Find unique
const user = await db.models.User.findUnique({
    where: { id: 1 },
    include: { posts: true }
});

// Update single record
const updated = await db.models.User.update({
    where: { id: 1 },
    data: { name: 'Alice Smith' }
});

// Delete single record
const deleted = await db.models.User.delete({
    where: { id: 1 }
});

// Batch operations
const createResult = await db.models.User.createMany({
    data: [
        { name: 'User1', email: 'user1@example.com' },
        { name: 'User2', email: 'user2@example.com' }
    ]
});

const updateResult = await db.models.User.updateMany({
    where: { email: { contains: '@example.com' } },
    data: { active: true }
});

const deleteResult = await db.models.User.deleteMany({
    where: { active: false }
});
```

### Advanced Queries

```javascript
// Complex where conditions
const results = await db.models.User.findMany({
    where: {
        AND: [
            { age: { gte: 18 } },
            { 
                OR: [
                    { email: { endsWith: '@company.com' } },
                    { role: { in: ['admin', 'moderator'] } }
                ]
            }
        ]
    }
});

// Nested includes
const users = await db.models.User.findMany({
    include: {
        posts: {
            include: {
                comments: {
                    include: {
                        author: true
                    }
                }
            }
        }
    }
});

// Include with options
const users = await db.models.User.findMany({
    include: {
        posts: {
            select: { id: true, title: true },
            where: { published: true },
            orderBy: { createdAt: 'desc' },
            take: 5
        }
    }
});
```

### Raw Queries

```javascript
// SQL queries (works with all databases including MongoDB)
const results = await db.queryRaw('SELECT * FROM users WHERE age > ?', 18);

// For MongoDB: SQL is automatically translated to MongoDB aggregation
const results = await db.queryRaw('SELECT name, COUNT(*) as postCount FROM users u JOIN posts p ON u.id = p.authorId GROUP BY u.id');

// MongoDB native commands (MongoDB only)
const results = await db.queryRaw(`{
    "find": "users",
    "filter": {"age": {"$gt": 18}},
    "sort": {"name": 1}
}`);

// MongoDB aggregation pipeline (MongoDB only)
const results = await db.queryRaw(`{
    "aggregate": "users",
    "pipeline": [
        {"$match": {"age": {"$gt": 18}}},
        {"$group": {"_id": "$department", "count": {"$sum": 1}}}
    ]
}`);

// Raw execution
const result = await db.executeRaw('INSERT INTO users (name) VALUES (?)', 'John');
console.log(`Inserted ${result.rowsAffected} rows`);
```

### Transactions

```javascript
await db.transaction(async (tx) => {
    // Use tx.models instead of db.models
    const user = await tx.models.User.create({
        data: { name: 'Alice', balance: 1000 }
    });
    
    await tx.models.Account.update({
        where: { userId: user.id },
        data: { balance: 900 }
    });
    
    await tx.models.Transaction.create({
        data: { 
            userId: user.id, 
            amount: 100,
            type: 'withdrawal'
        }
    });
    
    // All operations succeed or all are rolled back
});
```

### Logging

```javascript
// Create logger
const logger = createLogger('MyApp');

// Set log level
logger.setLevel(logger.levels.DEBUG);
logger.setLevel('info'); // or string

// Available levels
logger.levels.NONE    // No logging
logger.levels.ERROR   // Only errors
logger.levels.WARN    // Warnings and errors
logger.levels.INFO    // Info, warnings, and errors
logger.levels.DEBUG   // All logging including SQL queries

// Manual logging
logger.debug('Debug message');
logger.info('Info message');
logger.warn('Warning message');
logger.error('Error message');

// Attach to database for automatic SQL logging
db.setLogger(logger);
```

## CLI Commands

### Installation

```bash
# Install from Go
go install github.com/rediwo/redi-orm/cmd/redi-orm@latest

# Verify installation
redi-orm --version
```

### JavaScript Execution

```bash
# Run JavaScript files
redi-orm run script.js
redi-orm run script.js --timeout=30000  # 30 second timeout

# With database override
redi-orm run script.js --db=mysql://user:pass@localhost/db
```

### Migration Commands

```bash
# Apply migrations
redi-orm migrate --db=sqlite://./app.db --schema=./schema.prisma

# Generate migration
redi-orm migrate:generate --db=sqlite://./app.db --schema=./schema.prisma --name="add_users"

# Check migration status
redi-orm migrate:status --db=sqlite://./app.db

# Migration with logging
redi-orm migrate --db=sqlite://./app.db --schema=./schema.prisma --log-level=debug
```

### Server Commands

```bash
# Start GraphQL + REST API server
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma
redi-orm server --port=8080 --playground=true --log-level=info

# Start MCP server for AI
redi-orm mcp --db=sqlite://./app.db --schema=./schema.prisma
redi-orm mcp --port=3000 --read-only

# Combined server (GraphQL + REST + MCP)
redi-orm server --enable-mcp --mcp-port=3001
```

### Global Options

```bash
# Database connection
--db=URI                    # Database connection URI
--schema=FILE               # Path to Prisma schema file

# Logging
--log-level=LEVEL           # none, error, warn, info, debug
--verbose                   # Enable verbose output

# Timeouts
--timeout=MS                # Timeout in milliseconds

# Help
--help                      # Show help
--version                   # Show version
```

## Schema Definition

### Basic Model

```prisma
model User {
    id        Int      @id @default(autoincrement())
    email     String   @unique
    name      String
    age       Int?
    active    Boolean  @default(true)
    createdAt DateTime @default(now())
    updatedAt DateTime @updatedAt
}
```

### Field Types

```prisma
model Example {
    // Numbers
    intField     Int
    floatField   Float
    
    // Text
    stringField  String
    textField    String  // Mapped to TEXT in SQL
    
    // Boolean
    boolField    Boolean
    
    // Dates
    dateField    DateTime
    
    // JSON (PostgreSQL, MySQL 5.7+, MongoDB)
    jsonField    Json
    
    // Optional fields
    optional     String?
    
    // Arrays (MongoDB, PostgreSQL)
    tags         String[]
    numbers      Int[]
}
```

### Field Attributes

```prisma
model User {
    // Primary key
    id     Int @id @default(autoincrement())
    
    // Unique constraint
    email  String @unique
    
    // Default values
    active Boolean @default(true)
    count  Int     @default(0)
    
    // Column mapping
    firstName String @map("first_name")
    
    // Auto-update timestamp
    updatedAt DateTime @updatedAt
}
```

### Relations

```prisma
// One-to-Many
model User {
    id    Int    @id @default(autoincrement())
    posts Post[]
}

model Post {
    id       Int  @id @default(autoincrement())
    authorId Int
    author   User @relation(fields: [authorId], references: [id])
}

// Many-to-Many
model Post {
    id   Int   @id @default(autoincrement())
    tags Tag[]
}

model Tag {
    id    Int    @id @default(autoincrement())
    posts Post[]
}

// One-to-One
model User {
    id      Int      @id @default(autoincrement())
    profile Profile?
}

model Profile {
    id     Int  @id @default(autoincrement())
    userId Int  @unique
    user   User @relation(fields: [userId], references: [id])
}
```

### Composite Keys

```prisma
model PostTag {
    postId Int
    tagId  Int
    
    @@id([postId, tagId])
}
```

### Indexes

```prisma
model User {
    email String
    name  String
    
    @@index([email])
    @@index([name, email])
    @@unique([email])
}
```

## Type Mapping

### Go to Database Types

| Go Type | SQLite | MySQL | PostgreSQL | MongoDB |
|---------|--------|-------|------------|---------|
| `int`, `int64` | INTEGER | BIGINT | BIGINT | NumberLong |
| `int32` | INTEGER | INT | INTEGER | NumberInt |
| `float64` | REAL | DOUBLE | DOUBLE PRECISION | NumberDouble |
| `float32` | REAL | FLOAT | REAL | NumberDouble |
| `string` | TEXT | VARCHAR/TEXT | VARCHAR/TEXT | String |
| `bool` | BOOLEAN | BOOLEAN | BOOLEAN | Boolean |
| `time.Time` | DATETIME | DATETIME | TIMESTAMP | Date |
| `[]byte` | BLOB | BLOB | BYTEA | BinData |
| `interface{}` | JSON | JSON | JSONB | Object/Array |

### JavaScript to Go Types

| JavaScript | Go | Notes |
|------------|----|----|
| `number` | `int64`, `float64` | Auto-detected based on decimal |
| `string` | `string` | Direct mapping |
| `boolean` | `bool` | Direct mapping |
| `Date` | `time.Time` | ISO string or timestamp |
| `Array` | `[]interface{}` | JSON array |
| `Object` | `map[string]interface{}` | JSON object |
| `null` | `nil` | For optional fields |

### Schema Field Types

| Schema Type | Go Type | JavaScript Type | Notes |
|-------------|---------|-----------------|-------|
| `Int` | `int64` | `number` | 64-bit integer |
| `Float` | `float64` | `number` | Double precision |
| `String` | `string` | `string` | UTF-8 text |
| `Boolean` | `bool` | `boolean` | True/false |
| `DateTime` | `time.Time` | `Date` | ISO 8601 |
| `Json` | `interface{}` | `any` | JSON data |
| `Int[]` | `[]int64` | `number[]` | Integer array |
| `String[]` | `[]string` | `string[]` | String array |

### Filter Operators

| Operator | JavaScript | SQL | MongoDB | Description |
|----------|------------|-----|---------|-------------|
| `equals` | `{ field: value }` | `field = ?` | `{field: value}` | Exact match |
| `not` | `{ field: { not: value } }` | `field != ?` | `{field: {$ne: value}}` | Not equal |
| `in` | `{ field: { in: [1,2,3] } }` | `field IN (?,?,?)` | `{field: {$in: [1,2,3]}}` | In list |
| `notIn` | `{ field: { notIn: [1,2] } }` | `field NOT IN (?,?)` | `{field: {$nin: [1,2]}}` | Not in list |
| `contains` | `{ field: { contains: 'text' } }` | `field LIKE '%text%'` | `{field: /text/}` | Text contains |
| `startsWith` | `{ field: { startsWith: 'pre' } }` | `field LIKE 'pre%'` | `{field: /^pre/}` | Text starts with |
| `endsWith` | `{ field: { endsWith: 'suf' } }` | `field LIKE '%suf'` | `{field: /suf$/}` | Text ends with |
| `gt` | `{ field: { gt: 10 } }` | `field > ?` | `{field: {$gt: 10}}` | Greater than |
| `gte` | `{ field: { gte: 10 } }` | `field >= ?` | `{field: {$gte: 10}}` | Greater than or equal |
| `lt` | `{ field: { lt: 10 } }` | `field < ?` | `{field: {$lt: 10}}` | Less than |
| `lte` | `{ field: { lte: 10 } }` | `field <= ?` | `{field: {$lte: 10}}` | Less than or equal |

### Logical Operators

```javascript
// AND (default)
{
    name: 'Alice',
    age: 25
}

// OR
{
    OR: [
        { name: 'Alice' },
        { name: 'Bob' }
    ]
}

// NOT
{
    NOT: {
        email: { contains: '@spam.com' }
    }
}

// Complex combinations
{
    AND: [
        { age: { gte: 18 } },
        {
            OR: [
                { email: { endsWith: '@company.com' } },
                { role: 'admin' }
            ]
        }
    ]
}
```

---

For more examples and advanced usage, see:
- [Getting Started Guide](./getting-started.md)
- [Advanced Features](./advanced-features.md)
- [Database Guide](./database-guide.md)
- [APIs & Servers](./apis-and-servers.md)