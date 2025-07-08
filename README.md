# RediORM

A modern, schema-driven ORM for Go with a JavaScript runtime interface. RediORM provides a clean separation between Go's type-safe database operations and a Prisma-like JavaScript API for dynamic scripting.

## 🚀 Features

- **Dual API Design** - Native Go API for performance, JavaScript API for flexibility
- **Schema-Driven** - Define your data model once, use it everywhere
- **Multi-Database Support** - SQLite, MySQL, PostgreSQL, and MongoDB
- **Smart Field Mapping** - Automatic conversion between naming conventions
- **Migration System** - Track and manage database schema changes (SQL databases)
- **Relations Support** - One-to-one, one-to-many, many-to-one, and many-to-many relations
- **Raw Query Support** - Execute SQL or native database commands (including MongoDB)
- **SQL Parser for MongoDB** - Write familiar SQL queries that are automatically translated to MongoDB operations
- **Transaction Support** - Full ACID compliance with savepoints (SQL databases), basic transaction support for MongoDB
- **Type-Safe Queries** - Compile-time safety in Go, runtime validation in JS
- **Comprehensive Logging** - SQL/command logging with configurable levels and color output
- **Migration Warnings** - Preview migration changes before execution

## 📦 Installation

### Go Library

```bash
go get github.com/rediwo/redi-orm
```

### CLI Tool

#### Pre-built Binaries (Recommended)

Download the latest release for your platform:

**Linux (AMD64)**:
```bash
wget https://github.com/rediwo/redi-orm/releases/latest/download/redi-orm-linux-amd64.tar.gz
tar -xzf redi-orm-linux-amd64.tar.gz
sudo mv redi-orm-* /usr/local/bin/redi-orm
```

**macOS (Intel)**:
```bash
wget https://github.com/rediwo/redi-orm/releases/latest/download/redi-orm-darwin-amd64.tar.gz
tar -xzf redi-orm-darwin-amd64.tar.gz
sudo mv redi-orm-* /usr/local/bin/redi-orm
```

**macOS (Apple Silicon)**:
```bash
wget https://github.com/rediwo/redi-orm/releases/latest/download/redi-orm-darwin-arm64.tar.gz
tar -xzf redi-orm-darwin-arm64.tar.gz
sudo mv redi-orm-* /usr/local/bin/redi-orm
```

**Windows (AMD64)**:
1. Download `redi-orm-windows-amd64.zip` from [releases](https://github.com/rediwo/redi-orm/releases/latest)
2. Extract and add to your PATH

#### Build from Source

```bash
# Install latest version
go install github.com/rediwo/redi-orm/cmd/redi-orm@latest

# Or build from source
git clone https://github.com/rediwo/redi-orm.git
cd redi-orm
make release-build
```

#### Verify Installation

```bash
redi-orm version
```

## 🎯 Quick Start

### Go ORM API

The Go ORM API provides a simplified, high-level interface for rapid development using JSON-based queries:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/orm"
)

