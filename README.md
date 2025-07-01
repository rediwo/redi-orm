# RediORM - Schema-driven ORM for Go with JavaScript API

RediORM is a modern, schema-driven ORM for Go that provides a JavaScript interface for database operations using the Goja JavaScript engine. It's inspired by Prisma and offers both native Go schema definition and Prisma schema parsing capabilities, supporting SQLite, MySQL, and PostgreSQL.

## ✨ Key Features

### 🎯 **Dual Schema Approach**
- **Native Go Schema API** - Define schemas using fluent Go code
- **Prisma Schema Support** - Parse and use existing Prisma schemas directly
- **Seamless Integration** - Mix both approaches in the same project

### 🚀 **JavaScript Runtime Integration**
- **JavaScript API** - Database operations via clean JavaScript interface
- **Goja Engine** - Fast V8-compatible JavaScript runtime in Go
- **Type Safety** - Automatic validation based on schema definitions

### 🗃️ **Multi-Database Support**
- **SQLite** - Full implementation with in-memory and file-based databases
- **MySQL** - Complete driver implementation
- **PostgreSQL** - Full PostgreSQL support
- **URI-based Configuration** - Easy database switching

### 🔧 **Advanced Query Builder**
- **Chainable API** - Fluent query building with method chaining
- **Rich Operators** - Support for complex WHERE conditions
- **Aggregation** - Count, sum, and other aggregate functions
- **Pagination** - Built-in limit and offset support

### 🏗️ **Enterprise-Ready Features**
- **Transaction Support** - ACID-compliant transactions
- **Connection Pooling** - Efficient database connection management
- **Type Validation** - Runtime type checking and validation
- **Schema Validation** - Comprehensive schema integrity checks

## 📦 Installation

```bash
go get github.com/rediwo/redi-orm
```

## 🚀 Quick Start

### 🔄 Auto-Migration System

RediORM features a **sophisticated auto-migration system** that automatically manages your database schema:

#### **Two-Phase Migration Approach**

1. **Register Phase**: Define and register all your schemas without creating tables
2. **Migration Phase**: Call `EnsureSchema()` to analyze and apply all changes at once

```go
// Phase 1: Register schemas (fast, no database operations)
eng.RegisterSchema(userSchema)
eng.RegisterSchema(postSchema)
eng.RegisterSchema(commentSchema)

// Phase 2: Auto-migration (intelligent schema analysis and updates)
err := eng.EnsureSchema()
```

#### **Migration Features**

- 🆕 **Table Creation** - Automatically creates missing tables
- 🔄 **Schema Evolution** - Detects and applies schema changes
- 🛡️ **Safe Migrations** - Non-destructive operations by default
- 📊 **Migration History** - Tracks all schema changes in `redi_migrations` table
- 🔍 **Schema Introspection** - Compares current database with desired schemas
- ⚡ **Batch Operations** - Processes multiple changes efficiently
- 🎯 **Dependency Resolution** - Handles foreign key relationships correctly
- 🔄 **Idempotent** - Safe to call multiple times

#### **Migration Process**

```mermaid
graph TD
    A[Register Schemas] --> B[Call EnsureSchema]
    B --> C{Analyze Database}
    C --> D[Get Existing Tables]
    C --> E[Compare Schemas]
    D --> F[Generate Migration Plan]
    E --> F
    F --> G{Changes Needed?}
    G -->|Yes| H[Execute Migrations]
    G -->|No| I[Complete - No Changes]
    H --> J[Update Migration History]
    J --> I
```

#### **What Gets Migrated**

| Change Type | Status | Description |
|-------------|--------|-------------|
| 🆕 **New Tables** | ✅ Supported | Creates tables for new schemas |
| ➕ **Add Columns** | ✅ Supported | Adds new fields to existing tables |
| 🔧 **Modify Columns** | ✅ Supported* | Changes field types, constraints |
| ➖ **Drop Columns** | ✅ Supported* | Removes unused fields (opt-in) |
| 🔗 **Add Indexes** | 🚧 In Progress | Creates new database indexes |
| ❌ **Drop Indexes** | 🚧 In Progress | Removes unused indexes |
| 🔑 **Foreign Keys** | 🚧 In Progress | Manages relationships |

*_Note: For SQLite, MODIFY and DROP COLUMN operations are detected but require table recreation (SQLite limitation). Other databases support these operations directly._

#### **Column Migration Examples**

**Adding New Columns:**
```go
// Version 1: Basic user schema
userSchema := schema.New("User").
    AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
    AddField(schema.NewField("name").String().Build())

eng.RegisterSchema(userSchema)
eng.EnsureSchema() // Creates table with id, name

// Version 2: Add email column
userSchemaV2 := schema.New("User").
    AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
    AddField(schema.NewField("name").String().Build()).
    AddField(schema.NewField("email").String().Nullable().Build()) // New column

eng.RegisterSchema(userSchemaV2)
eng.EnsureSchema() // Automatically adds email column to existing table
```

