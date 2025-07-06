package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// DeleteQueryImpl implements the DeleteQuery interface
type DeleteQueryImpl struct {
	*ModelQueryImpl
	whereConditions []types.Condition
	returningFields []string
}

// NewDeleteQuery creates a new delete query
func NewDeleteQuery(baseQuery *ModelQueryImpl) *DeleteQueryImpl {
	return &DeleteQueryImpl{
		ModelQueryImpl:  baseQuery,
		whereConditions: []types.Condition{},
		returningFields: []string{},
	}
}

// Where adds a field condition to the delete query
func (q *DeleteQueryImpl) Where(fieldName string) types.FieldCondition {
	return types.NewFieldCondition(q.modelName, fieldName)
}

// WhereCondition adds a condition to the delete query
func (q *DeleteQueryImpl) WhereCondition(condition types.Condition) types.DeleteQuery {
	newQuery := q.clone()
	newQuery.whereConditions = append(newQuery.whereConditions, condition)
	return newQuery
}

// Returning sets fields to return after delete
func (q *DeleteQueryImpl) Returning(fieldNames ...string) types.DeleteQuery {
	newQuery := q.clone()
	newQuery.returningFields = append(newQuery.returningFields, fieldNames...)
	return newQuery
}

// Exec executes the delete query
func (q *DeleteQueryImpl) Exec(ctx context.Context) (types.Result, error) {
	sql, args, err := q.BuildSQL()
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to build SQL: %w", err)
	}

	rawQuery := q.database.Raw(sql, args...)
	result, err := rawQuery.Exec(ctx)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute delete: %w", err)
	}

	return result, nil
}

// BuildSQL builds the delete SQL query
func (q *DeleteQueryImpl) BuildSQL() (string, []any, error) {
	// Get table name
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve table name: %w", err)
	}

	var sql strings.Builder
	var args []any

	sql.WriteString(fmt.Sprintf("DELETE FROM %s", tableName))

	// Build WHERE clause
	allConditions := append(q.conditions, q.whereConditions...)
	if len(allConditions) > 0 {
		whereClause, whereArgs, err := q.buildWhereClause(allConditions)
		if err != nil {
			return "", nil, fmt.Errorf("failed to build WHERE clause: %w", err)
		}
		sql.WriteString(" " + whereClause)
		args = append(args, whereArgs...)
	} else {
		// Safety check: prevent DELETE without WHERE clause
		return "", nil, fmt.Errorf("DELETE without WHERE clause is not allowed")
	}

	// Add RETURNING clause if specified (for databases that support it)
	if len(q.returningFields) > 0 {
		returningColumns, err := q.fieldMapper.SchemaFieldsToColumns(q.modelName, q.returningFields)
		if err != nil {
			return "", nil, fmt.Errorf("failed to map returning fields: %w", err)
		}
		sql.WriteString(fmt.Sprintf(" RETURNING %s", strings.Join(returningColumns, ", ")))
	}

	return sql.String(), args, nil
}

// buildWhereClause builds the WHERE part of the query for delete
func (q *DeleteQueryImpl) buildWhereClause(conditions []types.Condition) (string, []any, error) {
	if len(conditions) == 0 {
		return "", nil, nil
	}

	// Create condition context (no table alias for DELETE)
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

// GetWhereConditions returns the where conditions
func (q *DeleteQueryImpl) GetWhereConditions() []types.Condition {
	return q.whereConditions
}

// GetReturningFields returns the returning fields
func (q *DeleteQueryImpl) GetReturningFields() []string {
	return q.returningFields
}

// clone creates a copy of the delete query
func (q *DeleteQueryImpl) clone() *DeleteQueryImpl {
	return &DeleteQueryImpl{
		ModelQueryImpl:  q.ModelQueryImpl.clone(),
		whereConditions: append([]types.Condition{}, q.whereConditions...),
		returningFields: append([]string{}, q.returningFields...),
	}
}
