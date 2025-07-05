package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// AggregationQueryImpl implements the AggregationQuery interface
type AggregationQueryImpl struct {
	*ModelQueryImpl
	selectedFields     []string
	aggregations       []aggregation
	groupByFields      []string
	havingCondition    types.Condition
	orderByClauses     []OrderClause
	aggregationOrders  []aggregationOrder
}

// aggregation represents a single aggregation function
type aggregation struct {
	Type      string // "COUNT", "SUM", "AVG", "MIN", "MAX"
	FieldName string // Field to aggregate on (empty for COUNT(*))
	Alias     string // Alias for the result
}

// aggregationOrder represents ordering by an aggregation result
type aggregationOrder struct {
	Type      string // "COUNT", "SUM", "AVG", "MIN", "MAX"
	FieldName string
	Direction types.Order
}

// NewAggregationQuery creates a new aggregation query
func NewAggregationQuery(baseQuery *ModelQueryImpl) *AggregationQueryImpl {
	return &AggregationQueryImpl{
		ModelQueryImpl: baseQuery,
	}
}

// GroupBy adds grouping fields
func (q *AggregationQueryImpl) GroupBy(fieldNames ...string) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.groupByFields = append(newQuery.groupByFields, fieldNames...)
	return newQuery
}

// Having adds having condition
func (q *AggregationQueryImpl) Having(condition types.Condition) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.havingCondition = condition
	return newQuery
}

// Count adds a COUNT aggregation
func (q *AggregationQueryImpl) Count(fieldName string, alias string) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.aggregations = append(newQuery.aggregations, aggregation{
		Type:      "COUNT",
		FieldName: fieldName,
		Alias:     alias,
	})
	return newQuery
}

// CountAll adds a COUNT(*) aggregation
func (q *AggregationQueryImpl) CountAll(alias string) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.aggregations = append(newQuery.aggregations, aggregation{
		Type:      "COUNT",
		FieldName: "",
		Alias:     alias,
	})
	return newQuery
}

// Sum adds a SUM aggregation
func (q *AggregationQueryImpl) Sum(fieldName string, alias string) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.aggregations = append(newQuery.aggregations, aggregation{
		Type:      "SUM",
		FieldName: fieldName,
		Alias:     alias,
	})
	return newQuery
}

// Avg adds an AVG aggregation
func (q *AggregationQueryImpl) Avg(fieldName string, alias string) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.aggregations = append(newQuery.aggregations, aggregation{
		Type:      "AVG",
		FieldName: fieldName,
		Alias:     alias,
	})
	return newQuery
}

// Min adds a MIN aggregation
func (q *AggregationQueryImpl) Min(fieldName string, alias string) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.aggregations = append(newQuery.aggregations, aggregation{
		Type:      "MIN",
		FieldName: fieldName,
		Alias:     alias,
	})
	return newQuery
}

// Max adds a MAX aggregation
func (q *AggregationQueryImpl) Max(fieldName string, alias string) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.aggregations = append(newQuery.aggregations, aggregation{
		Type:      "MAX",
		FieldName: fieldName,
		Alias:     alias,
	})
	return newQuery
}

// Select adds grouped fields to select
func (q *AggregationQueryImpl) Select(fieldNames ...string) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.selectedFields = append(newQuery.selectedFields, fieldNames...)
	return newQuery
}

// Where adds a field condition
func (q *AggregationQueryImpl) Where(fieldName string) types.FieldCondition {
	return types.NewFieldCondition(q.modelName, fieldName)
}

// WhereCondition adds a condition
func (q *AggregationQueryImpl) WhereCondition(condition types.Condition) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.conditions = append(newQuery.conditions, condition)
	return newQuery
}

// OrderBy adds ordering by a field
func (q *AggregationQueryImpl) OrderBy(fieldName string, direction types.Order) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.orderByClauses = append(newQuery.orderByClauses, OrderClause{
		FieldName: fieldName,
		Direction: direction,
	})
	return newQuery
}

// OrderByAggregation adds ordering by an aggregation result
func (q *AggregationQueryImpl) OrderByAggregation(aggregationType string, fieldName string, direction types.Order) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.aggregationOrders = append(newQuery.aggregationOrders, aggregationOrder{
		Type:      aggregationType,
		FieldName: fieldName,
		Direction: direction,
	})
	return newQuery
}

// Limit sets the limit
func (q *AggregationQueryImpl) Limit(limit int) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.limit = &limit
	return newQuery
}

// Offset sets the offset
func (q *AggregationQueryImpl) Offset(offset int) types.AggregationQuery {
	newQuery := q.clone()
	newQuery.offset = &offset
	return newQuery
}

// Exec executes the aggregation query
func (q *AggregationQueryImpl) Exec(ctx context.Context, dest any) error {
	sql, args, err := q.BuildSQL()
	if err != nil {
		return fmt.Errorf("failed to build SQL: %w", err)
	}

	// Execute the query using the database
	rawQuery := q.database.Raw(sql, args...)
	return rawQuery.Find(ctx, dest)
}

