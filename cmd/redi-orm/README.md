# RediORM CLI

Command-line interface for managing RediORM database migrations.

## Installation

```bash
# From the project root
go install ./cmd/redi-orm
```

## Usage

### Basic Commands

```bash
# Run migrations
redi-orm migrate --db=sqlite://./myapp.db --schema=./schema.prisma

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

- `--db` (required): Database connection URI
- `--schema`: Path to schema file (default: `./schema.prisma`)
- `--force`: Force destructive changes without confirmation
- `--help`: Show help message

### Migration Workflow

1. **Define your schema** in a `.prisma` file
2. **Run migrations** to sync your database with the schema
3. **Check status** to see applied migrations
4. **Use dry-run** to preview changes before applying them

### Safety Features

- **Dry Run**: Preview changes before applying them
- **Destructive Change Detection**: Warns about column/table drops
- **Force Flag Required**: Destructive operations require explicit confirmation
- **Transaction Support**: All migrations run in transactions (rollback on error)

### Examples

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