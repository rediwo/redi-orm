# MCP (Model Context Protocol) Guide

Complete guide to using RediORM's Model Context Protocol integration for AI assistants and applications.

## What is MCP?

The Model Context Protocol (MCP) is an open standard that enables AI systems to securely access and interact with data sources. RediORM's MCP implementation provides AI assistants with intelligent, schema-aware database access through a standardized JSON-RPC interface.

### Key Benefits

- **🤖 AI-Native Design** - Purpose-built for AI assistant integration
- **🔍 Schema-Aware Operations** - AI understands your data models and relationships
- **🛡️ Security-First** - Built-in authentication, rate limiting, and query validation
- **🌐 Multi-Transport** - Local (stdio) and remote (HTTP/SSE) access modes
- **📊 Advanced Analytics** - Statistical analysis and data sampling capabilities
- **🗄️ Multi-Database** - Unified interface across SQLite, MySQL, PostgreSQL, and MongoDB

### What AI Can Do

With MCP, AI assistants can:
- 🔍 **Discover your data** - "What tables do I have? What's their structure?"
- 🔎 **Smart queries** - "Find users who haven't logged in for 30 days"
- 📊 **Data analysis** - "Analyze sales trends by region and product category"
- ⚡ **Performance insights** - "Which queries are slow? How can I optimize them?"
- 🛠️ **Schema evolution** - "I need to add a field for user preferences"

## Quick Start

### Prerequisites

- RediORM CLI installed (`go install github.com/rediwo/redi-orm/cmd/redi-orm@latest`)
- A database with some data (SQLite is easiest for testing)
- (Optional) Prisma schema file

### 1. Start MCP Server (5 minutes)

```bash
# Basic setup with SQLite
redi-orm mcp --db=sqlite://./test.db --schema=./schema.prisma

# Or with existing database
redi-orm mcp --db=mysql://user:pass@localhost:3306/mydb
```

### 2. Test Basic Commands

Send JSON-RPC commands to test (copy/paste into terminal):

```json
{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}
```

List your tables:
```json
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"list_tables","arguments":{}},"id":2}
```

Query some data:
```json
{"jsonrpc":"2.0","method":"tools/call","params":{"name":"query","arguments":{"sql":"SELECT COUNT(*) as total FROM users"}},"id":3}
```

### 3. Web Mode (HTTP)

For web applications or remote access:

```bash
redi-orm mcp --db=sqlite://./test.db --port=3000
```

Test with curl:
```bash
curl -X POST http://localhost:3000/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}'
```

## Setup & Configuration

### Installation

MCP is included with RediORM CLI:

```bash
# Install from Go
go install github.com/rediwo/redi-orm/cmd/redi-orm@latest

# Verify installation
redi-orm --version
```

### Basic Configuration

```bash
# Minimal setup
redi-orm mcp --db=sqlite://./app.db

# With schema file
redi-orm mcp --db=sqlite://./app.db --schema=./schema.prisma

# Custom port
redi-orm mcp --db=sqlite://./app.db --port=8080

# With logging
redi-orm mcp --db=sqlite://./app.db --log-level=info
```

### Database URIs

```bash
# SQLite
redi-orm mcp --db=sqlite://./database.db
redi-orm mcp --db=sqlite://:memory:  # In-memory

# MySQL
redi-orm mcp --db=mysql://user:pass@host:3306/database

# PostgreSQL  
redi-orm mcp --db=postgresql://user:pass@host:5432/database

# MongoDB
redi-orm mcp --db=mongodb://host:27017/database
```

## Transport Modes

### Stdio Transport (Local AI)

**Best for**: Local development, desktop AI applications, secure environments

```bash
# Start server with stdio transport
redi-orm mcp --db=sqlite://./test.db --transport=stdio

# Communication via standard input/output
echo '{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}' | redi-orm mcp --db=sqlite://./test.db --transport=stdio
```

**Advantages**:
- Maximum security (no network exposure)
- Simple setup
- Perfect for local AI assistants

