package orm

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// executeOperation executes a database operation based on the method name
func executeOperation(db types.Database, modelName, methodName string, options map[string]any, typeConverter *TypeConverter) (any, error) {
	ctx := context.Background()
	model := db.Model(modelName)

	switch methodName {
	// Create operations
	case "create":
		return executeCreate(ctx, model, options, modelName, db, typeConverter)
	case "createMany":
		return executeCreateMany(ctx, model, modelName, options, db)
	case "createManyAndReturn":
		return executeCreateManyAndReturn(ctx, model, modelName, options, db)

	// Read operations
	case "findUnique":
		return executeFindUnique(ctx, model, options)
	case "findFirst":
		return executeFindFirst(ctx, model, options)
	case "findMany":
		return executeFindMany(ctx, model, options)
	case "count":
		return executeCount(ctx, model, options)
	case "aggregate":
		return executeAggregate(ctx, model, options)
	case "groupBy":
		return executeGroupBy(ctx, model, modelName, options, db)

	// Update operations
	case "update":
		return executeUpdate(ctx, model, options)
	case "updateMany":
		return executeUpdateMany(ctx, model, modelName, options)
	case "updateManyAndReturn":
		return executeUpdateManyAndReturn(ctx, model, modelName, options)
	case "upsert":
		return executeUpsert(ctx, model, options, modelName, db)

	// Delete operations
	case "delete":
		return executeDelete(ctx, model, options)
	case "deleteMany":
		return executeDeleteMany(ctx, model, modelName, options)

	default:
		return nil, fmt.Errorf("unknown method: %s", methodName)
	}
}

// Create operations

func executeCreate(ctx context.Context, model types.ModelQuery, options map[string]any, modelName string, db types.Database, typeConverter *TypeConverter) (any, error) {
	data, ok := options["data"]
	if !ok {
		return nil, fmt.Errorf("create requires 'data' field")
	}

	// Handle nested creates
	processedData := processNestedWrites(data, "create", modelName, db)

	query := model.Insert(processedData)

	// Add RETURNING clause for databases that support it
	if db.GetCapabilities().SupportsReturning() {
		// Get schema to determine which fields to return
		schema, err := db.GetSchema(modelName)
		if err != nil {
			return nil, fmt.Errorf("failed to get schema: %w", err)
		}

		// Build the list of fields to return (all fields by default)
		returningFields := make([]string, 0, len(schema.Fields))
		for _, field := range schema.Fields {
			returningFields = append(returningFields, field.Name)
		}

		query = query.Returning(returningFields...)

		// Handle returning specific fields if requested
		if selectFields, ok := options["select"]; ok {
			fields := extractFieldNames(selectFields)
			query = query.Returning(fields...)
		}
	}

	// Use ExecAndReturn for databases that support RETURNING clause
	var createdRecord map[string]any
	if db.GetCapabilities().SupportsReturning() {
		// Database supports RETURNING clause
		err := query.ExecAndReturn(ctx, &createdRecord)
		if err != nil {
			return nil, err
		}
	} else {
		// Database doesn't support RETURNING, use the traditional method
		result, err := query.Exec(ctx)
		if err != nil {
			return nil, err
		}

		// Fetch the created record using LastInsertID
		selectQuery := model.Select()
		if result.LastInsertID > 0 {
			selectQuery = applySimpleWhereConditions(selectQuery, map[string]any{"id": result.LastInsertID}).(types.SelectQuery)
		}

		err = selectQuery.FindFirst(ctx, &createdRecord)
		if err != nil {
			// If we can't fetch the created record, return what we have
			if dataMap, ok := processedData.(map[string]any); ok {
				dataMap["id"] = result.LastInsertID
				return dataMap, nil
			}
			return processedData, nil
		}
	}

	return createdRecord, nil
}

func executeCreateMany(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any, db types.Database) (any, error) {
	data, ok := options["data"]
	if !ok {
		return nil, fmt.Errorf("createMany requires 'data' field")
	}

	dataSlice, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("createMany 'data' must be an array")
	}

	skipDuplicates := false
	if skip, ok := options["skipDuplicates"].(bool); ok {
		skipDuplicates = skip
	}

	// Process each item
	var processedData []any
	for _, item := range dataSlice {
		processedData = append(processedData, processNestedWrites(item, "create", modelName, db))
	}

	// Create records one by one (batch insert would be more efficient)
	created := 0
	for _, item := range processedData {
		query := model.Insert(item)
		_, err := query.Exec(ctx)
		if err != nil {
			if skipDuplicates && isUniqueConstraintError(err) {
				continue
			}
			return nil, err
		}
		created++
	}

	return map[string]any{
		"count": created,
	}, nil
}

func executeCreateManyAndReturn(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any, db types.Database) (any, error) {
	// Similar to createMany but returns created records
	// This is a simplified implementation
	data, ok := options["data"]
	if !ok {
		return nil, fmt.Errorf("createManyAndReturn requires 'data' field")
	}

	dataSlice, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("createManyAndReturn 'data' must be an array")
	}

	var created []any
	for _, item := range dataSlice {
		processedItem := processNestedWrites(item, "create", modelName, db)
		query := model.Insert(processedItem)
		result, err := query.Exec(ctx)
		if err != nil {
			return nil, err
		}

		// Add ID to the created item
		if itemMap, ok := processedItem.(map[string]any); ok {
			itemMap["id"] = result.LastInsertID
			created = append(created, itemMap)
		}
	}

	return created, nil
}

// Read operations

