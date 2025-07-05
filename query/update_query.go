package query

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// UpdateQueryImpl implements the UpdateQuery interface
type UpdateQueryImpl struct {
	*ModelQueryImpl
	setData         map[string]any
	atomicOps       map[string]AtomicOperation
	whereConditions []types.Condition
	returningFields []string
}

type AtomicOperation struct {
	Type  string // "increment", "decrement"
	Value int64
}

// NewUpdateQuery creates a new update query
func NewUpdateQuery(baseQuery *ModelQueryImpl, data any) *UpdateQueryImpl {
	updateQuery := &UpdateQueryImpl{
		ModelQueryImpl:  baseQuery,
		setData:         make(map[string]any),
		atomicOps:       make(map[string]AtomicOperation),
		whereConditions: []types.Condition{},
		returningFields: []string{},
	}

	if data != nil {
		// Set doesn't modify the original query, it returns a new one
		// So we need to copy the data directly instead of using Set
		switch v := data.(type) {
		case map[string]any:
			for field, value := range v {
				updateQuery.setData[field] = value
			}
		default:
			// Handle struct by extracting fields and values
			fields, values, err := updateQuery.extractFieldsAndValues(data)
			if err == nil {
				for i, field := range fields {
					updateQuery.setData[field] = values[i]
				}
			}
		}
	}

	return updateQuery
}

// Set sets the data to update
func (q *UpdateQueryImpl) Set(data any) types.UpdateQuery {
	newQuery := q.clone()

	// Extract fields and values from data
	switch v := data.(type) {
	case map[string]any:
		for field, value := range v {
			newQuery.setData[field] = value
		}
	default:
		// Handle struct
		fields, values, err := q.extractFieldsAndValues(data)
		if err == nil {
			for i, field := range fields {
				newQuery.setData[field] = values[i]
			}
		}
	}

	return newQuery
}

// Where adds a field condition to the update query
func (q *UpdateQueryImpl) Where(fieldName string) types.FieldCondition {
	return types.NewFieldCondition(q.modelName, fieldName)
}

// WhereCondition adds a condition to the update query
func (q *UpdateQueryImpl) WhereCondition(condition types.Condition) types.UpdateQuery {
	newQuery := q.clone()
	newQuery.whereConditions = append(newQuery.whereConditions, condition)
	return newQuery
}

// Returning sets fields to return after update
func (q *UpdateQueryImpl) Returning(fieldNames ...string) types.UpdateQuery {
	newQuery := q.clone()
	newQuery.returningFields = append(newQuery.returningFields, fieldNames...)
	return newQuery
}

// Increment adds an atomic increment operation
func (q *UpdateQueryImpl) Increment(fieldName string, value int64) types.UpdateQuery {
	newQuery := q.clone()
	newQuery.atomicOps[fieldName] = AtomicOperation{
		Type:  "increment",
		Value: value,
	}
	return newQuery
}

// Decrement adds an atomic decrement operation
func (q *UpdateQueryImpl) Decrement(fieldName string, value int64) types.UpdateQuery {
	newQuery := q.clone()
	newQuery.atomicOps[fieldName] = AtomicOperation{
		Type:  "decrement",
		Value: value,
	}
	return newQuery
}

// Exec executes the update query
func (q *UpdateQueryImpl) Exec(ctx context.Context) (types.Result, error) {
	sql, args, err := q.BuildSQL()
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build SQL: %w", err)
	}

	rawQuery := q.database.Raw(sql, args...)
	result, err := rawQuery.Exec(ctx)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute update: %w", err)
	}

	return result, nil
}

