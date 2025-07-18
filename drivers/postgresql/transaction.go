package postgresql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rediwo/redi-orm/base"
	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// PostgreSQLTransaction implements the Transaction interface for PostgreSQL
type PostgreSQLTransaction struct {
	tx *sql.Tx
	db *PostgreSQLDB
}

// Commit commits the transaction
func (t *PostgreSQLTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *PostgreSQLTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

// Savepoint creates a savepoint with the given name
func (t *PostgreSQLTransaction) Savepoint(ctx context.Context, name string) error {
	_, err := t.tx.ExecContext(ctx, fmt.Sprintf("SAVEPOINT %s", t.db.quoteIdentifier(name)))
	if err != nil {
		return fmt.Errorf("failed to create savepoint: %w", err)
	}
	return nil
}

// RollbackTo rolls back to the savepoint with the given name
func (t *PostgreSQLTransaction) RollbackTo(ctx context.Context, name string) error {
	_, err := t.tx.ExecContext(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", t.db.quoteIdentifier(name)))
	if err != nil {
		return fmt.Errorf("failed to rollback to savepoint: %w", err)
	}
	return nil
}

// ReleaseSavepoint releases the savepoint with the given name
func (t *PostgreSQLTransaction) ReleaseSavepoint(ctx context.Context, name string) error {
	_, err := t.tx.ExecContext(ctx, fmt.Sprintf("RELEASE SAVEPOINT %s", t.db.quoteIdentifier(name)))
	if err != nil {
		return fmt.Errorf("failed to release savepoint: %w", err)
	}
	return nil
}

// Model creates a model query within the transaction
func (t *PostgreSQLTransaction) Model(modelName string) types.ModelQuery {
	// Create a transaction-aware database wrapper
	txDB := &PostgreSQLTransactionDB{
		PostgreSQLDB: t.db,
		tx:           t.tx,
	}
	return query.NewModelQuery(modelName, txDB, t.db.GetFieldMapper())
}

// CreateMany creates multiple records within the transaction
func (t *PostgreSQLTransaction) CreateMany(ctx context.Context, modelName string, data []any) (types.Result, error) {
	utils := base.NewTransactionUtils(t.tx, t.db, "postgresql")
	return utils.CreateMany(ctx, modelName, data)
}

// UpdateMany updates multiple records within the transaction
func (t *PostgreSQLTransaction) UpdateMany(ctx context.Context, modelName string, condition types.Condition, data any) (types.Result, error) {
	utils := base.NewTransactionUtils(t.tx, t.db, "postgresql")
	return utils.UpdateMany(ctx, modelName, condition, data)
}

// DeleteMany deletes multiple records within the transaction
func (t *PostgreSQLTransaction) DeleteMany(ctx context.Context, modelName string, condition types.Condition) (types.Result, error) {
	utils := base.NewTransactionUtils(t.tx, t.db, "postgresql")
	return utils.DeleteMany(ctx, modelName, condition)
}

// Raw creates a raw query within the transaction
func (t *PostgreSQLTransaction) Raw(query string, args ...any) types.RawQuery {
	return &PostgreSQLTransactionRawQuery{
		tx:   t.tx,
		sql:  query,
		args: args,
	}
}

// PostgreSQLTransactionRawQuery implements RawQuery for transactions
type PostgreSQLTransactionRawQuery struct {
	tx   *sql.Tx
	sql  string
	args []any
}

// Exec executes the query within the transaction
func (q *PostgreSQLTransactionRawQuery) Exec(ctx context.Context) (types.Result, error) {
	// Convert ? placeholders to $1, $2, etc.
	sql := convertPlaceholders(q.sql)
	result, err := q.tx.ExecContext(ctx, sql, q.args...)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute query: %w", err)
	}

	lastInsertID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	return types.Result{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
	}, nil
}

// Find executes the query and scans multiple rows into dest
func (q *PostgreSQLTransactionRawQuery) Find(ctx context.Context, dest any) error {
	// Convert ? placeholders to $1, $2, etc.
	sql := convertPlaceholders(q.sql)
	rows, err := q.tx.QueryContext(ctx, sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return utils.ScanRows(rows, dest)
}

// FindOne executes the query and scans a single row into dest
func (q *PostgreSQLTransactionRawQuery) FindOne(ctx context.Context, dest any) error {
	// Convert ? placeholders to $1, $2, etc.
	sql := convertPlaceholders(q.sql)
	return utils.ScanRowContext(q.tx, ctx, sql, q.args, dest)
}

// PostgreSQLTransactionDB wraps PostgreSQLDB for transaction context
type PostgreSQLTransactionDB struct {
	*PostgreSQLDB
	tx *sql.Tx
}

// GetDriverType returns the database driver type
func (t *PostgreSQLTransactionDB) GetDriverType() string {
	return t.PostgreSQLDB.GetDriverType()
}

// GetCapabilities returns driver capabilities
func (t *PostgreSQLTransactionDB) GetCapabilities() types.DriverCapabilities {
	return t.PostgreSQLDB.GetCapabilities()
}

// Raw creates a raw query within the transaction
func (t *PostgreSQLTransactionDB) Raw(query string, args ...any) types.RawQuery {
	return &PostgreSQLTransactionRawQuery{
		tx:   t.tx,
		sql:  query,
		args: args,
	}
}

// Transaction within a transaction is not supported
func (t *PostgreSQLTransactionDB) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	return fmt.Errorf("nested transactions are not supported, use savepoints instead")
}

// Begin within a transaction is not supported
func (t *PostgreSQLTransactionDB) Begin(ctx context.Context) (types.Transaction, error) {
	return nil, fmt.Errorf("nested transactions are not supported, use savepoints instead")
}

// RegisterSchema registers a schema with the database
func (t *PostgreSQLTransactionDB) RegisterSchema(modelName string, s *schema.Schema) error {
	return t.PostgreSQLDB.RegisterSchema(modelName, s)
}

// GetSchema returns the schema for a model
func (t *PostgreSQLTransactionDB) GetSchema(modelName string) (*schema.Schema, error) {
	return t.PostgreSQLDB.GetSchema(modelName)
}

// GetModelSchema returns the schema interface for a model
func (t *PostgreSQLTransactionDB) GetModelSchema(modelName string) (*schema.Schema, error) {
	return t.PostgreSQLDB.GetModelSchema(modelName)
}

// LoadSchema is not supported within a transaction
func (t *PostgreSQLTransactionDB) LoadSchema(ctx context.Context, schemaContent string) error {
	return fmt.Errorf("cannot load schema within a transaction")
}

// LoadSchemaFrom is not supported within a transaction
func (t *PostgreSQLTransactionDB) LoadSchemaFrom(ctx context.Context, filename string) error {
	return fmt.Errorf("cannot load schema from file within a transaction")
}

// SyncSchemas is not supported within a transaction
func (t *PostgreSQLTransactionDB) SyncSchemas(ctx context.Context) error {
	return fmt.Errorf("cannot sync schemas within a transaction")
}

// Model creates a model query within the transaction
func (t *PostgreSQLTransactionDB) Model(modelName string) types.ModelQuery {
	// Create a new transaction that wraps this one
	txWrapper := &PostgreSQLTransaction{
		tx: t.tx,
		db: t.PostgreSQLDB,
	}
	return txWrapper.Model(modelName)
}

// GetModels returns all registered model names
func (t *PostgreSQLTransactionDB) GetModels() []string {
	return t.PostgreSQLDB.GetModels()
}

// ResolveTableName resolves model name to table name
func (t *PostgreSQLTransactionDB) ResolveTableName(modelName string) (string, error) {
	return t.PostgreSQLDB.ResolveTableName(modelName)
}

// ResolveFieldName resolves schema field name to column name
func (t *PostgreSQLTransactionDB) ResolveFieldName(modelName, fieldName string) (string, error) {
	return t.PostgreSQLDB.ResolveFieldName(modelName, fieldName)
}

// ResolveFieldNames resolves multiple schema field names to column names
func (t *PostgreSQLTransactionDB) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return t.PostgreSQLDB.ResolveFieldNames(modelName, fieldNames)
}

