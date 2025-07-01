# RediORM

A modern, schema-driven ORM for Go with a JavaScript runtime interface. RediORM provides a clean separation between Go's type-safe database operations and a Prisma-like JavaScript API for dynamic scripting.

## 🚀 Features

- **Dual API Design** - Native Go API for performance, JavaScript API for flexibility
- **Schema-Driven** - Define your data model once, use it everywhere
- **Multi-Database Support** - SQLite, MySQL, and PostgreSQL
- **Smart Field Mapping** - Automatic conversion between naming conventions
- **Migration System** - Track and manage database schema changes
- **Raw SQL Support** - Execute arbitrary SQL when needed
- **Transaction Support** - Full ACID compliance with savepoints
- **Type-Safe Queries** - Compile-time safety in Go, runtime validation in JS

## 📦 Installation

```bash
go get github.com/rediwo/redi-orm
```

## 🎯 Quick Start

### Go API

The Go API provides type-safe, high-performance database operations:

```go
package main

import (
    "context"
    "log"
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/schema"
    "github.com/rediwo/redi-orm/utils"
)

func main() {
    // Create database from URI
    db, err := database.NewFromURI("sqlite://./myapp.db")
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Option 1: Load schema from string (Prisma-style)
    schemaContent := `
        model User {
            id    Int     @id @default(autoincrement())
            email String  @unique
            name  String
        }
        
        model Post {
            id      Int    @id @default(autoincrement())
            title   String
            content String
            userId  Int
        }
    `
    if err := db.LoadSchema(ctx, schemaContent); err != nil {
        log.Fatal(err)
    }
    
    // Option 2: Load schema from file
    // if err := db.LoadSchemaFrom(ctx, "./schema.prisma"); err != nil {
    //     log.Fatal(err)
    // }
    
    // Option 3: Define schema programmatically
    // userSchema := schema.New("User").
    //     AddField(schema.NewField("id").Int().PrimaryKey().AutoIncrement().Build()).
    //     AddField(schema.NewField("email").String().Unique().Build()).
    //     AddField(schema.NewField("name").String().Build())
    // db.RegisterSchema(userSchema)
    
    // Sync all loaded schemas with database
    if err := db.SyncSchemas(ctx); err != nil {
        log.Fatal(err)
    }
    
    // After sync, models are available
    models := db.GetModels() // ["User", "Post"]
    
    // Insert data
    result, err := db.Model("User").
        Insert(map[string]interface{}{
            "email": "alice@example.com",
            "name": "Alice",
        }).
        Exec(ctx)
    if err != nil {
        log.Fatal(err)
    }
    userID := result.LastInsertID
    
    // Insert with relations
    _, err = db.Model("Post").
        Insert(map[string]interface{}{
            "title": "Hello World",
            "content": "My first post",
            "userId": userID,
        }).
        Exec(ctx)
    
    // Query data
    var users []map[string]interface{}
    err = db.Model("User").
        Select("id", "email", "name").
        Where("email").Like("%example.com%").
        OrderBy("name", "ASC").
        FindMany(ctx, &users)
    
    // Count
    count, err := db.Model("Post").
        Where("userId").Equals(userID).
        Count(ctx)
    
    // Update
    _, err = db.Model("User").
        Update(map[string]interface{}{"name": "Alice Smith"}).
        Where("id").Equals(userID).
        Exec(ctx)
    
    // Raw SQL
    rows, err := db.Query("SELECT * FROM users WHERE created_at > ?", "2024-01-01")
    defer rows.Close()
    
    // Use utils for scanning
    results, err := utils.ScanRowsToMaps(rows)
}
```

### JavaScript API

The JavaScript API provides a Prisma-like interface for dynamic operations:

```javascript
// Import the ORM module
const { fromUri } = require('redi/orm');

async function main() {
    // Create database connection
    const db = fromUri('sqlite://./myapp.db');
    await db.connect();
    
    // Load schema from Prisma-style definition
    await db.loadSchema(`
        model User {
            id    Int     @id @default(autoincrement())
            email String  @unique
            name  String
            posts Post[]
        }
        
        model Post {
            id      Int    @id @default(autoincrement())
            title   String
            content String
            userId  Int
            user    User   @relation(fields: [userId], references: [id])
        }
    `);
    
    // Or load from file
    // await db.loadSchemaFrom('./schema.prisma');
    
    // Sync schemas with database
    await db.syncSchemas();
    
    // Use models via db.models
    const user = await db.models.User.create({
        data: {
            email: 'bob@example.com',
            name: 'Bob',
            posts: {
                create: [
                    { title: 'Hello World', content: 'My first post' }
                ]
            }
        }
    });
    
    // Find with relations
    const users = await db.models.User.findMany({
        where: {
            email: { contains: '@example.com' }
        },
        include: {
            posts: true
        },
        orderBy: {
            name: 'asc'
        }
    });
    
    // Update
    await db.models.User.update({
        where: { id: user.id },
        data: { name: 'Bob Smith' }
    });
    
    // Raw SQL queries
    const results = await db.queryRaw(
        'SELECT u.*, COUNT(p.id) as post_count FROM users u ' +
        'LEFT JOIN posts p ON u.id = p.user_id ' +
        'GROUP BY u.id'
    );
    
    // Execute raw SQL
    const { rowsAffected } = await db.executeRaw(
        'UPDATE users SET updated_at = ? WHERE id = ?',
        new Date(), user.id
    );
    
    await db.close();
}

main().catch(console.error);
```

