package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// MySQLTransaction implements types.Transaction for MySQL
type MySQLTransaction struct {
	tx *sql.Tx
	db *MySQLDB
}

// NewMySQLTransaction creates a new MySQL transaction
func NewMySQLTransaction(tx *sql.Tx, database *MySQLDB) types.Transaction {
	return &MySQLTransaction{
		tx: tx,
		db: database,
	}
}

// Model creates a new model query within the transaction
func (t *MySQLTransaction) Model(modelName string) types.ModelQuery {
	// Create a transaction database that uses the transaction
	txDB := &MySQLTransactionDB{
		db: t.db,
		tx: t,
	}
	return query.NewModelQuery(modelName, txDB, t.db.GetFieldMapper())
}

// Raw creates a new raw query within the transaction
func (t *MySQLTransaction) Raw(sql string, args ...interface{}) types.RawQuery {
	return &MySQLTransactionRawQuery{tx: t.tx, sql: sql, args: args}
}

// Commit commits the transaction
func (t *MySQLTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *MySQLTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

// Savepoint creates a new savepoint
func (t *MySQLTransaction) Savepoint(ctx context.Context, name string) error {
	_, err := t.tx.ExecContext(ctx, fmt.Sprintf("SAVEPOINT `%s`", name))
	if err != nil {
		return fmt.Errorf("failed to create savepoint: %w", err)
	}
	return nil
}

// RollbackTo rolls back to a specific savepoint
func (t *MySQLTransaction) RollbackTo(ctx context.Context, name string) error {
	_, err := t.tx.ExecContext(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT `%s`", name))
	if err != nil {
		return fmt.Errorf("failed to rollback to savepoint: %w", err)
	}
	return nil
}

// CreateMany creates multiple records within the transaction
func (t *MySQLTransaction) CreateMany(ctx context.Context, modelName string, data []interface{}) (types.Result, error) {
	// For now, return a simple implementation
	// In production, this would build a bulk INSERT statement
	return types.Result{}, fmt.Errorf("batch operations not yet implemented for MySQL")
}

// UpdateMany updates multiple records within the transaction
func (t *MySQLTransaction) UpdateMany(ctx context.Context, modelName string, condition types.Condition, data interface{}) (types.Result, error) {
	// For now, return a simple implementation
	return types.Result{}, fmt.Errorf("batch operations not yet implemented for MySQL")
}

// DeleteMany deletes multiple records within the transaction
func (t *MySQLTransaction) DeleteMany(ctx context.Context, modelName string, condition types.Condition) (types.Result, error) {
	// For now, return a simple implementation
	return types.Result{}, fmt.Errorf("batch operations not yet implemented for MySQL")
}

// MySQLTransactionDB implements types.Database for use within a transaction
type MySQLTransactionDB struct {
	db *MySQLDB
	tx *MySQLTransaction
}

// Connect - not supported in transaction
func (tdb *MySQLTransactionDB) Connect(ctx context.Context) error {
	return fmt.Errorf("cannot connect within a transaction")
}

// Close - not supported in transaction
func (tdb *MySQLTransactionDB) Close() error {
	return fmt.Errorf("cannot close within a transaction")
}

// Ping delegates to the main database
func (tdb *MySQLTransactionDB) Ping(ctx context.Context) error {
	return tdb.db.Ping(ctx)
}

// RegisterSchema delegates to the main database
func (tdb *MySQLTransactionDB) RegisterSchema(modelName string, s *schema.Schema) error {
	return tdb.db.RegisterSchema(modelName, s)
}

// GetSchema delegates to the main database
func (tdb *MySQLTransactionDB) GetSchema(modelName string) (*schema.Schema, error) {
	return tdb.db.GetSchema(modelName)
}

// GetModels delegates to the main database
func (tdb *MySQLTransactionDB) GetModels() []string {
	return tdb.db.GetModels()
}

// GetModelSchema delegates to the main database
func (tdb *MySQLTransactionDB) GetModelSchema(modelName string) (*schema.Schema, error) {
	return tdb.db.GetModelSchema(modelName)
}

// ResolveTableName delegates to the main database
func (tdb *MySQLTransactionDB) ResolveTableName(modelName string) (string, error) {
	return tdb.db.ResolveTableName(modelName)
}

// ResolveFieldName delegates to the main database
func (tdb *MySQLTransactionDB) ResolveFieldName(modelName, fieldName string) (string, error) {
	return tdb.db.ResolveFieldName(modelName, fieldName)
}

// ResolveFieldNames delegates to the main database
func (tdb *MySQLTransactionDB) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return tdb.db.ResolveFieldNames(modelName, fieldNames)
}

// GetFieldMapper delegates to the main database
func (tdb *MySQLTransactionDB) GetFieldMapper() types.FieldMapper {
	return tdb.db.GetFieldMapper()
}

// CreateModel - not supported in transaction
func (tdb *MySQLTransactionDB) CreateModel(ctx context.Context, modelName string) error {
	return fmt.Errorf("cannot create model within a transaction")
}

// DropModel - not supported in transaction
func (tdb *MySQLTransactionDB) DropModel(ctx context.Context, modelName string) error {
	return fmt.Errorf("cannot drop model within a transaction")
}

// Model creates a model query that uses the transaction
func (tdb *MySQLTransactionDB) Model(modelName string) types.ModelQuery {
	return tdb.tx.Model(modelName)
}

// Raw creates a raw query that uses the transaction
func (tdb *MySQLTransactionDB) Raw(sql string, args ...interface{}) types.RawQuery {
	return tdb.tx.Raw(sql, args...)
}

// Begin - not supported in transaction
func (tdb *MySQLTransactionDB) Begin(ctx context.Context) (types.Transaction, error) {
	return nil, fmt.Errorf("cannot begin transaction within a transaction")
}

// Transaction - not supported in transaction
func (tdb *MySQLTransactionDB) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	return fmt.Errorf("cannot start transaction within a transaction")
}

// GetMigrator delegates to the main database
func (tdb *MySQLTransactionDB) GetMigrator() types.DatabaseMigrator {
	return tdb.db.GetMigrator()
}

// Exec executes a query within the transaction
func (tdb *MySQLTransactionDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return tdb.tx.tx.Exec(query, args...)
}

// Query executes a query within the transaction
func (tdb *MySQLTransactionDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return tdb.tx.tx.Query(query, args...)
}

// QueryRow executes a query within the transaction
func (tdb *MySQLTransactionDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return tdb.tx.tx.QueryRow(query, args...)
}

// MySQLTransactionRawQuery implements RawQuery for MySQL transactions
type MySQLTransactionRawQuery struct {
	tx   *sql.Tx
	sql  string
	args []interface{}
}

// Exec executes the query within a transaction
func (q *MySQLTransactionRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.tx.ExecContext(ctx, q.sql, q.args...)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute query: %w", err)
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		lastInsertID = 0
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		rowsAffected = 0
	}

	return types.Result{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
	}, nil
}

// Find executes the query and scans results into dest within a transaction
func (q *MySQLTransactionRawQuery) Find(ctx context.Context, dest interface{}) error {
	rows, err := q.tx.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return utils.ScanRows(rows, dest)
}

// FindOne executes the query and scans a single result into dest within a transaction
func (q *MySQLTransactionRawQuery) FindOne(ctx context.Context, dest interface{}) error {
	return utils.ScanRowContext(q.tx, ctx, q.sql, q.args, dest)
}