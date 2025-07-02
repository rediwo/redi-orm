#!/bin/bash

# Production Mode Migration Test Case
# This script demonstrates the complete workflow for file-based migrations in production

set -e  # Exit on error

echo "=== Production Mode Migration Test Case ==="
echo "This test demonstrates the complete file-based migration workflow"
echo

# Configuration
TEST_DIR="./tmp-migrations"
DB_PATH="$TEST_DIR/production.db"
SCHEMA_DIR="$TEST_DIR/schemas"
MIGRATIONS_DIR="$TEST_DIR/migrations"
DB_URI="sqlite://$DB_PATH"

# Clean up previous test
echo "1. Setting up test environment..."
rm -rf $TEST_DIR
mkdir -p $TEST_DIR $SCHEMA_DIR $MIGRATIONS_DIR

# Initial schema
cat > $SCHEMA_DIR/v1-initial.prisma << 'EOF'
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String
  createdAt DateTime @default(now())
}

model Post {
  id        Int      @id @default(autoincrement())
  title     String
  content   String?
  authorId  Int
  author    User     @relation(fields: [authorId], references: [id])
  createdAt DateTime @default(now())
}
EOF

echo "✓ Created initial schema (v1)"
echo

# Step 1: Generate initial migration
echo "2. Generating initial migration..."
../../redi-orm migrate:generate \
  --db=$DB_URI \
  --schema=$SCHEMA_DIR/v1-initial.prisma \
  --migrations=$MIGRATIONS_DIR \
  --name="initial_schema"

echo "✓ Generated initial migration"
echo

# Show generated migration
echo "3. Review generated migration files:"
echo "=================================="
echo "Migration files in $MIGRATIONS_DIR:"
ls -la $MIGRATIONS_DIR/
echo

