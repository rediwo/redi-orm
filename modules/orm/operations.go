package orm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// executeOperation executes a database operation based on the method name
func (m *ModelsModule) executeOperation(db types.Database, modelName, methodName string, options map[string]any) (any, error) {
	ctx := context.Background()
	model := db.Model(modelName)

	switch methodName {
	// Create operations
	case "create":
		return m.executeCreate(ctx, model, options, modelName, db)
	case "createMany":
		return m.executeCreateMany(ctx, model, modelName, options, db)
	case "createManyAndReturn":
		return m.executeCreateManyAndReturn(ctx, model, modelName, options, db)

	// Read operations
	case "findUnique":
		return m.executeFindUnique(ctx, model, options)
	case "findFirst":
		return m.executeFindFirst(ctx, model, options)
	case "findMany":
		return m.executeFindMany(ctx, model, options)
	case "count":
		return m.executeCount(ctx, model, options)
	case "aggregate":
		return m.executeAggregate(ctx, model, options)
	case "groupBy":
		return m.executeGroupBy(ctx, model, modelName, options, db)

	// Update operations
	case "update":
		return m.executeUpdate(ctx, model, options)
	case "updateMany":
		return m.executeUpdateMany(ctx, model, modelName, options)
	case "updateManyAndReturn":
		return m.executeUpdateManyAndReturn(ctx, model, modelName, options)
	case "upsert":
		return m.executeUpsert(ctx, model, options, modelName, db)

	// Delete operations
	case "delete":
		return m.executeDelete(ctx, model, options)
	case "deleteMany":
		return m.executeDeleteMany(ctx, model, modelName, options)

	default:
		return nil, fmt.Errorf("unknown method: %s", methodName)
	}
}

// Create operations

