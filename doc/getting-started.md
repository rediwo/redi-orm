# Getting Started with RediORM

A complete guide to installing, configuring, and creating your first RediORM project.

## Prerequisites

- **Go 1.19+** for Go API usage
- **Node.js 16+** for JavaScript API (when using CLI)
- **Database**: SQLite (no setup), MySQL, PostgreSQL, or MongoDB

## Installation

### Method 1: Install CLI Tool (Recommended)

```bash
# Install from Go
go install github.com/rediwo/redi-orm/cmd/redi-orm@latest

# Verify installation
redi-orm --version
```

### Method 2: Download Pre-built Binary

```bash
# Linux AMD64
wget https://github.com/rediwo/redi-orm/releases/latest/download/redi-orm-linux-amd64.tar.gz
tar -xzf redi-orm-linux-amd64.tar.gz
sudo mv redi-orm /usr/local/bin/

# macOS AMD64
wget https://github.com/rediwo/redi-orm/releases/latest/download/redi-orm-darwin-amd64.tar.gz

# macOS ARM64 (Apple Silicon)
wget https://github.com/rediwo/redi-orm/releases/latest/download/redi-orm-darwin-arm64.tar.gz

# Windows AMD64
wget https://github.com/rediwo/redi-orm/releases/latest/download/redi-orm-windows-amd64.zip
```

### Method 3: Go Module Integration

```bash
# Add to your Go project
go mod init myproject
go get github.com/rediwo/redi-orm
```

## Quick Start Project

### 1. Create Project Structure

```bash
mkdir my-rediorm-project
cd my-rediorm-project

# Create schema file
touch schema.prisma

# Create main script
touch main.js
```

### 2. Define Your Schema

Create `schema.prisma`:

```prisma
// schema.prisma
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String
  createdAt DateTime @default(now())
  posts     Post[]
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  published Boolean  @default(false)
  createdAt DateTime @default(now())
  authorId  Int
  author    User     @relation(fields: [authorId], references: [id])
}
```

### 3. Create Your First Script

Create `main.js`:

```javascript
const { fromUri, createLogger } = require('redi/orm');

async function main() {
    // Create logger
    const logger = createLogger('MyApp');
    logger.setLevel(logger.levels.INFO);
    
    // Connect to database
    const db = fromUri('sqlite://./app.db');
    db.setLogger(logger);
    
    try {
        // Connect and setup
        await db.connect();
        await db.loadSchemaFrom('./schema.prisma');
        await db.syncSchemas();
        
        logger.info('Database connected and schemas synchronized');
        
        // Create a user with posts
        const user = await db.models.User.create({
            data: {
                name: 'Alice Johnson',
                email: 'alice@example.com',
                posts: {
                    create: [
                        {
                            title: 'Getting Started with RediORM',
                            content: 'This is my first post using RediORM!',
                            published: true
                        },
                        {
                            title: 'Advanced Features',
                            content: 'Exploring relations and transactions...',
                            published: false
                        }
                    ]
                }
            }
        });
        
        console.log('Created user:', user);
        
        // Query users with their posts
        const usersWithPosts = await db.models.User.findMany({
            include: { posts: true },
            where: { email: { contains: '@example.com' } }
        });
        
        console.log('Users with posts:', JSON.stringify(usersWithPosts, null, 2));
        
        // Update a post
        await db.models.Post.updateMany({
            where: { published: false },
            data: { published: true }
        });
        
        console.log('Published all draft posts');
        
    } catch (error) {
        logger.error('Error:', error.message);
    } finally {
        await db.disconnect();
    }
}

main().catch(console.error);
```

### 4. Run Your Project

```bash
# Run the script
redi-orm run main.js

# Expected output:
# [MyApp] INFO: Connected to SQLite database
# [MyApp] INFO: Database connected and schemas synchronized
# Created user: { id: 1, name: 'Alice Johnson', email: 'alice@example.com', ... }
# Users with posts: [...]
# Published all draft posts
```

## Database Configuration

### SQLite (Default - No Setup Required)

```javascript
// In-memory database (for testing)
const db = fromUri('sqlite://:memory:');

// File-based database
const db = fromUri('sqlite://./app.db');
const db = fromUri('sqlite:///absolute/path/to/database.db');
```

### MySQL Setup

1. **Install MySQL** (MySQL 8.0+ recommended)
2. **Create database and user**:

```sql
CREATE DATABASE myapp;
CREATE USER 'myuser'@'localhost' IDENTIFIED BY 'mypassword';
GRANT ALL PRIVILEGES ON myapp.* TO 'myuser'@'localhost';
FLUSH PRIVILEGES;
```

3. **Connect in your application**:

```javascript
const db = fromUri('mysql://myuser:mypassword@localhost:3306/myapp?charset=utf8mb4&parseTime=true');
```

### PostgreSQL Setup

1. **Install PostgreSQL** (PostgreSQL 12+ recommended)
2. **Create database and user**:

```sql
CREATE DATABASE myapp;
CREATE USER myuser WITH PASSWORD 'mypassword';
GRANT ALL PRIVILEGES ON DATABASE myapp TO myuser;
```

3. **Connect in your application**:

```javascript
const db = fromUri('postgresql://myuser:mypassword@localhost:5432/myapp?sslmode=prefer');
```

### MongoDB Setup

1. **Install MongoDB** (MongoDB 4.4+ recommended)
2. **Start MongoDB** with replica set (required for transactions):

