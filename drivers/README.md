# Database Driver API

This document covers the low-level database driver API for advanced usage and direct database operations. For the high-level ORM API, see the [main README](../README.md).

## Driver Architecture

RediORM uses a three-layer driver architecture:

1. **URI Layer** - Database connection URIs (e.g., `sqlite://./app.db`)
2. **Native DSN Layer** - Database-specific connection strings
3. **Connection Layer** - Actual database connections

### Supported Drivers

- **SQLite** - File-based database with in-memory support
- **MySQL** - Full-featured MySQL/MariaDB support
- **PostgreSQL** - Complete PostgreSQL support

## Direct Driver Usage

### Importing Drivers

Import only the drivers you need:

```go
import (
    "github.com/rediwo/redi-orm/database"
    _ "github.com/rediwo/redi-orm/drivers/sqlite"     // SQLite driver
    _ "github.com/rediwo/redi-orm/drivers/mysql"      // MySQL driver  
    _ "github.com/rediwo/redi-orm/drivers/postgresql" // PostgreSQL driver
)
```

### Creating Database Connections

#### Option 1: From URI (Recommended)

```go
// SQLite
db, err := database.NewFromURI("sqlite://./myapp.db")
db, err := database.NewFromURI("sqlite://:memory:")

// MySQL
db, err := database.NewFromURI("mysql://user:pass@localhost:3306/mydb")
db, err := database.NewFromURI("mysql://user:pass@localhost/mydb?charset=utf8mb4&parseTime=true")

// PostgreSQL
db, err := database.NewFromURI("postgresql://user:pass@localhost:5432/mydb")
db, err := database.NewFromURI("postgres://user:pass@localhost/mydb?sslmode=require")
```

#### Option 2: Direct Driver Instantiation

```go
import (
    "github.com/rediwo/redi-orm/drivers/sqlite"
    "github.com/rediwo/redi-orm/drivers/mysql" 
    "github.com/rediwo/redi-orm/drivers/postgresql"
)

// SQLite - pass native file path
sqliteDB, err := sqlite.NewSQLiteDB("./myapp.db")

// MySQL - pass native DSN
mysqlDB, err := mysql.NewMySQLDB("user:pass@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=true")

// PostgreSQL - pass native DSN  
pgDB, err := postgresql.NewPostgreSQLDB("host=localhost user=user password=pass dbname=mydb")
```

## Connection URIs and Options

### SQLite URIs

```go
// File database
"sqlite://./app.db"
"sqlite:///absolute/path/to/db.sqlite"

// In-memory database
"sqlite://:memory:"
```

SQLite doesn't support query parameters in URIs but accepts PRAGMA statements after connection.

### MySQL URIs

```go
// Basic connection
"mysql://user:password@localhost:3306/database"

// With options
"mysql://user:pass@localhost/mydb?charset=utf8mb4&parseTime=true&timeout=10s"

// Common MySQL options:
// - charset=utf8mb4 (character set)
// - parseTime=true (parse TIME/DATE values)
// - timeout=10s (connection timeout)
// - readTimeout=30s (I/O read timeout)
// - writeTimeout=30s (I/O write timeout)
// - tls=true|false|skip-verify (TLS configuration)
// - interpolateParams=true (client-side parameter interpolation)
// - multiStatements=true (allow multiple statements)
```

### PostgreSQL URIs

```go
// Basic connection
"postgresql://user:password@localhost:5432/database"
"postgres://user:password@localhost:5432/database"  // Alternative scheme

// With options
"postgresql://user:pass@localhost/mydb?sslmode=require&connect_timeout=10"

// Common PostgreSQL options:
// - sslmode=disable|require|verify-ca|verify-full
// - connect_timeout=10 (connection timeout in seconds)
// - application_name=myapp (application name)
// - search_path=schema1,schema2 (schema search path)
// - timezone=UTC (session timezone)
// - client_encoding=UTF8 (client encoding)
```

## Schema Management

### Schema Definition

Define schemas using the Go native API:

