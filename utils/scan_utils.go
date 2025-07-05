package utils

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// ScanRows scans multiple rows into a slice with smart field mapping
func ScanRows(rows *sql.Rows, dest any) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	sliceValue := destValue.Elem()
	elementType := sliceValue.Type().Elem()

	// Check if dest is *[]map[string]any
	if elementType.Kind() == reflect.Map &&
		elementType.Key().Kind() == reflect.String &&
		elementType.Elem().Kind() == reflect.Interface &&
		elementType.Elem().NumMethod() == 0 {
		// Use ScanRowsToMaps for map scanning
		results, err := ScanRowsToMaps(rows)
		if err != nil {
			return err
		}
		// Set the results to the destination slice
		resultValue := reflect.ValueOf(results)
		sliceValue.Set(resultValue)
		return nil
	}

	// Handle struct scanning
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
		scanDest, err := prepareScanDestinations(elem, elementType, columns)
		if err != nil {
			return err
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

// ScanRow scans a single row into dest with smart field mapping
func ScanRow(rows *sql.Rows, dest any) error {
	if !rows.Next() {
		return sql.ErrNoRows
	}

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	destType := reflect.TypeOf(dest)
	if destType.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	// Handle simple types
	if IsSimpleType(destType.Elem()) {
		return rows.Scan(dest)
	}

	// Handle map types
	if destType.Elem().Kind() == reflect.Map &&
		destType.Elem().Key().Kind() == reflect.String &&
		destType.Elem().Elem().Kind() == reflect.Interface {

		// Scan into a single map
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Create the result map
		rowMap := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			// Handle byte arrays (convert to string)
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}

		// Set the map to the destination
		destValue := reflect.ValueOf(dest).Elem()
		destValue.Set(reflect.ValueOf(rowMap))
		return nil
	}

	// Handle struct types
	destValue := reflect.ValueOf(dest).Elem()
	destType = destValue.Type()

	scanDest, err := prepareScanDestinations(destValue, destType, columns)
	if err != nil {
		return err
	}

	return rows.Scan(scanDest...)
}

// ScanRowSimple scans a sql.Row into dest (for simple queries)
func ScanRowSimple(row *sql.Row, dest any) error {
	return row.Scan(dest)
}

// ScanRowContext handles scanning from QueryRowContext for both simple and complex types
func ScanRowContext(db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
},
	ctx context.Context, query string, args []any, dest any) error {

	destType := reflect.TypeOf(dest)
	if destType.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	// Handle simple types with QueryRow
	if IsSimpleType(destType.Elem()) {
		switch d := db.(type) {
		case *sql.DB:
			return d.QueryRowContext(ctx, query, args...).Scan(dest)
		case *sql.Tx:
			return d.QueryRowContext(ctx, query, args...).Scan(dest)
		}
	}

	// Handle complex types with smart scanning
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return ScanRow(rows, dest)
}

// prepareScanDestinations prepares scan destinations with smart field mapping
func prepareScanDestinations(destValue reflect.Value, destType reflect.Type, columns []string) ([]any, error) {
	scanDest := make([]any, len(columns))

	for i, col := range columns {
		field := destValue.FieldByNameFunc(func(name string) bool {
			field, _ := destType.FieldByName(name)

			// 1. Check struct tag for db column name
			if tag := field.Tag.Get("db"); tag != "" {
				return tag == col
			}

			// 2. Check json tag as fallback
			if tag := field.Tag.Get("json"); tag != "" {
				return tag == col
			}

			// 3. Case-insensitive match
			if strings.EqualFold(field.Name, col) {
				return true
			}

			// 4. Try camelCase/snake_case conversion
			return ToCamelCase(col) == name || ToSnakeCase(name) == col
		})

		if field.IsValid() && field.CanSet() {
			scanDest[i] = field.Addr().Interface()
		} else {
			// Field not found or can't be set, use a dummy scanner
			var dummy any
			scanDest[i] = &dummy
		}
	}

	return scanDest, nil
}

// ScanRowsToMaps scans SQL rows into a slice of maps
// This is useful for raw queries where the result structure is not known at compile time
func ScanRowsToMaps(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Prepare result slice
	var results []map[string]any

	// Create a slice of any to hold column values
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan all rows
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create map for this row
		rowMap := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			// Handle byte arrays (convert to string)
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// IsSimpleType checks if a type is a simple type (not struct, slice, etc.)
func IsSimpleType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	default:
		return false
	}
}
