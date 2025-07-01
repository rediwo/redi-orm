package utils

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// ScanRows scans multiple rows into a slice with smart field mapping
func ScanRows(rows *sql.Rows, dest interface{}) error {
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
func ScanRow(rows *sql.Rows, dest interface{}) error {
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
func ScanRowSimple(row *sql.Row, dest interface{}) error {
	return row.Scan(dest)
}

// prepareScanDestinations prepares scan destinations with smart field mapping
func prepareScanDestinations(destValue reflect.Value, destType reflect.Type, columns []string) ([]interface{}, error) {
	scanDest := make([]interface{}, len(columns))
	
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
			var dummy interface{}
			scanDest[i] = &dummy
		}
	}
	
	return scanDest, nil
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