package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// SelectQueryImpl implements the SelectQuery interface
type SelectQueryImpl struct {
	*ModelQueryImpl
	selectedFields []string
	distinct       bool
	joinBuilder    *JoinBuilder
}

// NewSelectQuery creates a new select query
func NewSelectQuery(baseQuery *ModelQueryImpl, fields []string) *SelectQueryImpl {
	return &SelectQueryImpl{
		ModelQueryImpl: baseQuery,
		selectedFields: fields,
		distinct:       false,
		joinBuilder:    NewJoinBuilder(baseQuery.database),
	}
}

// Where adds a field condition to the select query
func (q *SelectQueryImpl) Where(fieldName string) types.FieldCondition {
	return types.NewFieldCondition(q.modelName, fieldName)
}

// WhereCondition adds a condition to the select query
func (q *SelectQueryImpl) WhereCondition(condition types.Condition) types.SelectQuery {
	newQuery := q.clone()
	newQuery.conditions = append(newQuery.conditions, condition)
	return newQuery
}

// Include adds relations to include
func (q *SelectQueryImpl) Include(relations ...string) types.SelectQuery {
	newQuery := q.clone()
	newQuery.includes = append(newQuery.includes, relations...)

	// Add joins for each included relation
	for _, relation := range relations {
		// Parse nested relations (e.g., "posts.comments")
		relationPath := strings.Split(relation, ".")

		// Add join for this relation path
		err := newQuery.joinBuilder.AddNestedRelationJoin(
			newQuery.tableAlias, // Use the main table alias
			newQuery.modelName,
			relationPath,
			LeftJoin, // Use LEFT JOIN to include records without relations
		)
		if err != nil {
			// Log error but continue - we might handle this differently in production
			fmt.Printf("Warning: failed to add join for relation %s: %v\n", relation, err)
		}
	}

	return newQuery
}

// OrderBy adds ordering
func (q *SelectQueryImpl) OrderBy(fieldName string, direction types.Order) types.SelectQuery {
	newQuery := q.clone()
	newQuery.orderBy = append(newQuery.orderBy, OrderClause{
		FieldName: fieldName,
		Direction: direction,
	})
	return newQuery
}

// GroupBy adds grouping
func (q *SelectQueryImpl) GroupBy(fieldNames ...string) types.SelectQuery {
	newQuery := q.clone()
	newQuery.groupBy = append(newQuery.groupBy, fieldNames...)
	return newQuery
}

// Having adds having condition
func (q *SelectQueryImpl) Having(condition types.Condition) types.SelectQuery {
	newQuery := q.clone()
	newQuery.having = condition
	return newQuery
}

// Limit sets the limit
func (q *SelectQueryImpl) Limit(limit int) types.SelectQuery {
	newQuery := q.clone()
	newQuery.limit = &limit
	return newQuery
}

// Offset sets the offset
func (q *SelectQueryImpl) Offset(offset int) types.SelectQuery {
	newQuery := q.clone()
	newQuery.offset = &offset
	return newQuery
}

// Distinct enables distinct selection
func (q *SelectQueryImpl) Distinct() types.SelectQuery {
	newQuery := q.clone()
	newQuery.distinct = true
	return newQuery
}

// FindMany executes the query and returns multiple results
func (q *SelectQueryImpl) FindMany(ctx context.Context, dest any) error {
	sql, args, err := q.BuildSQL()
	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	// Execute the query using the database
	rawQuery := q.database.Raw(sql, args...)
	return rawQuery.Find(ctx, dest)
}

// FindFirst executes the query and returns the first result
func (q *SelectQueryImpl) FindFirst(ctx context.Context, dest any) error {
	// Ensure limit is 1 for FindFirst
	limitedQuery := q.Limit(1)
	sql, args, err := limitedQuery.BuildSQL()
	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	// Execute the query using the database
	rawQuery := q.database.Raw(sql, args...)
	return rawQuery.FindOne(ctx, dest)
}

// Count returns the count of matching records
func (q *SelectQueryImpl) Count(ctx context.Context) (int64, error) {
	// Create a count query
	countSQL, args, err := q.buildCountSQL()
	if err != nil {
		return 0, fmt.Errorf("failed to build count SQL: %w", err)
	}

	// Execute count query
	var count int64
	rawQuery := q.database.Raw(countSQL, args...)
	err = rawQuery.FindOne(ctx, &count)
	if err != nil {
		return 0, fmt.Errorf("failed to execute count query: %w", err)
	}

	return count, nil
}

