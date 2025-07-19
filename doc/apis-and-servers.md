# APIs and Servers

Comprehensive guide to RediORM's auto-generated GraphQL and REST APIs, plus Model Context Protocol (MCP) integration for AI assistants.

## Table of Contents

- [GraphQL API](#graphql-api)
- [REST API](#rest-api)
- [MCP (Model Context Protocol)](#mcp-model-context-protocol)
- [Combined Server](#combined-server)
- [Production Configuration](#production-configuration)
- [Security](#security)
- [Monitoring and Logging](#monitoring-and-logging)

## GraphQL API

### Starting GraphQL Server

```bash
# Basic GraphQL server
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma

# With custom port and playground
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma --port=8080 --playground=true

# Production setup with logging
redi-orm server \
  --db=postgresql://user:pass@localhost/db \
  --schema=./schema.prisma \
  --port=4000 \
  --playground=false \
  --log-level=info
```

### Auto-Generated Schema

Given this Prisma schema:

```prisma
model User {
    id    Int     @id @default(autoincrement())
    email String  @unique
    name  String
    posts Post[]
}

model Post {
    id       Int     @id @default(autoincrement())
    title    String
    content  String?
    authorId Int
    author   User    @relation(fields: [authorId], references: [id])
}
```

RediORM automatically generates this GraphQL schema:

```graphql
type User {
    id: Int!
    email: String!
    name: String!
    posts: [Post!]!
}

type Post {
    id: Int!
    title: String!
    content: String
    authorId: Int!
    author: User!
}

# Input types for filtering
input UserWhereInput {
    id: IntFilter
    email: StringFilter
    name: StringFilter
    posts: PostListRelationFilter
}

input StringFilter {
    equals: String
    not: String
    contains: String
    startsWith: String
    endsWith: String
    in: [String!]
    notIn: [String!]
}

# Query operations
type Query {
    findUniqueUser(where: UserWhereUniqueInput!): User
    findManyUser(where: UserWhereInput, orderBy: [UserOrderByInput!], take: Int, skip: Int): [User!]!
    findUniquePost(where: PostWhereUniqueInput!): Post
    findManyPost(where: PostWhereInput, orderBy: [PostOrderByInput!], take: Int, skip: Int): [Post!]!
}

# Mutation operations
type Mutation {
    createUser(data: UserCreateInput!): User!
    updateUser(where: UserWhereUniqueInput!, data: UserUpdateInput!): User!
    deleteUser(where: UserWhereUniqueInput!): User!
    createManyUser(data: [UserCreateManyInput!]!): BatchPayload!
    updateManyUser(where: UserWhereInput!, data: UserUpdateManyInput!): BatchPayload!
    deleteManyUser(where: UserWhereInput!): BatchPayload!
    # ... similar operations for Post
}
```

### GraphQL Queries

```graphql
# Find users with posts
query FindUsersWithPosts {
    findManyUser(
        where: { posts: { some: { published: true } } }
        include: { posts: true }
    ) {
        id
        name
        email
        posts {
            id
            title
            published
        }
    }
}

# Complex filtering
query FilterUsers {
    findManyUser(
        where: {
            AND: [
                { email: { contains: "@company.com" } }
                { 
                    OR: [
                        { name: { startsWith: "A" } }
                        { posts: { some: { views: { gt: 1000 } } } }
                    ]
                }
            ]
        }
        orderBy: [{ createdAt: desc }]
        take: 10
    ) {
        id
        name
        email
        _count {
            posts
        }
    }
}

# Nested includes
query DeepIncludes {
    findManyUser {
        id
        name
        posts {
            id
            title
            comments {
                id
                content
                author {
                    id
                    name
                }
            }
        }
    }
}
```

### GraphQL Mutations

```graphql
# Create user with posts
mutation CreateUserWithPosts {
    createUser(
        data: {
            name: "Alice"
            email: "alice@example.com"
            posts: {
                create: [
                    { title: "First Post", content: "Hello World!" }
                    { title: "Second Post", content: "Learning GraphQL" }
                ]
            }
        }
    ) {
        id
        name
        posts {
            id
            title
        }
    }
}

# Batch operations
mutation BatchCreateUsers {
    createManyUser(
        data: [
            { name: "User 1", email: "user1@example.com" }
            { name: "User 2", email: "user2@example.com" }
            { name: "User 3", email: "user3@example.com" }
        ]
    ) {
        count
    }
}

# Update with relations
mutation UpdateUserPosts {
    updateUser(
        where: { id: 1 }
        data: {
            name: "Alice Smith"
            posts: {
                create: [{ title: "New Post", content: "Updated content" }]
                connect: [{ id: 5 }]
                disconnect: [{ id: 3 }]
            }
        }
    ) {
        id
        name
        posts {
            id
            title
        }
    }
}
```

### GraphQL Playground

Access GraphQL Playground at `http://localhost:4000/graphql` (when playground is enabled):

```javascript
// Example queries you can run in the playground
{
  findManyUser(take: 5) {
    id
    name
    email
    posts {
      id
      title
    }
  }
}
```

## REST API

### Starting REST Server

```bash
# REST API is included with GraphQL server
redi-orm server --db=sqlite://./app.db --schema=./schema.prisma

# Access REST endpoints at http://localhost:4000/api
```

### REST Endpoints

RediORM automatically generates RESTful endpoints for each model:

#### Users Endpoints

```bash
# GET /api/users - List users
curl "http://localhost:4000/api/users"
curl "http://localhost:4000/api/users?take=10&skip=0&orderBy=name:asc"

# GET /api/users/:id - Get user by ID
curl "http://localhost:4000/api/users/1"
curl "http://localhost:4000/api/users/1?include=posts"

# POST /api/users - Create user
curl -X POST "http://localhost:4000/api/users" \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "email": "alice@example.com"}'

# PUT /api/users/:id - Update user
curl -X PUT "http://localhost:4000/api/users/1" \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice Smith"}'

# DELETE /api/users/:id - Delete user
curl -X DELETE "http://localhost:4000/api/users/1"
```

#### Advanced REST Operations

```bash
# Complex filtering
curl "http://localhost:4000/api/users?where[email][contains]=@company.com&where[age][gte]=18"

# Include relations
curl "http://localhost:4000/api/users/1?include=posts,profile"

# Nested includes
curl "http://localhost:4000/api/users?include[posts][include]=comments"

# Batch operations
curl -X POST "http://localhost:4000/api/users/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "data": [
      {"name": "User1", "email": "user1@example.com"},
      {"name": "User2", "email": "user2@example.com"}
    ]
  }'

# Batch update
curl -X PUT "http://localhost:4000/api/users/batch" \
  -H "Content-Type: application/json" \
  -d '{
    "where": {"email": {"endsWith": "@oldcompany.com"}},
    "data": {"company": "New Company"}
  }'

# Batch delete
curl -X DELETE "http://localhost:4000/api/users/batch" \
  -H "Content-Type: application/json" \
  -d '{"where": {"active": false}}'
```

### REST Response Format

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "Alice",
      "email": "alice@example.com",
      "posts": [
        {
          "id": 1,
          "title": "Hello World",
          "authorId": 1
        }
      ]
    }
  ],
  "pagination": {
    "page": 1,
    "pageSize": 20,
    "total": 1,
    "totalPages": 1
  },
  "meta": {
    "requestId": "req_123",
    "executionTime": "45ms"
  }
}
```

### Error Handling

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Email already exists",
    "details": {
      "field": "email",
      "value": "alice@example.com"
    }
  },
  "meta": {
    "requestId": "req_124",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## MCP (Model Context Protocol)

MCP enables AI assistants to understand and manipulate your database through intelligent, schema-aware operations.

### Starting MCP Server

```bash
# Basic MCP server
redi-orm mcp --db=sqlite://./app.db --schema=./schema.prisma

# Production MCP with security
redi-orm mcp \
  --db=postgresql://readonly:pass@localhost/db \
  --schema=./schema.prisma \
  --port=3000 \
  --read-only \
  --allowed-tables=users,posts \
  --enable-auth

# Local development (stdio transport)
redi-orm mcp --db=sqlite://./app.db --schema=./schema.prisma --transport=stdio
```

### MCP Resources

MCP exposes your database through several resource types:

#### Model Resources
```
model://User
model://Post
model://Comment
```

#### Schema Resources
```
schema://database
schema://User
schema://Post
```

#### Data Resources
```
data://users
data://posts
data://users/1
data://posts?author=Alice
```

#### Meta Resources
```
meta://tables
meta://indexes
meta://relationships
```

### MCP Tools

AI assistants can use these tools to interact with your database:

#### Data Operations
```json
{
  "name": "data.findMany",
  "arguments": {
    "model": "User",
    "where": {"email": {"contains": "@company.com"}},
    "include": {"posts": true},
    "take": 10
  }
}

{
  "name": "data.create",
  "arguments": {
    "model": "User", 
    "data": {
      "name": "Alice",
      "email": "alice@example.com",
      "posts": {
        "create": [{"title": "Hello World"}]
      }
    }
  }
}

{
  "name": "data.update",
  "arguments": {
    "model": "User",
    "where": {"id": 1},
    "data": {"name": "Alice Smith"}
  }
}
```

#### Model Management
```json
{
  "name": "model.create",
  "arguments": {
    "name": "Category",
    "fields": [
      {"name": "id", "type": "Int", "primaryKey": true, "autoIncrement": true},
      {"name": "name", "type": "String", "unique": true},
      {"name": "posts", "type": "Post[]", "relation": true}
    ]
  }
}

{
  "name": "model.addField",
  "arguments": {
    "model": "User",
    "field": {
      "name": "avatar",
      "type": "String",
      "nullable": true
    }
  }
}
```

#### Schema Operations
```json
{
  "name": "schema.sync",
  "arguments": {
    "preview": true
  }
}

{
  "name": "schema.migrate",
  "arguments": {
    "name": "add_categories"
  }
}
```

### MCP Prompts

Pre-configured AI prompts for common operations:

```json
{
  "name": "analyze_user_engagement",
  "description": "Analyze user engagement metrics",
  "arguments": [
    {"name": "timeframe", "description": "Time period to analyze", "required": false}
  ]
}

{
  "name": "optimize_query_performance", 
  "description": "Suggest optimizations for slow queries",
  "arguments": [
    {"name": "query", "description": "Query to optimize", "required": true}
  ]
}

{
  "name": "suggest_schema_improvements",
  "description": "Recommend schema design improvements"
}
```

### Claude Desktop Integration

Configure Claude Desktop to use your MCP server:

```json
{
  "mcpServers": {
    "rediorm": {
      "command": "redi-orm",
      "args": [
        "mcp",
        "--db=sqlite:///path/to/your/database.db",
        "--schema=./schema.prisma",
        "--transport=stdio"
      ]
    }
  }
}
```

### AI Assistant Capabilities

With MCP, AI assistants can:

```
🔍 Discover Models: "What models do I have in my database?"
🔨 Create Models: "I need a model for tracking orders with fields for amount, customer, and status"
🔎 Smart Queries: "Find all users who have published posts in the last week"
⚡ Optimize Performance: "This query is slow, how can I improve it?"
📊 Generate Reports: "Create a summary of user activity by department"
🛡️ Security Analysis: "Are there any potential security issues with my schema?"
```

## Combined Server

Run all APIs from a single server:

```bash
# All APIs in one server
redi-orm server --enable-mcp --mcp-port=3001 \
  --db=postgresql://user:pass@localhost/db \
  --schema=./schema.prisma \
  --port=4000 \
  --log-level=info

# Available endpoints:
# GraphQL: http://localhost:4000/graphql
# REST API: http://localhost:4000/api  
# MCP: http://localhost:3001 (or stdio)
```

### JavaScript Integration

```javascript
// Use with your existing application
const { fromUri } = require('redi/orm');

async function startServers() {
    const db = fromUri('postgresql://user:pass@localhost/db');
    await db.connect();
    await db.loadSchemaFrom('./schema.prisma');
    await db.syncSchemas();
    
    // Start GraphQL + REST server
    const graphqlServer = await startGraphQLServer(db, { port: 4000 });
    
    // Start MCP server for AI
    const mcpServer = await startMCPServer(db, { port: 3001 });
    
    console.log('GraphQL: http://localhost:4000/graphql');
    console.log('REST API: http://localhost:4000/api');
    console.log('MCP: http://localhost:3001');
}
```

## Production Configuration

### Environment Variables

```bash
# .env file
DATABASE_URL=postgresql://user:pass@localhost:5432/production_db
SCHEMA_PATH=./schema.prisma
PORT=4000
LOG_LEVEL=info
ENABLE_PLAYGROUND=false
ENABLE_INTROSPECTION=false
CORS_ORIGINS=https://myapp.com,https://admin.myapp.com
RATE_LIMIT_MAX=1000
RATE_LIMIT_WINDOW=900000
```

### Production Startup

```bash
# Production server with all security features
redi-orm server \
  --db="$DATABASE_URL" \
  --schema="$SCHEMA_PATH" \
  --port="$PORT" \
  --log-level="$LOG_LEVEL" \
  --playground=false \
  --introspection=false \
  --cors-origins="$CORS_ORIGINS" \
  --rate-limit-max="$RATE_LIMIT_MAX" \
  --rate-limit-window="$RATE_LIMIT_WINDOW" \
  --enable-compression \
  --enable-query-timeout \
  --query-timeout=30000
```

### Docker Deployment

```dockerfile
# Dockerfile
FROM node:18-alpine

WORKDIR /app
COPY package*.json ./
RUN npm install

COPY . .
RUN npm run build

# Install redi-orm CLI
RUN npm install -g @rediwo/redi-orm

EXPOSE 4000

CMD ["redi-orm", "server", "--db=$DATABASE_URL", "--schema=./schema.prisma"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "4000:4000"
    environment:
      - DATABASE_URL=postgresql://user:pass@db:5432/app
      - LOG_LEVEL=info
    depends_on:
      - db
      
  db:
    image: postgres:15
    environment:
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
      - POSTGRES_DB=app
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

## Security

### Authentication & Authorization

```bash
# Enable authentication
redi-orm server \
  --enable-auth \
  --jwt-secret="your-secret-key" \
  --auth-header="Authorization"

# Role-based access control
redi-orm server \
  --enable-rbac \
  --admin-roles="admin,superuser" \
  --read-only-roles="viewer,analyst"
```

### API Key Protection

```bash
# Require API keys
redi-orm server \
  --require-api-key \
  --api-keys="key1,key2,key3"
```

```bash
# Usage with API key
curl -H "X-API-Key: key1" "http://localhost:4000/api/users"
```

### Rate Limiting

```javascript
// Configure rate limiting
{
  "rateLimiting": {
    "graphql": {
      "max": 100,
      "windowMs": 900000,
      "skipSuccessfulRequests": false
    },
    "rest": {
      "max": 1000, 
      "windowMs": 900000
    },
    "mcp": {
      "max": 50,
      "windowMs": 60000
    }
  }
}
```

### Query Complexity Analysis

```javascript
// Prevent complex queries that could cause performance issues
{
  "queryComplexity": {
    "maximumComplexity": 1000,
    "maximumDepth": 10,
    "scalarCost": 1,
    "objectCost": 2,
    "listFactor": 10
  }
}
```

### CORS Configuration

```bash
# Configure CORS for web applications
redi-orm server \
  --cors-origins="https://myapp.com,https://admin.myapp.com" \
  --cors-methods="GET,POST,PUT,DELETE" \
  --cors-headers="Content-Type,Authorization,X-API-Key"
```

### MCP Security

```bash
# Secure MCP for production
redi-orm mcp \
  --read-only \
  --allowed-tables="users,posts,comments" \
  --denied-operations="delete,drop,truncate" \
  --enable-auth \
  --auth-token="secure-token" \
  --max-query-results=1000
```

## Monitoring and Logging

### Structured Logging

```javascript
// Configure comprehensive logging
const logger = createLogger('APIServer');
logger.setLevel(logger.levels.INFO);

// Logs include:
// [APIServer] INFO: GraphQL server started on port 4000
// [APIServer] DEBUG: Query executed (123ms): findManyUser
// [APIServer] WARN: Rate limit exceeded for IP 192.168.1.1
// [APIServer] ERROR: Database connection failed
```

### Performance Monitoring

```bash
# Enable detailed performance tracking
redi-orm server \
  --enable-metrics \
  --metrics-endpoint="/metrics" \
  --slow-query-threshold=1000 \
  --log-slow-queries
```

### Health Checks

```bash
# Health check endpoint
curl "http://localhost:4000/health"

# Response:
{
  "status": "healthy",
  "database": "connected",
  "uptime": "2h 30m 15s",
  "version": "1.0.0",
  "timestamp": "2024-01-15T14:30:00Z"
}
```

### Request Tracking

```javascript
// Every request gets a unique ID for tracing
{
  "requestId": "req_abc123",
  "method": "POST",
  "endpoint": "/api/users",
  "executionTime": "45ms",
  "databaseQueries": 2,
  "cacheHits": 1
}
```

### Error Monitoring

```javascript
// Structured error reporting
{
  "error": {
    "type": "ValidationError",
    "message": "Email already exists",
    "code": "E_UNIQUE_CONSTRAINT",
    "requestId": "req_def456",
    "timestamp": "2024-01-15T14:30:00Z",
    "stack": "..." // Only in development
  }
}
```

---

For more information, see:
- [Advanced Features](./advanced-features.md) for complex query capabilities
- [Database Guide](./database-guide.md) for database-specific configurations
- [MCP Guide](./mcp-guide.md) for detailed AI integration
- [Getting Started](./getting-started.md) for initial setup