**Adding Columns with Defaults:**
```go
userSchemaV3 := schema.New("User").
    AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
    AddField(schema.NewField("name").String().Build()).
    AddField(schema.NewField("email").String().Nullable().Build()).
    AddField(schema.NewField("active").Bool().Default(true).Build()).  // Default value
    AddField(schema.NewField("score").Float().Default(0.0).Build())    // Default value

eng.RegisterSchema(userSchemaV3)
eng.EnsureSchema() // Adds active and score columns with defaults
```

**Schema Evolution Benefits:**
- 🛡️ **Data Preservation**: Existing data is never lost during column additions
- 🔄 **Backward Compatibility**: Old code continues to work with new schemas
- ⚡ **Automatic Detection**: Only necessary changes are applied
- 🎯 **Safe Defaults**: New columns with defaults don't break existing queries
- 🔧 **Production Ready**: Migrations can be applied to live databases safely

### Using Native Go API

```go
package main

import (
    "log"
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/schema"
)

func main() {
    // Create database connection using URI
    db, err := database.NewFromURI("sqlite://./example.db")
    if err != nil {
        log.Fatal(err)
    }
    
    if err := db.Connect(); err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Define schema using fluent API
    userSchema := schema.New("User").
        AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
        AddField(schema.NewField("name").String().Build()).
        AddField(schema.NewField("email").String().Unique().Build()).
        AddField(schema.NewField("age").Int().Nullable().Build()).
        AddField(schema.NewField("active").Bool().Default(true).Build())
    
    // Register schema for name conversion
    db.RegisterSchema("User", userSchema)
    
    // Create table
    if err := db.CreateTable(userSchema); err != nil {
        log.Fatal(err)
    }
    
    // Create user
    userData := map[string]interface{}{
        "name":  "Alice",
        "email": "alice@example.com", 
        "age":   30,
    }
    userID, err := db.Insert("User", userData)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Created user ID: %v", userID)
    
    // Get user by ID
    user, err := db.FindByID("User", userID)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("User: %+v", user)
    
    // Query with conditions
    users, err := db.Select("User", nil).
        Where("age", ">", 25).
        OrderBy("name", "ASC").
        Execute()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Adult users: %+v", users)
    
    // Update user
    updateData := map[string]interface{}{"age": 31}
    err = db.Update("User", userID, updateData)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Updated user age")
    
    // Count users
    count, err := db.Select("User", nil).Count()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Total users: %d", count)
    
    // Delete user
    err = db.Delete("User", userID)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Deleted user")
}
```

### Using JavaScript API

```go
package main

import (
    "log"
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/engine"
    "github.com/rediwo/redi-orm/schema"
)

func main() {
    // Create database connection using URI
    db, err := database.NewFromURI("sqlite://./example.db")
    if err != nil {
        log.Fatal(err)
    }
    
    if err := db.Connect(); err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Create engine for JavaScript API
    eng := engine.New(db)
    
    // Define schema using fluent API
    userSchema := schema.New("User").
        AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
        AddField(schema.NewField("name").String().Build()).
        AddField(schema.NewField("email").String().Unique().Build()).
        AddField(schema.NewField("age").Int().Nullable().Build()).
        AddField(schema.NewField("active").Bool().Default(true).Build())
    
    // Register schema with engine (does not create table yet)
    if err := eng.RegisterSchema(userSchema); err != nil {
        log.Fatal(err)
    }
    
    // Create all registered tables
    if err := eng.EnsureSchema(); err != nil {
        log.Fatal(err)
    }
    
    // Create user using JavaScript API
    userID, err := eng.Execute(`models.User.add({
        name: "Alice", 
        email: "alice@example.com",
        age: 30
    })`)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Created user ID: %v", userID)
    
    // Get user using JavaScript API
    user, err := eng.Execute(`models.User.get(1)`)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("User: %+v", user)
    
    // Query with conditions using JavaScript API
    users, err := eng.Execute(`
        models.User.select()
            .where("age", ">", 25)
            .orderBy("name", "ASC")
            .execute()
    `)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Adult users: %+v", users)
    
    // Update user using JavaScript API
    _, err = eng.Execute(`models.User.set(1, {age: 31})`)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Updated user age")
    
    // Count users using JavaScript API
    count, err := eng.Execute(`models.User.select().count()`)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Total users: %v", count)
    
    // Delete user using JavaScript API
    _, err = eng.Execute(`models.User.remove(1)`)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Deleted user")
}
```

