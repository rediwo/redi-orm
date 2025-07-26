# RediORM MCP (Model Context Protocol) Guide

## Overview

RediORM MCP is a Model Context Protocol server that enables AI assistants to interact with databases through natural language. It provides a secure, schema-aware interface for database operations using the official MCP SDK.

## What is MCP?

Model Context Protocol (MCP) is an open standard that allows AI assistants to interact with external tools and services. RediORM MCP implements this protocol to provide database capabilities to AI assistants.

### Key Features

- **ORM-Based Operations**: All database interactions use RediORM's Prisma-style query interface
- **Schema Intelligence**: Automatic schema discovery from existing database tables
- **Multi-Database Support**: SQLite, MySQL, PostgreSQL, and MongoDB
- **Secure by Default**: Authentication and rate limiting features available
- **Transport Flexibility**: Stdio for local AI assistants, HTTP for remote access
- **Schema Modification**: Create and modify database schemas through AI interactions
- **Transaction Support**: Execute multiple operations atomically

## Installation

### Download Binary

Download the appropriate binary from the [releases page](https://github.com/rediwo/redi-orm/releases):

```bash
# macOS (Apple Silicon)
wget https://github.com/rediwo/redi-orm/releases/latest/download/redi-mcp-darwin-arm64.tar.gz
tar -xzf redi-mcp-darwin-arm64.tar.gz
chmod +x redi-mcp
sudo mv redi-mcp /usr/local/bin/

# Linux (AMD64)
wget https://github.com/rediwo/redi-orm/releases/latest/download/redi-mcp-linux-amd64.tar.gz
tar -xzf redi-mcp-linux-amd64.tar.gz
chmod +x redi-mcp
sudo mv redi-mcp /usr/local/bin/

# Windows
# Download redi-mcp-windows-amd64.zip
# Extract and add to PATH
```

### Build from Source

```bash
git clone https://github.com/rediwo/redi-orm.git
cd redi-orm
go build -o redi-mcp ./cmd/redi-mcp
```

## Quick Start

### 1. Basic Setup

```bash
# Start with SQLite (stdio mode for local AI)
redi-mcp --db=sqlite://./myapp.db --schema=./schemas/

# Start with existing database
redi-mcp --db=mysql://user:pass@localhost:3306/mydb --schema=./schemas/

# Start HTTP server for remote access
redi-mcp --db=postgresql://user:pass@localhost/db --schema=./schemas/ --port=3000
```

### 2. Test the Server

For stdio mode, send JSON-RPC commands:
```json
{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}
```

For HTTP mode:
```bash
curl -X POST http://localhost:3000/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}'
```

## Command Line Options

```bash
redi-mcp [flags]

Required:
  --db              Database URI
                    Examples:
                    - sqlite://./myapp.db
                    - mysql://user:pass@localhost:3306/dbname
                    - postgresql://user:pass@localhost:5432/dbname
                    - mongodb://localhost:27017/dbname

Optional:
  --schema          Path to schema file or directory (default: ./schema.prisma)
                    Supports Prisma-style schema definitions
                    If directory: loads all .prisma files

  --port            Enable HTTP server on specified port
                    Default: 0 (stdio mode for local AI assistants)
                    Example: --port=3000

  --log-level       Logging level (debug|info|warn|error|none)
                    Default: info

Security:
  --api-key         API key for HTTP transport authentication
  --enable-auth     Enable authentication for HTTP transport
  --read-only       Enable read-only mode (default: false)
  --rate-limit      Requests per minute rate limit (default: 60)

Other:
  --help            Show help message
  --version         Show version information
```

## Transport Modes

### Stdio Mode (Default)

Used for local AI assistants like Claude Desktop. Communication happens via standard input/output using JSON-RPC.

```bash
redi-mcp --db=sqlite://./app.db --schema=./schemas/
```

**Characteristics:**
- No network exposure (most secure)
- Logs output to stderr to avoid polluting JSON-RPC stream
- Perfect for desktop AI applications

### HTTP Mode

Enable by specifying a port. Used for remote AI assistants and web applications.

```bash
redi-mcp --db=postgresql://user:pass@localhost/db --port=3000
```

**Endpoints:**
- `POST /` - JSON-RPC endpoint for MCP communication
- `GET /sse` - Server-Sent Events for streaming updates

## AI Assistant Integration

### Claude Desktop

Add to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "database": {
      "command": "redi-mcp",
      "args": [
        "--db=sqlite:///path/to/your/database.db",
        "--schema=./schemas/",
        "--log-level=info"
      ]
    }
  }
}
```

### Cursor

Add to `.cursor/config.json` in your project:

```json
{
  "mcpServers": {
    "database": {
      "url": "http://localhost:3000"
    }
  }
}
```

Then start the MCP server:
```bash
redi-mcp --db=sqlite://./app.db --schema=./schemas/ --port=3000
```

### Windsurf

Similar to Cursor, add to `.windsurf/config.json`:

```json
{
  "mcpServers": {
    "database": {
      "url": "http://localhost:3001"
    }
  }
}
```

Then start the MCP server:
```bash
redi-mcp --db=postgresql://user:pass@localhost/db --schema=./schemas/ --port=3001
```

## Available Tools

### Model Operations

#### model.findMany
Query multiple records with Prisma-style filters.

```json
{
  "model": "User",
  "where": {"active": true},
  "include": {"posts": true},
  "orderBy": {"createdAt": "desc"},
  "take": 10,
  "skip": 0
}
```

#### model.findUnique
Find a single record by unique field.

```json
{
  "model": "User",
  "where": {"id": 1},
  "include": {"posts": true}
}
```

#### model.create
Create a new record (requires `--read-only=false`).

```json
{
  "model": "User",
  "data": {
    "name": "Alice",
    "email": "alice@example.com"
  }
}
```

#### model.update
Update existing records (requires `--read-only=false`).

```json
{
  "model": "User",
  "where": {"id": 1},
  "data": {"name": "Alice Smith"}
}
```

#### model.delete
Delete records (requires `--read-only=false`).

```json
{
  "model": "User",
  "where": {"id": 1}
}
```

#### model.count
Count records with optional filters.

```json
{
  "model": "User",
  "where": {"active": true}
}
```

#### model.aggregate
Perform aggregation queries.

```json
{
  "model": "Order",
  "where": {"status": "completed"},
  "sum": {"amount": true},
  "avg": {"amount": true},
  "groupBy": ["customerId"]
}
```

### Schema Operations

#### schema.models
List all models with their fields and relationships.

#### schema.describe
Get detailed information about a specific model.

```json
{
  "model": "User"
}
```

#### schema.create
Create a new model schema (requires `--read-only=false`).

```json
{
  "model": "Product",
  "fields": [
    {"name": "id", "type": "Int", "primaryKey": true, "autoIncrement": true},
    {"name": "name", "type": "String"},
    {"name": "price", "type": "Float"},
    {"name": "categoryId", "type": "Int"}
  ],
  "relations": [
    {
      "name": "category",
      "type": "manyToOne",
      "model": "Category",
      "foreignKey": "categoryId",
      "references": "id"
    }
  ]
}
```

#### schema.update
Update an existing model schema (requires `--read-only=false`).

```json
{
  "model": "User",
  "addFields": [
    {"name": "bio", "type": "String", "optional": true}
  ],
  "removeFields": ["oldField"]
}
```

#### schema.addField
Add a field to an existing model (requires `--read-only=false`).

```json
{
  "model": "User",
  "field": {
    "name": "avatar",
    "type": "String",
    "optional": true
  }
}
```

#### schema.removeField
Remove a field from a model (requires `--read-only=false`).

```json
{
  "model": "User",
  "fieldName": "deprecatedField"
}
```

#### schema.addRelation
Add a relation between models (requires `--read-only=false`).

```json
{
  "model": "Post",
  "relation": {
    "name": "comments",
    "type": "oneToMany",
    "model": "Comment",
    "foreignKey": "postId",
    "references": "id"
  }
}
```

### Migration Operations

#### migration.create
Create a new migration based on schema changes.

```json
{
  "name": "add_user_bio",
  "preview": true
}
```

#### migration.apply
Apply pending migrations to the database.

```json
{
  "dry_run": false
}
```

#### migration.status
Show current migration status.

### Transaction Support

#### transaction
Execute multiple operations in a transaction (requires `--read-only=false`).

```json
{
  "operations": [
    {
      "tool": "model.create",
      "arguments": {
        "model": "User",
        "data": {"name": "Alice", "email": "alice@example.com"}
      }
    },
    {
      "tool": "model.create",
      "arguments": {
        "model": "Post",
        "data": {"title": "Hello", "authorEmail": "alice@example.com"}
      }
    }
  ]
}
```

## Security Best Practices

> **⚠️ WARNING**: By default, `--read-only` is `false`, meaning write operations are allowed! Always enable `--read-only` for production use unless you specifically need write access.

### 1. Read-Only Mode

When enabled with `--read-only`, only query operations are allowed:
- `model.findMany`, `model.findUnique`, `model.count`, `model.aggregate`
- `schema.models`, `schema.describe`
- `migration.status`

**IMPORTANT**: By default, write operations are allowed. To restrict to read-only:
```bash
redi-mcp --db=sqlite://./app.db --schema=./schemas/ --read-only
```

### 2. Authentication (HTTP Mode)

Enable API key authentication for HTTP transport:
```bash
redi-mcp --db=postgresql://user:pass@localhost/db \
  --schema=./schemas/ \
  --port=3000 \
  --enable-auth \
  --api-key=your-secret-key
