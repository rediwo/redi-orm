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

### Using Native Go Schema

```go
package main

import (
    "log"
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/engine"
    "github.com/rediwo/redi-orm/schema"
    "github.com/rediwo/redi-orm/types"
)

func main() {
    // Create database connection
    db, err := database.New(types.Config{
        Type:     types.SQLite,
        FilePath: "example.db",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    if err := db.Connect(); err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Create engine
    eng := engine.New(db)
    
    // Define schema using fluent API
    userSchema := schema.New("User").
        AddField(schema.NewField("id").Int64().PrimaryKey().AutoIncrement().Build()).
        AddField(schema.NewField("name").String().Build()).
        AddField(schema.NewField("email").String().Unique().Build()).
        AddField(schema.NewField("age").Int().Nullable().Build()).
        AddField(schema.NewField("active").Bool().Default(true).Build())
    
    // Register schema
    if err := eng.RegisterSchema(userSchema); err != nil {
        log.Fatal(err)
    }
    
    // JavaScript API for database operations
    // Create user
    userID, _ := eng.Execute(`models.User.add({
        name: "Alice", 
        email: "alice@example.com",
        age: 30
    })`)
    log.Printf("Created user ID: %v", userID)
    
    // Equivalent Go API
    userData := map[string]interface{}{
        "name":  "Alice",
        "email": "alice@example.com", 
        "age":   30,
    }
    userIDGo, _ := db.Insert("users", userData)
    log.Printf("Created user ID (Go): %v", userIDGo)
    
    // Get user - JavaScript API
    user, _ := eng.Execute(`models.User.get(1)`)
    log.Printf("User (JS): %+v", user)
    
    // Equivalent Go API
    userGo, _ := db.FindByID("users", 1)
    log.Printf("User (Go): %+v", userGo)
    
    // Query with conditions - JavaScript API
    users, _ := eng.Execute(`
        models.User.select()
            .where("age", ">", 25)
            .orderBy("name", "ASC")
            .execute()
    `)
    log.Printf("Users (JS): %+v", users)
    
    // Equivalent Go API
    qb := db.Select("users", nil).
        Where("age", ">", 25).
        OrderBy("name", "ASC")
    usersGo, _ := qb.Execute()
    log.Printf("Users (Go): %+v", usersGo)
}
```

### Using Prisma Schema

```go
package main

import (
    "log"
    "github.com/rediwo/redi-orm/database"
    "github.com/rediwo/redi-orm/engine"
    "github.com/rediwo/redi-orm/types"
)

func main() {
    // Create database connection
    db, err := database.New(types.Config{
        Type:     types.SQLite,
        FilePath: ":memory:",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    if err := db.Connect(); err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Create engine
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
    
    // JavaScript API usage
    userID, _ := eng.Execute(`models.User.add({
        name: "John Doe", 
        email: "john@example.com",
        role: "ADMIN"
    })`)
    log.Printf("Created user ID: %v", userID)
    
    // Equivalent Go API
    userData := map[string]interface{}{
        "name":  "John Doe",
        "email": "john@example.com",
        "role":  "ADMIN",
        "createdAt": "CURRENT_TIMESTAMP",
    }
    userIDGo, _ := db.Insert("users", userData)
    log.Printf("Created user ID (Go): %v", userIDGo)
    
    // Create post - JavaScript API
    postID, _ := eng.Execute(`models.Post.add({
        title: "My First Post",
        content: "Hello, world!",
        authorId: 1,
        published: true
    })`)
    log.Printf("Created post ID: %v", postID)
    
    // Equivalent Go API
    postData := map[string]interface{}{
        "title":     "My First Post",
        "content":   "Hello, world!",
        "authorId":  1,
        "published": true,
        "createdAt": "CURRENT_TIMESTAMP",
    }
    postIDGo, _ := db.Insert("posts", postData)
    log.Printf("Created post ID (Go): %v", postIDGo)
    
    // Query with relationships - JavaScript API
    posts, _ := eng.Execute(`
        models.Post.select()
            .where("published", "=", true)
            .orderBy("createdAt", "DESC")
            .execute()
    `)
    log.Printf("Posts (JS): %+v", posts)
    
    // Equivalent Go API
    postsGo, _ := db.Select("posts", nil).
        Where("published", "=", true).
        OrderBy("createdAt", "DESC").
        Execute()
    log.Printf("Posts (Go): %+v", postsGo)
}
```

## 🎮 API Reference - JavaScript vs Go

### Model Operations

