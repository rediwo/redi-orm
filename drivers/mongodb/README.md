# MongoDB Driver for RediORM

This driver provides MongoDB support for RediORM, enabling document database operations through the same familiar ORM interface.

## Features

### Core Capabilities
- ✅ **Full CRUD Operations** - Create, Read, Update, Delete
- ✅ **Schema Management** - Optional schema validation with JSON Schema
- ✅ **Transactions** - Multi-document ACID transactions (MongoDB 4.0+, requires replica set)
- ✅ **Connection Management** - Support for standard and SRV connection strings
- ✅ **Index Management** - Create and manage indexes through schema definitions
- ✅ **Field Mapping** - Automatic field name to column name conversion
- ✅ **Aggregation Queries** - Full support for GROUP BY, HAVING with aggregation pipeline

### MongoDB-Specific Features
- ✅ **SQL to MongoDB Translation** - Write SQL queries that are automatically translated to MongoDB operations
- ✅ **Nested Documents** - Full support for embedded documents
- ✅ **Array Fields** - Native array field support
- ✅ **ObjectId Support** - Native ObjectId type for `_id` fields
- ✅ **Aggregation Pipeline** - Through raw queries and SQL translation
- ✅ **Document Validation** - JSON Schema validation
- ✅ **Auto-increment IDs** - Emulated using sequence collection
- ✅ **String Operators** - Full regex support for startsWith, endsWith, contains
- ✅ **Schema Evolution** - Automatic handling of missing nullable fields
- ✅ **Collection Validation** - Existence checks for better error messages
- 🔧 **Geospatial Queries** - Planned
- 🔧 **Text Search** - Planned

## Installation

```go
import (
    "github.com/rediwo/redi-orm/database"
    _ "github.com/rediwo/redi-orm/drivers/mongodb"
)
```

## Connection

### Standard Connection
```go
db, err := database.NewFromURI("mongodb://localhost:27017/myapp")
```

### With Authentication
```go
db, err := database.NewFromURI("mongodb://user:password@localhost:27017/myapp?authSource=admin")
```

### MongoDB Atlas (SRV)
```go
db, err := database.NewFromURI("mongodb+srv://user:password@cluster.mongodb.net/myapp")
```

### Connection Options
- `authSource` - Authentication database
- `replicaSet` - Replica set name
- `readPreference` - Read preference mode
- `w` - Write concern
- `retryWrites` - Enable retryable writes

## Schema Definition

While MongoDB is schemaless, RediORM allows optional schema definitions for validation and type safety:

```go
userSchema := schema.New("User").
    AddField(schema.Field{
        Name:       "_id",
        Type:       schema.FieldTypeObjectId,
        PrimaryKey: true,
    }).
    AddField(schema.Field{
        Name: "email",
        Type: schema.FieldTypeString,
        Unique: true,
    }).
    AddField(schema.Field{
        Name: "profile",
        Type: schema.FieldTypeDocument, // Nested document
    }).
    AddField(schema.Field{
        Name: "tags",
        Type: schema.FieldTypeStringArray, // Array field
    })
```

### Supported Field Types
- `FieldTypeObjectId` - MongoDB ObjectId
- `FieldTypeString` - String
- `FieldTypeInt`, `FieldTypeInt64` - Integer types
- `FieldTypeFloat` - Floating point
- `FieldTypeBool` - Boolean
- `FieldTypeDateTime` - Date/Time
- `FieldTypeDocument` - Nested document (JSON)
- `FieldTypeArray` - Generic array
- `FieldTypeStringArray`, `FieldTypeIntArray`, etc. - Typed arrays
- `FieldTypeBinary` - Binary data
- `FieldTypeDecimal128` - High precision decimal

## Query Operations

### Basic Queries

The MongoDB driver translates SQL-like queries to MongoDB operations:

```go
// Find documents
users := db.Model("User").
    Select().
    Where("age").GreaterThan(18).
    Where("status").Equals("active").
    OrderBy("createdAt", types.DESC).
    Limit(10)

// The above translates to MongoDB filter:
// { age: { $gt: 18 }, status: "active" }
// With sort: { createdAt: -1 }
// And limit: 10
```

### Aggregation Queries

```go
// Group by with having clause
result := db.Model("Order").
    GroupBy("category").
    Having("SUM(amount) > ?", 1000).
    Select("category", "SUM(amount) as total", "COUNT(*) as count")

// Translates to MongoDB aggregation pipeline:
// [
//   { $group: { 
//     _id: "$category",
//     total: { $sum: "$amount" },
//     count: { $sum: 1 }
//   }},
//   { $match: { total: { $gt: 1000 } } }
// ]
```

### Nested Field Queries

```go
// Query nested fields using dot notation
users := db.Model("User").
    Select().
    Where("profile.location").Equals("San Francisco").
    Where("profile.age").GreaterThan(25)
```

### Array Operations

```go
// Query array fields
users := db.Model("User").
    Select().
    Where("tags").In("developer", "mongodb")
```

## Raw Queries

The MongoDB driver supports both SQL and native MongoDB commands:

