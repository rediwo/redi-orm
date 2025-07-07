package sqlite

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// SQLiteRawQuery implements the RawQuery interface for SQLite
type SQLiteRawQuery struct {
	driver *SQLiteDB
	sql    string
	args   []any
}

// NewSQLiteRawQuery creates a new SQLite raw query
func NewSQLiteRawQuery(driver *SQLiteDB, sql string, args ...any) types.RawQuery {
	return &SQLiteRawQuery{
		driver: driver,
		sql:    sql,
		args:   args,
	}
}

// Exec executes the raw query and returns the result
func (q *SQLiteRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.driver.Exec(q.sql, q.args...)
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

// Find executes the raw query and returns multiple results
func (q *SQLiteRawQuery) Find(ctx context.Context, dest any) error {
	rows, err := q.driver.Query(q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return utils.ScanRows(rows, dest)
}

// FindOne executes the raw query and returns a single result
func (q *SQLiteRawQuery) FindOne(ctx context.Context, dest any) error {
	// For INSERT with RETURNING, we need to handle the case where the INSERT fails
	// but the error is masked by "no rows in result set"
	// Use driver's Query method which includes logging
	rows, err := q.driver.Query(q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Check if we have any rows
	if !rows.Next() {
		// Special handling for INSERT ... RETURNING
		if strings.Contains(strings.ToUpper(q.sql), "INSERT") && strings.Contains(strings.ToUpper(q.sql), "RETURNING") {
			// Execute the query to get the actual error
			_, execErr := q.driver.Exec(q.sql, q.args...)
			if execErr != nil {
				return execErr // Return the actual error (e.g., unique constraint violation)
			}
		}
		return fmt.Errorf("sql: no rows in result set")
	}

	// Handle simple types directly
	destType := reflect.TypeOf(dest)
	if destType.Kind() == reflect.Ptr && utils.IsSimpleType(destType.Elem()) {
		return rows.Scan(dest)
	}

	// For complex types, delegate to ScanRow which expects Next() to have been called
	// Create a custom ScanRow implementation since rows.Next() was already called
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Handle map[string]any
	if destType.Kind() == reflect.Ptr && destType.Elem().Kind() == reflect.Map &&
		destType.Elem().Key().Kind() == reflect.String &&
		destType.Elem().Elem().Kind() == reflect.Interface {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		result := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			// Handle byte arrays (convert to string)
			if b, ok := val.([]byte); ok {
				result[col] = string(b)
			} else {
				result[col] = val
			}
		}

		reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(result))
		return nil
	}

	// For other types, handle directly
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return err
	}

	// For now, just handle the simple case directly
	// This is good enough for the COUNT(*) query
	if len(columns) == 1 && len(valuePtrs) == 1 {
		reflect.ValueOf(dest).Elem().Set(reflect.ValueOf(values[0]).Convert(destType.Elem()))
		return nil
	}

	return fmt.Errorf("complex type scanning not fully implemented in FindOne")
}
