# Agile Package

The `agile` package provides a Prisma-like experience in Go, using JSON strings for query definitions while supporting both type-safe and non-type-safe usage patterns.

## Overview

The agile package is an independent, high-level API that sits on top of the RediORM core. It provides:

- **JSON-based queries** - Use familiar JSON syntax instead of method chaining
- **Automatic type conversion** - Handles database-specific type conversions (e.g., MySQL string numbers)
- **Dual API support** - Both typed (with generics) and untyped (map[string]any) interfaces
- **Complete separation** - Independent from the ORM module for clean architecture

## Installation

```go
import "github.com/rediwo/redi-orm/agile"
```

## Basic Usage

### Creating a Client

```go
// Create client from database
client := agile.NewClient(db)

// Or with custom type converter
client := agile.NewClient(db, agile.WithTypeConverter(customConverter))
```

### CRUD Operations

#### Create
```go
user, err := client.Model("User").Create(`{
  "data": {
    "name": "John Doe",
    "email": "john@example.com"
  }
}`)
```

#### Find Many
```go
users, err := client.Model("User").FindMany(`{
  "where": { 
    "age": { "gte": 18 }
  },
  "orderBy": { "name": "asc" },
  "take": 10,
  "skip": 20
}`)
```

#### Update
```go
updated, err := client.Model("User").Update(`{
  "where": { "id": 1 },
  "data": { "name": "Jane Doe" }
}`)
```

#### Delete
```go
deleted, err := client.Model("User").Delete(`{
  "where": { "id": 1 }
}`)
```

### Complex Queries

#### With Conditions
```go
users, err := client.Model("User").FindMany(`{
  "where": { 
    "OR": [
      { "age": { "gte": 18 } },
      { "role": "admin" }
    ],
    "active": true
  }
}`)
```

#### With Relations
```go
users, err := client.Model("User").FindMany(`{
  "include": { 
    "posts": {
      "where": { "published": true },
      "orderBy": { "createdAt": "desc" },
      "take": 5
    }
  }
}`)
```

### Aggregations

```go
result, err := client.Model("Order").Aggregate(`{
  "_sum": { "amount": true },
  "_avg": { "amount": true },
  "_count": true,
  "where": { "status": "completed" }
}`)

// result["_sum"]["amount"] will be float64, not string (even for MySQL)
```

### Typed API

For type safety, use the typed variants:

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

var user User
err := client.Model("User").FindUniqueTyped(`{
  "where": { "email": "john@example.com" }
}`, &user)

var users []User
err = client.Model("User").FindManyTyped(`{
  "where": { "active": true }
}`, &users)
```

### Transactions

```go
err := client.Transaction(func(tx *agile.Client) error {
    // All operations use tx instead of client
    user, err := tx.Model("User").Create(`{
      "data": { "name": "Alice", "balance": 1000 }
    }`)
    if err != nil {
        return err // Will rollback
    }
    
    _, err = tx.Model("User").Update(`{
      "where": { "id": ` + fmt.Sprintf("%v", user["id"]) + ` },
      "data": { "balance": 900 }
    }`)
    if err != nil {
        return err // Will rollback
    }
    
    return nil // Will commit
})
```

### Raw Queries

```go
// Execute raw SQL
raw := client.Model("").Raw("SELECT * FROM users WHERE age > ?", 18)
results, err := raw.Find()

// For single result
result, err := raw.FindOne()
```

## Query Syntax

The agile package uses JSON for all queries. Here are the supported operations:

### Operations
- `create` - Create a single record
- `createMany` - Create multiple records
- `findUnique` - Find a single record by unique field
- `findFirst` - Find the first matching record
- `findMany` - Find multiple records
- `update` - Update a single record
- `updateMany` - Update multiple records
- `delete` - Delete a single record
- `deleteMany` - Delete multiple records
- `count` - Count matching records
- `aggregate` - Perform aggregations
- `upsert` - Update or create

### Query Options
- `where` - Filter conditions
- `data` - Data for create/update
- `orderBy` - Sort results
- `take` - Limit results
- `skip` - Skip results
- `select` - Select specific fields
- `include` - Include relations
- `distinct` - Distinct results

### Where Operators
- `equals` - Exact match (default)
- `not` - Not equal
- `in` - In array
- `notIn` - Not in array
- `lt` - Less than
- `lte` - Less than or equal
- `gt` - Greater than
- `gte` - Greater than or equal
- `contains` - Contains substring
- `startsWith` - Starts with
- `endsWith` - Ends with

### Logical Operators
- `AND` - All conditions must match
- `OR` - Any condition must match
- `NOT` - Negate condition

## Type Conversion

The agile package automatically handles database-specific type conversions:

- **MySQL**: Converts string numbers to proper numeric types
- **PostgreSQL**: Native numeric types preserved
- **SQLite**: Integer/float types preserved

This is especially important for aggregations where MySQL returns strings:

```go
// MySQL returns "123.45" as string, agile converts to float64
result, err := client.Model("Order").Aggregate(`{
  "_sum": { "amount": true }
}`)
sum := result["_sum"].(map[string]any)["amount"].(float64) // Always float64
```

## Testing

The agile package includes comprehensive conformance tests:

```go
suite := &agile.AgileConformanceTests{
    DriverName:  "SQLite",
    DatabaseURI: uri,
    NewDatabase: func(uri string) (types.Database, error) {
        // Create database instance
    },
    // ... other configuration
}

suite.RunAll(t)
```

## Differences from ORM Module

| Feature | ORM Module | Agile Package |
|---------|------------|---------------|
| API Style | Method chaining | JSON strings |
| Type Safety | Go types only | Dual (typed/untyped) |
| Complexity | Lower-level | Higher-level |
| Use Case | Direct control | Rapid development |

## Best Practices

1. **Use typed API when possible** - Provides compile-time safety
2. **Batch operations** - Use `createMany`, `updateMany` for bulk operations
3. **Limit includes** - Deep nesting can impact performance
4. **Use transactions** - For operations that must succeed together
5. **Handle errors** - Always check for errors, especially in typed API

## Examples

See the `agile_conformance_tests_*.go` files for comprehensive examples of all features.