```bash
# Start with replica set
mongod --replSet rs0

# Initialize replica set (run once)
mongosh --eval "rs.initiate()"
```

3. **Connect in your application**:

```javascript
// Local MongoDB
const db = fromUri('mongodb://localhost:27017/myapp');

// MongoDB Atlas
const db = fromUri('mongodb+srv://username:password@cluster.mongodb.net/myapp');

// With authentication
const db = fromUri('mongodb://myuser:mypassword@localhost:27017/myapp?authSource=admin');
```

## Go API Usage

### 1. Create Go Project

```bash
mkdir my-go-rediorm
cd my-go-rediorm
go mod init my-go-rediorm
go get github.com/rediwo/redi-orm
```

### 2. Basic Go Implementation

Create `main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/orm"
    _ "github.com/rediwo/redi-orm/drivers/sqlite"
)

func main() {
    ctx := context.Background()
    
    // Connect to database
    db, err := database.NewFromURI("sqlite://./app.db")
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    defer db.Close()
    
    err = db.Connect(ctx)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    
    // Load schema
    err = db.LoadSchemaFrom(ctx, "./schema.prisma")
    if err != nil {
        log.Fatal("Failed to load schema:", err)
    }
    
    // Sync schemas
    err = db.SyncSchemas(ctx)
    if err != nil {
        log.Fatal("Failed to sync schemas:", err)
    }
    
    // Create ORM client
    client := orm.NewClient(db)
    
    // Create user
    userResult, err := client.Model("User").Create(`{
        "data": {
            "name": "Bob Smith",
            "email": "bob@example.com"
        }
    }`)
    if err != nil {
        log.Fatal("Failed to create user:", err)
    }
    
    fmt.Printf("Created user: %v\n", userResult)
    
    // Find users
    users, err := client.Model("User").FindMany(`{
        "where": { "email": { "contains": "@example.com" } }
    }`)
    if err != nil {
        log.Fatal("Failed to find users:", err)
    }
    
    fmt.Printf("Found users: %v\n", users)
}
```

### 3. Run Go Application

```bash
go run main.go
```

## Development Workflow

### 1. Schema Development

```bash
# Edit your schema.prisma file
# Then sync changes to database
redi-orm migrate --db=sqlite://./app.db --schema=./schema.prisma
```

### 2. Interactive Development

```bash
# Start a development server with APIs
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma --log-level=debug

# Access your APIs:
# GraphQL Playground: http://localhost:4000/graphql
# REST API: http://localhost:4000/api
```

### 3. Testing with Different Databases

```bash
# Test with SQLite
redi-orm run main.js

# Test with MySQL (ensure MySQL is running)
redi-orm run main.js --db=mysql://user:pass@localhost/testdb

# Test with PostgreSQL
redi-orm run main.js --db=postgresql://user:pass@localhost/testdb
```

## Best Practices

### 1. Environment Configuration

Create a `.env` file:

```bash
# .env
DATABASE_URL=sqlite://./app.db
LOG_LEVEL=info
```

Use in your scripts:

```javascript
require('dotenv').config();

const db = fromUri(process.env.DATABASE_URL || 'sqlite://./app.db');
const logger = createLogger('MyApp');
logger.setLevel(process.env.LOG_LEVEL || 'info');
```

### 2. Error Handling

```javascript
async function robustDatabaseOperation() {
    let db;
    try {
        db = fromUri('sqlite://./app.db');
        await db.connect();
        
        // Your database operations
        const result = await db.models.User.create({
            data: { name: 'Test User', email: 'test@example.com' }
        });
        
        return result;
    } catch (error) {
        console.error('Database operation failed:', error);
        throw error;
    } finally {
        if (db) {
            await db.disconnect();
        }
    }
}
```

### 3. Transaction Best Practices

```javascript
// Always use transactions for related operations
await db.transaction(async (tx) => {
    const user = await tx.models.User.create({
        data: { name: 'Alice', email: 'alice@example.com' }
    });
    
    await tx.models.Post.create({
        data: {
            title: 'Hello World',
            authorId: user.id
        }
    });
    
    // Both operations succeed or both fail
});
```

## Next Steps

1. **Explore Relations**: Learn about [Advanced Features](./advanced-features.md)
2. **API Development**: Set up [GraphQL and REST servers](./apis-and-servers.md)
3. **AI Integration**: Configure [MCP for AI assistants](./mcp-guide.md)
4. **Production Setup**: Review [database-specific configurations](./database-guide.md)
5. **Contributing**: Open issues or PRs on the GitHub repository

## Troubleshooting

### Common Issues

**Issue**: `Command not found: redi-orm`
- **Solution**: Ensure `$GOPATH/bin` is in your `$PATH`, or use full path to binary

**Issue**: `Failed to connect to database`
- **Solution**: Verify database is running and connection string is correct

**Issue**: `Schema validation failed`
- **Solution**: Check your `schema.prisma` syntax and field types

**Issue**: `JavaScript runtime error`
- **Solution**: Always use `redi-orm run script.js`, not `node script.js`

### Getting Help

1. Check the [API Reference](./api-reference.md)
2. Review [database-specific guides](./database-guide.md)
3. See examples in the repository
4. Open an issue on GitHub

---

Ready to build something amazing? Continue with our [API Reference](./api-reference.md) or jump into [Advanced Features](./advanced-features.md)!