```go
import "github.com/rediwo/redi-orm/schema"

// Create a new schema
userSchema := schema.New("User").
    AddField(schema.Field{
        Name:          "id",
        Type:          schema.FieldTypeInt,
        PrimaryKey:    true,
        AutoIncrement: true,
    }).
    AddField(schema.Field{
        Name:     "email",
        Type:     schema.FieldTypeString,
        Unique:   true,
        Nullable: false,
    }).
    AddField(schema.Field{
        Name:     "name", 
        Type:     schema.FieldTypeString,
        Nullable: false,
    }).
    AddField(schema.Field{
        Name:     "age",
        Type:     schema.FieldTypeInt,
        Nullable: true,
    }).
    AddField(schema.Field{
        Name:    "createdAt",
        Type:    schema.FieldTypeDateTime,
        Default: "now()",
        Map:     "created_at", // Custom column name
    })
```

### Field Types

```go
// Available field types
schema.FieldTypeString    // TEXT/VARCHAR
schema.FieldTypeInt       // INTEGER/INT
schema.FieldTypeInt64     // BIGINT
schema.FieldTypeFloat     // REAL/FLOAT
schema.FieldTypeBool      // BOOLEAN (INTEGER in SQLite)
schema.FieldTypeDateTime  // DATETIME/TIMESTAMP
schema.FieldTypeJSON      // JSON (TEXT in SQLite)
schema.FieldTypeDecimal   // DECIMAL/NUMERIC
```

### Field Attributes

```go
schema.Field{
    Name:          "fieldName",      // Field name in schema
    Type:          schema.FieldTypeString,
    PrimaryKey:    true,             // Primary key field
    AutoIncrement: true,             // Auto-increment (integers only)
    Unique:        true,             // Unique constraint
    Nullable:      false,            // NOT NULL constraint
    Default:       "defaultValue",   // Default value
    Map:           "column_name",    // Custom column name (@map annotation)
    Index:         true,             // Create index
}
```

### Composite Primary Keys

```go
postTagSchema := schema.New("PostTag").
    AddField(schema.Field{Name: "postId", Type: schema.FieldTypeInt}).
    AddField(schema.Field{Name: "tagId", Type: schema.FieldTypeInt})

// Set composite primary key
postTagSchema.CompositeKey = []string{"postId", "tagId"}
```

### Relations

```go
// One-to-many relation (User has many Posts)
userSchema.AddRelation("posts", schema.Relation{
    Type:       schema.RelationOneToMany,
    Model:      "Post",
    ForeignKey: "userId",
    References: "id",
})

// Many-to-one relation (Post belongs to User)
postSchema.AddRelation("user", schema.Relation{
    Type:       schema.RelationManyToOne, 
    Model:      "User",
    ForeignKey: "userId",
    References: "id",
    OnDelete:   "CASCADE",  // Optional: CASCADE, SET NULL, RESTRICT
    OnUpdate:   "CASCADE",  // Optional: CASCADE, SET NULL, RESTRICT
})

// Many-to-many relation (Post has many Tags through PostTag)
postSchema.AddRelation("tags", schema.Relation{
    Type:       schema.RelationManyToMany,
    Model:      "Tag",
    Through:    "PostTag",
    ForeignKey: "postId",
    References: "id",
})
```

### Loading and Syncing Schemas

```go
ctx := context.Background()

// Add schema to database
db.AddSchema(userSchema)
db.AddSchema(postSchema)

// Sync all schemas with database (creates tables)
if err := db.SyncSchemas(ctx); err != nil {
    log.Fatal(err)
}

// Alternative: Sync individual schema
if err := db.CreateModel(ctx, "User"); err != nil {
    log.Fatal(err)
}
```

## Query Builder API

### Model Queries

Get a model query builder:

```go
// Get model query builder
User := db.Model("User")
Post := db.Model("Post")
```

### Insert Operations