```

### 3. Rate Limiting

Protect against excessive queries:
```bash
redi-mcp --db=sqlite://./app.db --schema=./schemas/ --rate-limit=30
```

### 4. Production Setup

```bash
# Use environment variables for sensitive data
export DB_URI="postgresql://readonly:pass@localhost/production"
export MCP_API_KEY="$(openssl rand -hex 32)"

redi-mcp \
  --db="$DB_URI" \
  --schema=./schemas/ \
  --port=3000 \
  --enable-auth \
  --api-key="$MCP_API_KEY" \
  --read-only \
  --rate-limit=100 \
  --log-level=info
```

## Schema Auto-Discovery

RediORM MCP automatically discovers and generates schemas from existing database tables:

1. **Load existing schemas** from files (`.prisma` files)
2. **Scan database tables** for tables without schemas
3. **Generate schemas** with proper field types and relations
4. **Save generated schemas** to the schema directory

This means you can use MCP with existing databases without writing schema files manually!

## Example Interactions

### With Claude Desktop

Once configured, you can ask Claude:

```
"Show me all users who registered this month"
"What's the structure of the User table?"
"Count posts by category"
"Create a new product with name 'Widget' and price 19.99" (if write enabled)
```

### Direct JSON-RPC Examples

#### Query Users
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "model.findMany",
    "arguments": {
      "model": "User",
      "where": {
        "createdAt": {
          "gte": "2024-01-01T00:00:00Z"
        }
      },
      "orderBy": {"createdAt": "desc"},
      "take": 10
    }
  },
  "id": 1
}
```

