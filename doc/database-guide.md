# Database Guide

Comprehensive guide to RediORM's multi-database support, configuration, and database-specific features.

## Table of Contents

- [Overview](#overview)
- [SQLite](#sqlite)
- [MySQL](#mysql) 
- [PostgreSQL](#postgresql)
- [MongoDB](#mongodb)
- [Connection Pooling](#connection-pooling)
- [Performance Optimization](#performance-optimization)
- [Migration Strategies](#migration-strategies)

## Overview

RediORM provides unified API across all supported databases while respecting each database's unique characteristics and capabilities.

### Supported Databases

| Database | Version | CRUD | Relations | Transactions | Migrations | Raw Queries |
|----------|---------|------|-----------|--------------|------------|-------------|
| SQLite | 3.35+ | ✅ | ✅ | ✅ | ✅ | SQL |
| MySQL | 8.0+ | ✅ | ✅ | ✅ | ✅ | SQL |
| PostgreSQL | 12+ | ✅ | ✅ | ✅ | ✅ | SQL |
| MongoDB | 4.4+ | ✅ | ✅ | ✅ | ❌ | SQL + MongoDB |

### URI-Based Configuration

All databases use URI-based connection strings:

```javascript
// General pattern
const db = fromUri('protocol://[user[:password]@]host[:port]/database[?options]');
```

## SQLite

### Connection Options

```javascript
// File database
const db = fromUri('sqlite://./app.db');
const db = fromUri('sqlite:///absolute/path/to/database.db');

// In-memory database (for testing)
const db = fromUri('sqlite://:memory:');

// With options
const db = fromUri('sqlite://./app.db?cache=shared&mode=rwc');
```

### SQLite-Specific Features

```javascript
// WAL mode for better concurrent access
const db = fromUri('sqlite://./app.db?_journal_mode=WAL');

// Foreign key constraints (enabled by default in RediORM)
const db = fromUri('sqlite://./app.db?_foreign_keys=1');

// Shared cache for multiple connections
const db = fromUri('sqlite://./app.db?cache=shared');
```

### Best Practices

```javascript
// Production configuration
const db = fromUri('sqlite://./production.db?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000');

// Enable connection pooling for concurrent access
await db.connect();
// SQLite uses a single connection with mutex protection
```

### Limitations

- Single writer at a time (readers can be concurrent)
- No built-in replication
- Limited concurrent write performance
- Database file size limits (depends on filesystem)

## MySQL

### Connection Options

```javascript
// Basic connection
const db = fromUri('mysql://user:password@localhost:3306/database');

// With SSL
const db = fromUri('mysql://user:password@localhost:3306/database?tls=true');

// Production configuration
const db = fromUri('mysql://user:password@localhost:3306/database?charset=utf8mb4&parseTime=true&loc=UTC');
```

### Common Options

| Option | Description | Example |
|--------|-------------|---------|
| `charset` | Character set | `utf8mb4` |
| `parseTime` | Parse time values | `true` |
| `loc` | Timezone location | `UTC`, `Local` |
| `timeout` | Connection timeout | `30s` |
| `readTimeout` | Read timeout | `30s` |
| `writeTimeout` | Write timeout | `30s` |
| `maxAllowedPacket` | Max packet size | `67108864` |
| `tls` | Enable TLS | `true`, `false`, `skip-verify` |

### Setup Guide

1. **Install MySQL 8.0+**

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install mysql-server

# macOS with Homebrew
brew install mysql

# Docker
docker run --name mysql-db -e MYSQL_ROOT_PASSWORD=root -p 3306:3306 -d mysql:8.0
```

2. **Create Database and User**

```sql
-- Connect as root
mysql -u root -p

-- Create database
CREATE DATABASE myapp CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Create user
CREATE USER 'myuser'@'localhost' IDENTIFIED BY 'mypassword';
CREATE USER 'myuser'@'%' IDENTIFIED BY 'mypassword'; -- For remote access

-- Grant privileges
GRANT ALL PRIVILEGES ON myapp.* TO 'myuser'@'localhost';
GRANT ALL PRIVILEGES ON myapp.* TO 'myuser'@'%';
FLUSH PRIVILEGES;
```

3. **Configuration Example**

```javascript
const db = fromUri(`mysql://myuser:mypassword@localhost:3306/myapp?charset=utf8mb4&parseTime=true&loc=UTC&timeout=30s`);
```

### MySQL-Specific Features

```javascript
// JSON support (MySQL 5.7+)
await db.loadSchema(`
  model User {
    id       Int  @id @default(autoincrement())
    metadata Json
  }
`);

// Full-text search
await db.queryRaw('SELECT * FROM posts WHERE MATCH(title, content) AGAINST(? IN NATURAL LANGUAGE MODE)', 'search term');

// Auto-increment with specific start value
await db.queryRaw('ALTER TABLE users AUTO_INCREMENT = 1000');
```

### Performance Tips

```javascript
// Connection pooling
const db = fromUri('mysql://user:pass@host/db?maxIdleConns=10&maxOpenConns=20&connMaxLifetime=300s');

// Batch inserts
await db.models.User.createMany({
    data: users // Insert multiple records in single query
});

// Use prepared statements (automatic in RediORM)
const users = await db.queryRaw('SELECT * FROM users WHERE age > ?', 18);
```

## PostgreSQL

### Connection Options

```javascript
// Basic connection
const db = fromUri('postgresql://user:password@localhost:5432/database');

// With SSL
const db = fromUri('postgresql://user:password@localhost:5432/database?sslmode=require');

// Production configuration
const db = fromUri('postgresql://user:password@localhost:5432/database?sslmode=require&connect_timeout=10&pool_max_conns=20');
```

### SSL Modes

| Mode | Description |
|------|-------------|
| `disable` | No SSL |
| `allow` | SSL if available |
| `prefer` | SSL preferred (default) |
| `require` | SSL required |
| `verify-ca` | SSL + verify CA |
| `verify-full` | SSL + verify CA + hostname |

### Setup Guide

1. **Install PostgreSQL 12+**

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install postgresql postgresql-contrib

# macOS with Homebrew
brew install postgresql

# Docker
docker run --name postgres-db -e POSTGRES_PASSWORD=postgres -p 5432:5432 -d postgres:15
```

2. **Create Database and User**

```bash
# Switch to postgres user
sudo -u postgres psql

-- Create database
CREATE DATABASE myapp;

-- Create user
CREATE USER myuser WITH PASSWORD 'mypassword';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE myapp TO myuser;

-- Connect to database and grant schema privileges
\c myapp
GRANT ALL ON SCHEMA public TO myuser;
```

3. **Configuration Example**

```javascript
const db = fromUri('postgresql://myuser:mypassword@localhost:5432/myapp?sslmode=prefer&connect_timeout=10');
```

### PostgreSQL-Specific Features

```javascript
// JSONB support
await db.loadSchema(`
  model User {
    id       Int  @id @default(autoincrement())
    metadata Json // Maps to JSONB in PostgreSQL
  }
`);

// Query JSONB data
const users = await db.queryRaw(`
  SELECT * FROM users 
  WHERE metadata->>'department' = ? 
  AND metadata->'settings'->>'theme' = ?
`, 'engineering', 'dark');

// Arrays support
await db.loadSchema(`
  model Post {
    id   Int      @id @default(autoincrement())
    tags String[]
  }
`);

// Query arrays
const posts = await db.queryRaw('SELECT * FROM posts WHERE ? = ANY(tags)', 'javascript');

// Full-text search
await db.queryRaw('SELECT * FROM posts WHERE to_tsvector(content) @@ plainto_tsquery(?)', 'search term');
```

### Performance Tips

```javascript
// Connection pooling
const db = fromUri('postgresql://user:pass@host/db?pool_max_conns=20&pool_min_conns=5&pool_max_conn_lifetime=1h');

// Batch operations
await db.models.User.createMany({
    data: users // Uses COPY for optimal performance
});

// Use RETURNING clause (automatic in RediORM)
const user = await db.models.User.create({
    data: { name: 'Alice' }
}); // Returns created record efficiently
```

### Advanced Configuration

```javascript
// Read replicas
const writeDb = fromUri('postgresql://user:pass@primary:5432/db');
const readDb = fromUri('postgresql://user:pass@replica:5432/db');

// Connection with specific application name
const db = fromUri('postgresql://user:pass@host/db?application_name=myapp&connect_timeout=10');
```

## MongoDB

### Connection Options

```javascript
// Local MongoDB
const db = fromUri('mongodb://localhost:27017/database');

// With authentication
const db = fromUri('mongodb://user:password@localhost:27017/database?authSource=admin');

// MongoDB Atlas
const db = fromUri('mongodb+srv://user:password@cluster.mongodb.net/database');

// Replica set (required for transactions)
const db = fromUri('mongodb://host1:27017,host2:27017,host3:27017/database?replicaSet=rs0');
```

### Setup Guide

1. **Install MongoDB 4.4+**

```bash
# Ubuntu/Debian
wget -qO - https://www.mongodb.org/static/pgp/server-6.0.asc | sudo apt-key add -
echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/6.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-6.0.list
sudo apt update
sudo apt install mongodb-org

# macOS with Homebrew
brew tap mongodb/brew
brew install mongodb-community

# Docker
docker run --name mongo-db -p 27017:27017 -d mongo:6.0
```

2. **Setup Replica Set (Required for Transactions)**

```bash
# Start MongoDB with replica set
mongod --replSet rs0

# Initialize replica set (run once)
mongosh --eval "rs.initiate()"

# Or with Docker
docker run --name mongo-rs -p 27017:27017 -d mongo:6.0 --replSet rs0
docker exec mongo-rs mongosh --eval "rs.initiate()"
```

3. **Create User**

```javascript
// Connect to MongoDB
mongosh

// Switch to admin database
use admin

// Create user
db.createUser({
  user: "myuser",
  pwd: "mypassword",
  roles: [
    { role: "readWrite", db: "myapp" },
    { role: "dbAdmin", db: "myapp" }
  ]
})
```

### MongoDB-Specific Features

#### Document Structure

```javascript
// Nested documents (no separate tables needed)
await db.models.User.create({
    data: {
        name: 'Alice',
        profile: {
            age: 25,
            address: {
                street: '123 Main St',
                city: 'Anytown',
                country: 'USA'
            }
        },
        tags: ['developer', 'javascript', 'mongodb']
    }
});
```

#### Raw MongoDB Commands

```javascript
// MongoDB find command
const users = await db.queryRaw(`{
    "find": "users",
    "filter": {"age": {"$gt": 18}},
    "sort": {"name": 1},
    "limit": 10
}`);

// Aggregation pipeline
const results = await db.queryRaw(`{
    "aggregate": "users",
    "pipeline": [
        {"$match": {"department": "engineering"}},
        {"$group": {"_id": "$level", "count": {"$sum": 1}, "avgSalary": {"$avg": "$salary"}}},
        {"$sort": {"avgSalary": -1}}
    ]
}`);

// Update operations
const result = await db.executeRaw(`{
    "updateMany": {
        "collection": "users",
        "filter": {"active": false},
        "update": {"$set": {"status": "inactive"}}
    }
}`);
```

#### SQL Translation

```javascript
// SQL queries are automatically translated to MongoDB
const users = await db.queryRaw('SELECT * FROM users WHERE age > ? ORDER BY name', 18);
// Translates to: db.users.find({age: {$gt: 18}}).sort({name: 1})

const analytics = await db.queryRaw(`
    SELECT department, COUNT(*) as count, AVG(salary) as avgSalary 
    FROM users 
    WHERE active = true 
    GROUP BY department 
    HAVING count > 5
    ORDER BY avgSalary DESC
`);
// Translates to aggregation pipeline with $match, $group, $match (having), $sort
```

### Performance Tips

```javascript
// Connection pooling
const db = fromUri('mongodb://user:pass@host/db?maxPoolSize=20&minPoolSize=5');

// Indexing
await db.queryRaw(`{
    "createIndex": {
        "collection": "users",
        "index": {"email": 1},
        "options": {"unique": true}
    }
}`);

// Compound indexes
await db.queryRaw(`{
    "createIndex": {
        "collection": "posts",
        "index": {"authorId": 1, "createdAt": -1}
    }
}`);

// Text search indexes
await db.queryRaw(`{
    "createIndex": {
        "collection": "posts",
        "index": {"title": "text", "content": "text"}
    }
}`);
```

### Limitations

- No migrations (MongoDB is schemaless)
- Transactions require replica set (MongoDB 4.0+)
- Cross-collection joins are less efficient than SQL
- No foreign key constraints (enforced at application level)

## Connection Pooling

### Default Pool Settings

```javascript
// SQLite: Single connection with mutex
// MySQL: Default pool size 10
// PostgreSQL: Default pool size 10  
// MongoDB: Default pool size 100
```

### Custom Pool Configuration

```javascript
// MySQL
const db = fromUri('mysql://user:pass@host/db?maxIdleConns=5&maxOpenConns=15&connMaxLifetime=300s');

// PostgreSQL  
const db = fromUri('postgresql://user:pass@host/db?pool_max_conns=20&pool_min_conns=5&pool_max_conn_lifetime=1h');

// MongoDB
const db = fromUri('mongodb://user:pass@host/db?maxPoolSize=50&minPoolSize=10&maxIdleTimeMS=30000');
```

### Pool Monitoring

```javascript
// Enable detailed logging to monitor pool usage
const logger = createLogger('Database');
logger.setLevel(logger.levels.DEBUG);
db.setLogger(logger);

// Monitor connection stats (database-specific)
const stats = await db.getConnectionStats(); // Implementation varies by driver
```

## Performance Optimization

### Query Optimization

```javascript
// Use indexes effectively
await db.models.User.findMany({
    where: { email: 'user@example.com' } // Ensure email has unique index
});

// Limit results
await db.models.User.findMany({
    take: 100,
    skip: 0
});

// Select specific fields (not implemented in current version)
// await db.models.User.findMany({
//     select: { id: true, name: true, email: true }
// });
```

### Eager Loading Optimization

```javascript
// Good: Single query with JOIN
await db.models.User.findMany({
    include: { posts: true }
});

// Avoid: N+1 queries
const users = await db.models.User.findMany();
for (const user of users) {
    user.posts = await db.models.Post.findMany({
        where: { authorId: user.id }
    });
}
```

### Batch Operations

```javascript
// Efficient batch inserts
await db.models.User.createMany({
    data: Array.from({ length: 1000 }, (_, i) => ({
        name: `User ${i}`,
        email: `user${i}@example.com`
    }))
});

// Batch updates
await db.models.User.updateMany({
    where: { active: false },
    data: { status: 'inactive' }
});
```

### Raw Query Optimization

```javascript
// Use prepared statements (automatic)
const users = await db.queryRaw('SELECT * FROM users WHERE department = ?', 'engineering');

// Batch raw operations in transactions
await db.transaction(async (tx) => {
    await tx.executeRaw('DELETE FROM temp_table');
    await tx.executeRaw('INSERT INTO temp_table SELECT * FROM source WHERE condition = ?', value);
    await tx.executeRaw('UPDATE main_table SET field = (SELECT field FROM temp_table WHERE id = main_table.id)');
});
```

## Migration Strategies

### Development to Production

```bash
# 1. Generate migration in development
redi-orm migrate:generate --db=sqlite://./dev.db --schema=./schema.prisma --name="add_user_table"

# 2. Test migration
redi-orm migrate --db=sqlite://./test.db --schema=./schema.prisma --dry-run

# 3. Apply to staging
redi-orm migrate --db=postgresql://user:pass@staging/db --schema=./schema.prisma

# 4. Apply to production (with backup)
pg_dump production_db > backup.sql
redi-orm migrate --db=postgresql://user:pass@production/db --schema=./schema.prisma
```

### Zero-Downtime Migrations

```bash
# 1. Add new column (nullable first)
# schema.prisma: Add field with ? (nullable)
redi-orm migrate --db=postgresql://user:pass@production/db --schema=./schema.prisma

# 2. Deploy application that populates new field
# Application code: Write to both old and new fields

# 3. Backfill existing data
redi-orm run backfill-script.js

# 4. Make field required
# schema.prisma: Remove ? from field
redi-orm migrate --db=postgresql://user:pass@production/db --schema=./schema.prisma

# 5. Remove old field (separate deployment)
```

### Cross-Database Migration

```javascript
// Migrate from SQLite to PostgreSQL
const sourceDb = fromUri('sqlite://./app.db');
const targetDb = fromUri('postgresql://user:pass@localhost/app');

await sourceDb.connect();
await targetDb.connect();

// Load same schema on both
await sourceDb.loadSchemaFrom('./schema.prisma');
await targetDb.loadSchemaFrom('./schema.prisma');
await targetDb.syncSchemas();

// Transfer data
const users = await sourceDb.models.User.findMany();
await targetDb.models.User.createMany({ data: users });

const posts = await sourceDb.models.Post.findMany();
await targetDb.models.Post.createMany({ data: posts });
```

## Troubleshooting

### Common Connection Issues

```javascript
// Connection timeout
try {
    await db.connect();
} catch (error) {
    if (error.message.includes('timeout')) {
        console.log('Increase connection timeout or check network');
    }
}

// SSL certificate issues (PostgreSQL)
const db = fromUri('postgresql://user:pass@host/db?sslmode=allow'); // Less strict SSL

// MongoDB replica set not initialized
// Error: "MongoServerError: Transaction numbers are only allowed on a replica set member"
// Solution: Initialize replica set with rs.initiate()
```

### Performance Issues

```javascript
// Enable query logging
const logger = createLogger('Performance');
logger.setLevel(logger.levels.DEBUG);
db.setLogger(logger);

// Monitor slow queries (MySQL)
await db.queryRaw('SET SESSION long_query_time = 1');
await db.queryRaw('SET SESSION log_queries_not_using_indexes = ON');

// Check PostgreSQL query plans
const plan = await db.queryRaw('EXPLAIN ANALYZE SELECT * FROM users WHERE email = ?', 'test@example.com');

// MongoDB query profiling
await db.queryRaw('{"profile": {"collection": "users", "level": 2}}');
```

### Memory Usage

```javascript
// Large result sets
const stream = db.queryRawStream('SELECT * FROM large_table');
stream.on('data', (row) => {
    // Process row by row instead of loading all into memory
});

// Limit batch sizes
const batchSize = 1000;
for (let offset = 0; ; offset += batchSize) {
    const batch = await db.models.User.findMany({
        take: batchSize,
        skip: offset
    });
    
    if (batch.length === 0) break;
    
    // Process batch
}
```

---

For more specific examples and advanced configurations, see:
- [Getting Started Guide](./getting-started.md) 
- [Advanced Features](./advanced-features.md)
- [APIs & Servers](./apis-and-servers.md)