```go
// Single insert
result, err := User.Insert(map[string]any{
    "name":  "John Doe",
    "email": "john@example.com",
    "age":   30,
}).Exec(ctx)

if err != nil {
    log.Fatal(err)
}

// Get inserted ID (if auto-increment)
id, err := result.LastInsertId()
rowsAffected, err := result.RowsAffected()

// Insert with RETURNING (PostgreSQL, SQLite with RETURNING support)
var insertedUser map[string]any
err = User.Insert(map[string]any{
    "name":  "Jane Doe", 
    "email": "jane@example.com",
}).Returning("id", "name", "createdAt").Scan(ctx, &insertedUser)

// Batch insert
users := []map[string]any{
    {"name": "Alice", "email": "alice@example.com"},
    {"name": "Bob", "email": "bob@example.com"},
}
result, err := User.InsertMany(users).Exec(ctx)
```

### Select Operations

```go
// Find single record
var user map[string]any
err := User.Select().Where("id", 1).FindOne(ctx, &user)

// Find multiple records  
var users []map[string]any
err := User.Select().Where("age", ">", 18).FindMany(ctx, &users)

// Scan into structs
type UserStruct struct {
    ID    int    `db:"id"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

var userStruct UserStruct
err := User.Select().Where("id", 1).FindOne(ctx, &userStruct)

var userStructs []UserStruct
err := User.Select().FindMany(ctx, &userStructs)
```

### Where Conditions

```go
// Simple conditions
User.Select().Where("name", "John")           // name = 'John'
User.Select().Where("age", ">", 18)           // age > 18
User.Select().Where("email", "!=", "test")    // email != 'test'

// Multiple conditions (AND)
User.Select().
    Where("age", ">", 18).
    Where("name", "LIKE", "John%")

// OR conditions
User.Select().
    Where("age", ">", 65).
    OrWhere("status", "premium")

// IN conditions
User.Select().WhereIn("id", []any{1, 2, 3, 4})
User.Select().WhereNotIn("status", []any{"banned", "inactive"})

// NULL conditions
User.Select().WhereNull("deletedAt")
User.Select().WhereNotNull("emailVerifiedAt")

// BETWEEN conditions
User.Select().WhereBetween("age", 18, 65)
User.Select().WhereNotBetween("created_at", "2023-01-01", "2023-12-31")
```

### Ordering and Limiting

```go
// Order by
User.Select().OrderBy("name", "ASC")
User.Select().OrderBy("createdAt", "DESC")

// Multiple order conditions
User.Select().
    OrderBy("status", "ASC").
    OrderBy("createdAt", "DESC")

// Limit and offset
User.Select().Limit(10)
User.Select().Limit(10).Offset(20)  // Pagination

// Count
count, err := User.Select().Where("active", true).Count(ctx)
```

### Update Operations

```go
// Update with WHERE
result, err := User.Update(map[string]any{
    "name": "John Smith",
    "updatedAt": "now()",
}).Where("id", 1).Exec(ctx)

rowsAffected, err := result.RowsAffected()

// Update with RETURNING  
var updatedUser map[string]any
err = User.Update(map[string]any{
    "name": "Jane Smith",
}).Where("id", 1).Returning("id", "name", "updatedAt").Scan(ctx, &updatedUser)

// Batch update
result, err := User.Update(map[string]any{
    "status": "inactive",
}).Where("lastLoginAt", "<", "2023-01-01").Exec(ctx)
```

### Delete Operations

```go
// Delete with WHERE
result, err := User.Delete().Where("id", 1).Exec(ctx)
rowsAffected, err := result.RowsAffected()

// Delete with RETURNING
var deletedUser map[string]any
err = User.Delete().Where("id", 1).Returning("id", "name").Scan(ctx, &deletedUser)

// Batch delete
result, err := User.Delete().Where("active", false).Exec(ctx)

// Delete all (use with caution)
result, err := User.Delete().Exec(ctx)
```

### Eager Loading (Relations)

```go
// Include related data
var usersWithPosts []map[string]any
err := User.Select().Include("posts").FindMany(ctx, &usersWithPosts)

// Nested includes
var postsWithAuthorAndTags []map[string]any  
err := Post.Select().
    Include("author").
    Include("tags").
    FindMany(ctx, &postsWithAuthorAndTags)

