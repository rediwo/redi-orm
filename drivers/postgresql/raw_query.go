package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// PostgreSQLRawQuery implements the RawQuery interface for PostgreSQL
type PostgreSQLRawQuery struct {
	db   *sql.DB
	sql  string
	args []interface{}
}

// Exec executes the query and returns the result
func (q *PostgreSQLRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.db.ExecContext(ctx, q.sql, q.args...)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute query: %w", err)
	}

	// PostgreSQL doesn't support LastInsertId in the standard way
	// We need to use RETURNING clause for that
	lastInsertID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	return types.Result{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
	}, nil
}

// Find executes the query and scans multiple rows into dest
func (q *PostgreSQLRawQuery) Find(ctx context.Context, dest interface{}) error {
	rows, err := q.db.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Use reflection to handle scanning into slice
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
		// Create new element
		elem := reflect.New(elementType).Elem()

		// Prepare scan destinations
		scanDest := make([]interface{}, len(columns))
		for i, col := range columns {
			field := elem.FieldByNameFunc(func(name string) bool {
				field, _ := elementType.FieldByName(name)
				// Check struct tag for db column name
				if tag := field.Tag.Get("db"); tag != "" {
					return tag == col
				}
				// Check json tag as fallback
				if tag := field.Tag.Get("json"); tag != "" {
					return tag == col
				}
				// Default to field name - case insensitive for simple fields
				if strings.EqualFold(field.Name, col) {
					return true
				}
				// Try camel case conversion
				return utils.ToCamelCase(col) == name ||
					   utils.ToSnakeCase(name) == col
			})
			
			if field.IsValid() && field.CanSet() {
				scanDest[i] = field.Addr().Interface()
			} else {
				// Field not found, use a dummy scanner
				var dummy interface{}
				scanDest[i] = &dummy
			}
		}

		if err := rows.Scan(scanDest...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Append to slice
		if isPtr {
			sliceValue.Set(reflect.Append(sliceValue, elem.Addr()))
		} else {
			sliceValue.Set(reflect.Append(sliceValue, elem))
		}
	}

	return rows.Err()
}

// FindOne executes the query and scans a single row into dest
func (q *PostgreSQLRawQuery) FindOne(ctx context.Context, dest interface{}) error {
	destType := reflect.TypeOf(dest)
	if destType.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	// Handle single value scanning
	if isSimpleType(destType.Elem()) {
		return q.db.QueryRowContext(ctx, q.sql, q.args...).Scan(dest)
	}

	// Handle struct scanning
	rows, err := q.db.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return sql.ErrNoRows
	}

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Use reflection to scan into struct
	destValue := reflect.ValueOf(dest).Elem()
	destType = destValue.Type()

	scanDest := make([]interface{}, len(columns))
	for i, col := range columns {
		field := destValue.FieldByNameFunc(func(name string) bool {
			field, _ := destType.FieldByName(name)
			// Check struct tag for db column name
			if tag := field.Tag.Get("db"); tag != "" {
				return tag == col
			}
			// Check json tag as fallback
			if tag := field.Tag.Get("json"); tag != "" {
				return tag == col
			}
			// Default to field name - case insensitive for simple fields
			if strings.EqualFold(field.Name, col) {
				return true
			}
			// Try camel case conversion
			return utils.ToCamelCase(col) == name ||
				   utils.ToSnakeCase(name) == col
		})
		
		if field.IsValid() && field.CanSet() {
			scanDest[i] = field.Addr().Interface()
		} else {
			// Field not found, use a dummy scanner
			var dummy interface{}
			scanDest[i] = &dummy
		}
	}

	return rows.Scan(scanDest...)
}

// Helper functions
func isSimpleType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	default:
		return false
	}
}