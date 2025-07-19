# RediORM

A modern, AI-native ORM for Go with Prisma-like JavaScript interface. RediORM bridges the gap between traditional database access and modern AI applications through sophisticated schema management and the Model Context Protocol (MCP).

## ✨ Key Features

- **🤖 AI-Native Design** - First-class MCP support for seamless AI assistant integration
- **🔄 Dual API** - Type-safe Go API + Prisma-like JavaScript interface  
- **🗄️ Multi-Database** - SQLite, MySQL, PostgreSQL, MongoDB with unified API
- **📊 Auto-Generated APIs** - GraphQL and REST servers from your schema
- **🔗 Smart Relations** - Eager loading, nested queries, relation management
- **🚀 Production Ready** - Migrations, transactions, connection pooling, logging

## 🚀 Quick Start

### Installation

```bash
# Install CLI tool
go install github.com/rediwo/redi-orm/cmd/redi-orm@latest

# Or download pre-built binary
wget https://github.com/rediwo/redi-orm/releases/latest/download/redi-orm-linux-amd64.tar.gz
```

### Define Your Schema

```prisma
// schema.prisma
model User {
  id    Int     @id @default(autoincrement())
  email String  @unique
  name  String
  posts Post[]
}

model Post {
  id      Int    @id @default(autoincrement())
  title   String
  content String?
  userId  Int
  user    User   @relation(fields: [userId], references: [id])
}
```

### Go API

```go
package main

import (
    "context"
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/orm"
    _ "github.com/rediwo/redi-orm/drivers/sqlite"
)

func main() {
    ctx := context.Background()
    
    // Connect and load schema
    db, _ := database.NewFromURI("sqlite://./app.db")
    db.Connect(ctx)
    db.LoadSchemaFrom(ctx, "./schema.prisma")
    db.SyncSchemas(ctx)
    
    // Use ORM
    client := orm.NewClient(db)
    user, _ := client.Model("User").Create(`{
        "data": {
            "name": "Alice",
            "email": "alice@example.com"
        }
    }`)
}
```

### JavaScript API

```javascript
const { fromUri } = require('redi/orm');

async function main() {
    const db = fromUri('sqlite://./app.db');
    await db.connect();
    await db.loadSchemaFrom('./schema.prisma');
    await db.syncSchemas();
    
    // Create user with posts
    const user = await db.models.User.create({
        data: {
            name: "Alice",
            email: "alice@example.com",
            posts: {
                create: [
                    { title: "Hello World", content: "My first post!" }
                ]
            }
        }
    });
    
    // Query with relations
    const users = await db.models.User.findMany({
        include: { posts: true },
        where: { email: { contains: "@example.com" } }
    });
}
```

## 🤖 AI Integration (MCP)

RediORM provides comprehensive Model Context Protocol support, enabling AI assistants to understand and manipulate your database through intelligent, schema-aware operations:

```bash
# Start MCP server for AI assistants
redi-orm mcp --db=sqlite://./app.db --schema=./schema.prisma

# With security for production
redi-orm mcp \
  --db=postgresql://readonly:pass@localhost/db \
  --enable-auth \
  --read-only \
  --allowed-tables=users,posts
```

**AI Can Now:**
- 🔍 **Discover Models** - "What models do I have in my database?"
- 🔨 **Create Models** - "I need a model for tracking orders"
- 🔎 **Smart Queries** - "Find all users who have published posts"
- ⚡ **Optimize Performance** - "This query is slow, how can I improve it?"

## 🌐 Auto-Generated APIs

### GraphQL Server

```bash
# Start GraphQL + REST API server
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma
# GraphQL: http://localhost:4000/graphql
# REST API: http://localhost:4000/api
```

### Example GraphQL Query

```graphql
query {
  findManyUser(
    where: { email: { contains: "@example.com" } }
    include: { posts: true }
  ) {
    id
    name
    email
    posts {
      title
      content
    }
  }
}
```

## 🗄️ Multi-Database Support

| Feature | SQLite | MySQL | PostgreSQL | MongoDB |
|---------|--------|-------|------------|---------|
| CRUD Operations | ✅ | ✅ | ✅ | ✅ |
| Relations | ✅ | ✅ | ✅ | ✅ |
| Transactions | ✅ | ✅ | ✅ | ✅ |
| Migrations | ✅ | ✅ | ✅ | ❌ |
| Aggregations | ✅ | ✅ | ✅ | ✅ |
| Raw Queries | ✅ | ✅ | ✅ | ✅ + MongoDB commands |

## 🔧 CLI Commands

```bash
# Run JavaScript with ORM
redi-orm run script.js

# Database migrations
redi-orm migrate --db=sqlite://./app.db --schema=./schema.prisma

# Start servers
redi-orm server --db=sqlite://./app.db    # GraphQL + REST
redi-orm mcp --db=sqlite://./app.db       # MCP for AI
```

## 📚 Documentation

### Getting Started
- **[Complete Guide](./doc/getting-started.md)** - Installation, configuration, first project
- **[API Reference](./doc/api-reference.md)** - Go, JavaScript, and CLI documentation

### Advanced Usage  
- **[Database Guide](./doc/database-guide.md)** - Multi-database setup and features
- **[Advanced Features](./doc/advanced-features.md)** - Relations, transactions, aggregations
- **[APIs & Servers](./doc/apis-and-servers.md)** - GraphQL, REST, and MCP servers


## 🎯 Why RediORM?

**Traditional ORMs** focus on mapping objects to database tables.

**RediORM** is designed for the AI era - where databases need to be **understandable** and **manipulable** by AI systems, while maintaining full type safety and performance for human developers.

- **Schema-Aware AI** - AI understands your data models, not just SQL tables
- **Unified Interface** - Same API across all databases (SQL + NoSQL)  
- **Production Ready** - Built-in servers, security, monitoring
- **Developer Friendly** - Prisma-like syntax developers already know

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Ready to build AI-native applications?** Start with our [Getting Started Guide](./doc/getting-started.md) or explore the [MCP Guide](./doc/mcp-guide.md) for AI integration.