func (m *ModelsModule) executeCreate(ctx context.Context, model types.ModelQuery, options map[string]any, modelName string, db types.Database) (any, error) {
	data, ok := options["data"]
	if !ok {
		return nil, fmt.Errorf("create requires 'data' field")
	}

	// Handle nested creates
	processedData := m.processNestedWrites(data, "create", modelName, db)

	query := model.Insert(processedData)

	// Handle returning specific fields
	if selectFields, ok := options["select"]; ok {
		fields := m.extractFieldNames(selectFields)
		query = query.Returning(fields...)
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// For now, return the created data with ID
	// In a real implementation, we'd fetch the created record
	if dataMap, ok := processedData.(map[string]any); ok {
		dataMap["id"] = result.LastInsertID
		return dataMap, nil
	}

	return processedData, nil
}

func (m *ModelsModule) executeCreateMany(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any, db types.Database) (any, error) {
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
		processedData = append(processedData, m.processNestedWrites(item, "create", modelName, db))
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

func (m *ModelsModule) executeCreateManyAndReturn(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any, db types.Database) (any, error) {
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
		processedItem := m.processNestedWrites(item, "create", modelName, db)
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

func (m *ModelsModule) executeFindUnique(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	where, ok := options["where"]
	if !ok {
		return nil, fmt.Errorf("findUnique requires 'where' field")
	}

	query := model.Select()

	// Apply where conditions
	query = m.applySimpleWhereConditions(query, where).(types.SelectQuery)

	// Handle select fields
	if selectFields, ok := options["select"]; ok {
		fields := m.extractFieldNames(selectFields)
		query = model.Select(fields...)
	}

	// Handle include (relations)
	if include, ok := options["include"]; ok {
		query = m.applyInclude(query, include).(types.SelectQuery)
	}

	result := make(map[string]any)
	err := query.FindFirst(ctx, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (m *ModelsModule) executeFindFirst(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	query := model.Select()

	// Apply where conditions
	if where, ok := options["where"]; ok {
		query = m.applySimpleWhereConditions(query, where).(types.SelectQuery)
	}

	// Apply orderBy if provided
	if orderBy, ok := options["orderBy"]; ok {
		m.applyOrderBy(query, orderBy)
	}

	// Handle select fields
	if selectFields, ok := options["select"]; ok {
		fields := m.extractFieldNames(selectFields)
		query = model.Select(fields...)
	}

	// Handle include (relations)
	if include, ok := options["include"]; ok {
		query = m.applyInclude(query, include).(types.SelectQuery)
	}

	result := make(map[string]any)
	err := query.FindFirst(ctx, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (m *ModelsModule) executeFindMany(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	query := model.Select()

	// Apply where conditions
	if where, ok := options["where"]; ok {
		query = m.applySimpleWhereConditions(query, where).(types.SelectQuery)
	}

	// Apply orderBy if provided
	if orderBy, ok := options["orderBy"]; ok {
		m.applyOrderBy(query, orderBy)
	}

	// Apply pagination
	if skip, ok := options["skip"]; ok {
		query = query.Offset(utils.ToInt(skip))
	}
	if take, ok := options["take"]; ok {
		query = query.Limit(utils.ToInt(take))
	}

	// Handle select fields
	if selectFields, ok := options["select"]; ok {
		fields := m.extractFieldNames(selectFields)
		query = model.Select(fields...)
	}

	// Handle include (relations)
	if include, ok := options["include"]; ok {
		query = m.applyInclude(query, include).(types.SelectQuery)
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
			// For now, just use general distinct if any fields are specified
			if len(d) > 0 {
				query = query.Distinct()
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

func (m *ModelsModule) executeCount(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
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
					condition := m.BuildCondition(map[string]any{field: value})
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
			model = model.WhereCondition(m.BuildCondition(where))
		}
	}

	count, err := model.Count(ctx)
	if err != nil {
		return nil, err
	}

	return count, nil
}

func (m *ModelsModule) executeAggregate(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	// Apply where conditions if provided
	if where, ok := options["where"]; ok {
		model = model.WhereCondition(m.BuildCondition(where))
	}

	result := make(map[string]any)

	// Handle different aggregation types
	if count, ok := options["_count"]; ok {
		if countMap, ok := count.(map[string]any); ok {
			for field := range countMap {
				// For simplicity, just count all records
				c, err := model.Count(ctx)
				if err != nil {
					return nil, err
				}
				if _, ok := result["_count"]; !ok {
					result["_count"] = make(map[string]any)
				}
				result["_count"].(map[string]any)[field] = c
			}
		}
	}

	if avg, ok := options["_avg"]; ok {
		if avgMap, ok := avg.(map[string]any); ok {
			for field := range avgMap {
				a, err := model.Avg(ctx, field)
				if err != nil {
					return nil, err
				}
				if _, ok := result["_avg"]; !ok {
					result["_avg"] = make(map[string]any)
				}
				result["_avg"].(map[string]any)[field] = a
			}
		}
	}

	if sum, ok := options["_sum"]; ok {
		if sumMap, ok := sum.(map[string]any); ok {
			for field := range sumMap {
				s, err := model.Sum(ctx, field)
				if err != nil {
					return nil, err
				}
				if _, ok := result["_sum"]; !ok {
					result["_sum"] = make(map[string]any)
				}
				result["_sum"].(map[string]any)[field] = s
			}
		}
	}

	if min, ok := options["_min"]; ok {
		if minMap, ok := min.(map[string]any); ok {
			for field := range minMap {
				m, err := model.Min(ctx, field)
				if err != nil {
					return nil, err
				}
				if _, ok := result["_min"]; !ok {
					result["_min"] = make(map[string]any)
				}
				result["_min"].(map[string]any)[field] = m
			}
		}
	}

	if max, ok := options["_max"]; ok {
		if maxMap, ok := max.(map[string]any); ok {
			for field := range maxMap {
				m, err := model.Max(ctx, field)
				if err != nil {
					return nil, err
				}
				if _, ok := result["_max"]; !ok {
					result["_max"] = make(map[string]any)
				}
				result["_max"].(map[string]any)[field] = m
			}
		}
	}

	return result, nil
}

func (m *ModelsModule) executeGroupBy(ctx context.Context, _ types.ModelQuery, modelName string, options map[string]any, db types.Database) (any, error) {
	// GroupBy implementation
	_ = ctx // We'll use it for raw queries

	// For now, we'll build raw SQL instead of using the query builder
	// since groupBy needs special handling for aggregations

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

	// Handle aggregations - we need to build raw SQL for aggregations
	// For now, let's execute a raw query instead of using the query builder

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
		selectParts = append(selectParts, fmt.Sprintf("%s AS %s", columnName, field))
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
						selectParts = append(selectParts, fmt.Sprintf("%s(%s) as %s_%s", aggFunc, columnName, field, agg))
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
		// For now, skip complex where conditions in groupBy
		_ = where
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
		// For now, skip having conditions
		_ = having
	}

	// Add ORDER BY if provided
	if orderBy, ok := options["orderBy"]; ok {
		if orderMap, ok := orderBy.(map[string]any); ok {
			var orderParts []string
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
			if len(orderParts) > 0 {
				sql += fmt.Sprintf(" ORDER BY %s", strings.Join(orderParts, ", "))
			}
		}
	}

	// Apply pagination
	if take, ok := options["take"]; ok {
		sql += fmt.Sprintf(" LIMIT %d", utils.ToInt64(take))
	}
	if skip, ok := options["skip"]; ok {
		sql += fmt.Sprintf(" OFFSET %d", utils.ToInt64(skip))
	}

	// Execute raw query
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

	return results, nil
}

// Update operations

func (m *ModelsModule) executeUpdate(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	where, ok := options["where"]
	if !ok {
		return nil, fmt.Errorf("update requires 'where' field")
	}

	data, ok := options["data"]
	if !ok {
		return nil, fmt.Errorf("update requires 'data' field")
	}

	// Process nested writes and atomic operations
	processedData := m.processUpdateData(data)

	query := model.Update(processedData)

	// Apply where conditions
	query = m.applySimpleWhereConditions(query, where).(types.UpdateQuery)

	// Handle returning specific fields
	if selectFields, ok := options["select"]; ok {
		fields := m.extractFieldNames(selectFields)
		query = query.Returning(fields...)
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// Return updated data (simplified - should fetch from DB)
	return map[string]any{
		"id":           result.LastInsertID,
		"rowsAffected": result.RowsAffected,
	}, nil
}

func (m *ModelsModule) executeUpdateMany(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any) (any, error) {
	_ = modelName // TODO: might be needed for future enhancements
	data, ok := options["data"]
	if !ok {
		return nil, fmt.Errorf("updateMany requires 'data' field")
	}

	// Process update data
	processedData := m.processUpdateData(data)

	query := model.Update(processedData)

	// Apply where conditions
	if where, ok := options["where"]; ok {
		query = m.applySimpleWhereConditions(query, where).(types.UpdateQuery)
	}

	result, err := query.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"count": result.RowsAffected,
	}, nil
}

func (m *ModelsModule) executeUpdateManyAndReturn(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any) (any, error) {
	// Similar to updateMany but returns updated records
	// This would require fetching the updated records
	_ = ctx
	_ = model
	_ = modelName
	_ = options
	return nil, fmt.Errorf("updateManyAndReturn not yet implemented")
}

func (m *ModelsModule) executeUpsert(ctx context.Context, model types.ModelQuery, options map[string]any, modelName string, db types.Database) (any, error) {
	where, ok := options["where"]
	if !ok {
		return nil, fmt.Errorf("upsert requires 'where' field")
	}

	create, ok := options["create"]
	if !ok {
		return nil, fmt.Errorf("upsert requires 'create' field")
	}

	update, ok := options["update"]
	if !ok {
		return nil, fmt.Errorf("upsert requires 'update' field")
	}

	// First try to find the record
	selectQuery := model.Select()
	selectQuery = m.applySimpleWhereConditions(selectQuery, where).(types.SelectQuery)

	var existing map[string]any
	err := selectQuery.FindFirst(ctx, &existing)

	if err == nil && existing != nil {
		// Record exists, update it
		updateData := m.processUpdateData(update)
		updateQuery := model.Update(updateData)
		updateQuery = m.applySimpleWhereConditions(updateQuery, where).(types.UpdateQuery)
		_, err := updateQuery.Exec(ctx)
		if err != nil {
			return nil, err
		}
		// Merge update data with existing
		for k, v := range updateData.(map[string]any) {
			existing[k] = v
		}
		return existing, nil
	} else {
		// Record doesn't exist, create it
		createData := m.processNestedWrites(create, "create", modelName, db)
		insertQuery := model.Insert(createData)
		result, err := insertQuery.Exec(ctx)
		if err != nil {
			return nil, err
		}
		// Add ID to created data
		if createMap, ok := createData.(map[string]any); ok {
			createMap["id"] = result.LastInsertID
			return createMap, nil
		}
		return createData, nil
	}
}

// Delete operations

func (m *ModelsModule) executeDelete(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	where, ok := options["where"]
	if !ok {
		return nil, fmt.Errorf("delete requires 'where' field")
	}

	// First fetch the record to return it
	selectQuery := model.Select()
	selectQuery = m.applySimpleWhereConditions(selectQuery, where).(types.SelectQuery)

	var existing map[string]any
	err := selectQuery.FindFirst(ctx, &existing)
	if err != nil {
		return nil, err
	}

	// Now delete it
	deleteQuery := model.Delete()
	deleteQuery = m.applySimpleWhereConditions(deleteQuery, where).(types.DeleteQuery)

	_, err = deleteQuery.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return existing, nil
}

func (m *ModelsModule) executeDeleteMany(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any) (any, error) {
	_ = modelName // TODO: might be needed for future enhancements
	deleteQuery := model.Delete()

	// Apply where conditions
	if where, ok := options["where"]; ok {
		deleteQuery = m.applySimpleWhereConditions(deleteQuery, where).(types.DeleteQuery)
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

// applySimpleWhereConditions applies simple field equality conditions to a query
// It takes a where clause (expected to be map[string]any) and applies each field=value condition
// The function is generic and works with any query type that has Where() and WhereCondition() methods
func (m *ModelsModule) applySimpleWhereConditions(query any, where any) any {
	// Build condition using our proper condition builder
	condition := m.BuildCondition(where)
	if condition == nil {
		return query
	}

	// Apply the condition to the query using type assertion instead of reflection
	switch q := query.(type) {
	case types.SelectQuery:
		return q.WhereCondition(condition)
	case types.UpdateQuery:
		return q.WhereCondition(condition)
	case types.DeleteQuery:
		return q.WhereCondition(condition)
	case types.ModelQuery:
		return q.WhereCondition(condition)
	default:
		return query
	}
}

func (m *ModelsModule) processNestedWrites(data any, operation string, modelName string, db types.Database) any {
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

func (m *ModelsModule) processNestedCreate(fieldName string, fieldValue any, relation schema.Relation) any {
	// Handle nested create operations for different relation types
	switch valueMap := fieldValue.(type) {
	case map[string]any:
		// Check for nested operations
		if createData, hasCreate := valueMap["create"]; hasCreate {
			// Handle create operation
			return m.handleNestedCreate(fieldName, createData, relation)
		}
		if connectData, hasConnect := valueMap["connect"]; hasConnect {
			// Handle connect operation (link existing records)
			return m.handleNestedConnect(fieldName, connectData, relation)
		}
		if createManyData, hasCreateMany := valueMap["createMany"]; hasCreateMany {
			// Handle createMany operation
			return m.handleNestedCreateMany(fieldName, createManyData, relation)
		}
		// If no nested operation keywords, treat as direct data
		return fieldValue
	default:
		// Direct value (e.g., for foreign key assignment)
		return fieldValue
	}
}

func (m *ModelsModule) processNestedUpdate(fieldName string, fieldValue any, relation schema.Relation) any {
	// Handle nested update operations
	switch valueMap := fieldValue.(type) {
	case map[string]any:
		// Check for nested operations
		if createData, hasCreate := valueMap["create"]; hasCreate {
			return m.handleNestedCreate(fieldName, createData, relation)
		}
		if updateData, hasUpdate := valueMap["update"]; hasUpdate {
			return m.handleNestedUpdate(fieldName, updateData, relation)
		}
		if connectData, hasConnect := valueMap["connect"]; hasConnect {
			return m.handleNestedConnect(fieldName, connectData, relation)
		}
		if disconnectData, hasDisconnect := valueMap["disconnect"]; hasDisconnect {
			return m.handleNestedDisconnect(fieldName, disconnectData, relation)
		}
		if setData, hasSet := valueMap["set"]; hasSet {
			// Replace all related records
			return m.handleNestedSet(fieldName, setData, relation)
		}
		return fieldValue
	default:
		return fieldValue
	}
}

func (m *ModelsModule) handleNestedCreate(fieldName string, createData any, relation schema.Relation) any {
	// For now, we'll store the nested write information
	// The actual creation will happen in the driver implementation
	return map[string]any{
		"__nested_create": true,
		"relation":        fieldName,
		"data":            createData,
		"type":            relation.Type,
		"model":           relation.Model,
		"foreignKey":      relation.ForeignKey,
		"references":      relation.References,
	}
}

func (m *ModelsModule) handleNestedConnect(fieldName string, connectData any, relation schema.Relation) any {
	return map[string]any{
		"__nested_connect": true,
		"relation":         fieldName,
		"data":             connectData,
		"type":             relation.Type,
		"model":            relation.Model,
		"foreignKey":       relation.ForeignKey,
		"references":       relation.References,
	}
}

func (m *ModelsModule) handleNestedCreateMany(fieldName string, createManyData any, relation schema.Relation) any {
	return map[string]any{
		"__nested_create_many": true,
		"relation":             fieldName,
		"data":                 createManyData,
		"type":                 relation.Type,
		"model":                relation.Model,
		"foreignKey":           relation.ForeignKey,
		"references":           relation.References,
	}
}

func (m *ModelsModule) handleNestedUpdate(fieldName string, updateData any, relation schema.Relation) any {
	return map[string]any{
		"__nested_update": true,
		"relation":        fieldName,
		"data":            updateData,
		"type":            relation.Type,
		"model":           relation.Model,
		"foreignKey":      relation.ForeignKey,
		"references":      relation.References,
	}
}

func (m *ModelsModule) handleNestedDisconnect(fieldName string, disconnectData any, relation schema.Relation) any {
	return map[string]any{
		"__nested_disconnect": true,
		"relation":            fieldName,
		"data":                disconnectData,
		"type":                relation.Type,
		"model":               relation.Model,
		"foreignKey":          relation.ForeignKey,
		"references":          relation.References,
	}
}

func (m *ModelsModule) handleNestedSet(fieldName string, setData any, relation schema.Relation) any {
	return map[string]any{
		"__nested_set": true,
		"relation":     fieldName,
		"data":         setData,
		"type":         relation.Type,
		"model":        relation.Model,
		"foreignKey":   relation.ForeignKey,
		"references":   relation.References,
	}
}

func (m *ModelsModule) processUpdateData(data any) any {
	dataMap, ok := data.(map[string]any)
	if !ok {
		// Not a map, return as-is
		return data
	}

	// For now, just return the data as-is since we don't have complex atomic operations
	// This ensures the data is passed through correctly
	return dataMap
}

func (m *ModelsModule) extractFieldNames(selectFields any) []string {
	var fields []string
	if selectMap, ok := selectFields.(map[string]any); ok {
		for field, value := range selectMap {
			if include, ok := value.(bool); ok && include {
				fields = append(fields, field)
			}
		}
	}
	return fields
}

func (m *ModelsModule) applyInclude(query any, include any) any {
	// Handle relation loading
	selectQuery, ok := query.(types.SelectQuery)
	if !ok {
		return query // Query doesn't support includes
	}

	// Parse include options
	switch inc := include.(type) {
	case bool:
		// Simple boolean include not supported at top level
		return query
	case map[string]any:
		// Object format: { relationName: true } or { relationName: { include: {...} } }
		for relationName, relationOptions := range inc {
			// Check if it's a simple include (true) or nested
			switch opts := relationOptions.(type) {
			case bool:
				if opts {
					// Simple include
					selectQuery = selectQuery.Include(relationName)
				}
			case map[string]any:
				// Handle nested include with options
				includeOpts := m.parseNestedIncludes(relationName, opts)
				// Apply include options to the query
				for path, opt := range includeOpts {
					selectQuery = m.applyIncludeOption(selectQuery, path, opt)
				}
			}
		}
	}
	return selectQuery
}

// applyIncludeOption applies a single include option to the query
func (m *ModelsModule) applyIncludeOption(query types.SelectQuery, path string, opt *types.IncludeOption) types.SelectQuery {
	// Use the new IncludeWithOptions method
	return query.IncludeWithOptions(path, opt)
}

// parseNestedIncludes parses nested include options and returns include options
func (m *ModelsModule) parseNestedIncludes(relationName string, options map[string]any) map[string]*types.IncludeOption {
	result := make(map[string]*types.IncludeOption)
	hasNestedIncludes := false

	// Create the include option for this relation
	includeOpt := &types.IncludeOption{
		Path: relationName,
	}

	// Check for select fields (for selective loading)
	if selectFields, hasSelect := options["select"]; hasSelect {
		if selectMap, ok := selectFields.(map[string]any); ok {
			var fields []string
			for field, included := range selectMap {
				if inc, ok := included.(bool); ok && inc {
					fields = append(fields, field)
				}
			}
			includeOpt.Select = fields
		}
	}

	// Check for where conditions (for filtered loading)
	if whereCondition, hasWhere := options["where"]; hasWhere {
		// Build condition from where clause
		includeOpt.Where = m.BuildCondition(whereCondition)
	}

	// Check for orderBy
	if orderBy, hasOrderBy := options["orderBy"]; hasOrderBy {
		if orderMap, ok := orderBy.(map[string]any); ok {
			var orders []types.OrderByOption
			for field, direction := range orderMap {
				dir := types.ASC
				if dirStr, ok := direction.(string); ok && strings.ToLower(dirStr) == "desc" {
					dir = types.DESC
				}
				orders = append(orders, types.OrderByOption{
					Field:     field,
					Direction: dir,
				})
			}
			includeOpt.OrderBy = orders
		}
	}

	// Check for limit
	if limit, hasLimit := options["take"]; hasLimit {
		l := utils.ToInt(limit)
		includeOpt.Limit = &l
	}

	// Check for offset
	if skip, hasSkip := options["skip"]; hasSkip {
		o := utils.ToInt(skip)
		includeOpt.Offset = &o
	}

	// Check for nested includes
	if nestedInclude, hasInclude := options["include"]; hasInclude {
		switch nested := nestedInclude.(type) {
		case map[string]any:
			// Parse nested relations
			for nestedRelation, nestedOpts := range nested {
				hasNestedIncludes = true
				fullPath := relationName + "." + nestedRelation
				switch opts := nestedOpts.(type) {
				case bool:
					if opts {
						result[fullPath] = &types.IncludeOption{
							Path: fullPath,
						}
					}
				case map[string]any:
					// Recursively parse deeper nesting
					deeperIncludes := m.parseNestedIncludes(fullPath, opts)
					for k, v := range deeperIncludes {
						result[k] = v
					}
				}
			}
		}
	}

	// Only include the parent relation if there are no nested includes
	// This prevents duplicate joins when we have both "posts" and "posts.comments"
	if !hasNestedIncludes {
		result[relationName] = includeOpt
	}

	return result
}

// Check if error is a unique constraint violation
func isUniqueConstraintError(err error) bool {
	errStr := err.Error()
	return reflect.ValueOf(errStr).String() == "UNIQUE constraint failed" ||
		reflect.ValueOf(errStr).String() == "duplicate key"
}
