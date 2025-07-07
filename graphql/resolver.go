package graphql

import (
	"context"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// createFindUniqueResolver creates a resolver for findUnique queries
func createFindUniqueResolver(db types.Database, modelName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		ctx := p.Context
		where := p.Args["where"].(map[string]any)

		// Build where conditions
		conditions := buildWhereConditions(where)

		// Execute query using the query builder
		query := db.Model(modelName).Select()

		// Apply where conditions
		for field, value := range conditions {
			// Handle operators
			if operators, ok := value.(map[string]any); ok {
				for op, val := range operators {
					switch op {
					case "equals":
						query = query.WhereCondition(query.Where(field).Equals(val))
					case "contains":
						query = query.WhereCondition(query.Where(field).Contains(val.(string)))
					case "startsWith":
						query = query.WhereCondition(query.Where(field).StartsWith(val.(string)))
					case "endsWith":
						query = query.WhereCondition(query.Where(field).EndsWith(val.(string)))
					case "gt":
						query = query.WhereCondition(query.Where(field).GreaterThan(val))
					case "gte":
						query = query.WhereCondition(query.Where(field).GreaterThanOrEqual(val))
					case "lt":
						query = query.WhereCondition(query.Where(field).LessThan(val))
					case "lte":
						query = query.WhereCondition(query.Where(field).LessThanOrEqual(val))
					case "in":
						query = query.WhereCondition(query.Where(field).In(val))
					case "notIn":
						query = query.WhereCondition(query.Where(field).NotIn(val))
					}
				}
			} else {
				// Direct equals
				query = query.WhereCondition(query.Where(field).Equals(value))
			}
		}

		// Execute and return first result
		var results []map[string]any
		err := query.Limit(1).FindMany(ctx, &results)
		if err != nil {
			return nil, err
		}

		if len(results) == 0 {
			return nil, nil
		}

		return results[0], nil
	}
}

// createFindManyResolver creates a resolver for findMany queries
func createFindManyResolver(db types.Database, modelName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		ctx := p.Context

		// Build query
		query := db.Model(modelName).Select()

		// Apply where conditions
		if where, ok := p.Args["where"].(map[string]any); ok {
			conditions := buildWhereConditions(where)
			for field, value := range conditions {
				// Handle operators
				if operators, ok := value.(map[string]any); ok {
					for op, val := range operators {
						switch op {
						case "equals":
							query = query.WhereCondition(query.Where(field).Equals(val))
						case "contains":
							query = query.WhereCondition(query.Where(field).Contains(val.(string)))
						case "startsWith":
							query = query.WhereCondition(query.Where(field).StartsWith(val.(string)))
						case "endsWith":
							query = query.WhereCondition(query.Where(field).EndsWith(val.(string)))
						case "gt":
							query = query.WhereCondition(query.Where(field).GreaterThan(val))
						case "gte":
							query = query.WhereCondition(query.Where(field).GreaterThanOrEqual(val))
						case "lt":
							query = query.WhereCondition(query.Where(field).LessThan(val))
						case "lte":
							query = query.WhereCondition(query.Where(field).LessThanOrEqual(val))
						case "in":
							query = query.WhereCondition(query.Where(field).In(val))
						case "notIn":
							query = query.WhereCondition(query.Where(field).NotIn(val))
						}
					}
				} else {
					// Direct equals
					query = query.WhereCondition(query.Where(field).Equals(value))
				}
			}
		}

		// Apply orderBy
		if orderBy, ok := p.Args["orderBy"].(map[string]any); ok {
			for field, direction := range orderBy {
				order := types.ASC
				if dir, ok := direction.(string); ok && dir == "DESC" {
					order = types.DESC
				}
				query = query.OrderBy(field, order)
			}
		}

		// Apply limit
		if limit, ok := p.Args["limit"].(int); ok {
			query = query.Limit(limit)
		}

		// Apply offset
		if offset, ok := p.Args["offset"].(int); ok {
			query = query.Offset(offset)
		}

		// Execute query
		var results []map[string]any
		err := query.FindMany(ctx, &results)
		if err != nil {
			return nil, err
		}

		return results, nil
	}
}

