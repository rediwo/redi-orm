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
**Note: Actual CRUD operations require struct handling improvements**

### `schema_test.js`
Tests schema loading and synchronization:
- Loading schema from string (`loadSchema`)
- Loading schema from file (`loadSchemaFrom`)
- Multiple schema loads
- Different database URI support
- Custom table/column mapping

### `transaction_test.js`
Tests transaction structure:
- Verifies transaction-related models exist
- Confirms required methods are available
**Note: Actual transaction operations require global `$transaction` function**

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

### Run JavaScript tests directly:
```bash
node basic_test.js
node schema_test.js
node transaction_test.js
node query_test.js
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
2. Add the test file name to the `testFiles` array in `orm_test.go`
3. Follow the existing test structure:
   - Setup database connection
   - Load and sync schemas
   - Run test cases with assertions
   - Clean up resources

## Test Utilities

Common patterns used across tests:
- `assert` module for validations
- `fromUri` for database connections
- Schema definition using Prisma syntax
- Async/await for database operations
- Console logging for test progress