### Using Prisma Schema with Go API

```go
package main

import (
    "log"
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/orm"
)

func main() {
    // Create database connection using URI
    db, err := database.NewFromURI("sqlite://:memory:")
    if err != nil {
        log.Fatal(err)
    }
    
    if err := db.Connect(); err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Define schema using Prisma syntax
    prismaSchema := `
    enum UserRole {
      ADMIN
      USER
      MODERATOR
    }

    model User {
      id        Int      @id @default(autoincrement())
      email     String   @unique
      name      String
      role      UserRole @default(USER)
      posts     Post[]
      createdAt DateTime @default(now())
      
      @@map("users")
    }

    model Post {
      id        Int     @id @default(autoincrement())
      title     String
      content   String
      published Boolean @default(false)
      authorId  Int
      author    User    @relation(fields: [authorId], references: [id])
      createdAt DateTime @default(now())
      
      @@index([published])
      @@index([authorId])
    }
    `
    
    // Initialize ORM with Prisma schema
    if err := orm.InitializeFromSchema(prismaSchema, db); err != nil {
        log.Fatal(err)
    }
    
    // Create user using Go API
    userData := map[string]interface{}{
        "name":  "John Doe",
        "email": "john@example.com",
        "role":  "ADMIN",
    }
    userID, err := db.Insert("User", userData)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Created user ID: %v", userID)
    
    // Create post using Go API
    postData := map[string]interface{}{
        "title":     "My First Post",
        "content":   "Hello, world!",
        "authorId":  userID,
        "published": true,
    }
    postID, err := db.Insert("Post", postData)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Created post ID: %v", postID)
    
    // Query posts using Go API
    posts, err := db.Select("Post", nil).
        Where("published", "=", true).
        OrderBy("createdAt", "DESC").
        Execute()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Published posts: %+v", posts)
    
    // Get user with complex query
    user, err := db.FindByID("User", userID)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("User: %+v", user)
}
```

### Using Prisma Schema with JavaScript API

```go
package main

import (
    "fmt"
    "log"
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/engine"
)

func main() {
    // Create database connection using URI
    db, err := database.NewFromURI("sqlite://:memory:")
    if err != nil {
        log.Fatal(err)
    }
    
    if err := db.Connect(); err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Create engine for JavaScript API
    eng := engine.New(db)
    
    // Define schema using Prisma syntax
    prismaSchema := `
    enum UserRole {
      ADMIN
      USER
      MODERATOR
    }

    model User {
      id        Int      @id @default(autoincrement())
      email     String   @unique
      name      String
      role      UserRole @default(USER)
      posts     Post[]
      createdAt DateTime @default(now())
      
      @@map("users")
    }

    model Post {
      id        Int     @id @default(autoincrement())
      title     String
      content   String
      published Boolean @default(false)
      authorId  Int
      author    User    @relation(fields: [authorId], references: [id])
      createdAt DateTime @default(now())
      
      @@index([published])
      @@index([authorId])
    }
    `
    
    // Load Prisma schema
    if err := eng.LoadPrismaSchema(prismaSchema); err != nil {
        log.Fatal(err)
    }
    
    // Create all registered tables
    if err := eng.EnsureSchema(); err != nil {
        log.Fatal(err)
    }
    
    // Create user using JavaScript API
    userID, err := eng.Execute(`models.User.add({
        name: "John Doe", 
        email: "john@example.com",
        role: "ADMIN"
    })`)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Created user ID: %v", userID)
    
    // Create post using JavaScript API
    postID, err := eng.Execute(`models.Post.add({
        title: "My First Post",
        content: "Hello, world!",
        authorId: ` + fmt.Sprintf("%v", userID) + `,
        published: true
    })`)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Created post ID: %v", postID)
    
    // Query posts using JavaScript API
    posts, err := eng.Execute(`
        models.Post.select()
            .where("published", "=", true)
            .orderBy("createdAt", "DESC")
            .execute()
    `)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Published posts: %+v", posts)
    
    // Complex query using JavaScript API
    userPosts, err := eng.Execute(`
        models.Post.select()
            .where("authorId", "=", ` + fmt.Sprintf("%v", userID) + `)
            .count()
    `)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("User's post count: %v", userPosts)
}
```

## 🎮 API Reference - JavaScript vs Go

### Model Operations

