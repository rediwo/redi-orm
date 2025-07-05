package agile

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// applySimpleWhereConditions applies simple field equality conditions to a query
func applySimpleWhereConditions(query any, where any) any {
	// Build condition using our proper condition builder
	condition := BuildCondition(where)
	if condition == nil {
		return query
	}

	// Apply the condition to the query using type assertion
	switch q := query.(type) {
	case types.SelectQuery:
		return q.WhereCondition(condition)
	case types.UpdateQuery:
		return q.WhereCondition(condition)
	case types.DeleteQuery:
		return q.WhereCondition(condition)
	case types.ModelQuery:
		return q.WhereCondition(condition)
	case types.AggregationQuery:
		return q.WhereCondition(condition)
	default:
		return query
	}
}

// BuildCondition builds a condition from where object
func BuildCondition(where any) types.Condition {
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
					orConds = append(orConds, BuildCondition(orCond))
				}
				if len(orConds) > 0 {
					conditions = append(conditions, types.NewOrCondition(orConds...))
				}
			}
		case "AND":
			if andConditions, ok := value.([]any); ok {
				var andConds []types.Condition
				for _, andCond := range andConditions {
					andConds = append(andConds, BuildCondition(andCond))
				}
				if len(andConds) > 0 {
					conditions = append(conditions, types.NewAndCondition(andConds...))
				}
			}
		case "NOT":
			notCond := BuildCondition(value)
			if notCond != nil {
				conditions = append(conditions, types.NewNotCondition(notCond))
			}
		default:
			// Regular field condition
			fieldCond := buildFieldCondition(field, value)
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
func buildFieldCondition(field string, value any) types.Condition {
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

// applyOrderBy applies orderBy conditions to a query
func applyOrderBy(query any, orderBy any) any {
	return applyOrderByToQuery(query, orderBy)
}

// applyOrderByToQuery handles different orderBy formats
func applyOrderByToQuery(query any, orderBy any) any {
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

// applyInclude applies include options to a query
func applyInclude(query any, include any) any {
	selectQuery, ok := query.(types.SelectQuery)
	if !ok {
		return query
	}

	// Handle different include formats
	if includeMap, ok := include.(map[string]any); ok {
		for relationName, opts := range includeMap {
			switch opts := opts.(type) {
			case bool:
				if opts {
					// Simple include
					selectQuery = selectQuery.Include(relationName)
				}
			case map[string]any:
				// Handle nested include with options
				includeOpts := parseNestedIncludes(relationName, opts)
				// Apply include options to the query
				for path, opt := range includeOpts {
					selectQuery = applyIncludeOption(selectQuery, path, opt)
				}
			}
		}
	}
	return selectQuery
}

// applyIncludeOption applies a single include option to the query
func applyIncludeOption(query types.SelectQuery, path string, opt *types.IncludeOption) types.SelectQuery {
	// Use the new IncludeWithOptions method
	return query.IncludeWithOptions(path, opt)
}

// parseNestedIncludes parses nested include options and returns include options
func parseNestedIncludes(relationName string, options map[string]any) map[string]*types.IncludeOption {
	result := make(map[string]*types.IncludeOption)

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
		includeOpt.Where = BuildCondition(whereCondition)
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
					deeperIncludes := parseNestedIncludes(fullPath, opts)
					for k, v := range deeperIncludes {
						result[k] = v
					}
				}
			}
		}
	}

	// Always include the parent relation
	// The join builder will handle deduplication if needed
	result[relationName] = includeOpt

	return result
}