// createCountResolver creates a resolver for count queries
func createCountResolver(db types.Database, modelName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		ctx := p.Context

		// Build query
		query := db.Model(modelName).Select()

		// Apply where conditions
		if where, ok := p.Args["where"].(map[string]any); ok {
			conditions := buildWhereConditions(where)
			for field, value := range conditions {
				// Handle operators
				if operators, ok := value.(map[string]any); ok {
					for op, val := range operators {
						switch op {
						case "equals":
							query = query.WhereCondition(query.Where(field).Equals(val))
						case "contains":
							query = query.WhereCondition(query.Where(field).Contains(val.(string)))
						case "startsWith":
							query = query.WhereCondition(query.Where(field).StartsWith(val.(string)))
						case "endsWith":
							query = query.WhereCondition(query.Where(field).EndsWith(val.(string)))
						case "gt":
							query = query.WhereCondition(query.Where(field).GreaterThan(val))
						case "gte":
							query = query.WhereCondition(query.Where(field).GreaterThanOrEqual(val))
						case "lt":
							query = query.WhereCondition(query.Where(field).LessThan(val))
						case "lte":
							query = query.WhereCondition(query.Where(field).LessThanOrEqual(val))
						case "in":
							query = query.WhereCondition(query.Where(field).In(val))
						case "notIn":
							query = query.WhereCondition(query.Where(field).NotIn(val))
						}
					}
				} else {
					// Direct equals
					query = query.WhereCondition(query.Where(field).Equals(value))
				}
			}
		}

		// Execute count
		count, err := query.Count(ctx)
		if err != nil {
			return nil, err
		}

		return count, nil
	}
}

// createCreateResolver creates a resolver for create mutations
func createCreateResolver(db types.Database, modelName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		ctx := p.Context
		data := p.Args["data"].(map[string]any)

		// Debug: log the data being inserted
		// fmt.Printf("Creating %s with data: %+v\n", modelName, data)

		// Execute insert
		result, err := db.Model(modelName).Insert(data).Exec(ctx)
		if err != nil {
			return nil, err
		}

		// Always fetch the created record to ensure proper field mapping
		if result.LastInsertID > 0 {
			query := db.Model(modelName).Select()
			query = query.WhereCondition(query.Where("id").Equals(result.LastInsertID))

			var results []map[string]any
			err = query.Limit(1).FindMany(ctx, &results)
			if err != nil {
				return nil, err
			}

			if len(results) > 0 {
				return results[0], nil
			}
		}

		// Return the data that was inserted (fallback)
		return data, nil
	}
}

// createUpdateResolver creates a resolver for update mutations
func createUpdateResolver(db types.Database, modelName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		ctx := p.Context
		where := p.Args["where"].(map[string]any)
		data := p.Args["data"].(map[string]any)

		// Build where conditions
		conditions := buildWhereConditions(where)

		// Execute update
		query := db.Model(modelName).Update(data)

		// Apply where conditions
		for field, value := range conditions {
			// Handle operators
			if operators, ok := value.(map[string]any); ok {
				for op, val := range operators {
					switch op {
					case "equals":
						query = query.WhereCondition(query.Where(field).Equals(val))
					default:
						// For update, we usually just need equals
						query = query.WhereCondition(query.Where(field).Equals(val))
					}
				}
			} else {
				// Direct equals
				query = query.WhereCondition(query.Where(field).Equals(value))
			}
		}

		// If the driver supports returning, use it
		if db.GetCapabilities().SupportsReturning() {
			// Add all fields to returning
			modelSchema, err := db.GetSchema(modelName)
			if err == nil && modelSchema != nil {
				var fieldNames []string
				for _, field := range modelSchema.Fields {
					fieldNames = append(fieldNames, field.Name)
				}
				query = query.Returning(fieldNames...)
			}

			var updated map[string]any
			err = query.ExecAndReturn(ctx, &updated)
			if err != nil {
				return nil, err
			}
			return updated, nil
		}

		// Otherwise, execute update and fetch the record
		_, err := query.Exec(ctx)
		if err != nil {
			return nil, err
		}

		// Fetch the updated record
		selectQuery := db.Model(modelName).Select()
		for field, value := range conditions {
			if operators, ok := value.(map[string]any); ok {
				for _, val := range operators {
					selectQuery = selectQuery.WhereCondition(selectQuery.Where(field).Equals(val))
				}
			} else {
				selectQuery = selectQuery.WhereCondition(selectQuery.Where(field).Equals(value))
			}
		}

		var results []map[string]any
		err = selectQuery.Limit(1).FindMany(ctx, &results)
		if err != nil {
			return nil, err
		}

		if len(results) > 0 {
			return results[0], nil
		}

		return nil, fmt.Errorf("record not found after update")
	}
}