| Operation | JavaScript API | Equivalent Go API |
|-----------|---------------|-------------------|
| **Create** | `models.User.add({name: "Alice", email: "alice@example.com"})` | `db.Insert("User", map[string]interface{}{"name": "Alice", "email": "alice@example.com"})` |
| **Read by ID** | `models.User.get(1)` | `db.FindByID("User", 1)` |
| **Update** | `models.User.set(1, {age: 31})` | `db.Update("User", 1, map[string]interface{}{"age": 31})` |
| **Delete** | `models.User.remove(1)` | `db.Delete("User", 1)` |
| **Select All** | `models.User.select().execute()` | `db.Select("User", nil).Execute()` |
| **Count** | `models.User.select().count()` | `db.Select("User", nil).Count()` |
| **First** | `models.User.select().first()` | `db.Select("User", nil).First()` |

### JavaScript API Examples

```javascript
// Create records
const userID = models.User.add({
    name: "Alice",
    email: "alice@example.com",
    age: 30
});

// Read records
const user = models.User.get(1);
const users = models.User.select().execute();

// Update records
models.User.set(1, { age: 31 });

// Delete records
models.User.remove(1);
```

### Equivalent Go API Examples

```go
// Create records
userData := map[string]interface{}{
    "name":  "Alice",
    "email": "alice@example.com",
    "age":   30,
}
userID, err := db.Insert("User", userData)

// Read records
user, err := db.FindByID("User", 1)
users, err := db.Select("User", nil).Execute()

// Update records
updateData := map[string]interface{}{"age": 31}
err := db.Update("User", 1, updateData)

// Delete records
err := db.Delete("User", 1)
```

### Query Builder Comparison

| Query Type | JavaScript API | Equivalent Go API |
|------------|---------------|-------------------|
| **Select All** | `models.User.select().execute()` | `db.Select("User", nil).Execute()` |
| **Select Columns** | `models.User.select(["name", "email"]).execute()` | `db.Select("User", []string{"name", "email"}).Execute()` |
| **WHERE Clause** | `models.User.select().where("age", ">", 18).execute()` | `db.Select("User", nil).Where("age", ">", 18).Execute()` |
| **ORDER BY** | `models.User.select().orderBy("name", "ASC").execute()` | `db.Select("User", nil).OrderBy("name", "ASC").Execute()` |
| **LIMIT/OFFSET** | `models.User.select().limit(10).offset(20).execute()` | `db.Select("User", nil).Limit(10).Offset(20).Execute()` |

### JavaScript Query Builder

```javascript
// Basic queries
models.User.select().execute()                    // SELECT * FROM users
models.User.select(["name", "email"]).execute()   // SELECT name, email FROM users

// WHERE clauses
models.User.select()
    .where("age", ">", 18)
    .where("active", "=", true)
    .execute()

// Ordering and pagination
models.User.select()
    .orderBy("name", "ASC")
    .limit(10)
    .offset(20)
    .execute()

// Aggregation
models.User.select().count()                      // Count all users
models.User.select().where("age", ">", 18).count() // Count adult users

// Get first result
models.User.select()
    .where("email", "=", "alice@example.com")
    .first()
```

### Equivalent Go Query Builder

```go
// Basic queries
users, err := db.Select("User", nil).Execute()                    // SELECT * FROM users
users, err := db.Select("User", []string{"name", "email"}).Execute() // SELECT name, email FROM users

// WHERE clauses
users, err := db.Select("User", nil).
    Where("age", ">", 18).
    Where("active", "=", true).
    Execute()

// Ordering and pagination
users, err := db.Select("User", nil).
    OrderBy("name", "ASC").
    Limit(10).
    Offset(20).
    Execute()

// Aggregation
count, err := db.Select("User", nil).Count()                      // Count all users
count, err := db.Select("User", nil).Where("age", ">", 18).Count() // Count adult users

// Get first result
user, err := db.Select("User", nil).
    Where("email", "=", "alice@example.com").
    First()
```

### Advanced Queries

```javascript
// Complex conditions
models.Post.select()
    .where("published", "=", true)
    .where("createdAt", ">", "2023-01-01")
    .orderBy("createdAt", "DESC")
    .limit(5)
    .execute()

// Pattern matching
models.User.select()
    .where("name", "like", "%john%")
    .execute()

// Multiple sort orders
models.Post.select()
    .orderBy("published", "DESC")
    .orderBy("createdAt", "DESC")
    .execute()
```

## 🏗️ Schema Definition

### Native Go Schema API

```go
// Complete schema example
userSchema := schema.New("User").
    WithTableName("users").  // Custom table name
    AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
    AddField(schema.NewField("email").String().Unique().Build()).
    AddField(schema.NewField("name").String().Build()).
    AddField(schema.NewField("age").Int().Nullable().Build()).
    AddField(schema.NewField("bio").String().Nullable().Build()).
    AddField(schema.NewField("active").Bool().Default(true).Build()).
    AddField(schema.NewField("createdAt").DateTime().Default("CURRENT_TIMESTAMP").Build()).
    AddField(schema.NewField("metadata").JSON().Nullable().Build()).
    AddIndex(schema.Index{
        Name:   "idx_email_active",
        Fields: []string{"email", "active"},
        Unique: false,
    }).
    AddRelation("posts", schema.Relation{
        Type:       schema.RelationOneToMany,
        Model:      "Post",
        ForeignKey: "user_id",
        References: "id",
    })
```