## 🏗️ Architecture

### Directory Structure

```
redi-orm/
├── database/          # Database abstraction layer
├── drivers/           # Database driver implementations
│   ├── base/         # Shared driver functionality
│   ├── sqlite/       # SQLite driver
│   ├── mysql/        # MySQL driver
│   └── postgresql/   # PostgreSQL driver
├── schema/           # Schema definition and parsing
├── migration/        # Migration system
├── modules/          # Feature modules
│   └── orm/         # JavaScript ORM interface
│       └── tests/   # JavaScript test suite
├── utils/           # Common utilities
├── types/           # Shared interfaces
└── registry/        # Driver registration
```

### Key Components

1. **Database Layer** - Provides connection management and query execution
2. **Schema System** - Handles model definitions and field mappings
3. **Query Builders** - Type-safe query construction for all operations
4. **Migration Engine** - Tracks and applies database schema changes
5. **JavaScript Runtime** - Goja-based JS engine for dynamic operations

## 📊 Database Support

| Feature | SQLite | MySQL | PostgreSQL |
|---------|--------|-------|------------|
| Basic CRUD | ✅ | ✅ | ✅ |
| Transactions | ✅ | ✅ | ✅ |
| Migrations | ✅ | ✅ | ✅ |
| Raw Queries | ✅ | ✅ | ✅ |
| Field Mapping | ✅ | ✅ | ✅ |
| Savepoints | ✅ | ✅ | ✅ |

## 🔧 Advanced Features

### Schema Loading

Both Go and JavaScript APIs support the same schema loading methods:

**Go:**
```go
// From string
err := db.LoadSchema(ctx, `
    model User {
        id    Int    @id @default(autoincrement())
        email String @unique @map("email_address")
        name  String
    }
`)

// From file
err := db.LoadSchemaFrom(ctx, "./schema.prisma")

// Multiple schemas
db.LoadSchema(ctx, userSchema)
db.LoadSchema(ctx, postSchema)
db.SyncSchemas(ctx) // Apply all at once
```

**JavaScript:**
```javascript
// From string
await db.loadSchema(`
    model User {
        id    Int    @id @default(autoincrement())
        email String @unique @map("email_address")
        name  String
    }
`);

// From file
await db.loadSchemaFrom('./prisma/schema.prisma');

// Multiple schemas
await db.loadSchema(userSchema);
await db.loadSchema(postSchema);
await db.syncSchemas(); // Apply all at once
```

### Field Mapping

RediORM automatically handles field name conversions:

- Schema: `firstName` → Database: `first_name`
- Database: `created_at` → Schema: `createdAt`
- Custom mapping: `@map("custom_name")`

### Smart Scanning

```go
// Scan into structs
var users []User
err := db.Model("User").FindMany(ctx, &users)

// Scan into maps (automatic)
var results []map[string]interface{}
err := db.Model("User").FindMany(ctx, &results)

// Raw query scanning
rows, _ := db.Query("SELECT * FROM users")
results, err := utils.ScanRowsToMaps(rows)
```

### Transactions

```javascript
// JavaScript transaction (coming soon)
await db.$transaction(async (tx) => {
    const user = await tx.user.create({ data: { name: 'Alice' } });
    const post = await tx.post.create({ 
        data: { title: 'Hello', userId: user.id } 
    });
    return { user, post };
});
```

```go
// Go transaction
err := db.Transaction(ctx, func(tx types.Transaction) error {
    // All operations in transaction
    _, err := tx.Model("User").Insert(userData).Exec(ctx)
    if err != nil {
        return err // Automatic rollback
    }
    _, err = tx.Model("Post").Insert(postData).Exec(ctx)
    return err
})
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run specific test suites
make test-orm        # JavaScript ORM tests
make test-sqlite     # SQLite driver tests
make test-mysql      # MySQL driver tests
make test-postgresql # PostgreSQL driver tests

# Additional options
make test-verbose    # Verbose output
make test-cover      # Coverage report
make test-race       # Race detection
```

## 🛠️ Development

```bash
# Format code
make fmt

# Run linter
make lint

# Run vet
make vet

# Full development cycle
make dev

# CI workflow
make ci
```

## 🚦 Roadmap

- [x] Core database operations
- [x] Schema management
- [x] Migration system
- [x] JavaScript runtime
- [x] Raw SQL support
- [ ] Relations and joins
- [ ] Query optimization
- [ ] Connection pooling
- [ ] Middleware support
- [ ] GraphQL integration

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Contributing

Contributions are welcome! Please ensure:

1. All tests pass (`make test`)
2. Code is formatted (`make fmt`)
3. No linter warnings (`make lint`)
4. New features include tests

---

Built with ❤️ by the RediORM team