// createDeleteResolver creates a resolver for delete mutations
func createDeleteResolver(db types.Database, modelName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		ctx := p.Context
		where := p.Args["where"].(map[string]any)

		// Build where conditions
		conditions := buildWhereConditions(where)

		// First fetch the record before deletion
		selectQuery := db.Model(modelName).Select()
		for field, value := range conditions {
			// Handle operators
			if operators, ok := value.(map[string]any); ok {
				for op, val := range operators {
					if op == "equals" {
						selectQuery = selectQuery.WhereCondition(selectQuery.Where(field).Equals(val))
					}
				}
			} else {
				// Direct equals
				selectQuery = selectQuery.WhereCondition(selectQuery.Where(field).Equals(value))
			}
		}

		var results []map[string]any
		err := selectQuery.Limit(1).FindMany(ctx, &results)
		if err != nil {
			return nil, err
		}

		if len(results) == 0 {
			return nil, fmt.Errorf("record not found")
		}

		record := results[0]

		// Execute delete
		deleteQuery := db.Model(modelName).Delete()
		for field, value := range conditions {
			// Handle operators
			if operators, ok := value.(map[string]any); ok {
				for op, val := range operators {
					if op == "equals" {
						deleteQuery = deleteQuery.WhereCondition(deleteQuery.Where(field).Equals(val))
					}
				}
			} else {
				// Direct equals
				deleteQuery = deleteQuery.WhereCondition(deleteQuery.Where(field).Equals(value))
			}
		}

		_, err = deleteQuery.Exec(ctx)
		if err != nil {
			return nil, err
		}

		return record, nil
	}
}

// createCreateManyResolver creates a resolver for createMany mutations
func createCreateManyResolver(db types.Database, modelName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		ctx := p.Context
		dataList := p.Args["data"].([]any)

		// Convert to slice of maps
		var records []map[string]any
		for _, item := range dataList {
			records = append(records, item.(map[string]any))
		}

		// Execute createMany in a transaction
		var count int64
		err := db.Transaction(ctx, func(tx types.Transaction) error {
			for _, record := range records {
				result, err := tx.Model(modelName).Insert(record).Exec(ctx)
				if err != nil {
					return err
				}
				count += result.RowsAffected
			}
			return nil
		})

		if err != nil {
			return nil, err
		}

		return map[string]any{"count": count}, nil
	}
}

// createUpdateManyResolver creates a resolver for updateMany mutations
func createUpdateManyResolver(db types.Database, modelName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		ctx := p.Context
		data := p.Args["data"].(map[string]any)

		// Build query
		query := db.Model(modelName).Update(data)

		// Apply where conditions
		if where, ok := p.Args["where"].(map[string]any); ok {
			conditions := buildWhereConditions(where)
			for field, value := range conditions {
				// Handle operators
				if operators, ok := value.(map[string]any); ok {
					for op, val := range operators {
						switch op {
						case "equals":
							query = query.WhereCondition(query.Where(field).Equals(val))
						case "contains":
							query = query.WhereCondition(query.Where(field).Contains(val.(string)))
						case "startsWith":
							query = query.WhereCondition(query.Where(field).StartsWith(val.(string)))
						case "endsWith":
							query = query.WhereCondition(query.Where(field).EndsWith(val.(string)))
						case "gt":
							query = query.WhereCondition(query.Where(field).GreaterThan(val))
						case "gte":
							query = query.WhereCondition(query.Where(field).GreaterThanOrEqual(val))
						case "lt":
							query = query.WhereCondition(query.Where(field).LessThan(val))
						case "lte":
							query = query.WhereCondition(query.Where(field).LessThanOrEqual(val))
						case "in":
							query = query.WhereCondition(query.Where(field).In(val))
						case "notIn":
							query = query.WhereCondition(query.Where(field).NotIn(val))
						}
					}
				} else {
					// Direct equals
					query = query.WhereCondition(query.Where(field).Equals(value))
				}
			}
		}

		// Execute update
		result, err := query.Exec(ctx)
		if err != nil {
			return nil, err
		}

		return map[string]any{"count": result.RowsAffected}, nil
	}
}