func main() {
    // Create database connection
    db, err := database.NewFromURI("sqlite://./myapp.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Connect to database
    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Create ORM client
    client := orm.NewClient(db)
    
    // Load schema from Prisma-style definition
    schemaContent := `
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
    `
    if err := db.LoadSchema(ctx, schemaContent); err != nil {
        log.Fatal(err)
    }
    
    if err := db.SyncSchemas(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Get model references for cleaner code
    User := client.Model("User")
    Post := client.Model("Post")
    
    // Create user (returns map[string]any)
    user, err := User.Create(`{
        "data": {
            "email": "alice@example.com",
            "name": "Alice"
        }
    }`)
    if err != nil {
        log.Fatal(err)
    }
    
    // Create post with relation
    post, err := Post.Create(fmt.Sprintf(`{
        "data": {
            "title": "Hello World",
            "content": "My first post",
            "userId": %v
        }
    }`, user["id"]))
    if err != nil {
        log.Fatal(err)
    }
    
    // Find with conditions (returns []map[string]any)
    users, err := User.FindMany(`{
        "where": {
            "email": { "contains": "@example.com" }
        },
        "orderBy": { "name": "asc" },
        "include": { "posts": true }
    }`)
    if err != nil {
        log.Fatal(err)
    }
    
    // Update data
    updatedUser, err := User.Update(fmt.Sprintf(`{
        "where": { "id": %v },
        "data": { "name": "Alice Smith" }
    }`, user["id"]))
    if err != nil {
        log.Fatal(err)
    }
    
    // Count records
    count, err := Post.Count(fmt.Sprintf(`{
        "where": { "userId": %v }
    }`, user["id"]))
    if err != nil {
        log.Fatal(err)
    }
    
    // Delete records
    deleted, err := Post.Delete(fmt.Sprintf(`{
        "where": { "id": %v }
    }`, post["id"]))
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d users\n", len(users))
    fmt.Printf("Updated user: %v\n", updatedUser)
    fmt.Printf("Post count: %v\n", count)
    fmt.Printf("Deleted: %v\n", deleted)
}
```

> **💡 Note**: The Go ORM API uses JSON strings for queries, similar to the JavaScript API. This provides a consistent experience across both languages. For low-level database operations and advanced features, see the [Driver API documentation](./drivers/README.md).

### JavaScript API with Logging

The JavaScript API provides a Prisma-like interface for dynamic operations with comprehensive logging support:

```javascript
// Import the ORM module
const { fromUri, createLogger } = require('redi/orm');

async function main() {
    // Create database connection
    const db = fromUri('sqlite://./myapp.db');
    
    // Set up logging (optional)
    const logger = createLogger('MyApp');
    logger.setLevel(logger.levels.INFO); // NONE, ERROR, WARN, INFO, DEBUG
    db.setLogger(logger);
    
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
    
    // Get model references for cleaner code
    const User = db.models.User;
    const Post = db.models.Post;
    
    // Create user with nested post
    const user = await User.create({
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
    const users = await User.findMany({
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
    await User.update({
        where: { id: user.id },
        data: { name: 'Bob Smith' }
    });
    
    // Raw SQL queries (works with all databases including MongoDB)
    // The logger will show: [MyApp] DEBUG: SQL (2.45ms): SELECT u.*, COUNT(p.id)...
    const results = await db.queryRaw(
        'SELECT u.*, COUNT(p.id) as post_count FROM users u ' +
        'LEFT JOIN posts p ON u.id = p.user_id ' +
        'GROUP BY u.id'
    );
    
    // MongoDB: You can also use native MongoDB commands
    if (db.getDriverType() === 'mongodb') {
        // Native MongoDB find command
        const users = await db.queryRaw(`{
            "find": "users",
            "filter": {"age": {"$gt": 18}},
            "sort": {"name": 1}
        }`);
        
        // MongoDB aggregation pipeline
        const stats = await db.queryRaw(`{
            "aggregate": "users",
            "pipeline": [
                {"$match": {"active": true}},
                {"$group": {
                    "_id": "$role",
                    "count": {"$sum": 1},
                    "avgAge": {"$avg": "$age"}
                }}
            ]
        }`);
    }
    
    // Execute raw SQL
    const { rowsAffected } = await db.executeRaw(
        'UPDATE users SET updated_at = ? WHERE id = ?',
        new Date(), user.id
    );
    
    await db.close();
}

main().catch(console.error);
```

## 🔨 CLI Tool

RediORM includes a powerful CLI tool for database migrations and running JavaScript files with ORM support.

### Running JavaScript Files

Execute JavaScript files with full ORM support:

```bash
# Basic usage
redi-orm run script.js

# With timeout for long-running scripts
redi-orm run --timeout=30000 batch-process.js  # 30 seconds
redi-orm run --timeout 60000 data-migration.js  # Alternative syntax

# Pass arguments to the script
redi-orm run process.js --input data.json --output results.json
```

The `--timeout` flag is useful for:
- Batch processing operations
- Data migration scripts
- Long-running synchronization tasks
- Scripts with multiple async operations

### Migration Commands

#### Development Mode (Auto-migration)

```bash
# Auto-migrate based on schema file (shows warnings before applying)
redi-orm migrate --db=sqlite://./myapp.db --schema=./schema.prisma

# Enable detailed migration logging to see what changes will be applied
redi-orm migrate --db=sqlite://./myapp.db --schema=./schema.prisma --log-level=debug

# Preview changes without applying (dry run)
redi-orm migrate:dry-run --db=sqlite://./myapp.db --schema=./schema.prisma

# Force destructive changes (drops columns/tables)
redi-orm migrate --db=sqlite://./myapp.db --schema=./schema.prisma --force
```

#### Production Mode (File-based migrations)

```bash
# Generate a new migration file
redi-orm migrate:generate --db=sqlite://./myapp.db --schema=./schema.prisma --name="add_user_table"

# Apply pending migrations
redi-orm migrate:apply --db=sqlite://./myapp.db --migrations=./migrations

# Rollback last migration
redi-orm migrate:rollback --db=sqlite://./myapp.db --migrations=./migrations

# Check migration status
redi-orm migrate:status --db=sqlite://./myapp.db

# Reset all migrations (dangerous!)
redi-orm migrate:reset --db=sqlite://./myapp.db --force
```

### CLI Examples

```bash
# Run a data processing script with 5 minute timeout
redi-orm run --timeout=300000 scripts/process-large-dataset.js

# Migrate development database
redi-orm migrate --db=sqlite://./dev.db --schema=./schema.prisma --mode=auto

# Generate production migration
redi-orm migrate:generate --db=postgresql://user:pass@localhost/prod --schema=./schema.prisma --name="add_indexes"

# Apply migrations in production
redi-orm migrate:apply --db=postgresql://user:pass@localhost/prod --migrations=./migrations
```

### GraphQL Server

RediORM includes an automatic GraphQL API generator that creates a fully-featured GraphQL server from your Prisma schemas:

```bash
# Start GraphQL server with default settings
redi-orm server --db=sqlite://./myapp.db --schema=./schema.prisma

# Custom port and options
redi-orm server --db=postgresql://user:pass@localhost/db --port=8080 --playground=true --cors=true

# Configurable logging for debugging and monitoring
redi-orm server --db=sqlite://./myapp.db --schema=./schema.prisma --log-level=debug
```

Features:
- ✅ Automatic schema generation from Prisma models
- ✅ Full CRUD operations (create, read, update, delete)
- ✅ Batch operations (createMany, updateMany, deleteMany)
- ✅ Complex where conditions and filtering
- ✅ Sorting and pagination
- ✅ **Relation support** - Query across model relationships with `include` syntax
- ✅ GraphQL Playground for testing
- ✅ CORS support for web applications
- ✅ Type-safe resolvers with automatic field mapping
- ✅ Count queries and aggregations
- ✅ HTTP handler compatible with any Go web framework
- ✅ **Configurable logging** - Monitor requests, performance, and errors

#### GraphQL Schema Generation

The GraphQL schema is automatically generated from your Prisma models:

- Each model becomes a GraphQL type
- CRUD operations are generated for each model
- Relations are automatically resolved
- Field types are mapped appropriately
- Non-nullable fields are enforced

#### Available Operations

For each model (e.g., `User`), the following operations are generated:

**Queries:**
- `findUniqueUser(where: UserWhereInput!): User` - Find a single user
- `findManyUser(where: UserWhereInput, orderBy: UserOrderByInput, limit: Int, offset: Int): [User!]!` - Find multiple users
- `countUser(where: UserWhereInput): Int!` - Count users

**Mutations:**
- `createUser(data: UserCreateInput!): User!` - Create a user
- `updateUser(where: UserWhereInput!, data: UserUpdateInput!): User!` - Update a user
- `deleteUser(where: UserWhereInput!): User!` - Delete a user
- `createManyUser(data: [UserCreateInput!]!): BatchPayload!` - Create multiple users
- `updateManyUser(where: UserWhereInput, data: UserUpdateInput!): BatchPayload!` - Update multiple users
- `deleteManyUser(where: UserWhereInput): BatchPayload!` - Delete multiple users

#### Where Conditions

The GraphQL API supports rich filtering with operators:

```graphql
where: {
  # Exact match
  email: { equals: "alice@example.com" }
  
  # String operators
  name: { contains: "John" }
  name: { startsWith: "J" }
  name: { endsWith: "son" }
  
  # Comparison operators
  age: { gt: 18 }
  age: { gte: 21 }
  age: { lt: 65 }
  age: { lte: 100 }
  
  # List operators
  role: { in: ["admin", "editor"] }
  role: { notIn: ["guest"] }
  
  # Logical operators
  AND: [
    { published: { equals: true } }
    { createdAt: { gt: "2024-01-01" } }
  ]
  OR: [
    { title: { contains: "GraphQL" } }
    { title: { contains: "API" } }
  ]
}
```

#### Example Queries

```graphql
# Find users with posts (relation support)
query {
  findManyUser(
    where: { email: { contains: "@example.com" } }
    orderBy: { createdAt: DESC }
    limit: 10
  ) {
    id
    name
    email
    posts {
      id
      title
      published
      author {
        name
        email
      }
    }
    _count {
      posts
      comments
    }
  }
}

# Create user
mutation {
  createUser(data: {
    name: "Alice"
    email: "alice@example.com"
  }) {
    id
    name
    email
  }
}

# Complex filtering with relations
query {
  findManyPost(
    where: {
      AND: [
        { published: { equals: true } }
        { author: { role: { equals: "admin" } } }
        { createdAt: { gte: "2024-01-01" } }
      ]
    }
    orderBy: { views: DESC }
    limit: 20
  ) {
    title
    author {
      name
      email
    }
    _count {
      comments
    }
  }
}

# Batch operations
mutation {
  createManyTag(data: [
    { name: "GraphQL" }
    { name: "API" }
    { name: "Tutorial" }
  ]) {
    count
  }
}
```

#### Logging Configuration

The GraphQL server includes comprehensive logging for monitoring and debugging:

```bash
# Production (minimal logging)
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma --log-level=info

# Development (full debugging)
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma --log-level=debug

# Silent (no logging)
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma --log-level=none
```

**Log Levels:**
- `debug` - Shows full requests, queries, variables, and responses
- `info` - Shows operation types, execution times, and success/failure
- `warn` - Shows warnings and errors
- `error` - Shows only errors

### Server with GraphQL and REST APIs

RediORM provides both GraphQL and REST APIs in a single server, giving you flexibility to use either or both:

```bash
# Start combined GraphQL and REST API server
redi-orm server --db=sqlite://./myapp.db --schema=./schema.prisma

# Custom port and options
redi-orm server --db=postgresql://user:pass@localhost/db --port=8080 --log-level=debug

# Access endpoints:
# GraphQL: http://localhost:8080/graphql
# GraphQL Playground: http://localhost:8080/
# REST API: http://localhost:8080/api
```

Features:
- ✅ Standard REST endpoints for all models
- ✅ JSON request/response format
- ✅ Complex filtering and query parameters
- ✅ Pagination with metadata
- ✅ Batch operations
- ✅ Relation loading with includes
- ✅ CORS support for web applications
- ✅ Multiple database connections
- ✅ Execution time tracking

#### Available Endpoints

For each model (e.g., `User`), REST endpoints use the exact model name from your Prisma schema:

- `GET /api/User` - List all users
- `GET /api/User/:id` - Get a specific user
- `POST /api/User` - Create a new user
- `PUT /api/User/:id` - Update a user
- `DELETE /api/User/:id` - Delete a user
- `POST /api/User/batch` - Create multiple users

#### Query Parameters

```bash
# Pagination
GET /api/User?page=2&limit=20

# Sorting
GET /api/User?sort=-age,name  # - prefix for DESC

# Field selection
GET /api/User?select=id,name,email

# Filtering
GET /api/User?filter[age][gt]=25&filter[name][contains]=John

# Complex where conditions (JSON)
GET /api/User?where={"age":{"gt":25},"name":{"contains":"John"}}

# Include relations
GET /api/User?include=posts
GET /api/Post?include=author,comments

# Search
GET /api/User?q=john
```

#### Request/Response Format

**Create User Request:**
```json
POST /api/users
{
  "data": {
    "name": "John Doe",
    "email": "john@example.com",
    "age": 30
  }
}
```

**Response Format:**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com",
    "age": 30
  },
  "meta": {
    "execution_time": "15.234ms",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

**Paginated Response:**
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 2,
    "limit": 20,
    "total": 156,
    "pages": 8
  },
  "meta": {
    "execution_time": "23.456ms",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "User not found",
    "details": "No user with id 123"
  },
  "meta": {
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

#### JavaScript/AJAX Example

```javascript
// Using fetch API
async function getUsers() {
  const response = await fetch('http://localhost:8080/api/User?include=posts&limit=10', {
    headers: {
      'Content-Type': 'application/json',
      'X-Connection-Name': 'default'  // Optional: specify database connection
    }
  });
  const data = await response.json();
  console.log(data);
}

// Create user
async function createUser(userData) {
  const response = await fetch('http://localhost:8080/api/User', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ data: userData })
  });
  const result = await response.json();
  return result.data;
}
```

See `examples/rest_api_demo.html` for a complete interactive demo.

### Logging

RediORM includes comprehensive logging support:

**Sample Output (INFO level):**
```
🚀 GraphQL server ready at http://localhost:4000/graphql
🎮 GraphQL Playground available at http://localhost:4000/graphql
[GraphQL] 14:31:14 INFO: mutation createUser
[GraphQL] 14:31:14 INFO: Success in 774µs
[GraphQL] 14:31:14 INFO: query findManyUser
[GraphQL] 14:31:14 INFO: Success in 237µs
```

#### Programmatic Usage

You can also use the GraphQL handler programmatically in your Go applications:

```go
import (
    "github.com/rediwo/redi-orm/graphql"
    "github.com/rediwo/redi-orm/database"
)

// Create GraphQL handler
db, _ := database.NewFromURI("sqlite://./app.db")
schemas := loadYourSchemas()
generator := graphql.NewSchemaGenerator(db, schemas)
schema, _ := generator.Generate()
handler := graphql.NewHandler(schema)

// Use with any HTTP framework
http.Handle("/graphql", handler)
```

See the [examples](./examples) directory for complete working examples.

## 📋 Logging and Monitoring

RediORM includes comprehensive logging support to help with debugging, performance monitoring, and production troubleshooting.

### Logging Levels

RediORM supports multiple log levels with color-coded output:

- **NONE** - Disables all logging
- **ERROR** - Shows only errors (red)
- **WARN** - Shows warnings and errors (yellow)
- **INFO** - Shows general information (green) - **Default level**
- **DEBUG** - Shows detailed debug information including SQL queries (gray)

### JavaScript Logging

```javascript
const { fromUri, createLogger } = require('redi/orm');

// Create logger with custom prefix
const logger = createLogger('MyApp');

// Set log level
logger.setLevel(logger.levels.DEBUG);

// Attach to database
const db = fromUri('sqlite://./app.db');
db.setLogger(logger);

// Now all database operations will be logged:
// [MyApp] INFO: Connected to database
// [MyApp] DEBUG: SQL (1.23ms): SELECT * FROM users WHERE age > ?
```

### Go Logging

```go
import (
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/utils"
)

// Create database connection
db, _ := database.NewFromURI("sqlite://./app.db")

// Create and configure logger
logger := utils.NewDefaultLogger("MyApp")
logger.SetLevel(utils.LogLevelDebug)

// Set logger on database
db.SetLogger(logger)

// All operations will now be logged
// [MyApp] DEBUG: SQL (2.45ms): INSERT INTO users (name, email) VALUES (?, ?)
```

### CLI Logging

```bash
# Set log level for CLI operations
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma --log-level=debug
redi-orm migrate --db=sqlite://./app.db --schema=./schema.prisma --log-level=info

# Available levels: none, error, warn, info, debug
```

### Migration Warnings

RediORM automatically shows detailed warnings before applying database migrations:

```bash
# Sample migration output with warnings
$ redi-orm server --db=sqlite://./app.db --schema=./schema.prisma --log-level=info

[RediORM] WARN: Table 'users' needs migration:
[RediORM] WARN:   - Adding column: bio
[RediORM] WARN:   - Modifying column 'age': nullable: false -> true
[RediORM] WARN:   - Dropping column: old_field
[RediORM] WARN:   - Adding index: idx_users_email
[RediORM] DEBUG: SQL (1.23ms): ALTER TABLE users ADD COLUMN bio TEXT
[RediORM] DEBUG: SQL (2.45ms): CREATE INDEX idx_users_email ON users(email)
```

### GraphQL Server Logging

The GraphQL server includes request/response logging:

```bash
# Sample GraphQL server logs
$ redi-orm server --db=sqlite://./app.db --schema=./schema.prisma --log-level=info

🚀 GraphQL server ready at http://localhost:4000/graphql
🎮 GraphQL Playground available at http://localhost:4000/graphql

[GraphQL] INFO: mutation createUser
[GraphQL] INFO: Success in 3.42ms
[GraphQL] INFO: query findManyUser  
[GraphQL] INFO: Success in 1.28ms

# With debug level, also shows queries and responses (truncated to 100 chars)
[GraphQL] DEBUG: Request body: {"query":"mutation { createUser(data: {name: \"Alice\", email: \"alice@ex...
[GraphQL] DEBUG: Query: mutation { createUser(data: {name: "Alice", email: "alice@example.com"}) { id...
[GraphQL] DEBUG: Response: {"data":{"createUser":{"id":1,"name":"Alice","email":"alice@example.com"}}...
```

### Log Output Examples

**SQL Query Logging:**
```
[RediORM] DEBUG: SQL (2.45ms): SELECT id, name, email FROM users WHERE age > ? [18]
[RediORM] DEBUG: SQL (1.23ms): INSERT INTO users (name, email) VALUES (?, ?) ["Alice", "alice@example.com"]
[RediORM] INFO: Connected to SQLite database: ./app.db
```

**MongoDB Command Logging:**
```
[RediORM] DEBUG: MongoDB (3.12ms): {"find": "users", "filter": {"age": {"$gt": 18}}}
[RediORM] DEBUG: MongoDB (1.87ms): {"insertOne": "users", "document": {"name": "Alice", "email": "alice@example.com"}}
```

**Error Logging:**
```
[RediORM] ERROR: Failed to connect to database: connection refused
[GraphQL] ERROR: Query failed: Field 'invalidField' not found in schema
```

### Best Practices

1. **Production**: Use `INFO` level for general monitoring
2. **Development**: Use `DEBUG` level to see all SQL queries
3. **CI/CD**: Use `WARN` level to catch issues without noise
4. **Performance Testing**: Use `DEBUG` to monitor query execution times
5. **Troubleshooting**: Use `DEBUG` to see exact SQL being generated

## 🏗️ Architecture

### Directory Structure

```
redi-orm/
├── database/          # Database abstraction layer
├── drivers/           # Database driver implementations
│   ├── base/         # Shared driver functionality
│   ├── sqlite/       # SQLite driver
│   ├── mysql/        # MySQL driver
│   ├── postgresql/   # PostgreSQL driver
│   └── mongodb/      # MongoDB driver
├── graphql/          # GraphQL API generator with relation support and logging
│   ├── handler.go    # HTTP handler with configurable logging (framework-agnostic)
│   ├── schema_generator.go # GraphQL schema generation with relation fields
│   ├── resolver.go   # Auto-generated resolvers with field mapping
│   └── server.go     # GraphQL server implementation
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
6. **GraphQL Server** - Automatic API generation from schemas with full CRUD support, relation fields, and configurable logging

## 📊 Database Support

| Feature | SQLite | MySQL | PostgreSQL | MongoDB |
|---------|--------|-------|------------|---------|
| Basic CRUD | ✅ | ✅ | ✅ | ✅ |
| Transactions | ✅ | ✅ | ✅ | ✅* |
| Migrations | ✅ | ✅ | ✅ | ❌ |
| Raw Queries | ✅ | ✅ | ✅ | ✅ |
| Field Mapping | ✅ | ✅ | ✅ | ✅ |
| Relations | ✅ | ✅ | ✅ | ✅ |
| Savepoints | ✅ | ✅ | ✅ | ❌ |
| Nested Documents | ❌ | ❌ | ❌ | ✅ |
| Array Fields | 🔧 | 🔧 | ✅ | ✅ |
| Aggregation Pipeline | ❌ | ❌ | ❌ | ✅ |
| GroupBy/Having | ✅ | ✅ | ✅ | ✅ |

> 🔧 = Partial support, ❌ = Not supported, ✅* = Supported with limitations

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

### Relations

Define relations in your schema:

```javascript
model User {
    id    Int     @id @default(autoincrement())
    email String  @unique
    posts Post[]  // One-to-many relation
}

model Post {
    id       Int      @id @default(autoincrement())
    title    String
    userId   Int
    user     User     @relation(fields: [userId], references: [id])
    comments Comment[]
}

model Comment {
    id     Int    @id @default(autoincrement())
    text   String
    postId Int
    post   Post   @relation(fields: [postId], references: [id])
}
```

Query with relations:

```javascript
// Find users with their posts
const users = await db.models.User.findMany({
    include: {
        posts: true
    }
});

// Find posts with user and comments
const posts = await db.models.Post.findMany({
    include: {
        user: true,
        comments: true
    }
});

// Filter by relation fields
const userPosts = await db.models.Post.findMany({
    where: {
        userId: 1
    }
});

// Count related records
const postCount = await db.models.Post.count({
    where: {
        userId: 1
    }
});
```

### Smart Scanning

```go
// Scan into structs
var users []User
err := db.Model("User").FindMany(ctx, &users)

// Scan into maps (automatic)
var results []map[string]any
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

The project includes comprehensive test coverage with a unified conformance test suite ensuring consistent behavior across all database drivers.

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
make test-short      # Skip long-running tests
```

### Type Conversion Utilities

RediORM provides safe type conversion utilities to handle driver-specific differences:

```go
import "github.com/rediwo/redi-orm/utils"

// Handle different driver representations
boolValue := utils.ToBool(result["active"])      // SQLite returns int64, MySQL returns bool
intValue := utils.ToInt64(result["count"])       // MySQL may return string for aggregates
floatValue := utils.ToFloat64(result["average"]) // Handle various numeric types
```

### MongoDB Support

RediORM includes full MongoDB support with special features for document databases:

#### Connection
```javascript
// MongoDB connection
const db = fromUri('mongodb://localhost:27017/myapp');
// MongoDB SRV connection  
const db = fromUri('mongodb+srv://user:pass@cluster.mongodb.net/myapp');
```

#### Document Schema
```javascript
await db.loadSchema(`
    model User {
        id      String   @id @default(auto()) @map("_id") @db.ObjectId
        email   String   @unique
        profile Json?    // Nested document
        tags    String[] // Array field
    }
`);
```

#### Nested Queries
```javascript
// Query nested fields
const users = await db.models.User.findMany({
    where: {
        'profile.location.city': 'San Francisco',
        tags: { in: ['developer'] }
    }
});

// Update nested fields
await db.models.User.update({
    where: { id: userId },
    data: {
        'profile.bio': 'Updated bio',
        tags: { push: 'mongodb' }
    }
});
```

#### Aggregation Pipeline
```javascript
// Raw aggregation pipeline
const results = await db.queryRaw(`{
    "aggregate": "users",
    "pipeline": [
        { "$match": { "age": { "$gte": 18 } } },
        { "$group": {
            "_id": "$location",
            "count": { "$sum": 1 },
            "avgAge": { "$avg": "$age" }
        }}
    ]
}`);
```

### MongoDB-Specific Features and Limitations

#### Supported Features
- ✅ Full CRUD operations with automatic ID generation
- ✅ SQL to MongoDB query translation
- ✅ Native MongoDB command execution
- ✅ Basic transactions (requires replica set)
- ✅ Field mapping with `_id` handling
- ✅ Distinct queries
- ✅ Full aggregation operations (COUNT, SUM, AVG, MIN, MAX)
- ✅ GroupBy and Having clauses with aggregation pipeline
- ✅ Index management
- ✅ String operators (startsWith, endsWith, contains) with regex
- ✅ Schema evolution support with nullable field handling
- ✅ Collection existence validation

#### Limitations
- ❌ Savepoints not supported
- ❌ Migrations not supported (schemaless database)

#### Best Practices
1. Use SQL syntax for simple queries
2. Use native MongoDB commands for complex aggregations
3. Ensure MongoDB replica set is configured for transactions
4. Test string matching carefully due to regex differences

## 🛠️ Development

### Local Development

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

### Docker Development

The project includes a docker-compose setup for testing with real databases:

```bash
# Start all databases (MySQL, PostgreSQL, MongoDB)
make docker-up

# Wait for databases to be ready
make docker-wait

# Run tests with Docker databases
make test-docker

# Stop databases
make docker-down
```

Database credentials (all databases use the same):
- User: `testuser`
- Password: `testpass`
- Database: `testdb`

Connection strings:
```bash
# MySQL
mysql://testuser:testpass@localhost:3306/testdb

# PostgreSQL
postgresql://testuser:testpass@localhost:5432/testdb

# MongoDB (with replica set for transactions)
mongodb://testuser:testpass@localhost:27017/testdb?authSource=admin

# Build with version injection
make release-build

# Show current version
make version
```


## 🚦 Roadmap

- [x] Core database operations (CRUD, transactions, raw SQL)
- [x] Schema management (Prisma-style definitions)
- [x] Migration system (auto & file-based)
- [x] JavaScript runtime with ORM interface
- [x] Multi-database support (SQLite, MySQL, PostgreSQL, MongoDB)
- [x] Relations support (one-to-one, one-to-many, many-to-many)
- [x] Advanced relation features (eager loading, nested writes, filtering)
- [x] CLI tool with timeout support
- [x] GitHub Actions CI/CD pipeline
- [x] Multi-platform binary releases
- [x] GraphQL integration with relation support and configurable logging
- [ ] Query optimization and caching
- [ ] Connection pooling configuration
- [ ] Middleware/plugin system
- [ ] Database introspection tools
- [ ] Performance monitoring and metrics

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