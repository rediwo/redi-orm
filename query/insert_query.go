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
	data            []interface{}
	conflictAction  types.ConflictAction
	returningFields []string
}

// NewInsertQuery creates a new insert query
func NewInsertQuery(baseQuery *ModelQueryImpl, data interface{}) *InsertQueryImpl {
	return &InsertQueryImpl{
		ModelQueryImpl:  baseQuery,
		data:            []interface{}{data},
		conflictAction:  types.ConflictIgnore,
		returningFields: []string{},
	}
}

// Values adds more data to insert
func (q *InsertQueryImpl) Values(data ...interface{}) types.InsertQuery {
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
func (q *InsertQueryImpl) ExecAndReturn(ctx context.Context, dest interface{}) error {
	if len(q.returningFields) == 0 {
		return fmt.Errorf("no returning fields specified")
	}

	sql, args, err := q.BuildSQL()
	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	rawQuery := q.database.Raw(sql, args...)
	return rawQuery.FindOne(ctx, dest)
}

// BuildSQL builds the insert SQL query
func (q *InsertQueryImpl) BuildSQL() (string, []interface{}, error) {
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

	// Map schema field names to column names
	columnNames, err := q.fieldMapper.SchemaFieldsToColumns(q.modelName, fields)
	if err != nil {
		return "", nil, fmt.Errorf("failed to map field names: %w", err)
	}

	// Build the basic INSERT statement
	var sql strings.Builder
	var args []interface{}

	sql.WriteString(fmt.Sprintf("INSERT INTO %s", tableName))

	// Handle conflict resolution
	switch q.conflictAction {
	case types.ConflictIgnore:
		// Different databases handle this differently
		// For now, use standard INSERT
	case types.ConflictReplace:
		sql.WriteString(" OR REPLACE")
	}

	// Add column names
	sql.WriteString(fmt.Sprintf(" (%s)", strings.Join(columnNames, ", ")))

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

	// Add RETURNING clause if specified
	if len(q.returningFields) > 0 {
		returningColumns, err := q.fieldMapper.SchemaFieldsToColumns(q.modelName, q.returningFields)
		if err != nil {
			return "", nil, fmt.Errorf("failed to map returning fields: %w", err)
		}
		sql.WriteString(fmt.Sprintf(" RETURNING %s", strings.Join(returningColumns, ", ")))
	}

	return sql.String(), args, nil
}

// extractFieldsAndValues extracts field names and values from data
func (q *InsertQueryImpl) extractFieldsAndValues(data interface{}) ([]string, []interface{}, error) {
	switch v := data.(type) {
	case map[string]interface{}:
		return q.extractFromMap(v)
	default:
		return q.extractFromStruct(data)
	}
}

// extractFromMap extracts fields and values from a map
func (q *InsertQueryImpl) extractFromMap(data map[string]interface{}) ([]string, []interface{}, error) {
	fields := make([]string, 0, len(data))
	values := make([]interface{}, 0, len(data))

	for field, value := range data {
		fields = append(fields, field)
		values = append(values, value)
	}

	return fields, values, nil
}

// extractFromStruct extracts fields and values from a struct
func (q *InsertQueryImpl) extractFromStruct(data interface{}) ([]string, []interface{}, error) {
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
	var values []interface{}

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
		data:            append([]interface{}{}, q.data...),
		conflictAction:  q.conflictAction,
		returningFields: append([]string{}, q.returningFields...),
	}
}
