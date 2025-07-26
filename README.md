# RediORM

A modern, AI-native ORM for Go with Prisma-like JavaScript interface. RediORM bridges the gap between traditional database access and modern AI applications through sophisticated schema management and the Model Context Protocol (MCP).

## ✨ Key Features

- **🤖 AI-Native Design** - First-class MCP support for seamless AI assistant integration
- **🔄 Dual API** - Type-safe Go API + Prisma-like JavaScript interface  
- **🗄️ Multi-Database** - SQLite, MySQL, PostgreSQL, MongoDB with unified API
- **📊 Auto-Generated APIs** - GraphQL and REST servers from your schema
- **🔗 Smart Relations** - Eager loading, nested queries, relation management
- **🚀 Production Ready** - Migrations, transactions, connection pooling, logging
- **📋 Schema Management** - Prisma-compatible schemas with auto-generation from databases
- **🎨 Developer Experience** - Color-coded logging, helpful error messages, consistent API

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
    
    // Create user
    user, _ := client.Model("User").Create(`{
        "data": {
            "name": "Alice",
            "email": "alice@example.com"
        }
    }`)
    
    // Find many with complex queries
    users, _ := client.Model("User").FindMany(`{
        "where": {
            "email": { "contains": "@example.com" }
        },
        "include": { "posts": true },
        "orderBy": { "name": "asc" }
    }`)
    
    // Advanced query with OR conditions
    adminsOr25, _ := client.Model("User").FindMany(`{
        "where": {
            "OR": [
                {"age": 25},
                {"role": "admin"}
            ]
        }
    }`)
    
    // Query with operators
    products, _ := client.Model("Product").FindMany(`{
        "where": {
            "AND": [
                {"price": {"gte": 100, "lte": 500}},
                {"name": {"startsWith": "Pro"}}
            ]
        },
        "orderBy": {"price": "desc"},
        "take": 10
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
        where: { email: { contains: "@example.com" } },
        include: { posts: true },
        orderBy: { name: "asc" }
    });
    
    // Advanced query with OR conditions
    const adminsOr25 = await db.models.User.findMany({
        where: {
            OR: [
                { age: 25 },
                { role: "admin" }
            ]
        }
    });
    
    // Complex query with operators
    const products = await db.models.Product.findMany({
        where: {
            AND: [
                { price: { gte: 100, lte: 500 } },
                { name: { startsWith: "Pro" } }
            ]
        },
        orderBy: { price: "desc" },
        take: 10
    });
}
```

### Query Operators

RediORM supports a rich set of query operators:

- **Comparison**: `equals`, `gt`, `gte`, `lt`, `lte`
- **List**: `in`, `notIn`
- **String**: `contains`, `startsWith`, `endsWith`
- **Logical**: `AND`, `OR`, `NOT`
- **Pagination**: `take`, `skip`
- **Sorting**: `orderBy` (with `asc`/`desc`)
- **Relations**: `include` (with nested support)

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
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma    # GraphQL + REST
redi-mcp --db=sqlite://./app.db --schema=./schema.prisma           # MCP for AI
```

## 🤖 AI Integration with MCP

### Model Context Protocol (MCP) Server

RediORM includes a built-in MCP server that enables AI assistants to understand and interact with your database through natural language.

```bash
# Install and run MCP server
go install github.com/rediwo/redi-orm/cmd/redi-mcp@latest

# Stdio mode (for Claude Desktop)
redi-mcp --db=sqlite://./app.db --schema=./schema.prisma

# HTTP streaming mode (for Cursor, Windsurf, web apps)
redi-mcp --db=sqlite://./app.db --schema=./schema.prisma --port=8080

# Production configuration with security
redi-mcp --db=postgresql://readonly:pass@localhost/myapp --schema=./prisma \
  --port=8080 \
  --log-level=info \
  --read-only=true \
  --rate-limit=100
```

### MCP Features

- **Natural Language Queries** - AI can understand requests like "find all users who posted this week"
- **Schema Understanding** - AI comprehends your data model relationships and constraints
- **Safe Operations** - Read-only mode by default, with granular permission control
- **Tool Integration** - Works with Claude, GitHub Copilot, and other MCP-compatible AI assistants

### MCP Server Modes

#### 1. Stdio Mode (Default)
Best for desktop AI applications like Claude Desktop:

```json
// Claude Desktop config (~/.claude/claude_desktop_config.json)
{
  "mcpServers": {
    "database": {
      "command": "redi-mcp",
      "args": ["--db=postgresql://localhost/myapp", "--schema=./prisma"]
    }
  }
}
```

#### 2. HTTP Streaming Mode
For web-based AI tools like Cursor, Windsurf, and remote access. HTTP mode uses streaming by default:

```bash
# Start HTTP server (streaming mode by default)
redi-mcp --db=postgresql://localhost/myapp --schema=./prisma --port=8080
```

Configure in Cursor/Windsurf:
```json
// .cursor/config.json or .windsurf/config.json
{
  "mcpServers": {
    "orm-mcp": {
      "url": "http://localhost:8080"
    }
  }
}
```

HTTP endpoints:
- `/` - Streaming MCP protocol endpoint
- `/sse` - Server-Sent Events endpoint

### What AI Assistants Can Do

Now your AI assistant can:
- Query data using natural language - "Show me users who signed up this month"
- Generate reports and analytics - "Create a weekly sales summary"
- Suggest schema improvements - "What indexes would improve performance?"
- Help debug data issues - "Why are some orders missing user references?"

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