package orm

import (
	"fmt"

	"github.com/rediwo/redi-orm/types"
)

// applyWhereConditions applies where conditions to a query
func (m *ModelsModule) applyWhereConditions(query any, where any) error {
	// Build conditions with proper field name resolution
	var conditions []types.Condition

	// Check if it's a simple field map
	if whereMap, ok := where.(map[string]any); ok {
		for field, value := range whereMap {
			// Check if it's a simple field equality (not an operator object)
			if _, isOperator := value.(map[string]any); !isOperator {
				// Get a field condition builder based on query type
				var fieldCond types.FieldCondition
				switch q := query.(type) {
				case types.SelectQuery:
					fieldCond = q.Where(field)
				case types.UpdateQuery:
					fieldCond = q.Where(field)
				case types.DeleteQuery:
					fieldCond = q.Where(field)
				case types.ModelQuery:
					fieldCond = q.Where(field)
				default:
					return fmt.Errorf("unsupported query type for where conditions")
				}

				// Build the condition
				condition := fieldCond.Equals(value)
				conditions = append(conditions, condition)
			} else {
				// For complex conditions, use the existing BuildCondition
				condition := m.BuildCondition(map[string]any{field: value})
				conditions = append(conditions, condition)
			}
		}
	} else {
		// For non-map where clauses, use buildCondition
		condition := m.BuildCondition(where)
		if condition != nil {
			conditions = append(conditions, condition)
		}
	}

	// Apply the conditions
	if len(conditions) > 0 {
		var finalCondition types.Condition
		if len(conditions) == 1 {
			finalCondition = conditions[0]
		} else {
			finalCondition = types.NewAndCondition(conditions...)
		}

		switch q := query.(type) {
		case types.SelectQuery:
			q.WhereCondition(finalCondition)
		case types.UpdateQuery:
			q.WhereCondition(finalCondition)
		case types.DeleteQuery:
			q.WhereCondition(finalCondition)
		case types.ModelQuery:
			q.WhereCondition(finalCondition)
		}
	}

	return nil
}

// BuildCondition builds a condition from JavaScript where object
func (m *ModelsModule) BuildCondition(where any) types.Condition {
	whereMap, ok := where.(map[string]any)
	if !ok {
		return nil
	}

	var conditions []types.Condition

	for field, value := range whereMap {
		// Handle special operators
		switch field {
		case "OR":
			if orConditions, ok := value.([]any); ok {
				var orConds []types.Condition
				for _, orCond := range orConditions {
					orConds = append(orConds, m.BuildCondition(orCond))
				}
				if len(orConds) > 0 {
					conditions = append(conditions, types.NewOrCondition(orConds...))
				}
			}
		case "AND":
			if andConditions, ok := value.([]any); ok {
				var andConds []types.Condition
				for _, andCond := range andConditions {
					andConds = append(andConds, m.BuildCondition(andCond))
				}
				if len(andConds) > 0 {
					conditions = append(conditions, types.NewAndCondition(andConds...))
				}
			}
		case "NOT":
			notCond := m.BuildCondition(value)
			if notCond != nil {
				conditions = append(conditions, types.NewNotCondition(notCond))
			}
		default:
			// Regular field condition
			fieldCond := m.buildFieldCondition(field, value)
			if fieldCond != nil {
				conditions = append(conditions, fieldCond)
			}
		}
	}

	if len(conditions) == 0 {
		return nil
	}
	if len(conditions) == 1 {
		return conditions[0]
	}
	return types.NewAndCondition(conditions...)
}