### Prisma Schema Features

RediORM supports comprehensive Prisma schema syntax:

```prisma
// Enums with mapping
enum Status {
  DRAFT     @map("draft")
  PUBLISHED @map("published")
  ARCHIVED  @map("archived")
}

// Models with advanced attributes
model User {
  id          Int      @id @default(autoincrement())
  email       String   @unique
  name        String
  bio         String?  @db.Text
  age         Int?
  balance     Decimal  @db.Decimal(10,2)
  status      Status   @default(DRAFT)
  preferences Json?
  tags        String[] // Scalar arrays
  createdAt   DateTime @default(now())
  updatedAt   DateTime @default(now()) @updatedAt
  
  // Relations
  posts       Post[]
  profile     Profile?
  
  // Block-level attributes
  @@unique([email])
  @@index([status, createdAt])
  @@map("users")
}

// Composite primary keys
model UserRole {
  userId   Int
  roleId   Int
  grantedAt DateTime @default(now())
  
  @@id([userId, roleId])
}

// Database-specific attributes
model Product {
  id       Int     @id @default(autoincrement())
  name     String  @db.VarChar(255)
  price    Decimal @db.Money
  metadata Json    @db.JsonB
}
```

## 🗃️ Database Configuration

### URI-based Configuration

```go
// SQLite
db, err := database.NewFromURI("sqlite://./database.db")
db, err := database.NewFromURI("sqlite://:memory:")

// MySQL
db, err := database.NewFromURI("mysql://user:password@localhost:3306/dbname")

// PostgreSQL
db, err := database.NewFromURI("postgresql://user:password@localhost:5432/dbname")
```

### Structured Configuration

```go
// SQLite
config := types.Config{
    Type:     "sqlite",
    FilePath: "database.db",
}

// MySQL
config := types.Config{
    Type:     "mysql",
    Host:     "localhost",
    Port:     3306,
    Database: "myapp",
    User:     "username",
    Password: "password",
}

// PostgreSQL
config := types.Config{
    Type:     "postgresql",
    Host:     "localhost",
    Port:     5432,
    Database: "myapp",
    User:     "username",
    Password: "password",
}

db, err := database.New(config)
```

## 🔄 Migration System Deep Dive

### Migration Workflow Examples

#### Basic Migration Scenario

```go
package main

import (
    "log"
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/engine"
    "github.com/rediwo/redi-orm/schema"
)

func main() {
    // Connect to database
    db, err := database.NewFromURI("sqlite://./app.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    if err := db.Connect(); err != nil {
        log.Fatal(err)
    }
    
    // Create engine
    eng := engine.New(db)
    
    // Define initial schema
    userSchema := schema.New("User").
        AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
        AddField(schema.NewField("email").String().Unique().Build()).
        AddField(schema.NewField("name").String().Build())
    
    // Register and migrate
    eng.RegisterSchema(userSchema)
    
    // First migration - creates tables
    if err := eng.EnsureSchema(); err != nil {
        log.Fatal("Migration failed:", err)
    }
    log.Println("✅ Initial migration completed")
    
    // Add more schemas later
    postSchema := schema.New("Post").
        AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
        AddField(schema.NewField("title").String().Build()).
        AddField(schema.NewField("content").String().Build()).
        AddField(schema.NewField("authorId").Int64().Build())
    
    eng.RegisterSchema(postSchema)
    
    // Second migration - adds new tables only
    if err := eng.EnsureSchema(); err != nil {
        log.Fatal("Migration failed:", err)
    }
    log.Println("✅ Schema evolution completed")
}
```

#### Migration History Tracking

RediORM automatically tracks migration history in the `redi_migrations` table:

```sql
CREATE TABLE redi_migrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    migration_name TEXT NOT NULL,
    schema_hash TEXT NOT NULL,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    database_type TEXT NOT NULL
);
```

#### Advanced Migration Features

```go
// Check migration status
migrator := db.GetMigrator()
if migrator != nil {
    tables, err := migrator.GetTables()
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Existing tables: %v", tables)
    
    // Get detailed table information
    for _, tableName := range tables {
        tableInfo, err := migrator.GetTableInfo(tableName)
        if err != nil {
            continue
        }
        log.Printf("Table %s has %d columns", tableName, len(tableInfo.Columns))
    }
}

// Manual migration SQL generation
schema := schema.New("CustomTable").
    AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
    AddField(schema.NewField("data").JSON().Build())

if migrator != nil {
    sql, err := migrator.GenerateCreateTableSQL(schema)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("Generated SQL: %s", sql)
}
```

