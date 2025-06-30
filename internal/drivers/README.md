# Database Drivers

This directory contains database drivers for ReORM, organized by database type.

## Structure

```
drivers/
├── sqlite/
│   ├── driver.go         # Main SQLite driver implementation
│   ├── query.go          # SQLite query builder
│   ├── transaction.go    # SQLite transaction support
│   ├── mapping.go        # SQLite type mapping
│   ├── driver_test.go    # SQLite driver tests
│   └── query_test.go     # Query builder tests
├── mysql/
│   ├── driver.go         # Main MySQL driver implementation
│   ├── query.go          # MySQL query builder
│   ├── transaction.go    # MySQL transaction support
│   ├── mapping.go        # MySQL type mapping
│   └── driver_test.go    # MySQL driver tests
├── postgresql/
│   ├── driver.go         # Main PostgreSQL driver implementation
│   ├── query.go          # PostgreSQL query builder
│   ├── transaction.go    # PostgreSQL transaction support
│   ├── mapping.go        # PostgreSQL type mapping
│   └── driver_test.go    # PostgreSQL driver tests
└── README.md             # This file
```

## Driver Interface

All drivers implement the `types.Database` interface, providing:

- **Connection Management**: `Connect()`, `Close()`
- **Schema Operations**: `CreateTable()`, `DropTable()`
- **CRUD Operations**: `Insert()`, `FindByID()`, `Find()`, `Update()`, `Delete()`
- **Query Building**: `Select()` returns a query builder
- **Transactions**: `Begin()` returns a transaction object
- **Raw Queries**: `Exec()`, `Query()`, `QueryRow()`

## Database-Specific Features

### SQLite
- Uses `?` placeholders for parameters
- Supports `AUTOINCREMENT` for primary keys
- File-based or in-memory databases

### MySQL
- Uses `?` placeholders for parameters
- Supports `AUTO_INCREMENT` for primary keys
- Requires network connection configuration
- Uses backticks for identifier quoting

### PostgreSQL
- Uses `$1`, `$2`, etc. placeholders for parameters
- Supports `SERIAL`/`BIGSERIAL` for primary keys
- Requires network connection configuration
- Uses double quotes for identifier quoting

## Testing

Each driver includes comprehensive tests covering:
- Connection management
- Table operations (CREATE, DROP)
- CRUD operations (INSERT, SELECT, UPDATE, DELETE)
- Query building and execution
- Transaction handling (COMMIT, ROLLBACK)

### Running Tests

Run tests for a specific driver:
```bash
go test ./internal/drivers/sqlite
go test ./internal/drivers/mysql    # Requires Docker with MySQL
go test ./internal/drivers/postgresql # Requires Docker with PostgreSQL
```

Run all driver tests:
```bash
go test ./internal/drivers/...
```

### Docker-based Testing

MySQL and PostgreSQL tests require Docker containers to be running:
```bash
# Start test databases
make docker-up

# Run specific database tests
make test-mysql
make test-postgresql

# Stop test databases
make docker-down
```

Tests gracefully skip if Docker databases are not available.

## Adding New Drivers

To add support for a new database:

1. Create a new subdirectory (e.g., `drivers/newdb/`)
2. Implement the `types.Database` interface
3. Create query builder and transaction types
4. Add comprehensive tests
5. Update `pkg/database/drivers.go` to include the new driver
6. Add the database type to `pkg/types/database.go`