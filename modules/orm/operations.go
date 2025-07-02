package orm

import (
	"context"
	"fmt"
	"reflect"

	"github.com/rediwo/redi-orm/types"
)

// executeOperation executes a database operation based on the method name
func (m *ModelsModule) executeOperation(db types.Database, modelName, methodName string, options map[string]any) (any, error) {
	ctx := context.Background()
	model := db.Model(modelName)

	switch methodName {
	// Create operations
	case "create":
		return m.executeCreate(ctx, model, options)
	case "createMany":
		return m.executeCreateMany(ctx, model, modelName, options)
	case "createManyAndReturn":
		return m.executeCreateManyAndReturn(ctx, model, modelName, options)

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
		return m.executeGroupBy(ctx, model, modelName, options)

	// Update operations
	case "update":
		return m.executeUpdate(ctx, model, options)
	case "updateMany":
		return m.executeUpdateMany(ctx, model, modelName, options)
	case "updateManyAndReturn":
		return m.executeUpdateManyAndReturn(ctx, model, modelName, options)
	case "upsert":
		return m.executeUpsert(ctx, model, options)

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

func (m *ModelsModule) executeCreate(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
	data, ok := options["data"]
	if !ok {
		return nil, fmt.Errorf("create requires 'data' field")
	}

	// Handle nested creates
	processedData := m.processNestedWrites(data, "create")

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

func (m *ModelsModule) executeCreateMany(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any) (any, error) {
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
		processedData = append(processedData, m.processNestedWrites(item, "create"))
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

func (m *ModelsModule) executeCreateManyAndReturn(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any) (any, error) {
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
		processedItem := m.processNestedWrites(item, "create")
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
	if err := m.applyWhereConditions(query, where); err != nil {
		return nil, err
	}

	// Handle select fields
	if selectFields, ok := options["select"]; ok {
		fields := m.extractFieldNames(selectFields)
		query = model.Select(fields...)
	}

	// Handle include (relations)
	if include, ok := options["include"]; ok {
		m.applyInclude(query, include)
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

	// Apply where conditions if provided
	if where, ok := options["where"]; ok {
		if err := m.applyWhereConditions(query, where); err != nil {
			return nil, err
		}
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
		m.applyInclude(query, include)
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

	// Apply where conditions if provided
	if where, ok := options["where"]; ok {
		if err := m.applyWhereConditions(query, where); err != nil {
			return nil, err
		}
	}

	// Apply orderBy if provided
	if orderBy, ok := options["orderBy"]; ok {
		m.applyOrderBy(query, orderBy)
	}

	// Apply pagination
	if skip, ok := options["skip"]; ok {
		query = query.Offset(int(toInt64(skip)))
	}
	if take, ok := options["take"]; ok {
		query = query.Limit(int(toInt64(take)))
	}

	// Handle select fields
	if selectFields, ok := options["select"]; ok {
		fields := m.extractFieldNames(selectFields)
		query = model.Select(fields...)
	}

	// Handle include (relations)
	if include, ok := options["include"]; ok {
		m.applyInclude(query, include)
	}

	// Handle distinct
	if distinct, ok := options["distinct"].([]any); ok && len(distinct) > 0 {
		// Note: Distinct support would need to be added to the query builder
		// For now, this is a placeholder
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
					condition := m.buildCondition(map[string]any{field: value})
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
			model = model.WhereCondition(m.buildCondition(where))
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
		model = model.WhereCondition(m.buildCondition(where))
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

func (m *ModelsModule) executeGroupBy(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any) (any, error) {
	// GroupBy is complex and would require significant query builder changes
	// For now, return a placeholder
	return nil, fmt.Errorf("groupBy not yet implemented")
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
	if err := m.applyWhereConditions(query, where); err != nil {
		return nil, err
	}

	// Handle returning specific fields
	if selectFields, ok := options["select"]; ok {
		fields := m.extractFieldNames(selectFields)
		updateQuery := query.(types.UpdateQuery)
		updateQuery = updateQuery.Returning(fields...)
		query = updateQuery
	}

	result, err := query.(types.UpdateQuery).Exec(ctx)
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
	data, ok := options["data"]
	if !ok {
		return nil, fmt.Errorf("updateMany requires 'data' field")
	}

	// Process update data
	processedData := m.processUpdateData(data)

	query := model.Update(processedData)

	// Apply where conditions if provided
	if where, ok := options["where"]; ok {
		if err := m.applyWhereConditions(query, where); err != nil {
			return nil, err
		}
	}

	result, err := query.(types.UpdateQuery).Exec(ctx)
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
	return nil, fmt.Errorf("updateManyAndReturn not yet implemented")
}

func (m *ModelsModule) executeUpsert(ctx context.Context, model types.ModelQuery, options map[string]any) (any, error) {
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
	if err := m.applyWhereConditions(selectQuery, where); err != nil {
		return nil, err
	}

	var existing map[string]any
	err := selectQuery.FindFirst(ctx, &existing)

	if err == nil && existing != nil {
		// Record exists, update it
		updateData := m.processUpdateData(update)
		updateQuery := model.Update(updateData)
		if err := m.applyWhereConditions(updateQuery, where); err != nil {
			return nil, err
		}
		_, err := updateQuery.(types.UpdateQuery).Exec(ctx)
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
		createData := m.processNestedWrites(create, "create")
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
	if err := m.applyWhereConditions(selectQuery, where); err != nil {
		return nil, err
	}

	var existing map[string]any
	err := selectQuery.FindFirst(ctx, &existing)
	if err != nil {
		return nil, err
	}

	// Now delete it
	deleteQuery := model.Delete()
	if err := m.applyWhereConditions(deleteQuery, where); err != nil {
		return nil, err
	}

	_, err = deleteQuery.(types.DeleteQuery).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return existing, nil
}

func (m *ModelsModule) executeDeleteMany(ctx context.Context, model types.ModelQuery, modelName string, options map[string]any) (any, error) {
	deleteQuery := model.Delete()

	// Apply where conditions if provided
	if where, ok := options["where"]; ok {
		if err := m.applyWhereConditions(deleteQuery, where); err != nil {
			return nil, err
		}
	}

	result, err := deleteQuery.(types.DeleteQuery).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"count": result.RowsAffected,
	}, nil
}

// Helper functions

func (m *ModelsModule) processNestedWrites(data any, operation string) any {
	// Process nested create/connect/disconnect operations
	// For now, just return the data as-is
	return data
}

func (m *ModelsModule) processUpdateData(data any) any {
	dataMap, ok := data.(map[string]any)
	if !ok {
		return data
	}

	processed := make(map[string]any)
	for key, value := range dataMap {
		// Check for atomic operations
		if valueMap, ok := value.(map[string]any); ok {
			if inc, ok := valueMap["increment"]; ok {
				// This would need special handling in the query builder
				processed[key] = inc
			} else if dec, ok := valueMap["decrement"]; ok {
				// This would need special handling in the query builder
				processed[key] = dec
			} else {
				processed[key] = value
			}
		} else {
			processed[key] = value
		}
	}

	return processed
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

func (m *ModelsModule) applyInclude(query any, include any) {
	// Handle relation loading
	// This would require relation support in the query builder
	// For now, this is a placeholder
}

// Helper to convert numeric types to int64
func toInt64(v any) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}

// Check if error is a unique constraint violation
func isUniqueConstraintError(err error) bool {
	errStr := err.Error()
	return reflect.ValueOf(errStr).String() == "UNIQUE constraint failed" ||
		reflect.ValueOf(errStr).String() == "duplicate key"
}