// Conditional includes
var users []map[string]any
err := User.Select().
    Include("posts", func(q types.SelectQuery) types.SelectQuery {
        return q.Where("published", true).OrderBy("createdAt", "DESC")
    }).
    FindMany(ctx, &users)
```

## Raw SQL Queries

### Raw Queries

```go
// SELECT queries
rawQuery := db.Raw("SELECT * FROM users WHERE age > ?", 18)
var users []map[string]any
err := rawQuery.Scan(ctx, &users)

// Scan single row
var user map[string]any
err := db.Raw("SELECT * FROM users WHERE id = ?", 1).ScanOne(ctx, &user)

// Execute non-SELECT queries
result, err := db.Raw("UPDATE users SET status = ? WHERE age < ?", "minor", 18).Exec(ctx)
rowsAffected, err := result.RowsAffected()
```

### Using Database/SQL Directly

Access the underlying `*sql.DB`:

```go
// Get underlying database connection
sqlDB := db.GetDB()

// Standard database/sql operations
rows, err := sqlDB.QueryContext(ctx, "SELECT * FROM users")
defer rows.Close()

for rows.Next() {
    var id int
    var name, email string
    err := rows.Scan(&id, &name, &email)
    // Process row...
}
```

## Transactions

### Transaction API

```go
// Execute function in transaction
err := db.Transaction(ctx, func(tx types.Transaction) error {
    // All operations use tx instead of db
    
    // Insert user
    userResult, err := tx.Model("User").Insert(map[string]any{
        "name": "John Doe",
        "email": "john@example.com",
    }).Exec(ctx)
    if err != nil {
        return err // Automatic rollback
    }
    
    userID, _ := userResult.LastInsertId()
    
    // Insert related post
    _, err = tx.Model("Post").Insert(map[string]any{
        "title": "Hello World",
        "userId": userID,
    }).Exec(ctx)
    if err != nil {
        return err // Automatic rollback
    }
    
    // If no error, transaction commits automatically
    return nil
})

if err != nil {
    log.Printf("Transaction failed: %v", err)
}
```

### Manual Transaction Control

```go
// Begin transaction manually
tx, err := db.Begin(ctx)
if err != nil {
    log.Fatal(err)
}

// Always rollback on defer (no-op if already committed)
defer tx.Rollback(ctx)

// Perform operations
_, err = tx.Model("User").Insert(userData).Exec(ctx)
if err != nil {
    // tx.Rollback(ctx) called by defer
    return err
}

_, err = tx.Model("Post").Insert(postData).Exec(ctx)  
if err != nil {
    // tx.Rollback(ctx) called by defer
    return err
}

// Commit transaction
if err := tx.Commit(ctx); err != nil {
    return err
}
```

### Savepoints

```go
err := db.Transaction(ctx, func(tx types.Transaction) error {
    // Insert user
    _, err := tx.Model("User").Insert(userData).Exec(ctx)
    if err != nil {
        return err
    }
    
    // Create savepoint
    savepoint := "user_posts"
    if err := tx.Savepoint(ctx, savepoint); err != nil {
        return err
    }
    
    // Try to insert posts
    for _, postData := range posts {
        _, err := tx.Model("Post").Insert(postData).Exec(ctx)
        if err != nil {
            // Rollback to savepoint (keep user, discard posts)
            if rollbackErr := tx.RollbackToSavepoint(ctx, savepoint); rollbackErr != nil {
                return rollbackErr
            }
            break
        }
    }
    
    return nil
})
```

## Migration System

### Database Migrator

```go
// Get migrator for database
migrator := db.GetMigrator()

// Check if migration table exists
exists, err := migrator.MigrationTableExists(ctx)

// Create migration table
if !exists {
    err := migrator.CreateMigrationTable(ctx) 
}

// Apply migration
migration := &types.Migration{
    Name: "create_users_table",
    UpSQL: `CREATE TABLE users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT UNIQUE NOT NULL
    )`,
    DownSQL: `DROP TABLE users`,
}