// BuildSQL builds the SQL query
func (q *SelectQueryImpl) BuildSQL() (string, []any, error) {
	// Get table name from model
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve table name: %w", err)
	}

	// Build SELECT clause (with table alias support)
	selectClause := q.buildSelectClause(tableName)

	// Build FROM clause with alias
	fromClause := fmt.Sprintf("FROM %s AS %s", tableName, q.tableAlias)

	// Add JOINs if any
	if q.joinBuilder != nil {
		joinSQL := q.joinBuilder.BuildSQL()
		if joinSQL != "" {
			fromClause += " " + joinSQL
		}
	}

	// Build WHERE clause
	whereClause, args, err := q.buildWhereClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build WHERE clause: %w", err)
	}

	// Build ORDER BY clause
	orderByClause, err := q.buildOrderByClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build ORDER BY clause: %w", err)
	}

	// Build GROUP BY clause
	groupByClause, err := q.buildGroupByClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build GROUP BY clause: %w", err)
	}

	// Build HAVING clause
	havingClause, havingArgs, err := q.buildHavingClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build HAVING clause: %w", err)
	}
	args = append(args, havingArgs...)

	// Build LIMIT and OFFSET clauses
	limitClause := q.buildLimitClause()
	offsetClause := q.buildOffsetClause()

	// Combine all parts
	sqlParts := []string{selectClause, fromClause}

	if whereClause != "" {
		sqlParts = append(sqlParts, whereClause)
	}
	if groupByClause != "" {
		sqlParts = append(sqlParts, groupByClause)
	}
	if havingClause != "" {
		sqlParts = append(sqlParts, havingClause)
	}
	if orderByClause != "" {
		sqlParts = append(sqlParts, orderByClause)
	}
	if limitClause != "" {
		sqlParts = append(sqlParts, limitClause)
	}
	if offsetClause != "" {
		sqlParts = append(sqlParts, offsetClause)
	}

	sql := strings.Join(sqlParts, " ")
	return sql, args, nil
}

// buildSelectClause builds the SELECT part of the query
func (q *SelectQueryImpl) buildSelectClause(tableName string) string {
	distinctStr := ""
	if q.distinct {
		distinctStr = "DISTINCT "
	}

	// If no specific fields selected, select all from main table and joined tables
	if len(q.selectedFields) == 0 {
		selectParts := []string{fmt.Sprintf("%s.*", q.tableAlias)}

		// Add fields from joined tables if includes are specified
		if q.joinBuilder != nil {
			for _, join := range q.joinBuilder.GetJoinedTables() {
				if join.Schema != nil {
					selectParts = append(selectParts, fmt.Sprintf("%s.*", join.Alias))
				}
			}
		}

		return fmt.Sprintf("SELECT %s%s", distinctStr, strings.Join(selectParts, ", "))
	}

	// Map schema field names to column names with table aliases
	columnNames := make([]string, len(q.selectedFields))
	for i, fieldName := range q.selectedFields {
		// Check if field includes table prefix (e.g., "posts.title")
		if strings.Contains(fieldName, ".") {
			columnNames[i] = fieldName // Use as-is
		} else {
			columnName, err := q.fieldMapper.SchemaToColumn(q.modelName, fieldName)
			if err != nil {
				// If mapping fails, use the original field name
				columnName = fieldName
			}
			// Add table alias
			columnNames[i] = fmt.Sprintf("%s.%s", q.tableAlias, columnName)
		}
	}

	return fmt.Sprintf("SELECT %s%s", distinctStr, strings.Join(columnNames, ", "))
}

// buildWhereClause builds the WHERE part of the query
func (q *SelectQueryImpl) buildWhereClause() (string, []any, error) {
	if len(q.conditions) == 0 {
		return "", nil, nil
	}

	// Combine all conditions with AND
	var conditionSQLs []string
	var args []any

	for _, condition := range q.conditions {
		sql, condArgs := condition.ToSQL()
		if sql != "" {
			// Map field names in the condition SQL
			mappedSQL, err := q.mapFieldNamesInSQL(sql)
			if err != nil {
				return "", nil, fmt.Errorf("failed to map field names in condition: %w", err)
			}
			conditionSQLs = append(conditionSQLs, mappedSQL)
			args = append(args, condArgs...)
		}
	}

	if len(conditionSQLs) == 0 {
		return "", nil, nil
	}

	whereSQL := fmt.Sprintf("WHERE %s", strings.Join(conditionSQLs, " AND "))
	return whereSQL, args, nil
}

// buildOrderByClause builds the ORDER BY part of the query
func (q *SelectQueryImpl) buildOrderByClause() (string, error) {
	if len(q.orderBy) == 0 {
		return "", nil
	}

	var orderParts []string
	for _, order := range q.orderBy {
		columnName, err := q.fieldMapper.SchemaToColumn(q.modelName, order.FieldName)
		if err != nil {
			return "", fmt.Errorf("failed to map field name %s: %w", order.FieldName, err)
		}

		direction := "ASC"
		if order.Direction == types.DESC {
			direction = "DESC"
		}

		orderParts = append(orderParts, fmt.Sprintf("%s %s", columnName, direction))
	}

	return fmt.Sprintf("ORDER BY %s", strings.Join(orderParts, ", ")), nil
}