// ExecAndReturn executes the update and returns the updated data
func (q *UpdateQueryImpl) ExecAndReturn(ctx context.Context, dest any) error {
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

// BuildSQL builds the update SQL query
func (q *UpdateQueryImpl) BuildSQL() (string, []any, error) {
	if len(q.setData) == 0 && len(q.atomicOps) == 0 {
		return "", nil, fmt.Errorf("no data to update")
	}

	// Get table name
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve table name: %w", err)
	}

	var sql strings.Builder
	var args []any

	sql.WriteString(fmt.Sprintf("UPDATE %s SET ", tableName))

	// Build SET clause
	var setParts []string

	// Add regular set operations
	for fieldName, value := range q.setData {
		columnName, err := q.fieldMapper.SchemaToColumn(q.modelName, fieldName)
		if err != nil {
			return "", nil, fmt.Errorf("failed to map field %s: %w", fieldName, err)
		}
		quotedColumnName := q.database.GetCapabilities().QuoteIdentifier(columnName)
		setParts = append(setParts, fmt.Sprintf("%s = ?", quotedColumnName))
		args = append(args, value)
	}

	// Add atomic operations
	for fieldName, op := range q.atomicOps {
		columnName, err := q.fieldMapper.SchemaToColumn(q.modelName, fieldName)
		if err != nil {
			return "", nil, fmt.Errorf("failed to map field %s: %w", fieldName, err)
		}
		quotedColumnName := q.database.GetCapabilities().QuoteIdentifier(columnName)

		switch op.Type {
		case "increment":
			setParts = append(setParts, fmt.Sprintf("%s = %s + ?", quotedColumnName, quotedColumnName))
		case "decrement":
			setParts = append(setParts, fmt.Sprintf("%s = %s - ?", quotedColumnName, quotedColumnName))
		}
		args = append(args, op.Value)
	}

	sql.WriteString(strings.Join(setParts, ", "))

	// Build WHERE clause
	allConditions := append(q.conditions, q.whereConditions...)
	if len(allConditions) > 0 {
		whereClause, whereArgs, err := q.buildWhereClause(allConditions)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build WHERE clause: %w", err)
		}
		sql.WriteString(" " + whereClause)
		args = append(args, whereArgs...)
	}

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

// buildWhereClause builds the WHERE part of the query for update
func (q *UpdateQueryImpl) buildWhereClause(conditions []types.Condition) (string, []any, error) {
	if len(conditions) == 0 {
		return "", nil, nil
	}

	// Create condition context (no table alias for UPDATE)
	ctx := types.NewConditionContext(q.fieldMapper, q.modelName, "")
	ctx.QuoteIdentifier = q.database.GetCapabilities().QuoteIdentifier

	var conditionSQLs []string
	var args []any

	for _, condition := range conditions {
		sql, condArgs := condition.ToSQL(ctx)
		if sql != "" {
			conditionSQLs = append(conditionSQLs, sql)
			args = append(args, condArgs...)
		}
	}

	if len(conditionSQLs) == 0 {
		return "", nil, nil
	}

	whereSQL := fmt.Sprintf("WHERE %s", strings.Join(conditionSQLs, " AND "))
	return whereSQL, args, nil
}

// extractFieldsAndValues extracts field names and values from data
func (q *UpdateQueryImpl) extractFieldsAndValues(data any) ([]string, []any, error) {
	switch v := data.(type) {
	case map[string]any:
		return q.extractFromMap(v)
	default:
		return q.extractFromStruct(data)
	}
}

// extractFromMap extracts fields and values from a map
func (q *UpdateQueryImpl) extractFromMap(data map[string]any) ([]string, []any, error) {
	fields := make([]string, 0, len(data))
	values := make([]any, 0, len(data))

	for field, value := range data {
		fields = append(fields, field)
		values = append(values, value)
	}

	return fields, values, nil
}

// extractFromStruct extracts fields and values from a struct
func (q *UpdateQueryImpl) extractFromStruct(data any) ([]string, []any, error) {
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

// clone creates a copy of the update query
func (q *UpdateQueryImpl) clone() *UpdateQueryImpl {
	newSetData := make(map[string]any)
	for k, v := range q.setData {
		newSetData[k] = v
	}

	newAtomicOps := make(map[string]AtomicOperation)
	for k, v := range q.atomicOps {
		newAtomicOps[k] = v
	}

	return &UpdateQueryImpl{
		ModelQueryImpl:  q.ModelQueryImpl.clone(),
		setData:         newSetData,
		atomicOps:       newAtomicOps,
		whereConditions: append([]types.Condition{}, q.whereConditions...),
		returningFields: append([]string{}, q.returningFields...),
	}
}