// buildFieldCondition builds a field condition
func (m *ModelsModule) buildFieldCondition(field string, value any) types.Condition {
	// Create a field condition that supports proper mapping
	fieldCond := types.NewFieldCondition("", field)

	// Check if value is an operator object
	if valueMap, ok := value.(map[string]any); ok {
		// Handle operators - collect all conditions for this field
		var fieldConditions []types.Condition
		
		for op, val := range valueMap {
			var cond types.Condition
			switch op {
			case "equals":
				cond = fieldCond.Equals(val)
			case "not":
				if val == nil {
					cond = fieldCond.IsNotNull()
				} else {
					cond = fieldCond.NotEquals(val)
				}
			case "in":
				if values, ok := val.([]any); ok {
					cond = fieldCond.In(values...)
				}
			case "notIn":
				if values, ok := val.([]any); ok {
					cond = fieldCond.NotIn(values...)
				}
			case "lt":
				cond = fieldCond.LessThan(val)
			case "lte":
				cond = fieldCond.LessThanOrEqual(val)
			case "gt":
				cond = fieldCond.GreaterThan(val)
			case "gte":
				cond = fieldCond.GreaterThanOrEqual(val)
			case "contains":
				if strVal, ok := val.(string); ok {
					cond = fieldCond.Contains(strVal)
				} else {
					cond = fieldCond.Contains(fmt.Sprintf("%v", val))
				}
			case "startsWith":
				if strVal, ok := val.(string); ok {
					cond = fieldCond.StartsWith(strVal)
				} else {
					cond = fieldCond.StartsWith(fmt.Sprintf("%v", val))
				}
			case "endsWith":
				if strVal, ok := val.(string); ok {
					cond = fieldCond.EndsWith(strVal)
				} else {
					cond = fieldCond.EndsWith(fmt.Sprintf("%v", val))
				}
			}
			
			if cond != nil {
				fieldConditions = append(fieldConditions, cond)
			}
		}
		
		// If we have multiple conditions for the same field, combine them with AND
		if len(fieldConditions) == 0 {
			return nil
		}
		if len(fieldConditions) == 1 {
			return fieldConditions[0]
		}
		return types.NewAndCondition(fieldConditions...)
	}

	// Direct value comparison
	if value == nil {
		return fieldCond.IsNull()
	}
	return fieldCond.Equals(value)
}

// buildInCondition builds an IN condition
func (m *ModelsModule) buildInCondition(field string, values []any) types.Condition {
	fieldCond := types.NewFieldCondition("", field)
	return fieldCond.In(values...)
}

// buildNotInCondition builds a NOT IN condition
func (m *ModelsModule) buildNotInCondition(field string, values []any) types.Condition {
	fieldCond := types.NewFieldCondition("", field)
	return fieldCond.NotIn(values...)
}

// applyOrderBy applies orderBy conditions to a query
func (m *ModelsModule) applyOrderBy(query any, orderBy any) any {
	return m.applyOrderByToQuery(query, orderBy)
}

// applyOrderByToQuery handles different orderBy formats
func (m *ModelsModule) applyOrderByToQuery(query any, orderBy any) any {
	// Handle single orderBy object: { field: 'asc' }
	if orderMap, ok := orderBy.(map[string]any); ok {
		for field, direction := range orderMap {
			dir := types.ASC
			if dirStr, ok := direction.(string); ok && dirStr == "desc" {
				dir = types.DESC
			}

			switch q := query.(type) {
			case types.SelectQuery:
				query = q.OrderBy(field, dir)
			case types.ModelQuery:
				query = q.OrderBy(field, dir)
			}
		}
		return query
	}

	// Handle array of orderBy objects: [{ field: 'asc' }, { field2: 'desc' }]
	if orderArray, ok := orderBy.([]any); ok {
		for _, item := range orderArray {
			if orderMap, ok := item.(map[string]any); ok {
				for field, direction := range orderMap {
					dir := types.ASC
					if dirStr, ok := direction.(string); ok && dirStr == "desc" {
						dir = types.DESC
					}

					switch q := query.(type) {
					case types.SelectQuery:
						query = q.OrderBy(field, dir)
					case types.ModelQuery:
						query = q.OrderBy(field, dir)
					}
				}
			}
		}
		return query
	}
	
	return query
}