// buildGroupByClause builds the GROUP BY part of the query
func (q *SelectQueryImpl) buildGroupByClause() (string, error) {
	if len(q.groupBy) == 0 {
		return "", nil
	}

	columnNames, err := q.fieldMapper.SchemaFieldsToColumns(q.modelName, q.groupBy)
	if err != nil {
		return "", fmt.Errorf("failed to map group by fields: %w", err)
	}

	return fmt.Sprintf("GROUP BY %s", strings.Join(columnNames, ", ")), nil
}

// buildHavingClause builds the HAVING part of the query
func (q *SelectQueryImpl) buildHavingClause() (string, []any, error) {
	if q.having == nil {
		return "", nil, nil
	}

	sql, args := q.having.ToSQL()
	if sql == "" {
		return "", nil, nil
	}

	mappedSQL, err := q.mapFieldNamesInSQL(sql)
	if err != nil {
		return "", nil, fmt.Errorf("failed to map field names in having clause: %w", err)
	}

	return fmt.Sprintf("HAVING %s", mappedSQL), args, nil
}

// buildLimitClause builds the LIMIT part of the query
func (q *SelectQueryImpl) buildLimitClause() string {
	if q.limit == nil {
		return ""
	}
	return fmt.Sprintf("LIMIT %d", *q.limit)
}

// buildOffsetClause builds the OFFSET part of the query
func (q *SelectQueryImpl) buildOffsetClause() string {
	if q.offset == nil {
		return ""
	}
	return fmt.Sprintf("OFFSET %d", *q.offset)
}

// buildCountSQL builds a count query
func (q *SelectQueryImpl) buildCountSQL() (string, []any, error) {
	// Get table name from model
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve table name: %w", err)
	}

	// Build WHERE clause
	whereClause, args, err := q.buildWhereClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build WHERE clause: %w", err)
	}

	// Build GROUP BY clause for count
	groupByClause, err := q.buildGroupByClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build GROUP BY clause: %w", err)
	}

	// Build HAVING clause
	havingClause, havingArgs, err := q.buildHavingClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build HAVING clause: %w", err)
	}
	args = append(args, havingArgs...)

	// Build count query
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)

	if whereClause != "" {
		countSQL += " " + whereClause
	}
	if groupByClause != "" {
		countSQL += " " + groupByClause
	}
	if havingClause != "" {
		countSQL += " " + havingClause
	}

	return countSQL, args, nil
}

// mapFieldNamesInSQL maps schema field names to column names in SQL
func (q *SelectQueryImpl) mapFieldNamesInSQL(sql string) (string, error) {
	// Get the schema to find all field mappings
	schema, err := q.database.GetSchema(q.modelName)
	if err != nil {
		// If we can't get the schema, return SQL as-is
		return sql, nil
	}

	// Replace field names with column names
	result := sql
	for _, field := range schema.Fields {
		fieldName := field.Name
		columnName := field.GetColumnName()

		// Only replace if they're different
		if fieldName != columnName {
			// Replace field name with column name
			// We need to be careful to match word boundaries
			// This is a simplified implementation - in production you'd use a proper SQL parser
			result = strings.ReplaceAll(result, fieldName+" ", columnName+" ")
			result = strings.ReplaceAll(result, fieldName+"=", columnName+"=")
			result = strings.ReplaceAll(result, fieldName+"<", columnName+"<")
			result = strings.ReplaceAll(result, fieldName+">", columnName+">")
			result = strings.ReplaceAll(result, fieldName+"!", columnName+"!")
			result = strings.ReplaceAll(result, fieldName+" IN", columnName+" IN")
			result = strings.ReplaceAll(result, fieldName+" NOT", columnName+" NOT")
			result = strings.ReplaceAll(result, fieldName+" LIKE", columnName+" LIKE")
			result = strings.ReplaceAll(result, fieldName+" BETWEEN", columnName+" BETWEEN")
			result = strings.ReplaceAll(result, fieldName+" IS", columnName+" IS")
		}
	}

	return result, nil
}

// clone creates a copy of the select query
func (q *SelectQueryImpl) clone() *SelectQueryImpl {
	newQuery := &SelectQueryImpl{
		ModelQueryImpl: q.ModelQueryImpl.clone(),
		selectedFields: append([]string{}, q.selectedFields...),
		distinct:       q.distinct,
		joinBuilder:    NewJoinBuilder(q.database),
	}

	// Copy existing joins if any
	if q.joinBuilder != nil && len(q.joinBuilder.joins) > 0 {
		// For now, create a new joinBuilder
		// In production, we'd want to properly clone the joins
		newQuery.joinBuilder = q.joinBuilder
	}

	return newQuery
}
