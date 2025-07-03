package query

import (
	"context"
	"fmt"
	"reflect"
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

	// Check if we have includes - if so, we need to use relation scanner
	if len(q.includes) > 0 && q.joinBuilder != nil && len(q.joinBuilder.GetJoinedTables()) > 0 {
		return q.findManyWithRelations(ctx, sql, args, dest)
	}

	// Check if dest is []map[string]any - if so, we need field mapping
	destType := reflect.TypeOf(dest)
	if destType.Kind() == reflect.Ptr && destType.Elem().Kind() == reflect.Slice {
		elemType := destType.Elem().Elem()
		if elemType.Kind() == reflect.Map &&
			elemType.Key().Kind() == reflect.String &&
			elemType.Elem().Kind() == reflect.Interface {
			// Scanning into []map[string]any, use field mapping
			return q.findManyMapsWithFieldMapping(ctx, sql, args, dest)
		}
	}

	// Execute the query using the database
	rawQuery := q.database.Raw(sql, args...)
	return rawQuery.Find(ctx, dest)
}

// FindFirst executes the query and returns the first result
func (q *SelectQueryImpl) FindFirst(ctx context.Context, dest any) error {
	// Don't add LIMIT 1 if we have includes, as we might need multiple rows
	var sql string
	var args []any
	var err error
	
	if len(q.includes) > 0 && q.joinBuilder != nil && len(q.joinBuilder.GetJoinedTables()) > 0 {
		// For includes, we need to get all rows for the first main record
		// We'll add a subquery or handle it differently
		sql, args, err = q.BuildSQL()
		if err != nil {
			return fmt.Errorf("failed to build SQL: %w", err)
		}
		
		// Use a slice to collect results
		destType := reflect.TypeOf(dest)
		if destType.Kind() != reflect.Ptr {
			return fmt.Errorf("dest must be a pointer")
		}
		
		// Create a slice of the same type
		elemType := destType.Elem()
		sliceType := reflect.SliceOf(elemType)
		slicePtr := reflect.New(sliceType)
		
		// Find all results
		err = q.findManyWithRelations(ctx, sql, args, slicePtr.Interface())
		if err != nil {
			return err
		}
		
		// Get the first result if any
		sliceValue := slicePtr.Elem()
		if sliceValue.Len() == 0 {
			return fmt.Errorf("no records found")
		}
		
		// Set the first element to dest
		reflect.ValueOf(dest).Elem().Set(sliceValue.Index(0))
		return nil
	} else {
		// Normal case without includes
		limitedQuery := q.Limit(1)
		sql, args, err = limitedQuery.BuildSQL()
		if err != nil {
			return fmt.Errorf("failed to build SQL: %w", err)
		}
		
		// Check if dest is map[string]any - if so, we need field mapping
		destType := reflect.TypeOf(dest)
		if destType.Kind() == reflect.Ptr && destType.Elem().Kind() == reflect.Map &&
			destType.Elem().Key().Kind() == reflect.String &&
			destType.Elem().Elem().Kind() == reflect.Interface {
			// Scanning into map[string]any, use field mapping
			return q.Limit(1).(*SelectQueryImpl).findOneMapsWithFieldMapping(ctx, sql, args, dest)
		}
		
		// Execute the query using the database
		rawQuery := q.database.Raw(sql, args...)
		return rawQuery.FindOne(ctx, dest)
	}
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
	selectClause := q.buildSelectClause()

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
func (q *SelectQueryImpl) buildSelectClause() string {
	distinctStr := ""
	if q.distinct {
		distinctStr = "DISTINCT "
	}

	// If no specific fields selected, select all from main table and joined tables
	if len(q.selectedFields) == 0 {
		// Check if we have joins - if so, we need to be more careful about column naming
		if q.joinBuilder != nil && len(q.joinBuilder.GetJoinedTables()) > 0 {
			// Build explicit column list to avoid ambiguity
			selectParts := []string{}
			
			// Get main table schema
			mainSchema, err := q.database.GetModelSchema(q.modelName)
			if err == nil {
				// Add main table columns with aliases
				for _, field := range mainSchema.Fields {
					columnName := field.GetColumnName()
					// Alias format: tableAlias.column AS tableAlias_column
					selectParts = append(selectParts, fmt.Sprintf("%s.%s AS %s_%s", 
						q.tableAlias, columnName, q.tableAlias, columnName))
				}
			} else {
				// Fallback to wildcard if schema not available
				selectParts = append(selectParts, fmt.Sprintf("%s.*", q.tableAlias))
			}

			// Add columns from joined tables
			for _, join := range q.joinBuilder.GetJoinedTables() {
				if join.Schema != nil {
					for _, field := range join.Schema.Fields {
						columnName := field.GetColumnName()
						// Alias format: joinAlias.column AS joinAlias_column
						selectParts = append(selectParts, fmt.Sprintf("%s.%s AS %s_%s", 
							join.Alias, columnName, join.Alias, columnName))
					}
				} else {
					// Fallback to wildcard if schema not available
					selectParts = append(selectParts, fmt.Sprintf("%s.*", join.Alias))
				}
			}
			
			return fmt.Sprintf("SELECT %s%s", distinctStr, strings.Join(selectParts, ", "))
		} else {
			// No joins, simple case
			return fmt.Sprintf("SELECT %s%s.*", distinctStr, q.tableAlias)
		}
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

	// Create condition context
	ctx := types.NewConditionContext(q.fieldMapper, q.modelName, q.tableAlias)

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
		nullsClause := ""
		if order.Direction == types.DESC {
			direction = "DESC"
			// PostgreSQL sorts NULL values first in DESC order by default
			// We want consistent behavior across all databases, so add NULLS LAST for PostgreSQL
			if q.database.GetDriverType() == "postgresql" {
				nullsClause = " NULLS LAST"
			}
		} else {
			// For ASC order, PostgreSQL sorts NULL values last by default, which is what we want
			// No need to add NULLS clause for ASC
		}

		// Add table alias if present to avoid ambiguity
		fullColumnName := columnName
		if q.tableAlias != "" {
			fullColumnName = fmt.Sprintf("%s.%s", q.tableAlias, columnName)
		}

		orderParts = append(orderParts, fmt.Sprintf("%s %s%s", fullColumnName, direction, nullsClause))
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

	// Create condition context
	ctx := types.NewConditionContext(q.fieldMapper, q.modelName, q.tableAlias)

	sql, args := q.having.ToSQL(ctx)
	if sql == "" {
		return "", nil, nil
	}

	return fmt.Sprintf("HAVING %s", sql), args, nil
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
	// SQLite requires LIMIT when using OFFSET
	// If no limit is set, use a very large number
	if q.limit == nil {
		limit := int(^uint(0) >> 1) // Max int value
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, *q.offset)
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

	// Create a temporary query without table alias for count
	countQuery := q.clone()
	countQuery.tableAlias = "" // Remove table alias for count queries

	// Build WHERE clause without table alias
	whereClause, args, err := countQuery.buildWhereClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build WHERE clause: %w", err)
	}

	// Build GROUP BY clause for count
	groupByClause, err := q.buildGroupByClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build GROUP BY clause: %w", err)
	}

	// Build HAVING clause without table alias
	havingClause, havingArgs, err := countQuery.buildHavingClause()
	if err != nil {
		return "", nil, fmt.Errorf("failed to build HAVING clause: %w", err)
	}
	args = append(args, havingArgs...)

	// Build count query
	var countExpr string
	if q.distinct && len(q.selectedFields) > 0 {
		// Handle DISTINCT with specific fields
		var distinctCols []string
		for _, field := range q.selectedFields {
			col, err := q.fieldMapper.SchemaToColumn(q.modelName, field)
			if err != nil {
				return "", nil, fmt.Errorf("failed to map field %s: %w", field, err)
			}
			distinctCols = append(distinctCols, col)
		}
		countExpr = fmt.Sprintf("COUNT(DISTINCT %s)", strings.Join(distinctCols, ", "))
	} else if q.distinct {
		// DISTINCT without specific fields - count distinct rows
		countExpr = "COUNT(DISTINCT *)"
	} else {
		// Regular count
		countExpr = "COUNT(*)"
	}
	
	countSQL := fmt.Sprintf("SELECT %s FROM %s", countExpr, tableName)

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


// findManyWithRelations executes the query and scans results with relation support
func (q *SelectQueryImpl) findManyWithRelations(_ context.Context, sql string, args []any, dest any) error {
	// Execute query using database's Query method
	rows, err := q.database.Query(sql, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Create relation scanner
	mainSchema, err := q.database.GetSchema(q.modelName)
	if err != nil {
		return fmt.Errorf("failed to get schema for model %s: %w", q.modelName, err)
	}

	scanner := NewRelationScanner(mainSchema, q.tableAlias)

	// Add joined table information to scanner
	for _, join := range q.joinBuilder.GetJoinedTables() {
		if join.Schema != nil && join.Relation != nil {
			scanner.AddJoinedTable(join.Alias, join.Schema, join.Relation, join.RelationName)
		}
	}

	// Use relation scanner to scan rows
	return scanner.ScanRowsWithRelations(rows, dest)
}

// findManyMapsWithFieldMapping executes the query and scans results into maps with field name mapping
func (q *SelectQueryImpl) findManyMapsWithFieldMapping(_ context.Context, sql string, args []any, dest any) error {
	// Execute query using database's Query method
	rows, err := q.database.Query(sql, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get schema for field mapping
	mainSchema, err := q.database.GetSchema(q.modelName)
	if err != nil {
		return fmt.Errorf("failed to get schema for model %s: %w", q.modelName, err)
	}

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Prepare result slice
	var results []map[string]any

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

		// Create map for this row
		rowMap := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			// Handle byte arrays (convert to string)
			if b, ok := val.([]byte); ok {
				val = string(b)
			}

			// Map column name back to field name
			fieldName := col
			// Remove table alias prefix if present (e.g., "t.first_name" -> "first_name")
			if strings.Contains(col, ".") {
				parts := strings.Split(col, ".")
				fieldName = parts[len(parts)-1]
			}

			// Try to map column name to field name
			if mapped, err := mainSchema.GetFieldNameByColumnName(fieldName); err == nil {
				fieldName = mapped
			}

			rowMap[fieldName] = val
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// Set results to destination
	destValue := reflect.ValueOf(dest).Elem()
	destValue.Set(reflect.ValueOf(results))

	return nil
}

// findOneMapsWithFieldMapping executes the query and scans a single result into a map with field name mapping
func (q *SelectQueryImpl) findOneMapsWithFieldMapping(_ context.Context, sql string, args []any, dest any) error {
	// Execute query using database's Query method
	rows, err := q.database.Query(sql, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("no rows found")
	}

	// Get schema for field mapping
	mainSchema, err := q.database.GetSchema(q.modelName)
	if err != nil {
		return fmt.Errorf("failed to get schema for model %s: %w", q.modelName, err)
	}

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	// Create value holders
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := rows.Scan(valuePtrs...); err != nil {
		return fmt.Errorf("failed to scan row: %w", err)
	}

	// Create map for this row
	rowMap := make(map[string]any)
	for i, col := range columns {
		val := values[i]
		// Handle byte arrays (convert to string)
		if b, ok := val.([]byte); ok {
			val = string(b)
		}

		// Map column name back to field name
		fieldName := col
		// Remove table alias prefix if present (e.g., "t.first_name" -> "first_name")
		if strings.Contains(col, ".") {
			parts := strings.Split(col, ".")
			fieldName = parts[len(parts)-1]
		}

		// Try to map column name to field name
		if mapped, err := mainSchema.GetFieldNameByColumnName(fieldName); err == nil {
			fieldName = mapped
		}

		rowMap[fieldName] = val
	}

	// Set the map to the destination
	destValue := reflect.ValueOf(dest).Elem()
	destValue.Set(reflect.ValueOf(rowMap))

	return nil
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
