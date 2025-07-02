package orm

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

// applyWhereConditions applies where conditions to a query
func (m *ModelsModule) applyWhereConditions(query interface{}, where interface{}) error {
	// Build conditions with proper field name resolution
	var conditions []types.Condition
	
	// Check if it's a simple field map
	if whereMap, ok := where.(map[string]interface{}); ok {
		for field, value := range whereMap {
			// Check if it's a simple field equality (not an operator object)
			if _, isOperator := value.(map[string]interface{}); !isOperator {
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
				// For complex conditions, use the existing buildCondition
				condition := m.buildCondition(map[string]interface{}{field: value})
				conditions = append(conditions, condition)
			}
		}
	} else {
		// For non-map where clauses, use buildCondition
		condition := m.buildCondition(where)
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

// buildCondition builds a condition from JavaScript where object
func (m *ModelsModule) buildCondition(where interface{}) types.Condition {
	whereMap, ok := where.(map[string]interface{})
	if !ok {
		return nil
	}

	var conditions []types.Condition

	for field, value := range whereMap {
		// Handle special operators
		switch field {
		case "OR":
			if orConditions, ok := value.([]interface{}); ok {
				var orConds []types.Condition
				for _, orCond := range orConditions {
					orConds = append(orConds, m.buildCondition(orCond))
				}
				if len(orConds) > 0 {
					conditions = append(conditions, types.NewOrCondition(orConds...))
				}
			}
		case "AND":
			if andConditions, ok := value.([]interface{}); ok {
				var andConds []types.Condition
				for _, andCond := range andConditions {
					andConds = append(andConds, m.buildCondition(andCond))
				}
				if len(andConds) > 0 {
					conditions = append(conditions, types.NewAndCondition(andConds...))
				}
			}
		case "NOT":
			notCond := m.buildCondition(value)
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
func (m *ModelsModule) buildFieldCondition(field string, value interface{}) types.Condition {
	// Check if value is an operator object
	if valueMap, ok := value.(map[string]interface{}); ok {
		// Handle operators
		for op, val := range valueMap {
			switch op {
			case "equals":
				return types.NewBaseCondition(field+" = ?", val)
			case "not":
				return types.NewBaseCondition(field+" != ?", val)
			case "in":
				if values, ok := val.([]interface{}); ok {
					return m.buildInCondition(field, values)
				}
				return nil
			case "notIn":
				if values, ok := val.([]interface{}); ok {
					return m.buildNotInCondition(field, values)
				}
				return nil
			case "lt":
				return types.NewBaseCondition(field+" < ?", val)
			case "lte":
				return types.NewBaseCondition(field+" <= ?", val)
			case "gt":
				return types.NewBaseCondition(field+" > ?", val)
			case "gte":
				return types.NewBaseCondition(field+" >= ?", val)
			case "contains":
				return types.NewBaseCondition(field+" LIKE ?", "%"+fmt.Sprintf("%v", val)+"%")
			case "startsWith":
				return types.NewBaseCondition(field+" LIKE ?", fmt.Sprintf("%v", val)+"%")
			case "endsWith":
				return types.NewBaseCondition(field+" LIKE ?", "%"+fmt.Sprintf("%v", val))
			}
		}
		return nil
	}

	// Direct value comparison
	if value == nil {
		return types.NewBaseCondition(field + " IS NULL")
	}
	return types.NewBaseCondition(field+" = ?", value)
}

// buildInCondition builds an IN condition
func (m *ModelsModule) buildInCondition(field string, values []interface{}) types.Condition {
	if len(values) == 0 {
		return types.NewBaseCondition("1 = 0") // Always false
	}
	
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
	}
	
	sql := fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", "))
	return types.NewBaseCondition(sql, values...)
}

// buildNotInCondition builds a NOT IN condition
func (m *ModelsModule) buildNotInCondition(field string, values []interface{}) types.Condition {
	if len(values) == 0 {
		return types.NewBaseCondition("1 = 1") // Always true
	}
	
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
	}
	
	sql := fmt.Sprintf("%s NOT IN (%s)", field, strings.Join(placeholders, ", "))
	return types.NewBaseCondition(sql, values...)
}

// applyOrderBy applies orderBy conditions to a query
func (m *ModelsModule) applyOrderBy(query interface{}, orderBy interface{}) {
	switch q := query.(type) {
	case types.SelectQuery:
		m.applyOrderByToQuery(q, orderBy)
	case types.ModelQuery:
		m.applyOrderByToQuery(q, orderBy)
	}
}

// applyOrderByToQuery handles different orderBy formats
func (m *ModelsModule) applyOrderByToQuery(query interface{}, orderBy interface{}) {
	// Handle single orderBy object: { field: 'asc' }
	if orderMap, ok := orderBy.(map[string]interface{}); ok {
		for field, direction := range orderMap {
			dir := types.ASC
			if dirStr, ok := direction.(string); ok && dirStr == "desc" {
				dir = types.DESC
			}
			
			switch q := query.(type) {
			case types.SelectQuery:
				q.OrderBy(field, dir)
			case types.ModelQuery:
				q.OrderBy(field, dir)
			}
		}
		return
	}

	// Handle array of orderBy objects: [{ field: 'asc' }, { field2: 'desc' }]
	if orderArray, ok := orderBy.([]interface{}); ok {
		for _, item := range orderArray {
			if orderMap, ok := item.(map[string]interface{}); ok {
				for field, direction := range orderMap {
					dir := types.ASC
					if dirStr, ok := direction.(string); ok && dirStr == "desc" {
						dir = types.DESC
					}
					
					switch q := query.(type) {
					case types.SelectQuery:
						q.OrderBy(field, dir)
					case types.ModelQuery:
						q.OrderBy(field, dir)
					}
				}
			}
		}
	}
}