### HTTP/SSE Transport (Web Applications)

**Best for**: Web applications, multi-user environments, cloud deployments

```bash
# Start HTTP server
redi-orm mcp --db=sqlite://./test.db --port=3000

# Endpoints available:
# POST / - JSON-RPC requests
# GET /events - Server-Sent Events stream
```

**JavaScript Example**:
```javascript
async function queryDatabase(sql) {
  const response = await fetch('http://localhost:3000/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      jsonrpc: '2.0',
      method: 'tools/call',
      params: {
        name: 'query',
        arguments: { sql }
      },
      id: Date.now()
    })
  });
  return response.json();
}

// Usage
const result = await queryDatabase('SELECT * FROM users LIMIT 5');
```

## Security

### Authentication

Enable API key authentication for production:

```bash
redi-orm mcp \
  --enable-auth \
  --api-key=your-secret-key-here \
  --port=3000
```

Include API key in requests:
```bash
curl -X POST http://localhost:3000/ \
  -H "Authorization: Bearer your-secret-key-here" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","params":{},"id":1}'
```

### Rate Limiting

Prevent abuse with configurable limits:

```bash
redi-orm mcp --rate-limit=60  # 60 requests per minute
```

### Read-Only Mode

Restrict to SELECT queries only (enabled by default):

```bash
redi-orm mcp --read-only  # Only SELECT queries allowed
```

### Table Access Control

Limit access to specific tables:

```bash
redi-orm mcp --allowed-tables=users,products,orders
```

### Production Security Setup

```bash
redi-orm mcp \
  --db="postgresql://readonly:pass@db:5432/production" \
  --port=8443 \
  --enable-auth \
  --api-key="$(cat /run/secrets/api-key)" \
  --read-only \
  --rate-limit=100 \
  --allowed-tables="users,products,orders" \
  --log-level=info
```

## Using MCP

### Resources

Resources provide access to database structure and data using URI-based addressing.

| Resource | URI Pattern | Description |
|----------|-------------|-------------|
| Schema | `schema://database` | Complete database schema |
| Table | `table://{name}` | Table structure and metadata |
| Data | `data://{table}?limit=N` | Table data with pagination |
| Model | `model://{name}` | Prisma model definition |

**List all resources**:
```json
{"jsonrpc":"2.0","method":"resources/list","params":{},"id":1}
```

**Read table structure**:
```json
{"jsonrpc":"2.0","method":"resources/read","params":{"uri":"table://users"},"id":2}
```

### Core Tools

#### 1. query - Execute SQL Queries
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "query",
    "arguments": {
      "sql": "SELECT * FROM users WHERE age > ?",
      "parameters": [18]
    }
  },
  "id": 1
}
```

#### 2. list_tables - Get All Tables
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "list_tables",
    "arguments": {}
  },
  "id": 2
}
```

#### 3. inspect_schema - Table Details
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "inspect_schema",
    "arguments": {
      "table": "users"
    }
  },
  "id": 3
}
```

#### 4. count_records - Count with Filtering
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "count_records",
    "arguments": {
      "table": "products",
      "where": {
        "category": "electronics",
        "in_stock": true
      }
    }
  },
  "id": 4
}
```

### Advanced Tools

#### batch_query - Multiple Queries
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "batch_query",
    "arguments": {
      "queries": [
        {
          "sql": "SELECT COUNT(*) as total FROM users",
          "label": "user_count"
        },
        {
          "sql": "SELECT category, COUNT(*) as count FROM products GROUP BY category",
          "label": "product_distribution"
        }
      ]
    }
  },
  "id": 5
}
```

#### analyze_table - Statistical Analysis
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "analyze_table",
    "arguments": {
      "table": "sales",
      "columns": ["amount", "quantity"],
      "sample_size": 1000
    }
  },
  "id": 6
}
```

#### generate_sample - Sample Data
```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "generate_sample",
    "arguments": {
      "table": "customers",
      "count": 10,
      "random": true,
      "where": {"country": "USA"}
    }
  },
  "id": 7
}
```

