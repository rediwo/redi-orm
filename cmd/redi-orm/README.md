# RediORM CLI

Command-line interface for managing RediORM database migrations and running JavaScript files with ORM support.

## Installation

```bash
# From the project root
go install ./cmd/redi-orm
```

## Usage

### Basic Commands

```bash
# Run JavaScript files with ORM support
redi-orm run script.js
redi-orm run --timeout=30000 long-running-script.js

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

#### Run Command Flags
- `--timeout`: Execution timeout in milliseconds (default: 0 = no timeout)
  - Example: `--timeout=30000` (30 seconds)
  - Alternative syntax: `--timeout 30000`

#### Migration Flags
- `--db` (required for migrations): Database connection URI
- `--schema`: Path to schema file (default: `./schema.prisma`)
- `--migrations`: Path to migrations directory (default: `./migrations`)
- `--mode`: Migration mode: `auto` or `file` (default: `auto`)
- `--name`: Migration name (required for `migrate:generate`)
- `--force`: Force destructive changes without confirmation

### Running JavaScript Files

The `run` command executes JavaScript files with full RediORM support:

```bash
# Basic usage
redi-orm run script.js

# With timeout for long-running scripts
redi-orm run --timeout=60000 data-migration.js  # 60 seconds

# Pass arguments to the script
redi-orm run process.js arg1 arg2 --option=value
```

**Timeout Feature:**
- Useful for batch processing, data migrations, and long-running scripts
- Default: 0 (no timeout) - script exits shortly after completion
- Set a timeout to allow async operations to complete
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
```

## Notes

- Migrations are tracked in a `redi_migrations` table
- Each migration has a version (timestamp) and checksum
- The CLI uses the same migration system as the programmatic API
- All database operations run in transactions for safety