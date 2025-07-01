package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

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
func (t *PostgreSQLTransaction) CreateMany(ctx context.Context, modelName string, data []interface{}) (types.Result, error) {
	// This would need the full model implementation
	return types.Result{}, fmt.Errorf("CreateMany not implemented")
}

// UpdateMany updates multiple records within the transaction
func (t *PostgreSQLTransaction) UpdateMany(ctx context.Context, modelName string, condition types.Condition, data interface{}) (types.Result, error) {
	// This would need the full model implementation
	return types.Result{}, fmt.Errorf("UpdateMany not implemented")
}

// DeleteMany deletes multiple records within the transaction
func (t *PostgreSQLTransaction) DeleteMany(ctx context.Context, modelName string, condition types.Condition) (types.Result, error) {
	// This would need the full model implementation
	return types.Result{}, fmt.Errorf("DeleteMany not implemented")
}

// Raw creates a raw query within the transaction
func (t *PostgreSQLTransaction) Raw(query string, args ...interface{}) types.RawQuery {
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
	args []interface{}
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
func (q *PostgreSQLTransactionRawQuery) Find(ctx context.Context, dest interface{}) error {
	rows, err := q.tx.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Use the shared scanning logic
	return scanRows(rows, dest)
}

// FindOne executes the query and scans a single row into dest
func (q *PostgreSQLTransactionRawQuery) FindOne(ctx context.Context, dest interface{}) error {
	// Handle single value scanning
	if isSimpleType(reflect.TypeOf(dest).Elem()) {
		return q.tx.QueryRowContext(ctx, q.sql, q.args...).Scan(dest)
	}

	// Handle struct scanning
	rows, err := q.tx.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return sql.ErrNoRows
	}

	return scanRow(rows, dest)
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
func (t *PostgreSQLTransactionDB) Raw(query string, args ...interface{}) types.RawQuery {
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

// Helper function to scan rows (shared between Find implementations)
func scanRows(rows *sql.Rows, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	sliceValue := destValue.Elem()
	elementType := sliceValue.Type().Elem()
	isPtr := elementType.Kind() == reflect.Ptr
	if isPtr {
		elementType = elementType.Elem()
	}

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	for rows.Next() {
		elem := reflect.New(elementType).Elem()
		scanDest := make([]interface{}, len(columns))
		
		for i, col := range columns {
			field := elem.FieldByNameFunc(func(name string) bool {
				field, _ := elementType.FieldByName(name)
				if tag := field.Tag.Get("db"); tag != "" {
					return tag == col
				}
				if tag := field.Tag.Get("json"); tag != "" {
					return tag == col
				}
				if strings.EqualFold(field.Name, col) {
					return true
				}
				return utils.ToCamelCase(col) == name ||
					   utils.ToSnakeCase(name) == col
			})
			
			if field.IsValid() && field.CanSet() {
				scanDest[i] = field.Addr().Interface()
			} else {
				var dummy interface{}
				scanDest[i] = &dummy
			}
		}

		if err := rows.Scan(scanDest...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		if isPtr {
			sliceValue.Set(reflect.Append(sliceValue, elem.Addr()))
		} else {
			sliceValue.Set(reflect.Append(sliceValue, elem))
		}
	}

	return rows.Err()
}

// Helper function to scan a single row
func scanRow(rows *sql.Rows, dest interface{}) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	destValue := reflect.ValueOf(dest).Elem()
	destType := destValue.Type()

	scanDest := make([]interface{}, len(columns))
	for i, col := range columns {
		field := destValue.FieldByNameFunc(func(name string) bool {
			field, _ := destType.FieldByName(name)
			if tag := field.Tag.Get("db"); tag != "" {
				return tag == col
			}
			if tag := field.Tag.Get("json"); tag != "" {
				return tag == col
			}
			if strings.EqualFold(field.Name, col) {
				return true
			}
			return utils.ToCamelCase(col) == name ||
				   utils.ToSnakeCase(name) == col
		})
		
		if field.IsValid() && field.CanSet() {
			scanDest[i] = field.Addr().Interface()
		} else {
			var dummy interface{}
			scanDest[i] = &dummy
		}
	}

	return rows.Scan(scanDest...)
}