#### Aggregate Sales
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "model.aggregate",
    "arguments": {
      "model": "Sale",
      "sum": {"amount": true},
      "avg": {"amount": true},
      "groupBy": ["productId"],
      "where": {
        "createdAt": {
          "gte": "2024-01-01T00:00:00Z"
        }
      }
    }
  },
  "id": 2
}
```

## Troubleshooting

### Common Issues

1. **"Cannot connect to database"**
   - Verify database URI is correct
   - Check database server is running
   - Ensure network connectivity

2. **"Permission denied"**
   - Check if operation requires write access (`--read-only=false`)
   - Verify database user permissions

3. **"Schema file not found"**
   - MCP will auto-generate schemas from database
   - Ensure schema directory is writable
   - Check `--schema` path is correct

4. **Logs polluting JSON-RPC output**
   - In stdio mode, logs go to stderr
   - Ensure your AI assistant reads from stdout only

### Debug Mode

Enable detailed logging:
```bash
redi-mcp --db=sqlite://./app.db --schema=./schemas/ --log-level=debug
```

### Testing MCP

Test stdio mode:
```bash
echo '{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}' | \
  redi-mcp --db=sqlite://:memory: --schema=./schemas/
```

Test HTTP mode:
```bash
# Start server
redi-mcp --db=sqlite://./app.db --schema=./schemas/ --port=3000

# In another terminal
curl -X POST http://localhost:3000/ \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}'
```

## Performance Tips

1. **Use Indexes**: Ensure frequently queried fields have indexes
2. **Limit Results**: Always use `take` parameter for large tables
3. **Use Aggregations**: Let the database do the heavy lifting
4. **Connection Pooling**: Add pool parameters to database URI
5. **Monitor Queries**: Use `--log-level=debug` to see SQL/MongoDB queries

## Advanced Features

### Working with Relations

```json
{
  "model": "User",
  "where": {"id": 1},
  "include": {
    "posts": {
      "include": {
        "comments": true
      }
    }
  }
}
```

### Complex Filters

```json
{
  "model": "Product",
  "where": {
    "OR": [
      {"price": {"lt": 20}},
      {"category": {"name": "Sale"}}
    ],
    "stock": {"gt": 0}
  }
}
```

### Schema Evolution

When write mode is enabled, MCP can modify your database schema:

1. **Add new models** with fields and relations
2. **Modify existing models** by adding/removing fields
3. **Create migrations** to track changes
4. **Apply migrations** to update the database

All schema changes are automatically saved to your schema files.
