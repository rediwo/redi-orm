package query

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// InsertQueryImpl implements the InsertQuery interface
type InsertQueryImpl struct {
	*ModelQueryImpl
	data            []any
	conflictAction  types.ConflictAction
	returningFields []string
}

// NewInsertQuery creates a new insert query
func NewInsertQuery(baseQuery *ModelQueryImpl, data any) *InsertQueryImpl {
	return &InsertQueryImpl{
		ModelQueryImpl:  baseQuery,
		data:            []any{data},
		conflictAction:  types.ConflictIgnore,
		returningFields: []string{},
	}
}

// Values adds more data to insert
func (q *InsertQueryImpl) Values(data ...any) types.InsertQuery {
	newQuery := q.clone()
	newQuery.data = append(newQuery.data, data...)
	return newQuery
}

// OnConflict sets the conflict resolution action
func (q *InsertQueryImpl) OnConflict(action types.ConflictAction) types.InsertQuery {
	newQuery := q.clone()
	newQuery.conflictAction = action
	return newQuery
}

// Returning sets fields to return after insert
func (q *InsertQueryImpl) Returning(fieldNames ...string) types.InsertQuery {
	newQuery := q.clone()
	newQuery.returningFields = append(newQuery.returningFields, fieldNames...)
	return newQuery
}

// Exec executes the insert query
func (q *InsertQueryImpl) Exec(ctx context.Context) (types.Result, error) {
	sql, args, err := q.BuildSQL()
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build SQL: %w", err)
	}

	rawQuery := q.database.Raw(sql, args...)
	result, err := rawQuery.Exec(ctx)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute insert: %w", err)
	}

	return result, nil
}

// ExecAndReturn executes the insert and returns the inserted data
func (q *InsertQueryImpl) ExecAndReturn(ctx context.Context, dest any) error {
	if len(q.returningFields) == 0 {
		return fmt.Errorf("no returning fields specified")
	}

	// Check if database supports RETURNING
	if !q.database.GetCapabilities().SupportsReturning() {
		return fmt.Errorf("database does not support RETURNING clause")
	}

	sql, args, err := q.BuildSQL()
	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	rawQuery := q.database.Raw(sql, args...)
	return rawQuery.FindOne(ctx, dest)
}

