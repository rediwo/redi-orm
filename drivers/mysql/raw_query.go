package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/rediwo/redi-orm/types"
)

// MySQLRawQuery implements types.RawQuery for MySQL
type MySQLRawQuery struct {
	db   *sql.DB
	sql  string
	args []interface{}
}

// NewMySQLRawQuery creates a new MySQL raw query
func NewMySQLRawQuery(db *sql.DB, sql string, args ...interface{}) types.RawQuery {
	return &MySQLRawQuery{
		db:   db,
		sql:  sql,
		args: args,
	}
}

// Exec executes the query and returns the result
func (q *MySQLRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.db.ExecContext(ctx, q.sql, q.args...)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute query: %w", err)
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		// MySQL should support LastInsertId, but handle error gracefully
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

// Find executes the query and scans results into dest
func (q *MySQLRawQuery) Find(ctx context.Context, dest interface{}) error {
	rows, err := q.db.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return scanRows(rows, dest)
}

// FindOne executes the query and scans a single result into dest
func (q *MySQLRawQuery) FindOne(ctx context.Context, dest interface{}) error {
	row := q.db.QueryRowContext(ctx, q.sql, q.args...)
	return scanRow(row, dest)
}

// scanRows scans multiple rows into dest (slice)
func scanRows(rows *sql.Rows, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	sliceVal := destVal.Elem()
	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("destination must be a pointer to slice")
	}

	elemType := sliceVal.Type().Elem()
	isPtr := elemType.Kind() == reflect.Ptr
	if isPtr {
		elemType = elemType.Elem()
	}

	for rows.Next() {
		elem := reflect.New(elemType)
		if err := scanRowToStruct(rows, elem.Interface()); err != nil {
			return err
		}

		if isPtr {
			sliceVal.Set(reflect.Append(sliceVal, elem))
		} else {
			sliceVal.Set(reflect.Append(sliceVal, elem.Elem()))
		}
	}

	return rows.Err()
}

// scanRow scans a single row into dest
func scanRow(row *sql.Row, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	return scanRowToStruct(row, dest)
}

// scanRowToStruct is a helper that handles both *sql.Row and *sql.Rows
func scanRowToStruct(scanner interface{}, dest interface{}) error {
	destVal := reflect.ValueOf(dest).Elem()
	destType := destVal.Type()

	// Handle simple types
	if destType.Kind() != reflect.Struct {
		switch s := scanner.(type) {
		case *sql.Row:
			return s.Scan(dest)
		case *sql.Rows:
			return s.Scan(dest)
		default:
			return fmt.Errorf("unsupported scanner type")
		}
	}

	// Handle struct types
	// This is a simplified implementation - in production, you'd want to
	// match column names to struct fields more intelligently
	numFields := destType.NumField()
	scanDest := make([]interface{}, numFields)

	for i := 0; i < numFields; i++ {
		field := destVal.Field(i)
		if field.CanSet() {
			scanDest[i] = field.Addr().Interface()
		} else {
			// Use a dummy scanner for unexported fields
			var dummy interface{}
			scanDest[i] = &dummy
		}
	}

	switch s := scanner.(type) {
	case *sql.Row:
		return s.Scan(scanDest...)
	case *sql.Rows:
		return s.Scan(scanDest...)
	default:
		return fmt.Errorf("unsupported scanner type")
	}
}