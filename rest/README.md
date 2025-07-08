# RediORM REST API

The REST API module provides a JSON-based HTTP API for RediORM, offering a traditional alternative to the GraphQL API.

## Features

- ✅ **Standard REST Endpoints** - CRUD operations for all models
- ✅ **Complex Filtering** - Support for operators like `gt`, `lt`, `contains`, `in`, etc.
- ✅ **Pagination** - Page-based pagination with metadata
- ✅ **Sorting** - Multi-field sorting with ASC/DESC support
- ✅ **Field Selection** - Choose which fields to return
- ✅ **Relation Loading** - Include related data to avoid N+1 queries
- ✅ **Batch Operations** - Create multiple records in a single request
- ✅ **Multiple Connections** - Support for multiple database connections
- ✅ **CORS Support** - Built-in CORS middleware for browser apps
- ✅ **Execution Time Tracking** - Performance monitoring in responses

## Architecture

```
/rest/
├── handlers/          # HTTP request handlers
│   ├── connection.go  # Database connection management
│   └── data.go        # CRUD operations handler
├── middleware/        # HTTP middleware
│   ├── cors.go        # CORS headers
│   ├── json.go        # JSON content type
│   └── logging.go     # Request logging
├── services/          # Business logic
│   └── query_builder.go # Convert REST params to ORM queries
├── types/             # Request/Response types
│   ├── request.go     # Request structures
│   └── response.go    # Response structures
├── router.go          # HTTP routing
└── server.go          # REST server implementation
```

## Usage

### Starting the Server

The REST API is now integrated with the GraphQL server. Both APIs start together:

```bash
# CLI command - starts both GraphQL and REST APIs
redi-orm server --db=sqlite://./myapp.db --schema=./schema.prisma --port=8080

# Endpoints:
# GraphQL: http://localhost:8080/graphql
# REST API: http://localhost:8080/api

# Or use REST API programmatically
import "github.com/rediwo/redi-orm/rest"

config := rest.ServerConfig{
    Database:   db,
    Port:       8080,
    LogLevel:   "info",
    SchemaFile: "./schema.prisma",
}

server, err := rest.NewServer(config)
if err != nil {
    log.Fatal(err)
}
```

### API Endpoints

For each model (e.g., `User`), the following endpoints are available:

- `GET /api/{Model}` - List all records
- `GET /api/{Model}/{id}` - Get a specific record
- `POST /api/{Model}` - Create a new record
- `PUT /api/{Model}/{id}` - Update a record
- `DELETE /api/{Model}/{id}` - Delete a record
- `POST /api/{Model}/batch` - Create multiple records

### Query Parameters

#### Filtering
```
# Simple equality
GET /api/User?filter[age]=25

# Operators
GET /api/User?filter[age][gt]=25
GET /api/User?filter[name][contains]=John

# JSON where clause
GET /api/User?where={"age":{"gt":25},"name":{"startsWith":"J"}}
```

#### Pagination
```
GET /api/User?page=2&limit=20
```

#### Sorting
```
# Ascending by name
GET /api/User?sort=name

# Descending by age, then ascending by name
GET /api/User?sort=-age,name
```

#### Field Selection
```
GET /api/User?select=id,name,email
```

#### Including Relations
```
# Include posts
GET /api/User?include=posts

# Multiple relations
GET /api/User?include=posts,profile

# Complex includes (JSON)
GET /api/User?include={"posts":{"include":{"comments":true}}}
```

### Request/Response Format

#### Create Request
```json
POST /api/User
{
  "data": {
    "name": "John Doe",
    "email": "john@example.com",
    "age": 30
  }
}
```

#### Success Response
```json
{
  "success": true,
  "data": {
    "id": 1,
    "name": "John Doe",
    "email": "john@example.com",
    "age": 30
  },
  "meta": {
    "execution_time": "15.234ms",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

#### Error Response
```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "User not found",
    "details": "No user with id 123"
  },
  "meta": {
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

#### Paginated Response
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 2,
    "limit": 20,
    "total": 156,
    "pages": 8
  },
  "meta": {
    "execution_time": "23.456ms",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

## JavaScript/AJAX Example

```javascript
// Using fetch API
async function getUsers() {
  const response = await fetch('http://localhost:8080/api/User?include=posts&limit=10');
  const data = await response.json();
  
  if (data.success) {
    console.log('Users:', data.data);
    console.log('Total:', data.pagination?.total);
  } else {
    console.error('Error:', data.error.message);
  }
}

// Create user
async function createUser(userData) {
  const response = await fetch('http://localhost:8080/api/User', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ data: userData })
  });
  
  const result = await response.json();
  return result.data;
}
```

## Multiple Database Connections

The REST API supports multiple database connections:

```javascript
// Connect to a database
await fetch('/api/connections/connect', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    uri: 'postgresql://user:pass@localhost/db',
    name: 'postgres-db',
    schema: schemaContent // Optional
  })
});

// Use a specific connection
await fetch('/api/User', {
  headers: {
    'X-Connection-Name': 'postgres-db'
  }
});

// List connections
const connections = await fetch('/api/connections').then(r => r.json());

// Disconnect
await fetch('/api/connections/disconnect?name=postgres-db', {
  method: 'DELETE'
});
```

## Testing

The REST API includes comprehensive tests:

```bash
# Run all REST tests
go test ./rest/tests -v

# Run specific test
go test ./rest/tests -run TestBasicRESTOperations -v

# Test with real databases
MYSQL_TEST_URI=mysql://user:pass@localhost/test go test ./rest/tests -v
```

## Performance Considerations

1. **Use field selection** to reduce payload size
2. **Enable pagination** for large datasets
3. **Use includes carefully** to avoid N+1 queries
4. **Monitor execution times** in response metadata
5. **Consider caching** for frequently accessed data

## Security

1. **CORS is enabled by default** - Configure allowed origins in production
2. **Use HTTPS in production** - The API doesn't enforce HTTPS
3. **Implement authentication** - Add auth middleware as needed
4. **Validate input** - The API validates basic types but not business rules
5. **Rate limiting** - Add rate limiting middleware in production