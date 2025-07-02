# ORM Module Tests

This directory contains comprehensive test suites for the RediORM module.

## Test Files

### `orm_test.go`
Go test harness that runs all JavaScript test files. Includes:
- Test suite runner that discovers all `*_test.js` files
- FromUri specific tests
- Temporary directory management

### `db_models_test.js`
Tests the `db.models` functionality:
- Verifies models are empty before sync
- Confirms models are populated after `syncSchemas()`
- Tests that all CRUD methods are available on models
- Tests multiple schema loads

### `basic_test.js`
Tests fundamental CRUD structure:
- Verifies models are accessible via `db.models`
- Confirms all CRUD methods exist on models
- Validates method signatures

### `simple_test.js`
Tests the basic fromUri functionality:
- Creating database from URI
- Connection and ping operations
- Schema loading from string and file
- Model availability after sync
- Error handling for invalid URIs and schemas

### `schema_test.js`
Tests schema loading and synchronization:
- Loading schema from string (`loadSchema`)
- Loading schema from file (`loadSchemaFrom`)
- Multiple schema loads
- Different database URI support
- Custom table/column mapping

### `transaction_test.js`
Tests transaction functionality:
- Verifies transaction-related models exist
- Confirms required methods are available
- Tests successful transactions with balance transfers
- Tests transaction rollback on errors
- Tests multiple operations in a single transaction
- Uses `db.transaction()` for transaction handling

### `query_test.js`
Tests query structure and raw queries:
- Verifies query models are accessible
- Confirms all query methods exist
- Tests `db.queryRaw()` for SELECT queries
- Tests `db.executeRaw()` for INSERT/UPDATE/DELETE operations

### `raw_query_test.js`
Comprehensive raw query tests:
- Basic queries with and without parameters
- CREATE TABLE, INSERT, UPDATE, DELETE operations
- Raw queries on schema-defined tables
- Aggregate queries
- Error handling

### `relation_test.js`
Tests relation functionality:
- Schema with relations (one-to-many, many-to-one)
- Creating records with foreign keys
- Querying with relation filters (where clause on foreign keys)
- Counting related records
- Include/eager loading (placeholder for future implementation)

## Running Tests

### Run all tests:
```bash
cd modules/orm/tests
go test -v
```

### Run specific test file:
```bash
cd modules/orm/tests
go test -v -run TestSuite/basic_test.js
```

### Run tests through make:
```bash
# From project root
make test-orm
```

## Test Database

Tests use SQLite in-memory database (`:memory:`) by default for:
- Fast execution
- No cleanup required
- Isolated test environment

## API Changes

The ORM module no longer automatically initializes a database connection. Instead:

1. Use `fromUri()` to create a database instance
2. Call `connect()` to establish connection
3. Load schemas with `loadSchema()` or `loadSchemaFrom()`
4. Call `syncSchemas()` to synchronize with database
5. Access models via `db.models.ModelName`

Example:
```javascript
const { fromUri } = require('redi/orm');

const db = fromUri('sqlite://./mydb.db');
await db.connect();
await db.loadSchema(schemaContent);
await db.syncSchemas();

// Now use models
await db.models.User.create({ data: { name: 'John' } });
```

## Adding New Tests

1. Create a new JavaScript test file following the naming pattern `*_test.js`
2. The test will be automatically discovered by the test runner
3. Follow the existing test structure:
   - Setup database connection using `fromUri()`
   - Load and sync schemas
   - Run test cases with assertions
   - Clean up resources with `db.close()`

## Test Utilities

### `assert.js`
Custom assertion module with methods:
- `assert(condition, message)` - Basic assertion
- `strictEqual(actual, expected, message)` - Strict equality check
- `equal(actual, expected, message)` - Alias for strictEqual
- `deepEqual(actual, expected, message)` - Deep object comparison
- `fail(message)` - Explicit test failure

### Common patterns:
- `fromUri` for database connections
- Schema definition using Prisma syntax
- Async/await for database operations
- Console logging for test progress

## Recent Updates

- Added relation support in schemas
- Field name to column name mapping is now properly handled in where clauses
- Junction table names are generated with alphabetical ordering for consistency
- String utility functions moved from internal package to utils package
- Transaction support implemented with `db.transaction()` method
- Fixed update operations to properly apply where conditions
- Added map support to scanning utilities for flexible result handling
- All ORM tests now pass successfully