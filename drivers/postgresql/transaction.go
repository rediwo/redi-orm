package postgresql

import (
	"context"
	"database/sql"
	"fmt"

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
	// This would need to be implemented with a transaction-aware model query
	// For now, return nil as we need to implement ModelQuery first
	panic("Model query not implemented for transactions")
}

// CreateMany creates multiple records within the transaction
func (t *PostgreSQLTransaction) CreateMany(ctx context.Context, modelName string, data []any) (types.Result, error) {
	// This would need the full model implementation
	return types.Result{}, fmt.Errorf("CreateMany not implemented")
}

// UpdateMany updates multiple records within the transaction
func (t *PostgreSQLTransaction) UpdateMany(ctx context.Context, modelName string, condition types.Condition, data any) (types.Result, error) {
	// This would need the full model implementation
	return types.Result{}, fmt.Errorf("UpdateMany not implemented")
}

// DeleteMany deletes multiple records within the transaction
func (t *PostgreSQLTransaction) DeleteMany(ctx context.Context, modelName string, condition types.Condition) (types.Result, error) {
	// This would need the full model implementation
	return types.Result{}, fmt.Errorf("DeleteMany not implemented")
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
	result, err := q.tx.ExecContext(ctx, q.sql, q.args...)
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
	rows, err := q.tx.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return utils.ScanRows(rows, dest)
}

// FindOne executes the query and scans a single row into dest
func (q *PostgreSQLTransactionRawQuery) FindOne(ctx context.Context, dest any) error {
	return utils.ScanRowContext(q.tx, ctx, q.sql, q.args, dest)
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