err := migrator.ApplyMigration(ctx, migration)

// Rollback migration  
err := migrator.RollbackMigration(ctx, "create_users_table")

// Get migration status
applied, err := migrator.GetAppliedMigrations(ctx)
for _, migration := range applied {
    fmt.Printf("Applied: %s at %s\n", migration.Name, migration.AppliedAt)
}
```

### Schema Comparison

```go
// Compare current database schema with loaded schemas
differences, err := migrator.CompareSchemas(ctx)

for _, diff := range differences {
    switch diff.Type {
    case types.DifferenceTypeTableMissing:
        fmt.Printf("Table missing: %s\n", diff.TableName)
    case types.DifferenceTypeColumnMissing:
        fmt.Printf("Column missing: %s.%s\n", diff.TableName, diff.ColumnName)
    case types.DifferenceTypeColumnTypeMismatch:
        fmt.Printf("Column type mismatch: %s.%s\n", diff.TableName, diff.ColumnName)
    }
}
```

## Driver Capabilities

### Checking Driver Capabilities

```go
caps := db.GetCapabilities()

// Check if driver supports RETURNING clause
if caps.SupportsReturning() {
    // Use RETURNING in INSERT/UPDATE/DELETE
    var result map[string]any
    err := User.Insert(data).Returning("id", "name").Scan(ctx, &result)
}

// Check if driver requires LIMIT for OFFSET
if caps.RequiresLimitForOffset() {
    // SQLite requires LIMIT when using OFFSET
    query = query.Limit(1000000).Offset(20)
} else {
    // MySQL/PostgreSQL can use OFFSET alone
    query = query.Offset(20) 
}

// Get NULLS ordering SQL
nullsSQL := caps.GetNullsOrderingSQL("ASC", "LAST")
// Returns " NULLS LAST" for PostgreSQL/SQLite, "" for MySQL
```

### Driver-Specific Capabilities

Each driver has different capabilities:

**SQLite:**
- Supports RETURNING (in newer versions)
- Requires LIMIT when using OFFSET
- Supports NULLS FIRST/LAST ordering
- Auto-increment uses AUTOINCREMENT

**MySQL:**
- Limited RETURNING support (MySQL 8.0+)
- Doesn't require LIMIT for OFFSET
- No NULLS ordering support
- Auto-increment uses AUTO_INCREMENT

**PostgreSQL:**
- Full RETURNING support
- Doesn't require LIMIT for OFFSET
- Full NULLS ordering support
- Auto-increment uses SERIAL/BIGSERIAL

## Type Conversion Utilities

Handle driver-specific type differences:

```go
import "github.com/rediwo/redi-orm/utils"

// Safe type conversions
boolValue := utils.ToBool(result["active"])      // Handles int64 from SQLite
intValue := utils.ToInt64(result["count"])       // Handles string from MySQL
floatValue := utils.ToFloat64(result["price"])   // Handles various numeric types
stringValue := utils.ToString(result["name"])    // Handles []byte, string, etc.

// Other conversions
intValue := utils.ToInt(result["age"])
floatValue := utils.ToFloat32(result["score"])
interfaceValue := utils.ToInterface(result["data"]) // Normalizes driver-specific types
```

## Error Handling

### Database-Specific Error Handling

```go
import (
    "database/sql"
    "errors"
    "strings"
)

// Check for common errors
if err != nil {
    // No rows found
    if errors.Is(err, sql.ErrNoRows) {
        // Handle not found
        return nil, fmt.Errorf("user not found")
    }
    
    // Constraint violations (driver-specific)
    if strings.Contains(err.Error(), "UNIQUE constraint") {  // SQLite
        return nil, fmt.Errorf("email already exists")
    }
    if strings.Contains(err.Error(), "Duplicate entry") {    // MySQL
        return nil, fmt.Errorf("email already exists") 
    }
    if strings.Contains(err.Error(), "duplicate key") {      // PostgreSQL
        return nil, fmt.Errorf("email already exists")
    }
    
    // Other database errors
    return nil, fmt.Errorf("database error: %w", err)
}
```

### Connection Error Handling

```go
// Test database connection
if err := db.Ping(ctx); err != nil {
    log.Printf("Database connection failed: %v", err)
    
    // Attempt reconnection
    if err := db.Connect(ctx); err != nil {
        log.Fatal("Failed to reconnect to database")
    }
}
```

## Best Practices

### 1. Resource Management

```go
// Always close database connections
defer db.Close()