### Migration Best Practices

#### 1. **Development Workflow**

```go
func setupDevelopment() {
    db, _ := database.NewFromURI("sqlite://:memory:")
    eng := engine.New(db)
    
    // Register all schemas at startup
    eng.RegisterSchema(userSchema)
    eng.RegisterSchema(postSchema)
    eng.RegisterSchema(commentSchema)
    
    // Single migration call
    if err := eng.EnsureSchema(); err != nil {
        log.Fatal("Development setup failed:", err)
    }
}
```

#### 2. **Production Deployment**

```go
func deployToProduction() {
    db, _ := database.NewFromURI("postgresql://user:pass@prod-db:5432/app")
    eng := engine.New(db)
    
    // Load all current schemas
    loadAllSchemas(eng)
    
    // Apply migrations safely
    if err := eng.EnsureSchema(); err != nil {
        log.Fatal("Production migration failed:", err)
    }
    
    log.Println("✅ Production deployment completed")
}
```

#### 3. **Testing with Migrations**

```go
func TestWithMigration(t *testing.T) {
    // Use in-memory database for tests
    db, err := database.NewFromURI("sqlite://:memory:")
    require.NoError(t, err)
    
    eng := engine.New(db)
    
    // Set up test schema
    testSchema := schema.New("TestModel").
        AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
        AddField(schema.NewField("name").String().Build())
    
    err = eng.RegisterSchema(testSchema)
    require.NoError(t, err)
    
    // Migrate before testing
    err = eng.EnsureSchema()
    require.NoError(t, err)
    
    // Now run tests...
    result, err := eng.Execute(`models.TestModel.add({name: "test"})`)
    require.NoError(t, err)
    assert.Equal(t, int64(1), result)
}
```

#### 4. **Schema Versioning Strategy**

```go
type SchemaVersion struct {
    Version int
    Schemas []*schema.Schema
}

func migrateToVersion(eng *engine.Engine, targetVersion int) error {
    versions := []SchemaVersion{
        {Version: 1, Schemas: []*schema.Schema{userSchemaV1}},
        {Version: 2, Schemas: []*schema.Schema{userSchemaV1, postSchemaV1}},
        {Version: 3, Schemas: []*schema.Schema{userSchemaV2, postSchemaV1, commentSchemaV1}},
    }
    
    for _, version := range versions {
        if version.Version <= targetVersion {
            for _, sch := range version.Schemas {
                if err := eng.RegisterSchema(sch); err != nil {
                    return err
                }
            }
        }
    }
    
    return eng.EnsureSchema()
}
```

### Migration Troubleshooting

#### Common Issues and Solutions

**Issue: "Table already exists" errors**
```go
// EnsureSchema is idempotent - this won't happen
// But if you're getting this error, check:
migrator := db.GetMigrator()
tables, _ := migrator.GetTables()
log.Printf("Existing tables: %v", tables)
```

**Issue: Schema registration errors**
```go
// Validate schema before registration
if err := schema.Validate(); err != nil {
    log.Printf("Schema validation failed: %v", err)
    return err
}
```

**Issue: Migration performance**
```go
// Register all schemas first, then migrate once
for _, sch := range allSchemas {
    eng.RegisterSchema(sch)  // Fast operation
}
eng.EnsureSchema()  // Single migration operation
```

## 🔧 Field Types and Modifiers

### Available Field Types

```go
// Scalar types
schema.NewField("name").String().Build()           // VARCHAR/TEXT
schema.NewField("age").Int().Build()               // INTEGER
schema.NewField("user_id").Int64().Build()         // BIGINT
schema.NewField("price").Float().Build()           // REAL/FLOAT
schema.NewField("amount").Decimal().Build()        // DECIMAL (precise)
schema.NewField("active").Bool().Build()           // BOOLEAN
schema.NewField("created").DateTime().Build()      // TIMESTAMP
schema.NewField("metadata").JSON().Build()         // JSON

// Array types (PostgreSQL)
schema.NewField("tags").StringArray().Build()      // TEXT[]
schema.NewField("scores").IntArray().Build()       // INTEGER[]
schema.NewField("prices").FloatArray().Build()     // REAL[]
schema.NewField("flags").BoolArray().Build()       // BOOLEAN[]
```

### Field Modifiers

