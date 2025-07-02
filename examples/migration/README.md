# Migration Examples

This directory contains comprehensive examples and test cases for RediORM's production migration system.

## Table of Contents

- [Quick Start](#quick-start)
- [Contents Overview](#contents-overview)
- [Migration Workflow](#migration-workflow)
- [Production Migration Guide](#production-migration-guide)
- [Complete Production Demo](#complete-production-demo)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## Quick Start

### 1. Run the Test Script

```bash
cd examples/migration
./test-production-migrations.sh
```

This will demonstrate:
- Creating initial schema
- Generating migrations
- Applying migrations
- Making schema changes
- Rolling back migrations
- Checking migration status

### 2. Run the JavaScript Example

```bash
# From project root
./redi-orm run examples/migration/production-migration-workflow.js
```

This shows:
- Connecting to database
- Loading schemas
- Simulating production workflow
- Best practices implementation

### 3. Simple Migration Demo

```bash
# From project root
./redi-orm run examples/migration/simple-migration-demo.js
```

## Contents Overview

### 🧪 Test Scripts

- **[test-production-migrations.sh](./test-production-migrations.sh)** - Comprehensive bash script that tests the complete migration workflow
  - Initial schema generation
  - Migration application
  - Schema evolution
  - Rollback testing
  - Status checking

### 💻 Code Examples

- **[simple-migration-demo.js](./simple-migration-demo.js)** - Simple example showing:
  - Basic schema evolution
  - Database cleanup on each run
  - Migration workflow overview

- **[production-migration-workflow.js](./production-migration-workflow.js)** - Comprehensive example demonstrating:
  - Programmatic migration workflow
  - Best practices checklist
  - Error handling
  - Team collaboration tips

## Migration Workflow

```mermaid
graph TD
    A[Define Schema Changes] --> B[Generate Migration Files]
    B --> C[Review Generated SQL]
    C --> D{SQL Correct?}
    D -->|No| A
    D -->|Yes| E[Test on Staging]
    E --> F{Tests Pass?}
    F -->|No| A
    F -->|Yes| G[Backup Production]
    G --> H[Apply to Production]
    H --> I{Success?}
    I -->|No| J[Rollback Migration]
    I -->|Yes| K[Verify & Monitor]
    J --> L[Fix Issues]
    L --> A
```

## Production Migration Guide

### Overview

File-based migrations provide version control, rollback capabilities, and team collaboration for database schema changes in production environments.

### Complete Workflow

#### 1. Define Your Schema Changes

Start by modifying your Prisma schema file to reflect the desired database structure:

```prisma
// schema.prisma - Initial version
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String
  createdAt DateTime @default(now())
}

// schema.prisma - Updated version
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String
  phone     String?  // New field
  isActive  Boolean  @default(true)  // New field
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt  // New field
}
```

#### 2. Generate Migration Files

Use descriptive names that clearly indicate what the migration does:

```bash
# Generate initial schema
redi-orm migrate:generate \
  --db=postgresql://user:pass@localhost/myapp \
  --schema=./schema.prisma \
  --migrations=./migrations \
  --name="initial_schema"

# Generate migration for adding user fields
redi-orm migrate:generate \
  --db=postgresql://user:pass@localhost/myapp \
  --schema=./schema.prisma \
  --migrations=./migrations \
  --name="add_user_phone_and_status"
```

This creates timestamped migration directories:
```
migrations/
├── 20240702150000_initial_schema/
│   ├── up.sql
│   ├── down.sql
│   └── metadata.json
└── 20240702151000_add_user_phone_and_status/
    ├── up.sql
    ├── down.sql
    └── metadata.json
```

#### 3. Review Generated SQL

Always review the generated SQL before applying to production:

**up.sql** (applies changes):
```sql
-- 20240702151000_add_user_phone_and_status/up.sql
ALTER TABLE users ADD COLUMN phone VARCHAR(255);
ALTER TABLE users ADD COLUMN is_active BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
```

**down.sql** (reverts changes):
```sql
-- 20240702151000_add_user_phone_and_status/down.sql
ALTER TABLE users DROP COLUMN phone;
ALTER TABLE users DROP COLUMN is_active;
ALTER TABLE users DROP COLUMN updated_at;
```

#### 4. Apply Migrations to Production

Follow these steps for safe production deployment:

##### Pre-deployment Checklist
- [ ] Backup production database
- [ ] Test migration on staging environment
- [ ] Review migration SQL with team
- [ ] Schedule maintenance window if needed
- [ ] Prepare rollback plan

##### Apply Migration
```bash
# Check current status
redi-orm migrate:status --db=postgresql://user:pass@localhost/prod_db

# Apply pending migrations
redi-orm migrate:apply \
  --db=postgresql://user:pass@localhost/prod_db \
  --migrations=./migrations
```

Output:
```
Found 1 pending migration(s):
  - 20240702151000: add_user_phone_and_status

Applying migration 20240702151000: add_user_phone_and_status
Successfully applied 1 migration(s).
```

#### 5. Rollback if Needed

If issues arise, rollback to the previous state:

```bash
# Rollback last migration
redi-orm migrate:rollback \
  --db=postgresql://user:pass@localhost/prod_db \
  --migrations=./migrations

# Check status after rollback
redi-orm migrate:status --db=postgresql://user:pass@localhost/prod_db
```

## Complete Production Demo

### Setup

First, let's set up our project structure:

```bash
myapp/
├── schema.prisma
├── migrations/
├── scripts/
│   └── migrate.sh
└── config/
    └── database.yml
```

### Step 1: Initial Schema

Create your initial schema (`schema.prisma`):

```prisma
// Initial schema for our e-commerce application
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String
  createdAt DateTime @default(now())
}

model Product {
  id          Int      @id @default(autoincrement())
  name        String
  description String?
  price       Float
  stock       Int      @default(0)
  createdAt   DateTime @default(now())
}
```

Generate the initial migration:

```bash
redi-orm migrate:generate \
  --db=postgresql://user:pass@localhost/myapp_dev \
  --schema=./schema.prisma \
  --migrations=./migrations \
  --name="initial_schema"
```

### Step 2: Production Deployment Script

Create deployment script (`scripts/migrate.sh`):

```bash
#!/bin/bash
set -e

# Configuration
DB_URI="postgresql://user:pass@prod.myapp.com/myapp"
MIGRATIONS_DIR="./migrations"
BACKUP_DIR="./backups"

echo "=== Production Migration ==="
echo "Database: $DB_URI"
echo

# Step 1: Backup
echo "1. Creating backup..."
BACKUP_FILE="$BACKUP_DIR/backup_$(date +%Y%m%d_%H%M%S).sql"
pg_dump $DB_URI > $BACKUP_FILE
echo "   Backup saved to: $BACKUP_FILE"

# Step 2: Check status
echo "2. Current migration status:"
redi-orm migrate:status --db=$DB_URI

# Step 3: Show pending migrations
echo "3. Pending migrations:"
redi-orm migrate:apply --db=$DB_URI --migrations=$MIGRATIONS_DIR --dry-run

# Step 4: Confirm
read -p "Continue with migration? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Migration cancelled."
    exit 1
fi

# Step 5: Apply migrations
echo "4. Applying migrations..."
redi-orm migrate:apply --db=$DB_URI --migrations=$MIGRATIONS_DIR

# Step 6: Verify
echo "5. Verifying..."
redi-orm migrate:status --db=$DB_URI

echo
echo "✓ Migration completed successfully!"
```

### Step 3: Rollback Script

If issues occur, use the rollback script:

```bash
#!/bin/bash
# scripts/rollback.sh

DB_URI="postgresql://user:pass@prod.myapp.com/myapp"
MIGRATIONS_DIR="./migrations"

echo "=== Production Rollback ==="
echo "WARNING: This will rollback the last migration!"
echo

# Show current status
redi-orm migrate:status --db=$DB_URI

# Confirm
read -p "Rollback last migration? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Rollback cancelled."
    exit 1
fi

# Rollback
redi-orm migrate:rollback --db=$DB_URI --migrations=$MIGRATIONS_DIR

# Verify
redi-orm migrate:status --db=$DB_URI

echo "✓ Rollback completed!"
```

## Best Practices

### Migration Naming
- Use descriptive names: `add_user_email_index`, `create_orders_table`
- Include ticket/issue numbers: `JIRA-123_add_customer_fields`
- Avoid generic names: `update`, `fix`, `change`

### Testing Strategy
1. **Development**: Test with `--mode=auto` for rapid iteration
2. **Staging**: Apply file-based migrations to production-like data
3. **Production**: Apply during low-traffic periods

### Version Control
```bash
# Commit migrations with schema changes
git add schema.prisma migrations/
git commit -m "Add user phone and active status fields"
```

### Team Collaboration
- Review migration files in pull requests
- Document breaking changes
- Coordinate deployment timing

### Safety Guidelines
1. **Always backup** before applying migrations
2. **Test thoroughly** on non-production environments
3. **Review SQL** before applying
4. **Monitor** application after migration
5. **Document** any manual steps required
6. **Keep migrations small** and focused
7. **Avoid** mixing schema and data migrations

## Key Commands

### Generate Migration
```bash
redi-orm migrate:generate \
  --db=<database-uri> \
  --schema=./schema.prisma \
  --migrations=./migrations \
  --name="descriptive_name"
```

### Apply Migrations
```bash
redi-orm migrate:apply \
  --db=<database-uri> \
  --migrations=./migrations
```

### Check Status
```bash
redi-orm migrate:status \
  --db=<database-uri>
```

### Rollback
```bash
redi-orm migrate:rollback \
  --db=<database-uri> \
  --migrations=./migrations
```

## File Structure

After running migrations, your project structure should look like:

```
myproject/
├── schema.prisma
├── migrations/
│   ├── 20240702150000_initial_schema/
│   │   ├── up.sql
│   │   ├── down.sql
│   │   └── metadata.json
│   ├── 20240702160000_add_user_fields/
│   │   ├── up.sql
│   │   ├── down.sql
│   │   └── metadata.json
│   └── 20240703120000_create_orders_table/
│       ├── up.sql
│       ├── down.sql
│       └── metadata.json
└── scripts/
    ├── migrate.sh
    └── rollback.sh
```

## Database Support

The migration system works with all supported databases:
- PostgreSQL
- MySQL
- SQLite

Each database driver generates appropriate SQL syntax for:
- Table creation/modification
- Column types
- Constraints
- Indexes

## Troubleshooting

### Common Issues

1. **Migration already applied**: Check status with `migrate:status`
2. **SQL syntax errors**: Review generated SQL and database compatibility
3. **Rollback fails**: May need manual intervention for complex changes
4. **Missing down migration**: Some changes (like data loss) require careful planning

### Migration Conflicts
If multiple developers create migrations:
1. Apply migrations in timestamp order
2. Regenerate if conflicts occur
3. Test combined changes thoroughly

### Failed Migrations
```bash
# Check which migrations are applied
redi-orm migrate:status --db=<uri>

# Manually fix issues, then retry
redi-orm migrate:apply --db=<uri> --migrations=./migrations
```

### Data Migration
For data transformations, create custom SQL migrations:
```sql
-- migrations/20240702160000_migrate_user_names/up.sql
UPDATE users SET full_name = first_name || ' ' || last_name;
ALTER TABLE users DROP COLUMN first_name;
ALTER TABLE users DROP COLUMN last_name;
```

### Migration History

Track all applied migrations:
```bash
# View detailed migration history
sqlite3 production.db "
SELECT version, name, applied_at 
FROM redi_migrations 
ORDER BY version DESC 
LIMIT 10;"
```

## Example: Complete Production Workflow

```bash
# 1. Make schema changes
vim schema.prisma

# 2. Generate migration
./redi-orm migrate:generate \
  --db=postgresql://localhost/myapp_dev \
  --schema=./schema.prisma \
  --migrations=./migrations \
  --name="add_user_preferences"

# 3. Review generated files
cat migrations/*/up.sql
cat migrations/*/down.sql

# 4. Test on staging
./redi-orm migrate:apply \
  --db=postgresql://staging/myapp \
  --migrations=./migrations

# 5. Deploy to production
./redi-orm migrate:apply \
  --db=postgresql://prod/myapp \
  --migrations=./migrations

# 6. Verify
./redi-orm migrate:status --db=postgresql://prod/myapp
```

## Migration History

After several deployments, check history:

```sql
SELECT 
  version,
  name,
  applied_at,
  checksum
FROM redi_migrations
ORDER BY version DESC;
```

Output:
```
version         | name                    | applied_at          | checksum
----------------|-------------------------|---------------------|----------
20240703160000  | add_inventory_tracking  | 2024-07-03 16:15:00| abc123...
20240702180000  | add_order_management    | 2024-07-02 18:30:00| def456...
20240702150000  | initial_schema          | 2024-07-02 15:45:00| ghi789...
```

## Contributing

When adding new migration examples:
1. Include both up and down migrations
2. Test with all supported databases
3. Document any special considerations
4. Add to the appropriate section in this README
5. Follow the established naming conventions
6. Include error handling in scripts
7. Test rollback functionality

## Conclusion

This migration system provides:
- ✅ Reproducible deployments
- ✅ Version-controlled changes
- ✅ Safe rollback capability
- ✅ Team collaboration
- ✅ Audit trail of all changes

The production migration system ensures database changes are managed as carefully as application code, following best practices for safe and reliable deployments.