package query

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/rediwo/redi-orm/schema"
)

// RelationScanner handles scanning results from queries with joins
type RelationScanner struct {
	mainSchema       *schema.Schema
	mainAlias        string                      // alias of the main table
	joinedSchemas    map[string]*schema.Schema   // alias -> schema
	relations        map[string]*schema.Relation // alias -> relation
	relationNames    map[string]string           // alias -> relation field name
	includeProcessor *IncludeProcessor           // processor for include options
}

// NewRelationScanner creates a new relation scanner
func NewRelationScanner(mainSchema *schema.Schema, mainAlias string) *RelationScanner {
	return &RelationScanner{
		mainSchema:    mainSchema,
		mainAlias:     mainAlias,
		joinedSchemas: make(map[string]*schema.Schema),
		relations:     make(map[string]*schema.Relation),
		relationNames: make(map[string]string),
	}
}

// SetIncludeProcessor sets the include processor for filtering and ordering
func (rs *RelationScanner) SetIncludeProcessor(processor *IncludeProcessor) {
	rs.includeProcessor = processor
}

// AddJoinedTable adds information about a joined table
func (rs *RelationScanner) AddJoinedTable(alias string, schema *schema.Schema, relation *schema.Relation, relationName string) {
	rs.joinedSchemas[alias] = schema
	rs.relations[alias] = relation
	rs.relationNames[alias] = relationName
}

// ScanRowsWithRelations scans rows that include joined data
func (rs *RelationScanner) ScanRowsWithRelations(rows *sql.Rows, dest any) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	sliceValue := destValue.Elem()
	elementType := sliceValue.Type().Elem()

	// Check if destination is []map[string]any
	if elementType.Kind() == reflect.Map &&
		elementType.Key().Kind() == reflect.String &&
		elementType.Elem().Kind() == reflect.Interface &&
		elementType.Elem().NumMethod() == 0 {
		return rs.scanRowsToMapsWithRelations(rows, dest)
	}

	// For struct scanning, we'd need more complex logic
	// For now, return an error
	return fmt.Errorf("struct scanning with relations not yet implemented")
}

