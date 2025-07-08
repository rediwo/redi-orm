package services

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/database"
	"github.com/rediwo/redi-orm/rest/types"
	ormTypes "github.com/rediwo/redi-orm/types"
)

// QueryBuilder builds ORM queries from REST parameters
type QueryBuilder struct{}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{}
}

// BuildFindQuery builds a select query from parameters
func (b *QueryBuilder) BuildFindQuery(db database.Database, modelName string, params *types.QueryParams) (ormTypes.SelectQuery, error) {
	query := db.Model(modelName).Select()

	// Apply field selection
	fields := b.getFields(params)
	if len(fields) > 0 {
		// The Select method is already called on db.Model(), so we need to recreate
		query = db.Model(modelName).Select(fields...)
	}

	// Apply where conditions
	if params.Where != nil {
		query = b.applyWhereConditions(query, params.Where)
	} else if params.Filter != nil {
		query = b.applyWhereConditions(query, params.Filter)
	}

	// Apply search
	if params.Search != "" || params.Q != "" {
		searchTerm := params.Search
		if searchTerm == "" {
			searchTerm = params.Q
		}
		query = b.applySearch(query, searchTerm)
	}

	// Apply sorting
	query = b.applySorting(query, params)

	// Apply pagination
	if params.Page > 0 {
		offset := (params.Page - 1) * params.Limit
		query = query.Offset(offset).Limit(params.Limit)
	} else if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}

	// Apply includes
	if params.Include != nil {
		query = b.ApplyIncludes(query, params.Include)
	}

	return query, nil
}

// BuildCountQuery builds a count query from parameters
func (b *QueryBuilder) BuildCountQuery(db database.Database, modelName string, params *types.QueryParams) ormTypes.SelectQuery {
	// Use a select query to build the count
	query := db.Model(modelName).Select()

	// Apply where conditions
	if params.Where != nil {
		query = b.applyWhereConditions(query, params.Where)
	} else if params.Filter != nil {
		query = b.applyWhereConditions(query, params.Filter)
	}

	// Apply search
	if params.Search != "" || params.Q != "" {
		searchTerm := params.Search
		if searchTerm == "" {
			searchTerm = params.Q
		}
		query = b.applySearch(query, searchTerm)
	}

	return query
}

// getFields extracts fields from parameters
func (b *QueryBuilder) getFields(params *types.QueryParams) []string {
	if len(params.Select) > 0 {
		return params.Select
	}
	if len(params.Fields) > 0 {
		return params.Fields
	}
	return nil
}

// applyWhereConditions applies where conditions to query
func (b *QueryBuilder) applyWhereConditions(query ormTypes.SelectQuery, where map[string]any) ormTypes.SelectQuery {
	for field, value := range where {
		switch v := value.(type) {
		case map[string]any:
			// Handle operators
			for op, val := range v {
				query = b.applyOperator(query, field, op, val)
			}
		default:
			// Simple equality
			query = query.WhereCondition(query.Where(field).Equals(value))
		}
	}
	return query
}

// applyOperator applies a specific operator
func (b *QueryBuilder) applyOperator(query ormTypes.SelectQuery, field, operator string, value any) ormTypes.SelectQuery {
	condition := query.Where(field)

	switch operator {
	case "eq", "equals":
		return query.WhereCondition(condition.Equals(value))
	case "ne", "not":
		// Use Equals and then Not on the condition
		return query.WhereCondition(condition.Equals(value).Not())
	case "gt":
		return query.WhereCondition(condition.GreaterThan(value))
	case "gte":
		return query.WhereCondition(condition.GreaterThanOrEqual(value))
	case "lt":
		return query.WhereCondition(condition.LessThan(value))
	case "lte":
		return query.WhereCondition(condition.LessThanOrEqual(value))
	case "in":
		if arr, ok := value.([]any); ok {
			return query.WhereCondition(condition.In(arr...))
		}
	case "notIn":
		if arr, ok := value.([]any); ok {
			return query.WhereCondition(condition.NotIn(arr...))
		}
	case "contains":
		return query.WhereCondition(condition.Contains(fmt.Sprintf("%v", value)))
	case "startsWith":
		return query.WhereCondition(condition.StartsWith(fmt.Sprintf("%v", value)))
	case "endsWith":
		return query.WhereCondition(condition.EndsWith(fmt.Sprintf("%v", value)))
	case "like":
		return query.WhereCondition(condition.Like(fmt.Sprintf("%v", value)))
	case "notNull":
		if b, ok := value.(bool); ok && b {
			return query.WhereCondition(condition.IsNotNull())
		}
	case "null":
		if b, ok := value.(bool); ok && b {
			return query.WhereCondition(condition.IsNull())
		}
	}

	// Default to equality
	return query.WhereCondition(condition.Equals(value))
}

// applySearch applies search conditions
func (b *QueryBuilder) applySearch(query ormTypes.SelectQuery, searchTerm string) ormTypes.SelectQuery {
	// This is a simple implementation - in practice, you'd want to
	// search across multiple text fields based on the model schema
	// For now, we'll leave it as a placeholder
	return query
}

// applySorting applies sorting to query
func (b *QueryBuilder) applySorting(query ormTypes.SelectQuery, params *types.QueryParams) ormTypes.SelectQuery {
	sortFields := params.Sort
	if len(sortFields) == 0 {
		sortFields = params.OrderBy
	}

	for _, field := range sortFields {
		if strings.HasPrefix(field, "-") {
			// Descending order
			query = query.OrderBy(field[1:], ormTypes.DESC)
		} else {
			// Ascending order
			query = query.OrderBy(field, ormTypes.ASC)
		}
	}

	return query
}

// ApplyIncludes applies include conditions to query
func (b *QueryBuilder) ApplyIncludes(query ormTypes.SelectQuery, include any) ormTypes.SelectQuery {
	switch v := include.(type) {
	case []string:
		// Simple string array of relations
		return query.Include(v...)
	case map[string]any:
		// For complex includes with options, we need to convert to simple strings for now
		// TODO: Support IncludeWithOptions for nested includes
		var relations []string
		for rel := range v {
			relations = append(relations, rel)
		}
		return query.Include(relations...)
	case []any:
		// Convert array to strings
		var relations []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				relations = append(relations, str)
			}
		}
		return query.Include(relations...)
	}
	return query
}