```go
schema.NewField("id").
    Int64().
    PrimaryKey().           // Primary key
    AutoIncrement().        // Auto-increment
    Build()

schema.NewField("email").
    String().
    Unique().              // Unique constraint
    Build()

schema.NewField("bio").
    String().
    Nullable().            // Allow NULL
    Build()

schema.NewField("active").
    Bool().
    Default(true).         // Default value
    Build()

schema.NewField("name").
    String().
    Index().               // Create index
    Build()
```

## ⚡ Advanced Features

### Transactions

```go
// Native Go transactions
tx, err := db.Begin()
if err != nil {
    return err
}

userID, err := tx.Insert("users", userData)
if err != nil {
    tx.Rollback()
    return err
}

err = tx.Update("profiles", profileID, profileData)
if err != nil {
    tx.Rollback()
    return err
}

return tx.Commit()
```

### Composite Primary Keys

```go
// Using Go API
userRoleSchema := schema.New("UserRole").
    AddField(schema.NewField("userId").Int().Build()).
    AddField(schema.NewField("roleId").Int().Build()).
    AddField(schema.NewField("grantedAt").DateTime().Build()).
    WithCompositeKey([]string{"userId", "roleId"})

// Using Prisma schema
`
model UserRole {
  userId   Int
  roleId   Int
  grantedAt DateTime @default(now())
  
  @@id([userId, roleId])
}
`
```

### Custom Indexes

```go
// Single field index
schema.AddIndex(schema.Index{
    Name:   "idx_email",
    Fields: []string{"email"},
    Unique: true,
})

// Multi-field index
schema.AddIndex(schema.Index{
    Name:   "idx_status_created",
    Fields: []string{"status", "createdAt"},
    Unique: false,
})
```

## 🧪 Testing

RediORM provides excellent testing support with in-memory databases:

```go
func TestUserOperations(t *testing.T) {
    // Create in-memory database for testing
    db, err := database.NewFromURI("sqlite://:memory:")
    require.NoError(t, err)
    
    err = db.Connect()
    require.NoError(t, err)
    defer db.Close()
    
    // Set up engine and schema
    eng := engine.New(db)
    userSchema := createUserSchema()
    err = eng.RegisterSchema(userSchema)
    require.NoError(t, err)
    
    // Create tables
    err = eng.EnsureSchema()
    require.NoError(t, err)
    
    // Test operations
    userID, err := eng.Execute(`models.User.add({name: "Test User", email: "test@example.com"})`)
    require.NoError(t, err)
    assert.Equal(t, int64(1), userID)
    
    user, err := eng.Execute(`models.User.get(1)`)
    require.NoError(t, err)
    
    userData := user.(map[string]interface{})
    assert.Equal(t, "Test User", userData["name"])
    
    // Test Go API as well
    userID2, err := db.Insert("User", map[string]interface{}{
        "name": "Test User 2", "email": "test2@example.com"})
    require.NoError(t, err)
    
    user2, err := db.FindByID("User", userID2)
    require.NoError(t, err)
    assert.Equal(t, "Test User 2", user2["name"])
}
```

## 🏗️ Development Commands

```bash
# Build the project
make build

# Run all tests
make test

# Test specific databases
make test-sqlite       # SQLite tests only
make test-mysql        # MySQL tests only  
make test-postgresql   # PostgreSQL tests only

# Docker-based testing
make docker-up         # Start test databases
make docker-down       # Stop test databases
make test-docker       # Run tests with Docker databases

# Test coverage and quality
make test-cover        # Run tests with coverage
make test-race         # Run tests with race detection
make test-benchmark    # Run benchmark tests

# Code quality
make fmt               # Format code
make lint              # Run linter
make vet               # Run go vet

# Development workflows
make dev               # fmt + vet + test
make ci                # race detection + coverage
make all               # complete build + test workflow
```

### 🐳 Docker Testing Setup

RediORM includes a complete Docker setup for testing with real databases:

```yaml
# docker-compose.yml
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: testpass
      MYSQL_DATABASE: testdb
      MYSQL_USER: testuser
      MYSQL_PASSWORD: testpass
    ports:
      - "3306:3306"

  postgresql:
    image: postgres:15
    environment:
      POSTGRES_DB: testdb
      POSTGRES_USER: testuser
      POSTGRES_PASSWORD: testpass
    ports:
      - "5432:5432"
```

```bash
# Start databases and run tests
make docker-up && make test-docker

# Or test individual databases
make docker-up
make test-mysql
make test-postgresql
make docker-down
```

## 📊 Performance

RediORM is designed for high performance:

- **Fast JavaScript Engine** - Goja provides near-native JavaScript performance
- **Connection Pooling** - Efficient database connection management
- **Query Optimization** - Optimized SQL generation
- **In-Memory Testing** - Lightning-fast test execution
- **Minimal Overhead** - Direct SQL execution without excessive abstraction