// scanRowsToMapsWithRelations scans rows with joins into maps
func (rs *RelationScanner) scanRowsToMapsWithRelations(rows *sql.Rows, dest any) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Parse column names to determine which table they belong to
	columnInfo := rs.parseColumns(columns)

	// Track main records by ID to handle one-to-many relations
	mainRecords := make(map[any]map[string]any)
	var recordOrder []any // Maintain order

	// Create value holders
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Scan all rows
	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Build record maps for each table
		recordMaps := make(map[string]map[string]any)

		for i, col := range columns {
			val := values[i]
			// Handle byte arrays
			if b, ok := val.([]byte); ok {
				val = string(b)
			}

			// Determine which table this column belongs to
			tableAlias, fieldName := rs.parseColumnName(col, columnInfo)

			if _, exists := recordMaps[tableAlias]; !exists {
				recordMaps[tableAlias] = make(map[string]any)
			}

			// Map column back to schema field name
			if tableAlias == rs.mainAlias {
				if rs.mainSchema != nil {
					if mapped, err := rs.mainSchema.GetFieldNameByColumnName(fieldName); err == nil {
						fieldName = mapped
					}
				}
			} else if schema, exists := rs.joinedSchemas[tableAlias]; exists {
				if mapped, err := schema.GetFieldNameByColumnName(fieldName); err == nil {
					fieldName = mapped
				}
			}

			recordMaps[tableAlias][fieldName] = val
		}

		// Get main record
		mainRecord := recordMaps[rs.mainAlias]
		if mainRecord == nil {
			// Skip this row if no main record
			continue
		}
		mainID := mainRecord["id"] // Assume "id" field exists

		// Check if we've seen this main record before (for one-to-many)
		if existingMain, exists := mainRecords[mainID]; exists {
			mainRecord = existingMain
		} else {
			mainRecords[mainID] = mainRecord
			recordOrder = append(recordOrder, mainID)

			// Initialize relation fields as empty slices/maps
			for alias, relation := range rs.relations {
				relationName := rs.relationNames[alias]
				if relation.Type == schema.RelationOneToMany {
					mainRecord[relationName] = []any{}
				}
			}
		}

		// Add related records
		for alias, relatedRecord := range recordMaps {
			if alias == rs.mainAlias {
				continue
			}

			relation, exists := rs.relations[alias]
			if !exists {
				continue
			}

			// Skip if related record is null (LEFT JOIN with no match)
			if isNullRecord(relatedRecord) {
				continue
			}

			relationName := rs.relationNames[alias]

			switch relation.Type {
			case schema.RelationOneToMany:
				// Append to array
				if arr, ok := mainRecord[relationName].([]any); ok {
					mainRecord[relationName] = append(arr, relatedRecord)
				}

			case schema.RelationManyToOne, schema.RelationOneToOne:
				// Set single value
				mainRecord[relationName] = relatedRecord
			}
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// Apply include processing to relation data if processor is set
	if rs.includeProcessor != nil {
		for _, mainRecord := range mainRecords {
			for alias, relation := range rs.relations {
				relationName := rs.relationNames[alias]

				// Process one-to-many relations
				if relation.Type == schema.RelationOneToMany {
					if arr, ok := mainRecord[relationName].([]any); ok && len(arr) > 0 {
						// Convert to map array for processing
						mapArray := make([]map[string]any, len(arr))
						for i, item := range arr {
							if m, ok := item.(map[string]any); ok {
								mapArray[i] = m
							}
						}

						// Process with include options
						mapArray = rs.includeProcessor.ProcessRelationData(relationName, mapArray)

						// Convert back to any array
						anyArray := make([]any, len(mapArray))
						for i, m := range mapArray {
							anyArray[i] = m
						}
						mainRecord[relationName] = anyArray
					}
				}
			}
		}
	}

	// Build final result array in original order
	results := make([]map[string]any, 0, len(recordOrder))
	for _, id := range recordOrder {
		results = append(results, mainRecords[id])
	}

	// Set results to destination
	destValue := reflect.ValueOf(dest).Elem()
	destValue.Set(reflect.ValueOf(results))

	return nil
}

// parseColumns analyzes column names to determine table aliases
func (rs *RelationScanner) parseColumns(columns []string) map[string]string {
	info := make(map[string]string)

	for _, col := range columns {
		// Try to parse table_column format (from aliased columns)
		if parts := strings.Split(col, "_"); len(parts) >= 2 {
			// First part is table alias
			info[col] = parts[0]
		} else if parts := strings.Split(col, "."); len(parts) == 2 {
			// Try to parse table.column format
			info[col] = parts[0]
		} else {
			// Assume main table if no prefix
			info[col] = rs.mainAlias
		}
	}

	return info
}

// parseColumnName extracts table alias and field name from a column
func (rs *RelationScanner) parseColumnName(column string, columnInfo map[string]string) (tableAlias, fieldName string) {
	if alias, exists := columnInfo[column]; exists {
		tableAlias = alias
	} else {
		tableAlias = rs.mainAlias
	}

	// Remove table prefix if present
	if parts := strings.Split(column, "_"); len(parts) >= 2 && parts[0] == tableAlias {
		// Handle aliased format: tableAlias_columnName
		fieldName = strings.Join(parts[1:], "_")
	} else if parts := strings.Split(column, "."); len(parts) == 2 {
		// Handle dot format: table.column
		fieldName = parts[1]
	} else {
		fieldName = column
	}

	return tableAlias, fieldName
}

// isNullRecord checks if all fields in a record are null
func isNullRecord(record map[string]any) bool {
	for _, val := range record {
		if val != nil {
			return false
		}
	}
	return true
}
