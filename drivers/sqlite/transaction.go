package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// SQLiteTransaction implements the Transaction interface for SQLite
type SQLiteTransaction struct {
	tx       *sql.Tx
	database *SQLiteDB
}

// NewSQLiteTransaction creates a new SQLite transaction
func NewSQLiteTransaction(tx *sql.Tx, database *SQLiteDB) *SQLiteTransaction {
	return &SQLiteTransaction{
		tx:       tx,
		database: database,
	}
}

// Model creates a new model query within the transaction
func (t *SQLiteTransaction) Model(modelName string) types.ModelQuery {
	// Create a transaction-aware database wrapper
	txDB := &SQLiteTransactionDB{
		transaction: t,
		database:    t.database,
	}
	return query.NewModelQuery(modelName, txDB, t.database.GetFieldMapper())
}

// Raw creates a new raw query within the transaction
func (t *SQLiteTransaction) Raw(sql string, args ...any) types.RawQuery {
	return &SQLiteTransactionRawQuery{
		tx:   t.tx,
		sql:  sql,
		args: args,
	}
}

// Commit commits the transaction
func (t *SQLiteTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *SQLiteTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

// Savepoint creates a savepoint (SQLite supports nested transactions via savepoints)
func (t *SQLiteTransaction) Savepoint(ctx context.Context, name string) error {
	_, err := t.tx.ExecContext(ctx, fmt.Sprintf("SAVEPOINT %s", name))
	return err
}

// RollbackTo rolls back to a savepoint
func (t *SQLiteTransaction) RollbackTo(ctx context.Context, name string) error {
	_, err := t.tx.ExecContext(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", name))
	return err
}

// CreateMany performs batch insert within the transaction
func (t *SQLiteTransaction) CreateMany(ctx context.Context, modelName string, data []any) (types.Result, error) {
	if len(data) == 0 {
		return types.Result{}, fmt.Errorf("no data to insert")
	}

	// Create an insert query and add all data
	insertQuery := t.Model(modelName).Insert(data[0])
	if len(data) > 1 {
		insertQuery = insertQuery.Values(data[1:]...)
	}

	return insertQuery.Exec(ctx)
}

// UpdateMany performs batch update within the transaction
func (t *SQLiteTransaction) UpdateMany(ctx context.Context, modelName string, condition types.Condition, data any) (types.Result, error) {
	updateQuery := t.Model(modelName).Update(data).WhereCondition(condition)
	return updateQuery.Exec(ctx)
}

// DeleteMany performs batch delete within the transaction
func (t *SQLiteTransaction) DeleteMany(ctx context.Context, modelName string, condition types.Condition) (types.Result, error) {
	deleteQuery := t.Model(modelName).Delete().WhereCondition(condition)
	return deleteQuery.Exec(ctx)
}

// SQLiteTransactionDB wraps the transaction for database operations
type SQLiteTransactionDB struct {
	transaction *SQLiteTransaction
	database    *SQLiteDB
}

// All Database interface methods that delegate to the transaction
func (td *SQLiteTransactionDB) Connect(ctx context.Context) error {
	return fmt.Errorf("cannot connect within a transaction")
}

func (td *SQLiteTransactionDB) Close() error {
	return fmt.Errorf("cannot close within a transaction")
}

func (td *SQLiteTransactionDB) Ping(ctx context.Context) error {
	return td.database.Ping(ctx)
}

func (td *SQLiteTransactionDB) RegisterSchema(modelName string, schema *schema.Schema) error {
	return td.database.RegisterSchema(modelName, schema)
}

func (td *SQLiteTransactionDB) GetSchema(modelName string) (*schema.Schema, error) {
	return td.database.GetSchema(modelName)
}

func (td *SQLiteTransactionDB) CreateModel(ctx context.Context, modelName string) error {
	return fmt.Errorf("cannot create model within a transaction")
}

func (td *SQLiteTransactionDB) DropModel(ctx context.Context, modelName string) error {
	return fmt.Errorf("cannot drop model within a transaction")
}

func (td *SQLiteTransactionDB) Model(modelName string) types.ModelQuery {
	return td.transaction.Model(modelName)
}

func (td *SQLiteTransactionDB) Raw(sql string, args ...any) types.RawQuery {
	return td.transaction.Raw(sql, args...)
}

func (td *SQLiteTransactionDB) Begin(ctx context.Context) (types.Transaction, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

func (td *SQLiteTransactionDB) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	return fn(td.transaction)
}

func (td *SQLiteTransactionDB) GetModels() []string {
	return td.database.GetModels()
}

func (td *SQLiteTransactionDB) GetModelSchema(modelName string) (*schema.Schema, error) {
	return td.database.GetModelSchema(modelName)
}

func (td *SQLiteTransactionDB) GetDriverType() string {
	return td.database.GetDriverType()
}

func (td *SQLiteTransactionDB) ResolveTableName(modelName string) (string, error) {
	return td.database.ResolveTableName(modelName)
}

func (td *SQLiteTransactionDB) ResolveFieldName(modelName, fieldName string) (string, error) {
	return td.database.ResolveFieldName(modelName, fieldName)
}

func (td *SQLiteTransactionDB) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return td.database.ResolveFieldNames(modelName, fieldNames)
}

func (td *SQLiteTransactionDB) Exec(query string, args ...any) (sql.Result, error) {
	return td.transaction.tx.Exec(query, args...)
}

func (td *SQLiteTransactionDB) Query(query string, args ...any) (*sql.Rows, error) {
	return td.transaction.tx.Query(query, args...)
}

func (td *SQLiteTransactionDB) QueryRow(query string, args ...any) *sql.Row {
	return td.transaction.tx.QueryRow(query, args...)
}

func (td *SQLiteTransactionDB) GetMigrator() types.DatabaseMigrator {
	return td.database.GetMigrator()
}

// LoadSchema is not supported within a transaction
func (td *SQLiteTransactionDB) LoadSchema(ctx context.Context, schemaContent string) error {
	return fmt.Errorf("cannot load schema within a transaction")
}

// LoadSchemaFrom is not supported within a transaction
func (td *SQLiteTransactionDB) LoadSchemaFrom(ctx context.Context, filename string) error {
	return fmt.Errorf("cannot load schema from file within a transaction")
}

// SyncSchemas is not supported within a transaction
func (td *SQLiteTransactionDB) SyncSchemas(ctx context.Context) error {
	return fmt.Errorf("cannot sync schemas within a transaction")
}

// SQLiteTransactionRawQuery implements RawQuery for transactions
type SQLiteTransactionRawQuery struct {
	tx   *sql.Tx
	sql  string
	args []any
}

func (q *SQLiteTransactionRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.tx.ExecContext(ctx, q.sql, q.args...)
	if err != nil {
		return types.Result{}, err
	}

	lastInsertID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	return types.Result{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
	}, nil
}

func (q *SQLiteTransactionRawQuery) Find(ctx context.Context, dest any) error {
	rows, err := q.tx.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return utils.ScanRows(rows, dest)
}

func (q *SQLiteTransactionRawQuery) FindOne(ctx context.Context, dest any) error {
	return utils.ScanRowContext(q.tx, ctx, q.sql, q.args, dest)
}