## 🚦 Production Ready

### ✅ Completed Features
- ✅ **Multi-Database Support** - SQLite, MySQL, PostgreSQL drivers with full feature parity
- ✅ **Schema-Aware Operations** - Automatic model-to-table and field-to-column name conversion
- ✅ **Comprehensive Prisma Integration** - Full Prisma schema parsing with advanced features
- ✅ **Dual API Design** - Both JavaScript and native Go APIs for maximum flexibility
- ✅ **Advanced Query Builder** - Chainable operations with complex WHERE conditions
- ✅ **Transaction Support** - ACID-compliant transactions across all databases
- ✅ **Type System** - Complete field type support with validation
- ✅ **Schema Migration** - Automatic database table generation and migration
- ✅ **URI-Based Configuration** - Flexible database connection management
- ✅ **Field Name Mapping** - Support for @map() annotations and custom field names
- ✅ **Composite Keys** - Multi-field primary keys and unique constraints
- ✅ **Database Introspection** - Schema diffing and migration history
- ✅ **Docker Integration** - Complete Docker setup for testing
- ✅ **Comprehensive Testing** - 100+ tests covering all features
- ✅ **Latest Dependencies** - Updated to latest github.com/rediwo/redi v0.3.1

### 🔬 Test Coverage
- ✅ **SQLite Driver** - 18/18 tests passing (100%)
- ✅ **MySQL Driver** - 13/13 tests passing (100%)  
- ✅ **PostgreSQL Driver** - 13/13 tests passing (100%)
- ✅ **Engine Integration** - All JavaScript API tests passing
- ✅ **Schema Conversion** - Model/field name mapping tests
- ✅ **Migration System** - Database schema migration tests
- ✅ **URI Parsing** - All database URI configuration tests

### 🔮 Future Enhancements
- [ ] Advanced relation loading (eager/lazy)
- [ ] Query result caching
- [ ] Batch operations optimization
- [ ] Advanced validation rules
- [ ] Hook system (beforeCreate, afterUpdate, etc.)
- [ ] Real-time subscriptions
- [ ] Schema versioning

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🌟 Why RediORM?

RediORM bridges the gap between Go's type safety and JavaScript's flexibility, providing:

- **Familiar API** - JavaScript developers feel at home
- **Type Safety** - Go's compile-time guarantees with runtime validation
- **Prisma Compatible** - Use existing Prisma schemas without modification
- **High Performance** - Native Go speed with JavaScript convenience
- **Multi-Database** - Write once, run on SQLite, MySQL, or PostgreSQL
- **Testing Friendly** - In-memory databases for fast, isolated tests

Perfect for applications that need the performance of Go with the flexibility of JavaScript for data operations.

---

## 🆕 Recent Updates

### v0.3.1 - January 2025

#### 🎯 **Database Driver Improvements**
- ✅ **Fixed MySQL String Handling** - Resolved byte slice to string conversion issues
- ✅ **Enhanced PostgreSQL Support** - Full Docker integration with proper authentication
- ✅ **Schema Registration** - Added automatic schema registration for all database operations
- ✅ **Type Conversion** - Improved JavaScript ↔ Go type handling in query operations

#### 🔧 **Infrastructure Enhancements**  
- ✅ **Updated Dependencies** - Upgraded to github.com/rediwo/redi v0.3.1
- ✅ **Docker Integration** - Complete Docker Compose setup for MySQL and PostgreSQL testing
- ✅ **Comprehensive Testing** - 100% test coverage across all database drivers
- ✅ **CI/CD Improvements** - Enhanced Makefile with Docker support and test isolation

#### 🚀 **Migration System**
- ✅ **Automatic Schema Generation** - Database tables created from schema definitions
- ✅ **Migration History** - Track schema changes with redi_migrations table  
- ✅ **Driver-Based Architecture** - Database-specific migration logic in individual drivers
- ✅ **Schema Introspection** - Compare current database state with desired schema

#### 🔄 **API Enhancements**
- ✅ **Field Name Mapping** - Full support for @map() annotations in Prisma schemas
- ✅ **Model Name Conversion** - Automatic User → users table name transformation
- ✅ **URI-Based Configuration** - Simplified database connection setup
- ✅ **Error Handling** - Improved error messages and debugging information

#### 📊 **Test Results**
- ✅ **SQLite**: 18/18 tests passing (100%) 
- ✅ **MySQL**: 13/13 tests passing (100%)
- ✅ **PostgreSQL**: 13/13 tests passing (100%)
- ✅ **Engine**: All JavaScript API integration tests passing
- ✅ **Migration**: All schema migration and conversion tests passing