## AI Integration

### Claude Desktop Setup

Configure Claude Desktop to use your MCP server by adding to your configuration:

```json
{
  "mcpServers": {
    "rediorm": {
      "command": "redi-orm",
      "args": [
        "mcp",
        "--db=sqlite:///path/to/your/database.db",
        "--schema=./schema.prisma",
        "--transport=stdio"
      ]
    }
  }
}
```

Now Claude can:
- Ask about your database structure
- Generate and execute queries
- Analyze your data
- Suggest optimizations
- Help with schema changes

### Example Conversations

**"What tables do I have?"**
Claude will call `list_tables` and show you all available tables.

**"Show me the structure of the users table"**
Claude will call `inspect_schema` with table="users" and explain the columns, types, and relationships.

**"Find users who haven't logged in for 30 days"**
Claude will generate appropriate SQL and execute it via the `query` tool.

**"Analyze my sales data trends"**
Claude will use `analyze_table` and `batch_query` to provide comprehensive analysis.

### Python Integration

```python
import json
import subprocess

class MCPClient:
    def __init__(self, db_uri, schema_path=None):
        args = ['redi-orm', 'mcp', f'--db={db_uri}']
        if schema_path:
            args.append(f'--schema={schema_path}')
        
        self.process = subprocess.Popen(
            args,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            text=True
        )
    
    def call_tool(self, tool_name, arguments):
        request = {
            'jsonrpc': '2.0',
            'method': 'tools/call',
            'params': {
                'name': tool_name,
                'arguments': arguments
            },
            'id': 1
        }
        
        self.process.stdin.write(json.dumps(request) + '\n')
        self.process.stdin.flush()
        
        response = self.process.stdout.readline()
        return json.loads(response)

# Usage
client = MCPClient('sqlite://./data.db', './schema.prisma')

# Query data
result = client.call_tool('query', {
    'sql': 'SELECT COUNT(*) as count FROM users WHERE active = ?',
    'parameters': [True]
})

print(f"Active users: {result}")
```

## Advanced Usage

### Data Analysis Workflow

```javascript
// Complete data analysis example
const mcpClient = {
  async call(method, params) {
    const response = await fetch('http://localhost:3000/', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer your-api-key'
      },
      body: JSON.stringify({
        jsonrpc: '2.0',
        method,
        params,
        id: Date.now()
      })
    });
    return response.json();
  }
};

// 1. Get overview of data
const tables = await mcpClient.call('tools/call', {
  name: 'list_tables',
  arguments: {}
});

// 2. Analyze sales table
const analysis = await mcpClient.call('tools/call', {
  name: 'analyze_table',
  arguments: {
    table: 'sales',
    columns: ['amount', 'quantity'],
    sample_size: 5000
  }
});

// 3. Get recent trends
const trends = await mcpClient.call('tools/call', {
  name: 'batch_query',
  arguments: {
    queries: [
      {
        sql: 'SELECT DATE(created_at) as date, SUM(amount) as total FROM sales WHERE created_at >= DATE("now", "-30 days") GROUP BY DATE(created_at) ORDER BY date',
        label: 'daily_sales_30d'
      },
      {
        sql: 'SELECT product_category, SUM(amount) as revenue FROM sales s JOIN products p ON s.product_id = p.id GROUP BY product_category ORDER BY revenue DESC',
        label: 'category_revenue'
      }
    ]
  }
});

console.log('Analysis complete:', { tables, analysis, trends });
```

### Streaming Large Results

```bash
# For large datasets, use streaming
echo '{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "stream_query",
    "arguments": {
      "sql": "SELECT * FROM large_table ORDER BY id",
      "batch_size": 1000
    }
  },
  "id": 1
}' | redi-orm mcp --db=sqlite://./large.db
```

## Production Deployment

### Docker Setup

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o redi-orm ./cmd/redi-orm

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/redi-orm .
COPY schema.prisma .