// CreateModel is not supported within a transaction
func (t *PostgreSQLTransactionDB) CreateModel(ctx context.Context, modelName string) error {
	return fmt.Errorf("cannot create model within a transaction")
}

// DropModel is not supported within a transaction
func (t *PostgreSQLTransactionDB) DropModel(ctx context.Context, modelName string) error {
	return fmt.Errorf("cannot drop model within a transaction")
}

// GetMigrator is not supported within a transaction
func (t *PostgreSQLTransactionDB) GetMigrator() types.DatabaseMigrator {
	return nil
}

// Exec executes a raw SQL statement within the transaction
func (t *PostgreSQLTransactionDB) Exec(query string, args ...any) (sql.Result, error) {
	// Convert ? placeholders to $1, $2, etc.
	query = convertPlaceholders(query)
	return t.tx.Exec(query, args...)
}

// Query executes a raw SQL query that returns rows within the transaction
func (t *PostgreSQLTransactionDB) Query(query string, args ...any) (*sql.Rows, error) {
	// Convert ? placeholders to $1, $2, etc.
	query = convertPlaceholders(query)
	return t.tx.Query(query, args...)
}

// QueryRow executes a raw SQL query that returns a single row within the transaction
func (t *PostgreSQLTransactionDB) QueryRow(query string, args ...any) *sql.Row {
	// Convert ? placeholders to $1, $2, etc.
	query = convertPlaceholders(query)
	return t.tx.QueryRow(query, args...)
}

// Connect is not supported within a transaction
func (t *PostgreSQLTransactionDB) Connect(ctx context.Context) error {
	return fmt.Errorf("cannot connect within a transaction")
}

// Close is not supported within a transaction
func (t *PostgreSQLTransactionDB) Close() error {
	return fmt.Errorf("cannot close within a transaction")
}

// Ping is not supported within a transaction
func (t *PostgreSQLTransactionDB) Ping(ctx context.Context) error {
	return fmt.Errorf("cannot ping within a transaction")
}