// BuildSQL builds the SQL query
func (q *AggregationQueryImpl) BuildSQL() (string, []any, error) {
	// Get table name from model
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve table name: %w", err)
	}

	// Build SELECT clause
	selectParts := []string{}
	
	// Add selected fields (grouped fields)
	for _, field := range q.selectedFields {
		columnName, err := q.fieldMapper.SchemaToColumn(q.modelName, field)
		if err != nil {
			return "", nil, fmt.Errorf("failed to map field %s: %w", field, err)
		}
		selectParts = append(selectParts, columnName)
	}
	
	// Add aggregations
	for _, agg := range q.aggregations {
		var aggExpr string
		if agg.FieldName == "" {
			// COUNT(*)
			aggExpr = fmt.Sprintf("COUNT(*) AS %s", agg.Alias)
		} else {
			columnName, err := q.fieldMapper.SchemaToColumn(q.modelName, agg.FieldName)
			if err != nil {
				return "", nil, fmt.Errorf("failed to map field %s: %w", agg.FieldName, err)
			}
			aggExpr = fmt.Sprintf("%s(%s) AS %s", agg.Type, columnName, agg.Alias)
		}
		selectParts = append(selectParts, aggExpr)
	}
	
	if len(selectParts) == 0 {
		return "", nil, fmt.Errorf("no fields or aggregations specified")
	}
	
	selectClause := fmt.Sprintf("SELECT %s", strings.Join(selectParts, ", "))
	
	// Build FROM clause
	fromClause := fmt.Sprintf("FROM %s", tableName)
	
	// Build WHERE clause
	whereClause, args, err := q.buildWhereClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build WHERE clause: %w", err)
	}
	
	// Build GROUP BY clause
	groupByClause := ""
	if len(q.groupByFields) > 0 {
		columnNames, err := q.fieldMapper.SchemaFieldsToColumns(q.modelName, q.groupByFields)
		if err != nil {
			return "", nil, fmt.Errorf("failed to map group by fields: %w", err)
		}
		groupByClause = fmt.Sprintf("GROUP BY %s", strings.Join(columnNames, ", "))
	}
	
	// Build HAVING clause
	havingClause := ""
	var havingArgs []any
	if q.havingCondition != nil {
		// Create condition context without table alias for aggregations
		ctx := types.NewConditionContext(q.fieldMapper, q.modelName, "")
		ctx.QuoteIdentifier = q.database.QuoteIdentifier
		
		sql, args := q.havingCondition.ToSQL(ctx)
		if sql != "" {
			havingClause = fmt.Sprintf("HAVING %s", sql)
			havingArgs = args
		}
	}
	
	// Build ORDER BY clause
	orderByClause, err := q.buildOrderByClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build ORDER BY clause: %w", err)
	}
	
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
		args = append(args, havingArgs...)
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
	// Debug logging
	// fmt.Printf("[DEBUG AggregationQuery] SQL: %s\n", sql)
	// fmt.Printf("[DEBUG AggregationQuery] Args: %v\n", args)
	return sql, args, nil
}

// buildWhereClause builds the WHERE clause
func (q *AggregationQueryImpl) buildWhereClause() (string, []any, error) {
	if len(q.conditions) == 0 {
		return "", nil, nil
	}

	// Create condition context
	ctx := types.NewConditionContext(q.fieldMapper, q.modelName, "")
	ctx.QuoteIdentifier = q.database.QuoteIdentifier

	// Combine all conditions with AND
	var conditionSQLs []string
	var args []any

	for _, condition := range q.conditions {
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

// buildOrderByClause builds the ORDER BY clause
func (q *AggregationQueryImpl) buildOrderByClause() (string, error) {
	var orderParts []string
	
	// Add regular field ordering
	for _, order := range q.orderByClauses {
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
	
	// Add aggregation ordering
	for _, aggOrder := range q.aggregationOrders {
		var aggExpr string
		if aggOrder.FieldName == "" {
			// COUNT(*)
			aggExpr = "COUNT(*)"
		} else {
			columnName, err := q.fieldMapper.SchemaToColumn(q.modelName, aggOrder.FieldName)
			if err != nil {
				return "", fmt.Errorf("failed to map field name %s: %w", aggOrder.FieldName, err)
			}
			aggExpr = fmt.Sprintf("%s(%s)", aggOrder.Type, columnName)
		}
		
		direction := "ASC"
		if aggOrder.Direction == types.DESC {
			direction = "DESC"
		}
		
		orderParts = append(orderParts, fmt.Sprintf("%s %s", aggExpr, direction))
	}
	
	if len(orderParts) == 0 {
		return "", nil
	}
	
	return fmt.Sprintf("ORDER BY %s", strings.Join(orderParts, ", ")), nil
}

// buildLimitClause builds the LIMIT clause
func (q *AggregationQueryImpl) buildLimitClause() string {
	if q.limit == nil {
		return ""
	}
	return fmt.Sprintf("LIMIT %d", *q.limit)
}

// buildOffsetClause builds the OFFSET clause
func (q *AggregationQueryImpl) buildOffsetClause() string {
	if q.offset == nil {
		return ""
	}
	// Some databases require LIMIT when using OFFSET
	if q.limit == nil && q.database.RequiresLimitForOffset() {
		limit := int(^uint(0) >> 1) // Max int value
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, *q.offset)
	}
	return fmt.Sprintf("OFFSET %d", *q.offset)
}

// GetModelName returns the model name
func (q *AggregationQueryImpl) GetModelName() string {
	return q.modelName
}

// clone creates a copy of the aggregation query
func (q *AggregationQueryImpl) clone() *AggregationQueryImpl {
	newQuery := &AggregationQueryImpl{
		ModelQueryImpl:    q.ModelQueryImpl.clone(),
		selectedFields:    append([]string{}, q.selectedFields...),
		aggregations:      append([]aggregation{}, q.aggregations...),
		groupByFields:     append([]string{}, q.groupByFields...),
		havingCondition:   q.havingCondition,
		orderByClauses:    append([]OrderClause{}, q.orderByClauses...),
		aggregationOrders: append([]aggregationOrder{}, q.aggregationOrders...),
	}
	return newQuery
}