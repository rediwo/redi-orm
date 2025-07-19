# RediORM CLI

Command-line interface for RediORM - run JavaScript files, manage migrations, start API servers, and integrate with AI assistants.

## Installation

```bash
# From the project root
go install ./cmd/redi-orm
```

## Usage

### Basic Commands

```bash
# Run JavaScript files with built-in ORM runtime
redi-orm run script.js
redi-orm run --timeout=30000 long-running-script.js

# Start GraphQL and REST API server
redi-orm server --db=sqlite://./myapp.db --schema=./schema.prisma
redi-orm server --db=postgresql://user:pass@localhost/db --port=8080

# Start MCP server for AI assistants (provides both stdio and HTTP access)
redi-orm mcp --db=sqlite://./myapp.db --schema=./schema.prisma
redi-orm mcp --db=postgresql://user:pass@localhost/db --port=3000

# Run migrations
redi-orm migrate --db=sqlite://./myapp.db --schema=./schema.prisma

# Generate migration file (production mode)
redi-orm migrate:generate --db=sqlite://./myapp.db --schema=./schema.prisma --name="add_user_table"

# Apply migrations from directory
redi-orm migrate:apply --db=sqlite://./myapp.db --migrations=./migrations

# Rollback last migration
redi-orm migrate:rollback --db=sqlite://./myapp.db --migrations=./migrations

# Check migration status
redi-orm migrate:status --db=sqlite://./myapp.db

# Preview changes without applying them (dry run)
redi-orm migrate:dry-run --db=sqlite://./myapp.db --schema=./schema.prisma

# Reset database (dangerous - drops all tables!)
redi-orm migrate:reset --db=sqlite://./myapp.db --force

# Show version
redi-orm version
```

### Database URIs

The CLI supports all databases that RediORM supports:

- **SQLite**: `sqlite://./path/to/database.db` or `sqlite://:memory:`
- **MySQL**: `mysql://user:password@localhost:3306/database`
- **PostgreSQL**: `postgresql://user:password@localhost:5432/database`
- **MongoDB**: `mongodb://user:password@localhost:27017/database` or `mongodb+srv://cluster.mongodb.net/database`

### Schema Files

The CLI uses Prisma-style schema files. By default, it looks for `./schema.prisma`, but you can specify a different path with the `--schema` flag.

Example schema:

```prisma
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  posts     Post[]
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  authorId  Int      @map("author_id")
  author    User     @relation(fields: [authorId], references: [id])
}
```

### Flags

#### Common Flags
- `--help`: Show help message
- `--db`: Database connection URI (required for most commands)
- `--schema`: Path to schema file (default: `./schema.prisma`)

#### Run Command Flags
- `--timeout`: Execution timeout in milliseconds (default: 0 = no timeout)
  - Example: `--timeout=30000` (30 seconds)
  - Alternative syntax: `--timeout 30000`

#### Server Command Flags
- `--port`: Server port (default: 4000)
- `--playground`: Enable GraphQL playground (default: true)
- `--cors`: Enable CORS (default: true)
- `--log-level`: Logging level: debug|info|warn|error|none (default: info)

#### MCP Command Flags
- `--port`: HTTP port (default: 3000)
- `--log-level`: Logging level: debug|info|warn|error|none (default: info)
- `--read-only`: Enable read-only mode (default: true)
- `--enable-auth`: Enable API key authentication
- `--api-key`: API key for authentication
- `--rate-limit`: Requests per minute rate limit (default: 60)
- `--allowed-tables`: Comma-separated list of allowed tables

#### Migration Flags
- `--migrations`: Path to migrations directory (default: `./migrations`)
- `--mode`: Migration mode: `auto` or `file` (default: `auto`)
- `--name`: Migration name (required for `migrate:generate`)
- `--force`: Force destructive changes without confirmation

### Running JavaScript Files

The `run` command executes JavaScript files with RediORM's built-in JavaScript runtime (Goja engine). **No Node.js required!**

```bash
# Basic usage
redi-orm run script.js

# With timeout for long-running scripts
redi-orm run --timeout=60000 data-migration.js  # 60 seconds

# Pass arguments to the script
redi-orm run process.js arg1 arg2 --option=value
```

**Key Features:**
- Built-in JavaScript runtime - no Node.js dependency
- Full ORM module support via `require('redi/orm')`
- Environment variables automatically available
- Timeout support for long-running operations
- Supports both `--timeout=5000` and `--timeout 5000` syntax