func executeFindUnique(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	where, ok := options["where"]
	if !ok {
		return nil, fmt.Errorf("findUnique requires 'where' field")
	}

	query := model.Select()

	// Apply where conditions
	query = applySimpleWhereConditions(query, where).(types.SelectQuery)

	// Handle select fields
	if selectFields, ok := options["select"]; ok {
		fields := extractFieldNames(selectFields)
		query = model.Select(fields...)
	}

	// Handle include (relations)
	if include, ok := options["include"]; ok {
		query = applyInclude(query, include).(types.SelectQuery)
	}

	result := make(map[string]any)
	err := query.FindFirst(ctx, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func executeFindFirst(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	query := model.Select()

	// Apply where conditions
	if where, ok := options["where"]; ok {
		query = applySimpleWhereConditions(query, where).(types.SelectQuery)
	}

	// Apply orderBy if provided
	if orderBy, ok := options["orderBy"]; ok {
		query = applyOrderBy(query, orderBy).(types.SelectQuery)
	}

	// Handle select fields
	if selectFields, ok := options["select"]; ok {
		fields := extractFieldNames(selectFields)
		query = model.Select(fields...)
	}

	// Handle include (relations)
	if include, ok := options["include"]; ok {
		query = applyInclude(query, include).(types.SelectQuery)
	}

	result := make(map[string]any)
	err := query.FindFirst(ctx, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func executeFindMany(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	// First determine which fields to select
	var selectedFields []string
	var includesFromSelect map[string]any
	if selectFields, ok := options["select"]; ok {
		selectedFields = extractFieldNames(selectFields)

		// Extract nested includes from select
		if selectMap, ok := selectFields.(map[string]any); ok {
			includesFromSelect = make(map[string]any)
			for field, value := range selectMap {
				if valueMap, ok := value.(map[string]any); ok {
					// This is a nested include with select
					includesFromSelect[field] = valueMap
				}
			}
		}
	}

	// Create query with selected fields (or all fields if none specified)
	var query types.SelectQuery
	if len(selectedFields) > 0 {
		query = model.Select(selectedFields...)
	} else {
		query = model.Select()
	}

	// Apply where conditions
	if where, ok := options["where"]; ok {
		query = applySimpleWhereConditions(query, where).(types.SelectQuery)
	}

	// Apply orderBy if provided
	if orderBy, ok := options["orderBy"]; ok {
		query = applyOrderBy(query, orderBy).(types.SelectQuery)
	}

	// Apply pagination
	if skip, ok := options["skip"]; ok {
		query = query.Offset(utils.ToInt(skip))
	}
	if take, ok := options["take"]; ok {
		query = query.Limit(utils.ToInt(take))
	}

	// Handle include (relations)
	if include, ok := options["include"]; ok {
		query = applyInclude(query, include).(types.SelectQuery)
	}

	// Apply includes from select if any
	if includesFromSelect != nil && len(includesFromSelect) > 0 {
		query = applyInclude(query, includesFromSelect).(types.SelectQuery)
	}

	// Handle distinct
	if distinct, ok := options["distinct"]; ok {
		switch d := distinct.(type) {
		case bool:
			if d {
				query = query.Distinct()
			}
		case []any:
			// Distinct on specific fields
			if len(d) > 0 {
				// Convert []any to []string
				fields := make([]string, 0, len(d))
				for _, field := range d {
					if fieldStr, ok := field.(string); ok {
						fields = append(fields, fieldStr)
					}
				}
				if len(fields) > 0 {
					query = query.DistinctOn(fields...)
				} else {
					// Fallback to general distinct if no valid fields
					query = query.Distinct()
				}
			}
		}
	}

	results := []map[string]any{}
	err := query.FindMany(ctx, &results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func executeCount(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	// Apply where conditions if provided
	if where, ok := options["where"]; ok {
		// For simple field equality, use the model's Where method which handles field resolution
		if whereMap, ok := where.(map[string]any); ok {
			var conditions []types.Condition
			for field, value := range whereMap {
				// Check if it's a simple field equality (not an operator object)
				if _, isOperator := value.(map[string]any); !isOperator {
					// Use the model's Where method which will handle field name resolution
					condition := model.Where(field).Equals(value)
					conditions = append(conditions, condition)
				} else {
					// For complex conditions, use the existing buildCondition
					condition := BuildCondition(map[string]any{field: value})
					conditions = append(conditions, condition)
				}
			}
			// Combine all conditions with AND
			if len(conditions) > 0 {
				var finalCondition types.Condition
				if len(conditions) == 1 {
					finalCondition = conditions[0]
				} else {
					finalCondition = types.NewAndCondition(conditions...)
				}
				model = model.WhereCondition(finalCondition)
			}
		} else {
			model = model.WhereCondition(BuildCondition(where))
		}
	}

	count, err := model.Count(ctx)
	if err != nil {
		return nil, err
	}

	return count, nil
}

func executeAggregate(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	// Apply where conditions if provided
	if where, ok := options["where"]; ok {
		model = model.WhereCondition(BuildCondition(where))
	}

	result := make(map[string]any)

	// Handle different aggregation types
	if count, ok := options["_count"]; ok {
		switch c := count.(type) {
		case bool:
			if c {
				// Simple count
				cnt, err := model.Count(ctx)
				if err != nil {
					return nil, err
				}
				result["_count"] = cnt
			}
		case map[string]any:
			// Field-specific count
			for field := range c {
				// For simplicity, just count all records
				cnt, err := model.Count(ctx)
				if err != nil {
					return nil, err
				}
				if _, ok := result["_count"]; !ok {
					result["_count"] = make(map[string]any)
				}
				result["_count"].(map[string]any)[field] = cnt
			}
		}
	}

	if avg, ok := options["_avg"]; ok {
		if avgMap, ok := avg.(map[string]any); ok {
			result["_avg"] = make(map[string]any)
			for field, val := range avgMap {
				if enabled, ok := val.(bool); ok && enabled {
					a, err := model.Avg(ctx, field)
					if err != nil {
						return nil, err
					}
					result["_avg"].(map[string]any)[field] = a
				}
			}
		}
	}

	if sum, ok := options["_sum"]; ok {
		if sumMap, ok := sum.(map[string]any); ok {
			result["_sum"] = make(map[string]any)
			for field, val := range sumMap {
				if enabled, ok := val.(bool); ok && enabled {
					s, err := model.Sum(ctx, field)
					if err != nil {
						return nil, err
					}
					result["_sum"].(map[string]any)[field] = s
				}
			}
		}
	}

	if min, ok := options["_min"]; ok {
		if minMap, ok := min.(map[string]any); ok {
			result["_min"] = make(map[string]any)
			for field, val := range minMap {
				if enabled, ok := val.(bool); ok && enabled {
					m, err := model.Min(ctx, field)
					if err != nil {
						return nil, err
					}
					result["_min"].(map[string]any)[field] = m
				}
			}
		}
	}

	if max, ok := options["_max"]; ok {
		if maxMap, ok := max.(map[string]any); ok {
			result["_max"] = make(map[string]any)
			for field, val := range maxMap {
				if enabled, ok := val.(bool); ok && enabled {
					m, err := model.Max(ctx, field)
					if err != nil {
						return nil, err
					}
					result["_max"].(map[string]any)[field] = m
				}
			}
		}
	}

	return result, nil
}

// Update operations

func executeUpdate(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	where, ok := options["where"]
	if !ok {
		return nil, fmt.Errorf("update requires 'where' field")
	}

	data, ok := options["data"]
	if !ok {
		return nil, fmt.Errorf("update requires 'data' field")
	}

	// First fetch the existing record
	selectQuery := model.Select()
	selectQuery = applySimpleWhereConditions(selectQuery, where).(types.SelectQuery)

	var existing map[string]any
	err := selectQuery.FindFirst(ctx, &existing)
	if err != nil {
		return nil, err
	}

	// Now update it
	updateQuery := model.Update(data)
	updateQuery = applySimpleWhereConditions(updateQuery, where).(types.UpdateQuery)

	_, err = updateQuery.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch the updated record
	var updated map[string]any
	err = selectQuery.FindFirst(ctx, &updated)
	if err != nil {
		// Return the data we attempted to update merged with existing
		existingMap := existing
		if dataMap, ok := data.(map[string]any); ok {
			for k, v := range dataMap {
				existingMap[k] = v
			}
		}
		return existingMap, nil
	}

	return updated, nil
}

func executeUpdateMany(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any) (any, error) {
	_ = modelName // TODO: might be needed for future enhancements
	data, ok := options["data"]
	if !ok {
		return nil, fmt.Errorf("updateMany requires 'data' field")
	}

	updateQuery := model.Update(data)

	// Apply where conditions
	if where, ok := options["where"]; ok {
		updateQuery = applySimpleWhereConditions(updateQuery, where).(types.UpdateQuery)
	}

	result, err := updateQuery.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"count": result.RowsAffected,
	}, nil
}

func executeUpdateManyAndReturn(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any) (any, error) {
	// This is a placeholder - proper implementation would require batch update with returning
	return nil, fmt.Errorf("updateManyAndReturn not yet implemented")
}

func executeUpsert(ctx context.Context, model types.ModelQuery, options map[string]any, modelName string, db types.Database) (any, error) {
	where, ok := options["where"]
	if !ok {
		return nil, fmt.Errorf("upsert requires 'where' field")
	}

	createData, hasCreate := options["create"]
	updateData, hasUpdate := options["update"]

	if !hasCreate || !hasUpdate {
		return nil, fmt.Errorf("upsert requires both 'create' and 'update' fields")
	}

	// First, try to find the existing record
	selectQuery := model.Select()
	selectQuery = applySimpleWhereConditions(selectQuery, where).(types.SelectQuery)

	var existing map[string]any
	err := selectQuery.FindFirst(ctx, &existing)

	if err != nil {
		// Record doesn't exist, create it
		query := model.Insert(createData)
		result, err := query.Exec(ctx)
		if err != nil {
			return nil, err
		}
		// Add ID to created data
		if createMap, ok := createData.(map[string]any); ok {
			createMap["id"] = result.LastInsertID
			return createMap, nil
		}
		return createData, nil
	} else {
		// Record exists, update it
		updateQuery := model.Update(updateData)
		updateQuery = applySimpleWhereConditions(updateQuery, where).(types.UpdateQuery)

		_, err = updateQuery.Exec(ctx)
		if err != nil {
			return nil, err
		}

		// Fetch the updated record
		var updated map[string]any
		err = selectQuery.FindFirst(ctx, &updated)
		if err != nil {
			// Merge update data with existing
			existingMap := existing
			if updateMap, ok := updateData.(map[string]any); ok {
				for k, v := range updateMap {
					existingMap[k] = v
				}
			}
			return existingMap, nil
		}
		return updated, nil
	}
}

// Delete operations

func executeDelete(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	where, ok := options["where"]
	if !ok {
		return nil, fmt.Errorf("delete requires 'where' field")
	}

	// First fetch the record to return it
	selectQuery := model.Select()
	selectQuery = applySimpleWhereConditions(selectQuery, where).(types.SelectQuery)

	var existing map[string]any
	err := selectQuery.FindFirst(ctx, &existing)
	if err != nil {
		return nil, err
	}

	// Now delete it
	deleteQuery := model.Delete()
	deleteQuery = applySimpleWhereConditions(deleteQuery, where).(types.DeleteQuery)

	_, err = deleteQuery.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return existing, nil
}

func executeDeleteMany(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any) (any, error) {
	_ = modelName // TODO: might be needed for future enhancements
	deleteQuery := model.Delete()

	// Apply where conditions
	if where, ok := options["where"]; ok {
		deleteQuery = applySimpleWhereConditions(deleteQuery, where).(types.DeleteQuery)
	}

	result, err := deleteQuery.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"count": result.RowsAffected,
	}, nil
}

// Helper functions

func processNestedWrites(data any, operation string, modelName string, db types.Database) any {
	// Process nested create/connect/disconnect operations
	dataMap, ok := data.(map[string]any)
	if !ok {
		return data // Not a map, return as-is
	}

	processedData := make(map[string]any)

	// Get schema for the current model to check relations
	schema, err := db.GetSchema(modelName)
	if err != nil {
		// No schema found, return data as-is
		return data
	}

	for fieldName, fieldValue := range dataMap {
		// Check if this field is a relation
		_, exists := schema.Relations[fieldName]
		if !exists {
			// Not a relation, copy as-is
			processedData[fieldName] = fieldValue
			continue
		}

		// Handle nested writes based on operation type
		// For now, we'll skip relation fields entirely as they need special handling
		// The actual nested write implementation should happen in the query builders
		// This prevents the field mapper from trying to map relation fields as columns
		continue
	}

	return processedData
}

// extractFieldNames extracts field names from select options
func extractFieldNames(selectFields any) []string {
	var fields []string

	switch v := selectFields.(type) {
	case map[string]any:
		for field, value := range v {
			// Check if it's a simple boolean selection
			if boolVal, ok := value.(bool); ok && boolVal {
				fields = append(fields, field)
			}
			// Could also handle nested selections here in the future
		}
	case []any:
		for _, field := range v {
			if fieldStr, ok := field.(string); ok {
				fields = append(fields, fieldStr)
			}
		}
	case []string:
		fields = v
	}

	return fields
}

// Check if error is a unique constraint violation
func isUniqueConstraintError(err error) bool {
	errStr := err.Error()
	return reflect.ValueOf(errStr).String() == "UNIQUE constraint failed" ||
		reflect.ValueOf(errStr).String() == "duplicate key"
}

// executeGroupBy handles groupBy queries
func executeGroupBy(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any, db types.Database) (any, error) {
	// Check if database supports raw SQL queries
	// For NoSQL databases like MongoDB, use the query builder instead of generating SQL
	capabilities := db.GetCapabilities()
	if capabilities.IsNoSQL() {
		return executeAggregationQuery(ctx, model, modelName, options, db)
	}

	// Parse groupBy fields
	var groupByFields []string
	if by, ok := options["by"]; ok {
		switch b := by.(type) {
		case string:
			groupByFields = []string{b}
		case []any:
			for _, field := range b {
				if fieldStr, ok := field.(string); ok {
					groupByFields = append(groupByFields, fieldStr)
				}
			}
		}
	}

	if len(groupByFields) == 0 {
		return nil, fmt.Errorf("groupBy requires 'by' field")
	}

	// Build SELECT clause
	var selectParts []string

	// Add grouped fields
	for _, field := range groupByFields {
		// Resolve field name to column name
		columnName, err := db.ResolveFieldName(modelName, field)
		if err != nil {
			// Fall back to field name if not found
			columnName = field
		}
		// Use column AS field to maintain the original field name in results
		// Quote the alias to preserve case in PostgreSQL
		selectParts = append(selectParts, fmt.Sprintf("%s AS \"%s\"", columnName, field))
	}

	// Handle _count, _sum, _avg, _min, _max aggregations
	aggregations := []string{"_count", "_sum", "_avg", "_min", "_max"}
	for _, agg := range aggregations {
		if aggValue, ok := options[agg]; ok {
			// Parse aggregation options
			switch av := aggValue.(type) {
			case bool:
				if av && agg == "_count" {
					// Simple count(*)
					selectParts = append(selectParts, "COUNT(*) as _count")
				}
			case map[string]any:
				// Field-specific aggregations
				for field, enabled := range av {
					if e, ok := enabled.(bool); ok && e {
						aggFunc := strings.ToUpper(strings.TrimPrefix(agg, "_"))
						// Resolve field name to column name
						columnName, err := db.ResolveFieldName(modelName, field)
						if err != nil {
							// Fall back to field name
							columnName = field
						}
						selectParts = append(selectParts, fmt.Sprintf("%s(%s) as %s%s", aggFunc, columnName, field, agg))
					}
				}
			}
		}
	}

	// Build the SQL query
	tableName, err := db.ResolveTableName(modelName)
	if err != nil {
		return nil, err
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectParts, ", "), tableName)

	// Add WHERE clause if provided
	if where, ok := options["where"]; ok {
		// Build simple WHERE conditions for groupBy
		whereSQL := buildSimpleWhereSQL(where, modelName, db)
		if whereSQL != "" {
			sql += " WHERE " + whereSQL
		}
	}

	// Add GROUP BY clause
	if len(groupByFields) > 0 {
		var groupByColumns []string
		for _, field := range groupByFields {
			columnName, err := db.ResolveFieldName(modelName, field)
			if err != nil {
				columnName = field
			}
			groupByColumns = append(groupByColumns, columnName)
		}
		sql += fmt.Sprintf(" GROUP BY %s", strings.Join(groupByColumns, ", "))
	}

	// Add HAVING clause if provided
	if having, ok := options["having"]; ok {
		// Build simple HAVING conditions
		havingSQL := buildSimpleHavingSQL(having)
		if havingSQL != "" {
			sql += " HAVING " + havingSQL
		}
	}

	// Add ORDER BY if provided
	if orderBy, ok := options["orderBy"]; ok {
		orderSQL := buildOrderBySQL(orderBy, modelName, db)
		if orderSQL != "" {
			sql += " ORDER BY " + orderSQL
		}
	}

	// Apply pagination
	if take, ok := options["take"]; ok {
		sql += fmt.Sprintf(" LIMIT %d", int(utils.ToInt64(take)))
	}
	if skip, ok := options["skip"]; ok {
		sql += fmt.Sprintf(" OFFSET %d", int(utils.ToInt64(skip)))
	}

	// Execute raw query for SQL databases
	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Scan results into maps
	results, err := utils.ScanRowsToMaps(rows)
	if err != nil {
		return nil, err
	}

	// Post-process results to convert field_agg format to nested objects
	for i, result := range results {
		processedResult := make(map[string]any)

		// Copy grouped fields and _count
		for k, v := range result {
			if !strings.Contains(k, "_") {
				// Regular field
				processedResult[k] = v
			} else if k == "_count" {
				// Convert _count to int
				processedResult[k] = utils.ToInt64(v)
			}
		}

		// Transform field_agg to nested format
		aggregations := []string{"_sum", "_avg", "_min", "_max"}
		for _, agg := range aggregations {
			aggMap := make(map[string]any)
			for k, v := range result {
				// Check if this is a field_agg pattern
				if strings.HasSuffix(k, agg) && strings.Contains(k, "_") {
					// Remove the _agg suffix to get the field name
					fieldName := strings.TrimSuffix(k, agg)
					// Remove the trailing underscore
					fieldName = strings.TrimSuffix(fieldName, "_")
					// Convert to proper numeric type for aggregations
					if agg != "_count" {
						aggMap[fieldName] = utils.ToFloat64(v)
					} else {
						aggMap[fieldName] = utils.ToInt64(v)
					}
				}
			}
			if len(aggMap) > 0 {
				processedResult[agg] = aggMap
			}
		}

		results[i] = processedResult
	}

	return results, nil
}

// buildOrderBySQL builds ORDER BY SQL from orderBy options
func buildOrderBySQL(orderBy any, modelName string, db types.Database) string {
	var orderParts []string

	switch ob := orderBy.(type) {
	case map[string]any:
		// Single orderBy object: {field: "asc"|"desc"} or {_sum: {field: "asc"}}
		for field, direction := range ob {
			// Check if it's an aggregation orderBy
			if strings.HasPrefix(field, "_") {
				// Handle aggregation ordering like _sum, _avg, etc.
				if dirMap, ok := direction.(map[string]any); ok {
					for aggField, dir := range dirMap {
						direction := "ASC"
						if dirStr, ok := dir.(string); ok && strings.ToLower(dirStr) == "desc" {
							direction = "DESC"
						}
						// Use the aliased column name from SELECT
						orderParts = append(orderParts, fmt.Sprintf("%s%s %s", aggField, field, direction))
					}
				}
			} else {
				// Regular field ordering
				columnName, err := db.ResolveFieldName(modelName, field)
				if err != nil {
					columnName = field
				}
				dir := "ASC"
				if dirStr, ok := direction.(string); ok && strings.ToLower(dirStr) == "desc" {
					dir = "DESC"
				}
				orderParts = append(orderParts, fmt.Sprintf("%s %s", columnName, dir))
			}
		}
	case []any:
		// Array of orderBy objects: [{field: "asc"}, {field2: "desc"}]
		for _, item := range ob {
			if orderMap, ok := item.(map[string]any); ok {
				for field, direction := range orderMap {
					columnName, err := db.ResolveFieldName(modelName, field)
					if err != nil {
						columnName = field
					}
					dir := "ASC"
					if dirStr, ok := direction.(string); ok && strings.ToLower(dirStr) == "desc" {
						dir = "DESC"
					}
					orderParts = append(orderParts, fmt.Sprintf("%s %s", columnName, dir))
				}
			}
		}
	}

	return strings.Join(orderParts, ", ")
}

// buildSimpleWhereSQL builds WHERE SQL from simple where conditions (for raw SQL queries)
func buildSimpleWhereSQL(where any, modelName string, db types.Database) string {
	whereMap, ok := where.(map[string]any)
	if !ok {
		return ""
	}

	var whereParts []string

	for field, value := range whereMap {
		// Skip complex operators for now
		if _, isMap := value.(map[string]any); isMap {
			continue
		}

		// Resolve field name to column name
		columnName, err := db.ResolveFieldName(modelName, field)
		if err != nil {
			columnName = field
		}

		// Format value based on type
		var valueStr string
		switch v := value.(type) {
		case string:
			// Escape single quotes in string values
			valueStr = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
		case nil:
			valueStr = "NULL"
		default:
			valueStr = fmt.Sprintf("%v", v)
		}

		whereParts = append(whereParts, fmt.Sprintf("%s = %s", columnName, valueStr))
	}

	return strings.Join(whereParts, " AND ")
}

// buildSimpleHavingSQL builds HAVING SQL from having conditions (for raw SQL queries)
func buildSimpleHavingSQL(having any) string {
	havingMap, ok := having.(map[string]any)
	if !ok {
		return ""
	}

	var havingParts []string

	// Handle aggregation conditions like _sum, _avg, etc.
	for aggType, conditions := range havingMap {
		if !strings.HasPrefix(aggType, "_") {
			continue
		}

		// Get the aggregation function name
		aggFunc := strings.ToUpper(strings.TrimPrefix(aggType, "_"))

		if condMap, ok := conditions.(map[string]any); ok {
			for field, operators := range condMap {
				// Build the aggregation expression
				var aggExpr string
				if field == "_all" && aggType == "_count" {
					// Special case for COUNT(*)
					aggExpr = "COUNT(*)"
				} else {
					aggExpr = fmt.Sprintf("%s(%s)", aggFunc, field)
				}

				if opMap, ok := operators.(map[string]any); ok {
					for op, value := range opMap {
						// Handle different operators
						var condition string
						switch op {
						case "gte":
							condition = fmt.Sprintf("%s >= %v", aggExpr, value)
						case "gt":
							condition = fmt.Sprintf("%s > %v", aggExpr, value)
						case "lte":
							condition = fmt.Sprintf("%s <= %v", aggExpr, value)
						case "lt":
							condition = fmt.Sprintf("%s < %v", aggExpr, value)
						case "equals":
							condition = fmt.Sprintf("%s = %v", aggExpr, value)
						default:
							// Default to equals
							condition = fmt.Sprintf("%s = %v", aggExpr, value)
						}

						if condition != "" {
							havingParts = append(havingParts, condition)
						}
					}
				}
			}
		}
	}

	return strings.Join(havingParts, " AND ")
}

// executeAggregationQuery executes groupBy using the query builder for NoSQL databases
func executeAggregationQuery(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any, db types.Database) (any, error) {
	// For MongoDB, we need to manually build the aggregation pipeline
	// since the standard SelectQuery doesn't handle aggregation functions properly

	// Parse groupBy fields
	var groupByFields []string
	if by, ok := options["by"]; ok {
		switch b := by.(type) {
		case string:
			groupByFields = []string{b}
		case []any:
			for _, field := range b {
				if fieldStr, ok := field.(string); ok {
					groupByFields = append(groupByFields, fieldStr)
				}
			}
		}
	}

	if len(groupByFields) == 0 {
		return nil, fmt.Errorf("groupBy requires 'by' field")
	}

	// Build MongoDB aggregation pipeline manually
	return executeMongoDBGroupBy(ctx, modelName, groupByFields, options, db)
}

// applyAggregationWhereConditions applies where conditions to aggregation query
func applyAggregationWhereConditions(query types.AggregationQuery, where any) types.AggregationQuery {
	if whereMap, ok := where.(map[string]any); ok {
		for field, value := range whereMap {
			// For simple field equality
			if _, isOperator := value.(map[string]any); !isOperator {
				// Simple equality condition
				query = query.WhereCondition(query.Where(field).Equals(value))
			} else {
				// Complex condition - use existing condition builder
				condition := BuildCondition(map[string]any{field: value})
				query = query.WhereCondition(condition)
			}
		}
	}
	return query
}

// buildAggregationHavingCondition builds having condition for aggregation
func buildAggregationHavingCondition(having map[string]any) types.Condition {
	// This is a simplified implementation
	// Full implementation would need to handle aggregation function references
	// For now, return nil to skip having conditions in NoSQL
	return nil
}

// applyAggregationOrderBy applies order by to aggregation query
func applyAggregationOrderBy(query types.AggregationQuery, orderBy any) types.AggregationQuery {
	switch ob := orderBy.(type) {
	case map[string]any:
		for field, direction := range ob {
			dir := types.ASC
			if dirStr, ok := direction.(string); ok && strings.ToLower(dirStr) == "desc" {
				dir = types.DESC
			}

			// Check if it's an aggregation field ordering
			if strings.HasPrefix(field, "_") {
				// This would be aggregation ordering - simplified for now
				query = query.OrderBy(field, dir)
			} else {
				// Regular field ordering
				query = query.OrderBy(field, dir)
			}
		}
	case []any:
		for _, item := range ob {
			if orderMap, ok := item.(map[string]any); ok {
				for field, direction := range orderMap {
					dir := types.ASC
					if dirStr, ok := direction.(string); ok && strings.ToLower(dirStr) == "desc" {
						dir = types.DESC
					}
					query = query.OrderBy(field, dir)
				}
			}
		}
	}
	return query
}

// processAggregationResults processes raw aggregation results to match expected format
func processAggregationResults(results []map[string]any) []map[string]any {
	processed := make([]map[string]any, len(results))

	for i, result := range results {
		processedResult := make(map[string]any)

		// Copy regular grouped fields
		for k, v := range result {
			if !strings.HasPrefix(k, "_") {
				processedResult[k] = v
			}
		}

		// Process aggregation fields - convert from _sum_field format to nested format
		aggregations := map[string]map[string]any{
			"_count": make(map[string]any),
			"_sum":   make(map[string]any),
			"_avg":   make(map[string]any),
			"_min":   make(map[string]any),
			"_max":   make(map[string]any),
		}

		for k, v := range result {
			if strings.HasPrefix(k, "_") {
				if k == "_count" {
					// Simple count
					processedResult["_count"] = utils.ToInt64(v)
				} else {
					// Parse field-specific aggregations like _sum_amount
					for aggType := range aggregations {
						prefix := aggType + "_"
						if strings.HasPrefix(k, prefix) {
							fieldName := strings.TrimPrefix(k, prefix)
							if aggType == "_count" {
								aggregations[aggType][fieldName] = utils.ToInt64(v)
							} else {
								aggregations[aggType][fieldName] = utils.ToFloat64(v)
							}
						}
					}
				}
			}
		}

		// Add non-empty aggregation groups to result
		for aggType, aggData := range aggregations {
			if len(aggData) > 0 {
				processedResult[aggType] = aggData
			}
		}

		processed[i] = processedResult
	}

	return processed
}

// executeMongoDBGroupBy manually builds MongoDB aggregation pipeline for groupBy operations
func executeMongoDBGroupBy(ctx context.Context, modelName string, groupByFields []string, options map[string]any, db types.Database) (any, error) {
	// Get collection name
	tableName, err := db.ResolveTableName(modelName)
	if err != nil {
		return nil, err
	}

	// Build aggregation pipeline
	pipeline := []any{}

	// Add $match stage for WHERE conditions
	if where, ok := options["where"]; ok {
		matchFilter := buildMongoDBMatchFilter(where, modelName, db)
		if matchFilter != nil {
			pipeline = append(pipeline, map[string]any{"$match": matchFilter})
		}
	}

	// Build $group stage
	groupStage := buildMongoDBGroupStage(groupByFields, options, modelName, db)
	pipeline = append(pipeline, map[string]any{"$group": groupStage})

	// Add $match stage for HAVING conditions
	if having, ok := options["having"]; ok {
		havingFilter := buildMongoDBHavingFilter(having)
		if havingFilter != nil {
			pipeline = append(pipeline, map[string]any{"$match": havingFilter})
		}
	}

	// Add $sort stage
	if orderBy, ok := options["orderBy"]; ok {
		sortStage := buildMongoDBSortStage(orderBy, modelName, db)
		if sortStage != nil {
			pipeline = append(pipeline, map[string]any{"$sort": sortStage})
		}
	}

	// Add $skip and $limit stages
	if skip, ok := options["skip"]; ok {
		pipeline = append(pipeline, map[string]any{"$skip": utils.ToInt64(skip)})
	}
	if take, ok := options["take"]; ok {
		pipeline = append(pipeline, map[string]any{"$limit": utils.ToInt64(take)})
	}

	// Execute aggregation
	pipelineJSON := mustMarshalJSON(pipeline)
	rawSQL := fmt.Sprintf(`{"operation": "aggregate", "collection": "%s", "pipeline": %s}`,
		tableName, pipelineJSON)

	rawQuery := db.Raw(rawSQL)
	results := []map[string]any{}
	err = rawQuery.Find(ctx, &results)
	if err != nil {
		return nil, err
	}

	// Post-process results to match expected format
	return processMongoDBGroupByResults(results, groupByFields), nil
}

// buildMongoDBMatchFilter builds MongoDB match filter from WHERE conditions
func buildMongoDBMatchFilter(where any, modelName string, db types.Database) map[string]any {
	if whereMap, ok := where.(map[string]any); ok {
		filter := make(map[string]any)
		for field, value := range whereMap {
			columnName, err := db.ResolveFieldName(modelName, field)
			if err != nil {
				columnName = field
			}
			filter[columnName] = value
		}
		return filter
	}
	return nil
}

// buildMongoDBGroupStage builds MongoDB group stage with aggregations
func buildMongoDBGroupStage(groupByFields []string, options map[string]any, modelName string, db types.Database) map[string]any {
	// Build _id for grouping
	groupID := make(map[string]any)
	for _, field := range groupByFields {
		columnName, err := db.ResolveFieldName(modelName, field)
		if err != nil {
			columnName = field
		}
		groupID[field] = "$" + columnName
	}

	groupStage := map[string]any{
		"_id": groupID,
	}

	// Add grouped fields to the result
	for _, field := range groupByFields {
		columnName, err := db.ResolveFieldName(modelName, field)
		if err != nil {
			columnName = field
		}
		groupStage[field] = map[string]any{"$first": "$" + columnName}
	}

	// Add aggregation functions
	if count, ok := options["_count"]; ok {
		switch c := count.(type) {
		case bool:
			if c {
				groupStage["_count"] = map[string]any{"$sum": 1}
			}
		case map[string]any:
			for field, enabled := range c {
				if e, ok := enabled.(bool); ok && e {
					groupStage["_count_"+field] = map[string]any{"$sum": 1}
				}
			}
		}
	}

	if sum, ok := options["_sum"]; ok {
		if sumMap, ok := sum.(map[string]any); ok {
			for field, enabled := range sumMap {
				if e, ok := enabled.(bool); ok && e {
					columnName, err := db.ResolveFieldName(modelName, field)
					if err != nil {
						columnName = field
					}
					groupStage["_sum_"+field] = map[string]any{"$sum": "$" + columnName}
				}
			}
		}
	}

	if avg, ok := options["_avg"]; ok {
		if avgMap, ok := avg.(map[string]any); ok {
			for field, enabled := range avgMap {
				if e, ok := enabled.(bool); ok && e {
					columnName, err := db.ResolveFieldName(modelName, field)
					if err != nil {
						columnName = field
					}
					groupStage["_avg_"+field] = map[string]any{"$avg": "$" + columnName}
				}
			}
		}
	}

	if min, ok := options["_min"]; ok {
		if minMap, ok := min.(map[string]any); ok {
			for field, enabled := range minMap {
				if e, ok := enabled.(bool); ok && e {
					columnName, err := db.ResolveFieldName(modelName, field)
					if err != nil {
						columnName = field
					}
					groupStage["_min_"+field] = map[string]any{"$min": "$" + columnName}
				}
			}
		}
	}

	if max, ok := options["_max"]; ok {
		if maxMap, ok := max.(map[string]any); ok {
			for field, enabled := range maxMap {
				if e, ok := enabled.(bool); ok && e {
					columnName, err := db.ResolveFieldName(modelName, field)
					if err != nil {
						columnName = field
					}
					groupStage["_max_"+field] = map[string]any{"$max": "$" + columnName}
				}
			}
		}
	}

	return groupStage
}

// buildMongoDBHavingFilter builds MongoDB filter for HAVING conditions
func buildMongoDBHavingFilter(having any) map[string]any {
	if havingMap, ok := having.(map[string]any); ok {
		filter := make(map[string]any)

		for aggType, conditions := range havingMap {
			if condMap, ok := conditions.(map[string]any); ok {
				for field, operators := range condMap {
					if opMap, ok := operators.(map[string]any); ok {
						for op, value := range opMap {
							// Build the field name for the aggregation result
							var fieldName string
							if field == "_all" && aggType == "_count" {
								fieldName = "_count"
							} else {
								fieldName = aggType + "_" + field
							}

							// Build the condition
							switch op {
							case "gte":
								filter[fieldName] = map[string]any{"$gte": value}
							case "gt":
								filter[fieldName] = map[string]any{"$gt": value}
							case "lte":
								filter[fieldName] = map[string]any{"$lte": value}
							case "lt":
								filter[fieldName] = map[string]any{"$lt": value}
							case "equals":
								filter[fieldName] = value
							default:
								filter[fieldName] = value
							}
						}
					}
				}
			}
		}

		if len(filter) == 0 {
			return nil
		}
		return filter
	}
	return nil
}

// buildMongoDBSortStage builds MongoDB sort stage
func buildMongoDBSortStage(orderBy any, modelName string, db types.Database) map[string]any {
	sort := make(map[string]any)

	switch ob := orderBy.(type) {
	case map[string]any:
		for field, direction := range ob {
			dir := 1
			if dirStr, ok := direction.(string); ok && strings.ToLower(dirStr) == "desc" {
				dir = -1
			}

			// Check if it's a regular field or aggregation result
			if strings.HasPrefix(field, "_") {
				// Handle nested aggregation ordering like {_sum: {amount: 'desc'}}
				if dirMap, ok := direction.(map[string]any); ok {
					// This is nested aggregation ordering
					for aggField, aggDir := range dirMap {
						aggDirInt := 1
						if dirStr, ok := aggDir.(string); ok && strings.ToLower(dirStr) == "desc" {
							aggDirInt = -1
						}
						// Create the aggregation field name: field_aggField (e.g., _sum_amount)
						aggFieldName := fmt.Sprintf("%s_%s", field, aggField)
						sort[aggFieldName] = aggDirInt
					}
				} else {
					// Simple aggregation field - use as is
					sort[field] = dir
				}
			} else {
				// Regular field - resolve column name
				columnName, err := db.ResolveFieldName(modelName, field)
				if err != nil {
					columnName = field
				}
				sort[columnName] = dir
			}
		}
	}

	if len(sort) == 0 {
		return nil
	}
	return sort
}

// processMongoDBGroupByResults processes aggregation results to match expected format
func processMongoDBGroupByResults(results []map[string]any, groupByFields []string) []map[string]any {
	processed := make([]map[string]any, len(results))

	for i, result := range results {
		processedResult := make(map[string]any)

		// Extract grouped fields from _id
		if id, ok := result["_id"]; ok {
			if idMap, ok := id.(map[string]any); ok {
				for _, field := range groupByFields {
					if value, exists := idMap[field]; exists {
						processedResult[field] = value
					}
				}
			}
		}

		// Process aggregation results
		aggregations := map[string]map[string]any{
			"_sum": make(map[string]any),
			"_avg": make(map[string]any),
			"_min": make(map[string]any),
			"_max": make(map[string]any),
		}

		for k, v := range result {
			if k == "_id" {
				continue
			}

			if k == "_count" {
				processedResult["_count"] = utils.ToInt64(v)
			} else {
				// Parse field-specific aggregations like _sum_amount
				for aggType := range aggregations {
					prefix := aggType + "_"
					if strings.HasPrefix(k, prefix) {
						fieldName := strings.TrimPrefix(k, prefix)
						if aggType == "_count" {
							aggregations[aggType][fieldName] = utils.ToInt64(v)
						} else {
							aggregations[aggType][fieldName] = utils.ToFloat64(v)
						}
					}
				}
			}
		}

		// Add non-empty aggregation groups to result
		for aggType, aggData := range aggregations {
			if len(aggData) > 0 {
				processedResult[aggType] = aggData
			}
		}

		processed[i] = processedResult
	}

	return processed
}

// mustMarshalJSON marshals to JSON or panics
func mustMarshalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return string(b)
}