// Use context with timeouts
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Close result sets
rows, err := db.Raw("SELECT * FROM users").Query(ctx)
if err != nil {
    return err
}
defer rows.Close() // Important!
```

### 2. Connection Pooling

```go
// Configure connection pool (after Connect)
sqlDB := db.GetDB()
sqlDB.SetMaxOpenConns(25)                // Maximum open connections
sqlDB.SetMaxIdleConns(25)                // Maximum idle connections  
sqlDB.SetConnMaxLifetime(5 * time.Minute) // Connection lifetime
```

### 3. Prepared Statements

```go
// Raw queries automatically use prepared statements
for _, userID := range userIDs {
    var user map[string]any
    err := db.Raw("SELECT * FROM users WHERE id = ?", userID).ScanOne(ctx, &user)
    if err != nil {
        log.Printf("Failed to get user %d: %v", userID, err)
    }
}

// Model queries also use prepared statements internally
for _, userData := range users {
    _, err := User.Insert(userData).Exec(ctx)
    if err != nil {
        log.Printf("Failed to insert user: %v", err)
    }
}
```

### 4. Error Recovery

```go
func performDatabaseOperation() error {
    maxRetries := 3
    for i := 0; i < maxRetries; i++ {
        err := db.Transaction(ctx, func(tx types.Transaction) error {
            // Database operations...
            return nil
        })
        
        if err == nil {
            return nil // Success
        }
        
        // Check if error is retryable
        if isRetryableError(err) && i < maxRetries-1 {
            time.Sleep(time.Duration(i+1) * time.Second)
            continue
        }
        
        return err
    }
    return nil
}

func isRetryableError(err error) bool {
    errStr := err.Error()
    return strings.Contains(errStr, "connection") ||
           strings.Contains(errStr, "timeout") ||
           strings.Contains(errStr, "deadlock")
}
```

## Testing

### Test Database Setup

```go
func setupTestDB(t *testing.T) types.Database {
    // Use in-memory SQLite for tests
    db, err := database.NewFromURI("sqlite://:memory:")
    require.NoError(t, err)
    
    err = db.Connect(context.Background())
    require.NoError(t, err)
    
    // Load test schemas
    db.AddSchema(userSchema)
    db.AddSchema(postSchema)
    
    err = db.SyncSchemas(context.Background())
    require.NoError(t, err)
    
    t.Cleanup(func() {
        db.Close()
    })
    
    return db
}

func TestUserOperations(t *testing.T) {
    db := setupTestDB(t)
    ctx := context.Background()
    
    // Test operations...
    User := db.Model("User")
    
    result, err := User.Insert(map[string]any{
        "name": "Test User",
        "email": "test@example.com",
    }).Exec(ctx)
    
    assert.NoError(t, err)
    
    id, err := result.LastInsertId()
    assert.NoError(t, err)
    assert.Greater(t, id, int64(0))
}
```

### Mock Testing

```go
// For unit testing without database
import "github.com/rediwo/redi-orm/test/mocks"

func TestBusinessLogic(t *testing.T) {
    mockDB := mocks.NewMockDatabase()
    
    // Configure mock behavior
    mockDB.On("Model", "User").Return(mocks.NewMockModelQuery())
    
    // Test your business logic
    service := NewUserService(mockDB)
    user, err := service.CreateUser("John", "john@example.com")
    
    assert.NoError(t, err)
    assert.Equal(t, "John", user.Name)
    
    mockDB.AssertExpectations(t)
}
```

This covers the comprehensive low-level driver API. For higher-level operations and rapid development, use the [ORM API](../README.md) instead.