# Find the migration directory
MIGRATION_DIR=$(ls -d $MIGRATIONS_DIR/*_initial_schema 2>/dev/null | head -1)
if [ -n "$MIGRATION_DIR" ]; then
  echo "Up migration:"
  echo "----------------------------------"
  cat "$MIGRATION_DIR/up.sql"
  echo
  echo "Down migration:"
  echo "----------------------------------"
  cat "$MIGRATION_DIR/down.sql"
fi
echo "=================================="
echo

# Step 2: Apply initial migration
echo "4. Applying initial migration..."
../../redi-orm migrate:apply \
  --db=$DB_URI \
  --migrations=$MIGRATIONS_DIR

echo "✓ Applied initial migration"
echo

# Check migration status
echo "5. Checking migration status..."
../../redi-orm migrate:status --db=$DB_URI
echo

# Step 3: Create schema v2 with changes
cat > $SCHEMA_DIR/v2-add-fields.prisma << 'EOF'
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String
  bio       String?  // New field
  isActive  Boolean  @default(true)  // New field
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt  // New field
}

model Post {
  id          Int      @id @default(autoincrement())
  title       String
  content     String?
  published   Boolean  @default(false)  // New field
  viewCount   Int      @default(0)      // New field
  authorId    Int
  author      User     @relation(fields: [authorId], references: [id])
  createdAt   DateTime @default(now())
  publishedAt DateTime?  // New field
}

model Comment {  // New model
  id        Int      @id @default(autoincrement())
  content   String
  postId    Int
  post      Post     @relation(fields: [postId], references: [id])
  authorId  Int
  author    User     @relation(fields: [authorId], references: [id])
  createdAt DateTime @default(now())
}
EOF

echo "6. Created schema v2 with changes:"
echo "   - Added fields to User: bio, isActive, updatedAt"
echo "   - Added fields to Post: published, viewCount, publishedAt"
echo "   - Added new model: Comment"
echo

# Generate migration for changes
echo "7. Generating migration for schema changes..."
../../redi-orm migrate:generate \
  --db=$DB_URI \
  --schema=$SCHEMA_DIR/v2-add-fields.prisma \
  --migrations=$MIGRATIONS_DIR \
  --name="add_fields_and_comments"

echo "✓ Generated migration for changes"
echo

# Show the new migration
echo "8. Review new migration files:"
echo "=================================="
MIGRATION_DIR_2=$(ls -d $MIGRATIONS_DIR/*_add_fields_and_comments 2>/dev/null | head -1)
if [ -n "$MIGRATION_DIR_2" ]; then
  echo "Up migration:"
  echo "----------------------------------"
  cat "$MIGRATION_DIR_2/up.sql"
  echo
  echo "Down migration:"
  echo "----------------------------------"
  cat "$MIGRATION_DIR_2/down.sql"
fi
echo "=================================="
echo

# Apply the new migration
echo "9. Applying new migration..."
../../redi-orm migrate:apply \
  --db=$DB_URI \
  --migrations=$MIGRATIONS_DIR

echo "✓ Applied new migration"
echo

# Check status again
echo "10. Checking migration status after second migration..."
../../redi-orm migrate:status --db=$DB_URI
echo

# Insert some test data
echo "11. Inserting test data..."
sqlite3 $DB_PATH << 'EOF'
INSERT INTO users (email, name, bio, is_active) VALUES 
  ('alice@example.com', 'Alice', 'Software developer', 1),
  ('bob@example.com', 'Bob', 'Designer', 1);

INSERT INTO posts (title, content, published, view_count, author_id) VALUES 
  ('First Post', 'Hello World', 1, 10, 1),
  ('Draft Post', 'Work in progress', 0, 0, 2);

INSERT INTO comments (content, post_id, author_id) VALUES 
  ('Great post!', 1, 2),
  ('Thanks for sharing', 1, 1);

SELECT COUNT(*) as user_count FROM users;
SELECT COUNT(*) as post_count FROM posts;
SELECT COUNT(*) as comment_count FROM comments;
EOF
echo

# Create schema v3 with breaking changes
cat > $SCHEMA_DIR/v3-breaking-changes.prisma << 'EOF'
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String   // Keep original field
  username  String?  // New optional field instead of rename
  bio       String?
  isActive  Boolean  @default(true)
  role      String   @default("user")  // New field
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}

model Post {
  id          Int        @id @default(autoincrement())
  title       String
  content     String?
  published   Boolean    @default(false)
  viewCount   Int        @default(0)
  authorId    Int
  author      User       @relation(fields: [authorId], references: [id])
  tags        PostTag[]  // New relation
  createdAt   DateTime   @default(now())
  publishedAt DateTime?
}

model Comment {
  id        Int      @id @default(autoincrement())
  content   String
  postId    Int
  post      Post     @relation(fields: [postId], references: [id])
  authorId  Int
  author    User     @relation(fields: [authorId], references: [id])
  createdAt DateTime @default(now())
}

model Tag {  // New model
  id    Int       @id @default(autoincrement())
  name  String    @unique
  posts PostTag[]
}

model PostTag {  // New junction table
  postId Int
  tagId  Int
  post   Post @relation(fields: [postId], references: [id])
  tag    Tag  @relation(fields: [tagId], references: [id])
  
  @@id([postId, tagId])
}
EOF

echo "12. Created schema v3 with changes:"
echo "    - Added optional User.username field"
echo "    - Added User.role field"
echo "    - Added Tag model and many-to-many relation with Post"
echo

# Generate migration with breaking changes
echo "13. Generating migration for breaking changes..."
../../redi-orm migrate:generate \
  --db=$DB_URI \
  --schema=$SCHEMA_DIR/v3-breaking-changes.prisma \
  --migrations=$MIGRATIONS_DIR \
  --name="rename_fields_add_tags" \
  --force || true  # Allow it to fail if it detects breaking changes

echo

# Show the migration with warnings
MIGRATION_FILE_3=$(ls -1 $MIGRATIONS_DIR/*_rename_fields_add_tags.up.sql 2>/dev/null | head -1 || echo "")
if [ -n "$MIGRATION_FILE_3" ]; then
  echo "14. Review migration with breaking changes:"
  echo "=================================="
  echo "File: $(basename $MIGRATION_FILE_3)"
  echo "----------------------------------"
  cat $MIGRATION_FILE_3
  echo "=================================="
  echo
fi

# Test rollback functionality
echo "15. Testing rollback functionality..."
echo "    Current migrations:"
../../redi-orm migrate:status --db=$DB_URI | grep -E "(Total migrations|Version:|Name:|Applied:)" || true

echo
echo "16. Rolling back last migration..."
../../redi-orm migrate:rollback \
  --db=$DB_URI \
  --migrations=$MIGRATIONS_DIR

echo "✓ Rolled back last migration"
echo

echo "17. Checking status after rollback..."
../../redi-orm migrate:status --db=$DB_URI
echo

# Verify data is intact after rollback
echo "18. Verifying data after rollback..."
sqlite3 $DB_PATH << 'EOF'
SELECT 'Users:', COUNT(*) FROM users;
SELECT 'Posts:', COUNT(*) FROM posts;
SELECT 'Tables:', GROUP_CONCAT(name) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name != 'redi_migrations';
EOF
echo

# Re-apply the rolled back migration
echo "19. Re-applying the rolled back migration..."
../../redi-orm migrate:apply \
  --db=$DB_URI \
  --migrations=$MIGRATIONS_DIR

echo "✓ Re-applied migration"
echo

# Final status
echo "20. Final migration status..."
../../redi-orm migrate:status --db=$DB_URI
echo

# Show migration history
echo "21. Migration history from database:"
sqlite3 $DB_PATH << 'EOF'
.mode column
.headers on
SELECT 
  version,
  name,
  datetime(applied_at, 'localtime') as applied_at,
  checksum
FROM redi_migrations
ORDER BY version;
EOF
echo

# List all migration files
echo "22. All migration files in directory:"
ls -la $MIGRATIONS_DIR/ | grep -E '^d' | awk '{print "    " $9}'
echo

echo "=== Test Complete ==="
echo "This test demonstrated:"
echo "✓ Generating initial migration"
echo "✓ Applying migrations"
echo "✓ Checking migration status"
echo "✓ Generating migrations for schema changes"
echo "✓ Handling new fields and models"
echo "✓ Rolling back migrations"
echo "✓ Re-applying migrations"
echo "✓ Migration history tracking"
echo
echo "Test artifacts are in: $TEST_DIR"