| Operation | JavaScript API | Equivalent Go API |
|-----------|---------------|-------------------|
| **Create** | `models.User.add({name: "Alice", email: "alice@example.com"})` | `db.Insert("users", map[string]interface{}{"name": "Alice", "email": "alice@example.com"})` |
| **Read by ID** | `models.User.get(1)` | `db.FindByID("users", 1)` |
| **Update** | `models.User.set(1, {age: 31})` | `db.Update("users", 1, map[string]interface{}{"age": 31})` |
| **Delete** | `models.User.remove(1)` | `db.Delete("users", 1)` |
| **Select All** | `models.User.select().execute()` | `db.Select("users", nil).Execute()` |
| **Count** | `models.User.select().count()` | `db.Select("users", nil).Count()` |
| **First** | `models.User.select().first()` | `db.Select("users", nil).First()` |

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
userID, err := db.Insert("users", userData)

// Read records
user, err := db.FindByID("users", 1)
users, err := db.Select("users", nil).Execute()

// Update records
updateData := map[string]interface{}{"age": 31}
err := db.Update("users", 1, updateData)

// Delete records
err := db.Delete("users", 1)
```

### Query Builder Comparison

| Query Type | JavaScript API | Equivalent Go API |
|------------|---------------|-------------------|
| **Select All** | `models.User.select().execute()` | `db.Select("users", nil).Execute()` |
| **Select Columns** | `models.User.select(["name", "email"]).execute()` | `db.Select("users", []string{"name", "email"}).Execute()` |
| **WHERE Clause** | `models.User.select().where("age", ">", 18).execute()` | `db.Select("users", nil).Where("age", ">", 18).Execute()` |
| **ORDER BY** | `models.User.select().orderBy("name", "ASC").execute()` | `db.Select("users", nil).OrderBy("name", "ASC").Execute()` |
| **LIMIT/OFFSET** | `models.User.select().limit(10).offset(20).execute()` | `db.Select("users", nil).Limit(10).Offset(20).Execute()` |

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
users, err := db.Select("users", nil).Execute()                    // SELECT * FROM users
users, err := db.Select("users", []string{"name", "email"}).Execute() // SELECT name, email FROM users

// WHERE clauses
users, err := db.Select("users", nil).
    Where("age", ">", 18).
    Where("active", "=", true).
    Execute()

// Ordering and pagination
users, err := db.Select("users", nil).
    OrderBy("name", "ASC").
    Limit(10).
    Offset(20).
    Execute()

// Aggregation
count, err := db.Select("users", nil).Count()                      // Count all users
count, err := db.Select("users", nil).Where("age", ">", 18).Count() // Count adult users

// Get first result
user, err := db.Select("users", nil).
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
    Type:     types.SQLite,
    FilePath: "database.db",
}

// MySQL
config := types.Config{
    Type:     types.MySQL,
    Host:     "localhost",
    Port:     3306,
    Database: "myapp",
    User:     "username",
    Password: "password",
}

// PostgreSQL
config := types.Config{
    Type:     types.PostgreSQL,
    Host:     "localhost",
    Port:     5432,
    Database: "myapp",
    User:     "username",
    Password: "password",
}

db, err := database.New(config)
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
    db, err := database.New(types.Config{
        Type:     types.SQLite,
        FilePath: ":memory:",
    })
    require.NoError(t, err)
    
    err = db.Connect()
    require.NoError(t, err)
    defer db.Close()
    
    // Set up engine and schema
    eng := engine.New(db)
    userSchema := createUserSchema()
    err = eng.RegisterSchema(userSchema)
    require.NoError(t, err)
    
    // Test operations
    userID, err := eng.Execute(`models.User.add({name: "Test User", email: "test@example.com"})`)
    require.NoError(t, err)
    assert.Equal(t, int64(1), userID)
    
    user, err := eng.Execute(`models.User.get(1)`)
    require.NoError(t, err)
    
    userData := user.(map[string]interface{})
    assert.Equal(t, "Test User", userData["name"])
}
```

## 🏗️ Development Commands

```bash
# Build the project
make build

# Run all tests
make test

# Run tests with coverage
make test-cover

# Run benchmarks
make test-benchmark

# Format code
make fmt

# Run linter
make lint

# Development workflow
make dev               # fmt + vet + test

# Full CI workflow
make ci                # race detection + coverage
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
- ✅ SQLite, MySQL, PostgreSQL drivers
- ✅ Comprehensive Prisma schema parsing
- ✅ JavaScript API with full query builder
- ✅ Transaction support
- ✅ Schema validation and type checking
- ✅ Composite primary keys
- ✅ Scalar arrays (PostgreSQL)
- ✅ Enum value mapping
- ✅ Database-specific attributes
- ✅ Connection pooling
- ✅ Comprehensive test coverage
- ✅ Benchmark suite

### 🔮 Future Enhancements
- [ ] Schema migrations
- [ ] Relation loading (eager/lazy)
- [ ] Advanced validation rules
- [ ] Hook system (beforeCreate, afterUpdate, etc.)
- [ ] Query caching
- [ ] Batch operations
- [ ] Real-time subscriptions

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