# RediORM MCP Server

MCP (Model Context Protocol) server that allows AI assistants to interact with databases through RediORM.

## Overview

The `redi-mcp` command provides a standalone MCP server that exposes database operations to AI assistants. By default, it uses stdio for local AI assistants. You can enable HTTP server by specifying a port.

## Installation

```bash
# Build from source
go build -o redi-mcp ./cmd/redi-mcp

# Or install globally
go install github.com/rediwo/redi-orm/cmd/redi-mcp@latest
```

## Usage

### Basic Usage

```bash
# Start with stdio (default for local AI assistants)
redi-mcp --db=sqlite://./myapp.db

# Start with HTTP server
redi-mcp --db=postgresql://user:pass@localhost/db --port=3000

# Use a directory of schema files
redi-mcp --db=sqlite://./myapp.db --schema=./schemas/
```

### Security Options

```bash
# Enable authentication
redi-mcp --db=mysql://user:pass@localhost/db --enable-auth --api-key=secret

# Set rate limiting
redi-mcp --db=sqlite://./myapp.db --rate-limit=100

# Restrict to specific tables
redi-mcp --db=sqlite://./myapp.db --allowed-tables=users,posts,comments

# Disable read-only mode (use with caution!)
redi-mcp --db=sqlite://./myapp.db --read-only=false
```

## Command Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `--db` | Database URI (required) | - |
| `--schema` | Path to Prisma schema file or directory | `./schema.prisma` |
| `--port` | Enable HTTP server on specified port (0 = stdio mode) | `0` (stdio) |
| `--log-level` | Logging level: `debug`, `info`, `warn`, `error`, `none` | `info` |
| `--api-key` | API key for HTTP authentication | - |
| `--enable-auth` | Enable HTTP authentication | `false` |
| `--read-only` | Enable read-only mode | `true` |
| `--rate-limit` | Requests per minute rate limit (0 = disabled) | `60` |
| `--allowed-tables` | Comma-separated list of allowed tables | - |

## AI Assistant Integration

### For Local AI Assistants (stdio)

Configure your AI assistant to run:
```bash
redi-mcp --db=<your-database-uri>
```

The MCP server will communicate via standard input/output using JSON-RPC.

### For Remote AI Assistants (HTTP)

Start the HTTP server by specifying a port:
```bash
redi-mcp --db=<your-database-uri> --port=3000
```

Configure your AI assistant to connect to:
- Endpoint: `http://localhost:3000`
- POST `/` for JSON-RPC requests
- GET `/sse` for Server-Sent Events

## Available MCP Tools

The server exposes the following tools to AI assistants:

### Model Operations
- `model.findMany` - Query multiple records with Prisma-style filters
- `model.findUnique` - Find a single record by unique field
- `model.create` - Create a new record
- `model.update` - Update existing records
- `model.delete` - Delete records
- `model.count` - Count records with optional filters
- `model.aggregate` - Perform aggregation queries (sum, avg, min, max, groupBy)

### Schema Operations
- `schema.models` - List all models with their fields and relationships
- `schema.describe` - Get detailed information about a specific model
- `schema.create` - Create a new model schema
- `schema.update` - Update an existing model schema
- `schema.addField` - Add a field to an existing model
- `schema.removeField` - Remove a field from a model
- `schema.addRelation` - Add a relation between models

### Migration Operations
- `migration.create` - Create a new migration based on schema changes
- `migration.apply` - Apply pending migrations to the database
- `migration.status` - Show current migration status

### Transaction Support
- `transaction` - Execute multiple operations in a transaction

## Security Best Practices

1. **Always use read-only mode in production** unless you specifically need write access
2. **Enable authentication** when using HTTP server (--port specified)
3. **Use rate limiting** to prevent abuse
4. **Restrict allowed tables** to limit exposure
5. **Use secure database credentials** and consider using environment variables

Example secure setup:
```bash
export DB_URI="postgresql://user:pass@localhost/db"
export MCP_API_KEY="your-secret-key"

redi-mcp \
  --db="$DB_URI" \
  --port=3000 \
  --enable-auth \
  --api-key="$MCP_API_KEY" \
  --rate-limit=30 \
  --allowed-tables=users,posts \
  --read-only=true
```

### Schema Modification (requires --read-only=false)

When write access is enabled, you can modify schemas:

```json
// Create a new model
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "schema.create",
    "arguments": {
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
  },
  "id": 1
}

// Add a field to existing model
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "schema.addField",
    "arguments": {
      "model": "Product",
      "field": {
        "name": "description",
        "type": "String",
        "optional": true
      }
    }
  },
  "id": 2
}
```

**Note**: Schema modifications are automatically saved to the schema files specified by `--schema`.

## Examples

### Basic Query Example
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "model.findMany",
    "arguments": {
      "model": "User",
      "where": {"active": true},
      "orderBy": {"createdAt": "desc"},
      "take": 10
    }
  },
  "id": 1
}
```

### Transaction Example
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "transaction",
    "arguments": {
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
  },
  "id": 2
}
```

## Troubleshooting

### Connection Issues
- Ensure the database URI is correct and the database is accessible
- Check firewall settings when using HTTP server
- Verify authentication credentials if enabled

### Performance
- Use appropriate rate limits based on your database capacity
- Consider using connection pooling in your database URI
- Monitor database query performance with `--log-level=debug`

### Security
- Review allowed tables regularly
- Rotate API keys periodically
- Monitor access logs for suspicious activity