// BuildSQL builds the insert SQL query
func (q *InsertQueryImpl) BuildSQL() (string, []any, error) {
	if len(q.data) == 0 {
		return "", nil, fmt.Errorf("no data to insert")
	}

	// Get table name
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve table name: %w", err)
	}

	// Extract fields and values from the first data item
	firstItem := q.data[0]
	fields, values, err := q.extractFieldsAndValues(firstItem)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract fields and values: %w", err)
	}

	// Build the basic INSERT statement
	var sql strings.Builder
	var args []any

	sql.WriteString(fmt.Sprintf("INSERT INTO %s", tableName))

	// Handle conflict resolution
	switch q.conflictAction {
	case types.ConflictIgnore:
		// Different databases handle this differently
		// For now, use standard INSERT
	case types.ConflictReplace:
		sql.WriteString(" OR REPLACE")
	}

	// Check if we have any fields to insert
	if len(fields) == 0 {
		// Handle empty insert based on database capabilities
		if q.database.GetCapabilities().SupportsDefaultValues() {
			// Use DEFAULT VALUES for databases that support it (PostgreSQL, SQLite)
			sql.WriteString(" DEFAULT VALUES")
		} else {
			// For databases like MySQL that don't support DEFAULT VALUES,
			// use the () VALUES () syntax for empty insert
			sql.WriteString(" () VALUES ()")
		}

		// RETURNING clause is still supported with empty insert if database supports it
		if len(q.returningFields) > 0 && q.database.GetCapabilities().SupportsReturning() {
			returningColumns, err := q.fieldMapper.SchemaFieldsToColumns(q.modelName, q.returningFields)
			if err != nil {
				return "", nil, fmt.Errorf("failed to map returning fields: %w", err)
			}
			quotedReturningColumns := make([]string, len(returningColumns))
			for i, columnName := range returningColumns {
				quotedReturningColumns[i] = q.database.GetCapabilities().QuoteIdentifier(columnName)
			}
			sql.WriteString(fmt.Sprintf(" RETURNING %s", strings.Join(quotedReturningColumns, ", ")))
		}

		return sql.String(), args, nil
	}

	// Map schema field names to column names
	columnNames, err := q.fieldMapper.SchemaFieldsToColumns(q.modelName, fields)
	if err != nil {
		return "", nil, fmt.Errorf("failed to map field names: %w", err)
	}

	// Quote column names to handle reserved keywords
	quotedColumnNames := make([]string, len(columnNames))
	for i, columnName := range columnNames {
		quotedColumnNames[i] = q.database.GetCapabilities().QuoteIdentifier(columnName)
	}

	// Add column names
	sql.WriteString(fmt.Sprintf(" (%s)", strings.Join(quotedColumnNames, ", ")))

	// Add VALUES clause
	sql.WriteString(" VALUES ")

	// Process all data items
	var valuePlaceholders []string
	for i, dataItem := range q.data {
		if i == 0 {
			// First item - we already have its values
			args = append(args, values...)
		} else {
			// Additional items - extract their values
			_, itemValues, err := q.extractFieldsAndValues(dataItem)
			if err != nil {
				return "", nil, fmt.Errorf("failed to extract values from item %d: %w", i, err)
			}
			args = append(args, itemValues...)
		}

		// Create placeholders for this row
		placeholders := make([]string, len(fields))
		for j := range placeholders {
			placeholders[j] = "?"
		}
		valuePlaceholders = append(valuePlaceholders, fmt.Sprintf("(%s)", strings.Join(placeholders, ", ")))
	}

	sql.WriteString(strings.Join(valuePlaceholders, ", "))

	// Add RETURNING clause if specified and supported
	if len(q.returningFields) > 0 && q.database.GetCapabilities().SupportsReturning() {
		returningColumns, err := q.fieldMapper.SchemaFieldsToColumns(q.modelName, q.returningFields)
		if err != nil {
			return "", nil, fmt.Errorf("failed to map returning fields: %w", err)
		}
		quotedReturningColumns := make([]string, len(returningColumns))
		for i, columnName := range returningColumns {
			quotedReturningColumns[i] = q.database.GetCapabilities().QuoteIdentifier(columnName)
		}
		sql.WriteString(fmt.Sprintf(" RETURNING %s", strings.Join(quotedReturningColumns, ", ")))
	}

	return sql.String(), args, nil
}

// extractFieldsAndValues extracts field names and values from data
func (q *InsertQueryImpl) extractFieldsAndValues(data any) ([]string, []any, error) {
	switch v := data.(type) {
	case map[string]any:
		return q.extractFromMap(v)
	default:
		return q.extractFromStruct(data)
	}
}

// extractFromMap extracts fields and values from a map
func (q *InsertQueryImpl) extractFromMap(data map[string]any) ([]string, []any, error) {
	fields := make([]string, 0, len(data))
	values := make([]any, 0, len(data))

	for field, value := range data {
		fields = append(fields, field)
		values = append(values, value)
	}

	return fields, values, nil
}

// extractFromStruct extracts fields and values from a struct
func (q *InsertQueryImpl) extractFromStruct(data any) ([]string, []any, error) {
	v := reflect.ValueOf(data)
	t := reflect.TypeOf(data)

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("data must be a struct or map")
	}

	var fields []string
	var values []any

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name (could use tags for mapping)
		fieldName := field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			// Use json tag as field name
			if commaIdx := strings.Index(tag, ","); commaIdx != -1 {
				fieldName = tag[:commaIdx]
			} else {
				fieldName = tag
			}
		}

		fields = append(fields, fieldName)
		values = append(values, value.Interface())
	}

	return fields, values, nil
}

// clone creates a copy of the insert query
func (q *InsertQueryImpl) clone() *InsertQueryImpl {
	return &InsertQueryImpl{
		ModelQueryImpl:  q.ModelQueryImpl.clone(),
		data:            append([]any{}, q.data...),
		conflictAction:  q.conflictAction,
		returningFields: append([]string{}, q.returningFields...),
	}
}