### SQL Queries
```go
// SQL queries are automatically translated to MongoDB
var users []map[string]any
err := db.Raw("SELECT * FROM users WHERE age > ? ORDER BY name", 18).Find(ctx, &users)

// Complex SQL with JOINs (translated to $lookup)
err := db.Raw(`
    SELECT u.*, COUNT(p.id) as post_count 
    FROM users u 
    LEFT JOIN posts p ON u.id = p.user_id 
    GROUP BY u.id
`).Find(ctx, &results)
```

### Native MongoDB Commands
```go
// Execute MongoDB commands directly
result, err := db.Raw(`{
    "aggregate": "users",
    "pipeline": [
        { "$match": { "age": { "$gte": 18 } } },
        { "$group": {
            "_id": "$location",
            "count": { "$sum": 1 }
        }}
    ]
}`).Exec(ctx)

// Find with native MongoDB syntax
var user map[string]any
err := db.Raw(`{
    "find": "users",
    "filter": {"email": "john@example.com"},
    "limit": 1
}`).FindOne(ctx, &user)
```

## Transactions

MongoDB supports multi-document transactions (requires MongoDB 4.0+ and replica set):

```go
err := db.Transaction(ctx, func(tx types.Transaction) error {
    // All operations in transaction
    _, err := tx.Model("User").Insert(userData).Exec(ctx)
    if err != nil {
        return err // Automatic rollback
    }
    
    _, err = tx.Model("Order").Insert(orderData).Exec(ctx)
    return err
})
```

## Indexes

Define indexes through schema:

```go
schema.AddIndex(schema.Index{
    Name:   "email_idx",
    Fields: []string{"email"},
    Unique: true,
})

// Compound index
schema.AddIndex(schema.Index{
    Name:   "location_age_idx",
    Fields: []string{"location", "age"},
})
```

## Limitations and Differences

### Feature Limitations
1. **No Savepoints** - MongoDB doesn't support savepoints in transactions
2. **Limited JOIN Support** - JOINs are translated to $lookup (only LEFT JOIN supported)
3. **Schema Migrations** - MongoDB is schemaless, no ALTER TABLE equivalent
4. **Foreign Keys** - No native foreign key constraints

### Behavioral Differences
1. **String Matching** - MongoDB uses case-sensitive regex by default
2. **NULL Handling** - MongoDB treats missing fields and null differently
3. **Transactions** - Require MongoDB 4.0+ with replica set
4. **Auto-increment** - Emulated using a sequence collection

## Query Translation

The driver translates SQL-like conditions to MongoDB filters:

| SQL Operation | MongoDB Equivalent |
|--------------|-------------------|
| `= value` | `{ field: value }` |
| `!= value` | `{ field: { $ne: value } }` |
| `> value` | `{ field: { $gt: value } }` |
| `>= value` | `{ field: { $gte: value } }` |
| `< value` | `{ field: { $lt: value } }` |
| `<= value` | `{ field: { $lte: value } }` |
| `IN (...)` | `{ field: { $in: [...] } }` |
| `NOT IN (...)` | `{ field: { $nin: [...] } }` |
| `LIKE '%text%'` | `{ field: { $regex: ".*text.*" } }` |
| `IS NULL` | `{ field: null }` |
| `IS NOT NULL` | `{ field: { $ne: null } }` |

## Best Practices

1. **Use Indexes** - Create indexes for frequently queried fields
2. **Limit Projections** - Select only needed fields to reduce network traffic
3. **Batch Operations** - Use `InsertMany` for bulk inserts
4. **Connection Pooling** - The driver handles connection pooling automatically
5. **Schema Validation** - Use schemas for data integrity even though MongoDB is flexible
6. **Query Choice** - Use SQL for simple queries, native MongoDB for complex aggregations
7. **Transactions** - Ensure replica set is configured for transaction support

## Example

```go
package main

import (
    "context"
    "log"
    
    "github.com/rediwo/redi-orm/database"
    _ "github.com/rediwo/redi-orm/drivers/mongodb"
    "github.com/rediwo/redi-orm/schema"
)

func main() {
    // Connect
    db, err := database.NewFromURI("mongodb://localhost:27017/myapp")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Define schema
    userSchema := schema.New("User").
        AddField(schema.Field{
            Name:       "_id",
            Type:       schema.FieldTypeObjectId,
            PrimaryKey: true,
        }).
        AddField(schema.Field{
            Name: "name",
            Type: schema.FieldTypeString,
        }).
        AddField(schema.Field{
            Name: "email",
            Type: schema.FieldTypeString,
            Unique: true,
        })
    
    // Register and sync
    db.AddSchema(userSchema)
    if err := db.SyncSchemas(ctx); err != nil {
        log.Fatal(err)
    }
    
    // Use the model
    User := db.Model("User")
    
    // Insert
    result, err := User.Insert(map[string]any{
        "name": "John Doe",
        "email": "john@example.com",
    }).Exec(ctx)
    
    // Query
    var users []map[string]any
    err = User.Select().
        Where("name").Contains("John").
        FindMany(ctx, &users)
}
```

## Contributing

When adding features to the MongoDB driver:

1. Maintain compatibility with the RediORM interface
2. Add appropriate type conversions for BSON types
3. Include tests for MongoDB-specific features
4. Document any limitations or differences from SQL databases
5. Update this README with new features