// createDeleteManyResolver creates a resolver for deleteMany mutations
func createDeleteManyResolver(db types.Database, modelName string) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		ctx := p.Context

		// Build query
		query := db.Model(modelName).Delete()

		// Apply where conditions
		if where, ok := p.Args["where"].(map[string]any); ok {
			conditions := buildWhereConditions(where)
			for field, value := range conditions {
				// Handle operators
				if operators, ok := value.(map[string]any); ok {
					for op, val := range operators {
						switch op {
						case "equals":
							query = query.WhereCondition(query.Where(field).Equals(val))
						case "contains":
							query = query.WhereCondition(query.Where(field).Contains(val.(string)))
						case "startsWith":
							query = query.WhereCondition(query.Where(field).StartsWith(val.(string)))
						case "endsWith":
							query = query.WhereCondition(query.Where(field).EndsWith(val.(string)))
						case "gt":
							query = query.WhereCondition(query.Where(field).GreaterThan(val))
						case "gte":
							query = query.WhereCondition(query.Where(field).GreaterThanOrEqual(val))
						case "lt":
							query = query.WhereCondition(query.Where(field).LessThan(val))
						case "lte":
							query = query.WhereCondition(query.Where(field).LessThanOrEqual(val))
						case "in":
							query = query.WhereCondition(query.Where(field).In(val))
						case "notIn":
							query = query.WhereCondition(query.Where(field).NotIn(val))
						}
					}
				} else {
					// Direct equals
					query = query.WhereCondition(query.Where(field).Equals(value))
				}
			}
		}

		// Execute delete
		result, err := query.Exec(ctx)
		if err != nil {
			return nil, err
		}

		return map[string]any{"count": result.RowsAffected}, nil
	}
}

// createRelationResolver creates a resolver for relation fields
func createRelationResolver(db types.Database, modelName string, relation schema.Relation) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (any, error) {
		ctx := context.Background()

		// Get the parent record
		parent, ok := p.Source.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid parent object")
		}

		// Get the foreign key value from parent
		var foreignKeyValue any
		if relation.Type == schema.RelationOneToMany || relation.Type == schema.RelationManyToMany {
			// For one-to-many, the foreign key is on the child model
			foreignKeyValue = parent[relation.References]
			if foreignKeyValue == nil {
				// Try "id" as default
				foreignKeyValue = parent["id"]
			}
		} else {
			// For many-to-one or one-to-one, the foreign key is on the parent model
			foreignKeyValue = parent[relation.ForeignKey]
		}

		if foreignKeyValue == nil {
			return nil, nil
		}

		// Build query for related model
		query := db.Model(relation.Model).Select()

		if relation.Type == schema.RelationOneToMany {
			// Find all children where foreign key matches parent's primary key
			query = query.WhereCondition(query.Where(relation.ForeignKey).Equals(foreignKeyValue))

			var results []map[string]any
			err := query.FindMany(ctx, &results)
			if err != nil {
				return nil, err
			}
			return results, nil
		} else if relation.Type == schema.RelationManyToOne || relation.Type == schema.RelationOneToOne {
			// Find parent where primary key matches child's foreign key
			referencesField := relation.References
			if referencesField == "" {
				referencesField = "id" // Default to id
			}
			query = query.WhereCondition(query.Where(referencesField).Equals(foreignKeyValue))

			var results []map[string]any
			err := query.Limit(1).FindMany(ctx, &results)
			if err != nil {
				return nil, err
			}

			if len(results) > 0 {
				return results[0], nil
			}
			return nil, nil
		}

		// TODO: Handle many-to-many relations
		return nil, fmt.Errorf("many-to-many relations not yet implemented")
	}
}

// buildWhereConditions builds where conditions from GraphQL input
func buildWhereConditions(where map[string]any) map[string]any {
	conditions := make(map[string]any)

	for field, value := range where {
		// Handle special operators
		if field == "AND" || field == "OR" {
			// TODO: Handle AND/OR operators
			continue
		}

		// Handle field filters
		if filterMap, ok := value.(map[string]any); ok {
			// Pass the filter map as is, it contains the operators
			conditions[field] = filterMap
		} else {
			// Direct value - wrap in equals operator
			conditions[field] = map[string]any{"equals": value}
		}
	}

	return conditions
}