CMD ["./redi-orm", "mcp", "--db=${DATABASE_URL}", "--schema=./schema.prisma", "--port=3000"]
```

### Environment Variables

```bash
# Environment configuration
DATABASE_URL=postgresql://user:pass@db:5432/production
SCHEMA_PATH=./schema.prisma
MCP_PORT=3000
MCP_API_KEY=your-production-api-key
MCP_RATE_LIMIT=100
MCP_ALLOWED_TABLES=users,products,orders
LOG_LEVEL=info
```

### Health Checks

```bash
# Health check endpoint
curl http://localhost:3000/health

# Response
{
  "status": "healthy",
  "database": "connected",
  "uptime": "2h 30m",
  "version": "1.0.0"
}
```

### Monitoring

```bash
# Enable detailed logging
redi-orm mcp \
  --db="$DATABASE_URL" \
  --log-level=debug \
  --log-queries \
  --log-slow-queries=1000
```

## Testing & Development

### Quick Local Testing

```bash
# 1. Start test server
redi-orm mcp --db=sqlite://:memory: --schema=./test-schema.prisma &

# 2. Run test queries
echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"query","arguments":{"sql":"SELECT 1 as test"}},"id":1}' | redi-orm mcp --db=sqlite://:memory:

# 3. Test with sample data
./scripts/populate-test-data.sh
```

### Integration Testing

```bash
# Test against real databases with Docker
docker-compose up -d postgres mysql mongo

# Run comprehensive tests
make test-mcp

# Test security features
./scripts/test-security.sh
```

## Troubleshooting

### Common Issues

**"Permission denied" error**
```bash
# Solutions:
# 1. Use read-only database user
# 2. Check --read-only flag
# 3. Verify table access with --allowed-tables
```

**"Rate limit exceeded"**
```bash
# Solutions:
# 1. Increase rate limit
redi-orm mcp --rate-limit=120

# 2. Implement client-side backoff
# 3. Use batch_query for multiple operations
```

**"Database connection failed"**
```bash
# Debug steps:
# 1. Test connection string
# 2. Check database is running
# 3. Verify network connectivity
# 4. Enable debug logging
redi-orm mcp --db=sqlite://./test.db --log-level=debug
```

**"Method not found"**
```bash
# Check JSON-RPC format:
{
  "jsonrpc": "2.0",        # Required
  "method": "tools/call",  # Correct method name
  "params": {...},         # Valid parameters
  "id": 1                  # Required ID
}
```

### Debug Mode

```bash
# Enable comprehensive debugging
redi-orm mcp \
  --db=sqlite://./test.db \
  --log-level=debug \
  --log-queries \
  --log-errors
```

### Validation

```bash
# Validate your setup
redi-orm mcp --db=sqlite://./test.db --validate-only

# Test all tools
./scripts/test-all-tools.sh
```

## API Reference

### JSON-RPC Methods

| Method | Description | Parameters |
|--------|-------------|------------|
| `resources/list` | List available resources | None |
| `resources/read` | Read resource content | `uri` |
| `tools/list` | List available tools | None |
| `tools/call` | Execute a tool | `name`, `arguments` |
| `prompts/list` | List available prompts | None |
| `prompts/get` | Get prompt template | `name`, `arguments` |

### Tool Reference

| Tool | Purpose | Key Arguments |
|------|---------|---------------|
| `query` | Execute SQL | `sql`, `parameters` |
| `list_tables` | Get all tables | None |
| `inspect_schema` | Table details | `table` |
| `count_records` | Count with filter | `table`, `where` |
| `batch_query` | Multiple queries | `queries` |
| `analyze_table` | Statistics | `table`, `columns`, `sample_size` |
| `generate_sample` | Sample data | `table`, `count`, `where` |

### Security Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--enable-auth` | Enable API key auth | false |
| `--api-key` | API key | "" |
| `--read-only` | SELECT only | true |
| `--rate-limit` | Requests per minute | 60 |
| `--allowed-tables` | Table whitelist | All |

---

Ready to integrate AI with your database? Start with the [Quick Start](#quick-start) section and have MCP running in 5 minutes!