**Example Scripts:**
```javascript
// batch-process.js - Process records with timeout
const { fromUri } = require('redi/orm');

async function main() {
    const db = fromUri('sqlite://./data.db');
    await db.connect();
    
    // Long-running batch processing...
    const records = await db.models.Record.findMany({ where: { status: 'pending' } });
    
    for (const record of records) {
        // Process each record
        await processRecord(record);
        await db.models.Record.update({
            where: { id: record.id },
            data: { status: 'processed' }
        });
    }
    
    await db.close();
}

main().catch(console.error);
```

Run with: `redi-orm run --timeout=300000 batch-process.js` (5 minutes)

### Migration Workflow

#### Development Mode (Auto-migration)

1. **Define your schema** in a `.prisma` file
2. **Run migrations** to sync your database with the schema
3. **Check status** to see applied migrations
4. **Use dry-run** to preview changes before applying them

#### Production Mode (File-based migrations)

1. **Define your schema** changes
2. **Generate migration files** with descriptive names
3. **Review the generated SQL** in the migration files
4. **Apply migrations** to your production database
5. **Rollback if needed** to the previous state

### Safety Features

- **Dry Run**: Preview changes before applying them
- **Destructive Change Detection**: Warns about column/table drops
- **Force Flag Required**: Destructive operations require explicit confirmation
- **Transaction Support**: All migrations run in transactions (rollback on error)

### Examples

#### Running JavaScript Scripts

```bash
# Simple script
redi-orm run hello.js

# Data processing with 2 minute timeout
redi-orm run --timeout=120000 process-users.js

# Migration script with arguments
redi-orm run migrate-data.js --source=old.db --target=new.db

# Long-running sync job
redi-orm run --timeout 600000 sync-external-api.js
```

#### Initial Setup

```bash
# Create initial schema
cat > schema.prisma << EOF
model User {
  id    Int    @id @default(autoincrement())
  email String @unique
  name  String
}
EOF

# Run initial migration
redi-orm migrate --db=sqlite://./myapp.db
```

#### Adding a New Model

```bash
# Update schema.prisma to add a Post model
# Then run migration
redi-orm migrate --db=sqlite://./myapp.db
```

#### Checking Migration Status

```bash
redi-orm migrate:status --db=sqlite://./myapp.db

# Output:
# === Migration Status ===
# Database: sqlite://./myapp.db
# Tables: 2
# 
# Existing tables:
#   - users
#   - posts
# 
# Last migration:
#   Version: 20240115123456
#   Name: auto-migration
#   Applied: 2024-01-15 12:34:56
# 
# Total migrations applied: 1
```

#### Safe Migration Preview

```bash
# Preview changes before applying
redi-orm migrate:dry-run --db=sqlite://./myapp.db --schema=./new-schema.prisma

# If changes look good, apply them
redi-orm migrate --db=sqlite://./myapp.db --schema=./new-schema.prisma
```

## Development

To work on the CLI:

```bash
# Build
go build -o redi-orm ./cmd/redi-orm

# Run directly
go run ./cmd/redi-orm/main.go migrate --db=sqlite://./test.db

# Install globally
go install ./cmd/redi-orm

# Build with version
go build -ldflags "-X main.version=v1.0.0" -o redi-orm ./cmd/redi-orm
```

## Server Commands

### GraphQL + REST API Server

Start a server with auto-generated GraphQL and REST APIs based on your schema:

```bash
# Basic server
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma

# Production setup
redi-orm server \
  --db=postgresql://user:pass@localhost/db \
  --schema=./schema.prisma \
  --port=8080 \
  --playground=false \
  --log-level=info
```

**Endpoints:**
- GraphQL: `http://localhost:{port}/graphql`
- GraphQL Playground: `http://localhost:{port}/` (if enabled)
- REST API: `http://localhost:{port}/api`

### MCP Server for AI Assistants

Start an MCP server that provides both stdio (for local AI) and HTTP (for web apps) access:

```bash
# Basic MCP server
redi-orm mcp --db=sqlite://./app.db --schema=./schema.prisma

# Production MCP with security
redi-orm mcp \
  --db=postgresql://readonly:pass@localhost/db \
  --port=3000 \
  --read-only \
  --allowed-tables=users,posts \
  --enable-auth \
  --api-key=your-secret-key
```

**Access Methods:**
- Stdio: For local AI assistants like Claude Desktop
- HTTP: `http://localhost:{port}/` for web applications
- SSE: `http://localhost:{port}/events` for real-time updates

## Notes

- **JavaScript Runtime**: RediORM includes Goja engine - no Node.js required
- **Migrations**: Tracked in a `redi_migrations` table
- **Transactions**: All database operations run in transactions for safety
- **MongoDB**: Requires replica set for transactions, migrations limited to indexes
- **MCP**: Automatically provides both stdio and HTTP access simultaneously