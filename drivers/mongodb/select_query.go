package mongodb

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDBSelectQuery implements SelectQuery for MongoDB
type MongoDBSelectQuery struct {
	*query.SelectQueryImpl
	db          *MongoDB
	fieldMapper types.FieldMapper
	modelName   string
}

// NewMongoDBSelectQuery creates a new MongoDB select query
func NewMongoDBSelectQuery(baseQuery *query.ModelQueryImpl, fieldNames []string, db *MongoDB, fieldMapper types.FieldMapper, modelName string) types.SelectQuery {
	selectQuery := query.NewSelectQuery(baseQuery, fieldNames)
	return &MongoDBSelectQuery{
		SelectQueryImpl: selectQuery,
		db:              db,
		fieldMapper:     fieldMapper,
		modelName:       modelName,
	}
}

// BuildSQL builds a MongoDB find/aggregate command instead of SQL
func (q *MongoDBSelectQuery) BuildSQL() (string, []any, error) {
	// Get collection name
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve collection name: %w", err)
	}

	// Check if we have includes - use aggregation with $lookup for relations
	includes := q.GetIncludes()
	includeOptions := q.GetIncludeOptions()
	if len(includes) > 0 || len(includeOptions) > 0 {
		return q.buildIncludeCommand(tableName)
	}

	// Check if we need aggregation pipeline
	if q.hasAggregation() {
		return q.buildAggregateCommand(tableName)
	}

	// Build simple find command
	return q.buildFindCommand(tableName)
}

// buildFindCommand builds a simple find command
func (q *MongoDBSelectQuery) buildFindCommand(collection string) (string, []any, error) {
	// Build filter from conditions
	filter, err := q.buildFilter()
	if err != nil {
		return "", nil, err
	}

	// Build options
	options := bson.M{}

	// Add sort
	if sortDoc := q.buildSort(); sortDoc != nil {
		options["sort"] = sortDoc
	}

	// Add limit
	if limit := q.GetLimit(); limit > 0 {
		options["limit"] = int64(limit)
	}

	// Add skip
	if offset := q.GetOffset(); offset > 0 {
		options["skip"] = int64(offset)
	}

	// Get selected fields for projection
	fields := q.GetSelectedFields()

	// Map field names to column names for projection
	var projectedColumns []string
	if len(fields) > 0 {
		projectedColumns = make([]string, 0, len(fields))
		for _, field := range fields {
			columnName, err := q.GetFieldMapper().SchemaToColumn(q.GetModelName(), field)
			if err != nil {
				columnName = field
			}
			projectedColumns = append(projectedColumns, columnName)
		}
	}

	// Create MongoDB command
	cmd := MongoDBCommand{
		Operation:  "find",
		Collection: collection,
		Filter:     filter,
		Options:    options,
		Fields:     projectedColumns, // Use column names for projection
	}

	// Convert to JSON
	jsonCmd, err := cmd.ToJSON()
	if err != nil {
		return "", nil, err
	}

	return jsonCmd, nil, nil
}

// buildAggregateCommand builds an aggregation pipeline command
func (q *MongoDBSelectQuery) buildAggregateCommand(collection string) (string, []any, error) {
	pipeline := []bson.M{}

	// Add $match stage for WHERE conditions
	if filter, err := q.buildFilter(); err == nil && len(filter) > 0 {
		pipeline = append(pipeline, bson.M{"$match": filter})
	}

	// Add $group stage for GROUP BY
	if groupStage := q.buildGroupStage(); groupStage != nil {
		pipeline = append(pipeline, groupStage)

		// Add $match stage for HAVING conditions after $group
		if havingFilter := q.buildHavingFilter(); havingFilter != nil {
			pipeline = append(pipeline, bson.M{"$match": havingFilter})
		}
	}

	// Handle DISTINCT by adding a $group stage
	if q.GetDistinct() {
		// Check if we have specific distinct fields
		distinctFields := q.SelectQueryImpl.GetDistinctOn()
		fields := distinctFields
		if len(fields) == 0 {
			// If no specific fields, use all selected fields
			fields = q.GetSelectedFields()

			// If still no fields (distinct: true with no select), get all fields from schema
			if len(fields) == 0 {
				// For distinct: true without select, we need to get all fields from schema
				schema, err := q.db.GetSchema(q.modelName)
				if err != nil {
					return "", nil, fmt.Errorf("failed to get schema for distinct: %w", err)
				}

				// Use all fields from schema
				fields = make([]string, 0, len(schema.Fields))
				for _, field := range schema.Fields {
					fields = append(fields, field.Name)
				}
			}
		}

		if len(fields) > 0 {
			groupID := bson.M{}
			for _, field := range fields {
				// Map field name to column name
				columnName, err := q.GetFieldMapper().SchemaToColumn(q.GetModelName(), field)
				if err != nil {
					columnName = field
				}
				groupID[field] = "$" + columnName
			}

			// Add $group stage to get distinct values
			groupStage := bson.M{
				"$group": bson.M{
					"_id": groupID,
				},
			}
			pipeline = append(pipeline, groupStage)

			// Add $replaceRoot to flatten the result
			replaceRootStage := bson.M{
				"$replaceRoot": bson.M{
					"newRoot": "$_id",
				},
			}
			pipeline = append(pipeline, replaceRootStage)
		}
	} else {
		// Add $project stage for field selection (only if not using DISTINCT)
		if projectStage := q.buildProjectStage(); projectStage != nil {
			pipeline = append(pipeline, projectStage)
		}
	}

	// Add $sort stage (after DISTINCT processing to ensure proper field references)
	if sortDoc := q.buildSort(); sortDoc != nil {
		// sortDoc is already a bson.D, which is the correct format for $sort
		sortStage := bson.M{"$sort": sortDoc}
		pipeline = append(pipeline, sortStage)
	}

	// Add $skip stage
	if offset := q.GetOffset(); offset > 0 {
		pipeline = append(pipeline, bson.M{"$skip": offset})
	}

	// Add $limit stage
	if limit := q.GetLimit(); limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": limit})
	}

	// Create MongoDB command
	cmd := MongoDBCommand{
		Operation:  "aggregate",
		Collection: collection,
		Pipeline:   pipeline,
	}

	// Convert to JSON
	jsonCmd, err := cmd.ToJSON()
	if err != nil {
		return "", nil, err
	}

	return jsonCmd, nil, nil
}

// buildFilter builds MongoDB filter from WHERE conditions
func (q *MongoDBSelectQuery) buildFilter() (bson.M, error) {
	conditions := q.GetConditions()
	if len(conditions) == 0 {
		return bson.M{}, nil
	}

	// Use query builder
	qb := NewMongoDBQueryBuilder(q.db)

	// Combine all conditions with AND
	var combined types.Condition
	for i, cond := range conditions {
		if i == 0 {
			combined = cond
		} else {
			combined = combined.And(cond)
		}
	}

	return qb.ConditionToFilter(combined, q.modelName)
}

// buildSort builds MongoDB sort document
func (q *MongoDBSelectQuery) buildSort() bson.D {
	orderBy := q.GetOrderBy()
	if len(orderBy) == 0 {
		return nil
	}

	// Use query builder
	qb := NewMongoDBQueryBuilder(q.db)

	sort, err := qb.ConvertOrderBy(orderBy, q.GetModelName())
	if err != nil {
		// Log error and return nil
		return nil
	}
	return sort
}

// buildGroupStage builds $group stage for aggregation
func (q *MongoDBSelectQuery) buildGroupStage() bson.M {
	groupBy := q.GetGroupBy()
	if len(groupBy) == 0 {
		return nil
	}

	// Build _id for grouping
	groupID := bson.M{}
	for _, field := range groupBy {
		// Map field name to column name
		columnName, err := q.GetFieldMapper().SchemaToColumn(q.GetModelName(), field)
		if err != nil {
			columnName = field
		}
		groupID[field] = "$" + columnName
	}

	// Build group stage
	group := bson.M{
		"_id": groupID,
	}

	// Add aggregation fields based on selected fields
	// This is a simplified version - in reality, we'd parse the selected fields
	// to determine which aggregations to perform

	// For now, include all non-grouped fields as $first
	for _, field := range q.GetSelectedFields() {
		if !contains(groupBy, field) {
			columnName, err := q.GetFieldMapper().SchemaToColumn(q.GetModelName(), field)
			if err != nil {
				columnName = field
			}
			group[field] = bson.M{"$first": "$" + columnName}
		}
	}

	return bson.M{"$group": group}
}

// buildHavingFilter builds filter for HAVING conditions
func (q *MongoDBSelectQuery) buildHavingFilter() bson.M {
	// TODO: Implement HAVING support
	// This requires parsing HAVING conditions which reference aggregated values
	return nil
}

// buildProjectStage builds $project stage for field selection
func (q *MongoDBSelectQuery) buildProjectStage() bson.M {
	fields := q.GetSelectedFields()
	if len(fields) == 0 {
		return nil
	}

	// Check if we have a GROUP BY - if so, we need to handle projection differently
	groupBy := q.GetGroupBy()
	if len(groupBy) > 0 {
		// After grouping, fields are in _id or as aggregated fields
		projection := bson.M{}

		for _, field := range fields {
			if contains(groupBy, field) {
				// Grouped fields are in _id
				projection[field] = "$_id." + field
			} else {
				// Non-grouped fields are direct
				projection[field] = 1
			}
		}

		// Exclude _id
		projection["_id"] = 0

		return bson.M{"$project": projection}
	}

	// Normal projection (no grouping)
	projection := bson.M{}
	columnToField := make(map[string]string) // Track column to field mapping

	for _, field := range fields {
		if field == "*" {
			// Select all fields
			return nil
		}
		// Map field name to column name
		columnName, err := q.GetFieldMapper().SchemaToColumn(q.GetModelName(), field)
		if err != nil {
			columnName = field
		}
		projection[columnName] = 1
		// Store the mapping so we can use field names in the result
		columnToField[columnName] = field
	}

	// Always exclude _id unless explicitly requested
	if _, hasID := projection["_id"]; !hasID && projection["id"] == nil {
		projection["_id"] = 0
	}

	// For mapped fields, we need to add a rename stage after projection
	if len(columnToField) > 0 && len(fields) != len(projection)-1 { // -1 for _id:0
		// We have field mapping, need to rename columns back to field names
		// This is handled by the field mapper during result processing
		return bson.M{"$project": projection}
	}

	return bson.M{"$project": projection}
}

// hasAggregation checks if query requires aggregation pipeline
func (q *MongoDBSelectQuery) hasAggregation() bool {
	// Need aggregation for GROUP BY, HAVING, or complex operations
	return len(q.GetGroupBy()) > 0 || q.GetDistinct()
}

// Helper functions to access query properties
func (q *MongoDBSelectQuery) GetConditions() []types.Condition {
	return q.SelectQueryImpl.GetConditions()
}

func (q *MongoDBSelectQuery) GetOrderBy() []types.OrderByClause {
	return q.SelectQueryImpl.GetOrderBy()
}

func (q *MongoDBSelectQuery) GetGroupBy() []string {
	return q.SelectQueryImpl.GetGroupBy()
}

func (q *MongoDBSelectQuery) GetIncludes() []string {
	return q.SelectQueryImpl.GetIncludes()
}

func (q *MongoDBSelectQuery) GetLimit() int {
	return q.SelectQueryImpl.GetLimit()
}

func (q *MongoDBSelectQuery) GetOffset() int {
	return q.SelectQueryImpl.GetOffset()
}

func (q *MongoDBSelectQuery) GetDistinct() bool {
	return q.SelectQueryImpl.GetDistinct()
}

func (q *MongoDBSelectQuery) GetSelectedFields() []string {
	return q.SelectQueryImpl.GetSelectedFields()
}

// buildIncludeCommand builds a $lookup aggregation command for includes/relations
func (q *MongoDBSelectQuery) buildIncludeCommand(collection string) (string, []any, error) {
	pipeline := []bson.M{}

	// Add $match stage for WHERE conditions
	if filter, err := q.buildFilter(); err == nil && len(filter) > 0 {
		pipeline = append(pipeline, bson.M{"$match": filter})
	}

	// Add $lookup stages for each include (including nested ones)
	includeOptions := q.SelectQueryImpl.GetIncludeOptions()
	simpleIncludes := q.GetIncludes()

	// Group includes by their root relation to avoid conflicts
	// For example, both "comments" and "comments.user" should be handled together
	rootIncludes := make(map[string][]string)

	// First, process include options to identify all paths
	for includePath := range includeOptions {
		if strings.Contains(includePath, ".") {
			// Extract root relation (e.g., "comments" from "comments.user")
			root := strings.Split(includePath, ".")[0]
			rootIncludes[root] = append(rootIncludes[root], includePath)
		} else {
			// Simple include from include options
			rootIncludes[includePath] = append(rootIncludes[includePath], includePath)
		}
	}

	// Then, add simple includes only if they don't already have nested versions
	for _, includePath := range simpleIncludes {
		if _, exists := rootIncludes[includePath]; !exists {
			// Only add if this root doesn't already have entries from include options
			rootIncludes[includePath] = append(rootIncludes[includePath], includePath)
		}
	}

	// Process each root relation once, prioritizing nested includes
	processedIncludes := make(map[string]bool)
	for rootRelation, paths := range rootIncludes {
		// Skip if this is actually a nested path (e.g., "comments.user")
		if strings.Contains(rootRelation, ".") {
			continue
		}
		// Check if we have nested includes for this root relation
		var nestedPath string
		hasNested := false
		for _, path := range paths {
			if strings.Contains(path, ".") {
				nestedPath = path
				hasNested = true
				break // Use the first nested path we find
			}
		}

		if hasNested {
			// Process nested include (this will handle the root relation too)
			lookupStages, err := q.buildNestedLookupStages(nestedPath)
			if err != nil {
				// Skip problematic includes for now, but continue processing
				continue
			}
			pipeline = append(pipeline, lookupStages...)

			// Mark all paths for this root as processed
			for _, path := range paths {
				processedIncludes[path] = true
			}
		} else {
			// Process simple include
			lookupStage, err := q.buildLookupStage(rootRelation)
			if err != nil {
				// Skip problematic includes for now, but continue processing
				continue
			}
			if lookupStage != nil {
				pipeline = append(pipeline, lookupStage)

				// Add unwind stage for many-to-one and one-to-one relations
				shouldUnwind, err := q.shouldUnwindRelation(rootRelation)
				if err == nil && shouldUnwind {
					unwindStage := bson.M{
						"$unwind": bson.M{
							"path":                       "$" + rootRelation,
							"preserveNullAndEmptyArrays": true,
						},
					}
					pipeline = append(pipeline, unwindStage)
				}

				processedIncludes[rootRelation] = true
			}
		}
	}

	// Note: We already add unwind stages inline after each lookup
	// so we don't need addUnwindStages here

	// Add $sort stage
	if sortDoc := q.buildSort(); sortDoc != nil {
		// sortDoc is already a bson.D, which is the correct format for $sort
		sortStage := bson.M{"$sort": sortDoc}
		pipeline = append(pipeline, sortStage)
	}

	// Add $skip stage
	if offset := q.GetOffset(); offset > 0 {
		pipeline = append(pipeline, bson.M{"$skip": offset})
	}

	// Add $limit stage
	if limit := q.GetLimit(); limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": limit})
	}

	// Create MongoDB command
	cmd := MongoDBCommand{
		Operation:  "aggregate",
		Collection: collection,
		Pipeline:   pipeline,
	}

	// Convert to JSON
	jsonCmd, err := cmd.ToJSON()
	if err != nil {
		return "", nil, err
	}

	return jsonCmd, nil, nil
}

// buildLookupStage creates a $lookup stage for a given relation
func (q *MongoDBSelectQuery) buildLookupStage(relationName string) (bson.M, error) {
	// Get the current model's schema
	currentSchema, err := q.db.GetSchema(q.modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for model %s: %w", q.modelName, err)
	}

	// Get the relation definition
	relation, exists := currentSchema.Relations[relationName]
	if !exists {
		return nil, fmt.Errorf("relation %s not found in model %s", relationName, q.modelName)
	}

	// Get the related model's collection name
	relatedCollection, err := q.fieldMapper.ModelToTable(relation.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection name for model %s: %w", relation.Model, err)
	}

	// Check if there are include options for this relation
	includeOptions := q.SelectQueryImpl.GetIncludeOptions()
	includeOpt, hasIncludeOpt := includeOptions[relationName]
	var whereFilter bson.M
	var orderBy bson.D
	var limitValue *int
	var skipValue *int
	var projection bson.M

	if hasIncludeOpt {
		// Build where filter if present
		if includeOpt.Where != nil {
			qb := NewMongoDBQueryBuilder(q.db)
			whereFilter, err = qb.ConditionToFilter(includeOpt.Where, relation.Model)
			if err != nil {
				return nil, fmt.Errorf("failed to build where filter for include %s: %w", relationName, err)
			}
		}

		// Build order by if present
		if len(includeOpt.OrderBy) > 0 {
			orderBy = bson.D{}
			for _, order := range includeOpt.OrderBy {
				// Map field name to column name
				columnName, err := q.fieldMapper.SchemaToColumn(relation.Model, order.Field)
				if err != nil {
					columnName = order.Field
				}

				direction := 1
				if order.Direction == types.DESC {
					direction = -1
				}

				orderBy = append(orderBy, bson.E{Key: columnName, Value: direction})
			}
		}

		// Build projection if select fields are specified
		if len(includeOpt.Select) > 0 {
			projection = bson.M{}
			for _, field := range includeOpt.Select {
				// Map field name to column name
				columnName, err := q.fieldMapper.SchemaToColumn(relation.Model, field)
				if err != nil {
					columnName = field
				}
				projection[columnName] = 1
			}
		}

		// Get limit and skip values
		limitValue = includeOpt.Limit
		skipValue = includeOpt.Offset
	}

	// Build the $lookup stage based on relation type
	switch relation.Type {
	case schema.RelationOneToMany:
		// For one-to-many, we look up from the related collection
		// where the foreign key in the related collection matches our primary key

		// Get the local field (usually the primary key)
		localField := relation.References
		if localField == "" {
			localField = "id"
		}
		// Map to column name
		localColumn, err := q.fieldMapper.SchemaToColumn(q.modelName, localField)
		if err != nil {
			localColumn = localField
		}
		// MongoDB uses _id for primary key
		if localColumn == "id" {
			localColumn = "_id"
		}

		// Get the foreign key column in the related collection
		foreignColumn, err := q.fieldMapper.SchemaToColumn(relation.Model, relation.ForeignKey)
		if err != nil {
			foreignColumn = relation.ForeignKey
		}

		// Check if the foreign key is part of a composite primary key in the related model
		relatedSchema, err := q.db.GetSchema(relation.Model)
		if err == nil && len(relatedSchema.CompositeKey) > 1 {
			// Check if the foreign key is part of the composite key
			for _, keyField := range relatedSchema.CompositeKey {
				if keyField == relation.ForeignKey {
					// For composite keys, MongoDB stores them under _id
					foreignColumn = "_id." + foreignColumn
					break
				}
			}
		}

		// Check if we need pipeline style lookup (for filtering, ordering, pagination, or projection)
		needsPipeline := len(whereFilter) > 0 || len(orderBy) > 0 || limitValue != nil || skipValue != nil || len(projection) > 0

		if needsPipeline {
			// Build pipeline with stages
			pipeline := []bson.M{}

			// Add match stage for the join condition
			pipeline = append(pipeline, bson.M{
				"$match": bson.M{
					"$expr": bson.M{
						"$eq": []any{"$" + foreignColumn, "$$localField"},
					},
				},
			})

			// Add match stage for where conditions
			if len(whereFilter) > 0 {
				pipeline = append(pipeline, bson.M{
					"$match": whereFilter,
				})
			}

			// Add sort stage
			if len(orderBy) > 0 {
				pipeline = append(pipeline, bson.M{
					"$sort": orderBy,
				})
			}

			// Add skip stage
			if skipValue != nil && *skipValue > 0 {
				pipeline = append(pipeline, bson.M{
					"$skip": *skipValue,
				})
			}

			// Add limit stage
			if limitValue != nil && *limitValue > 0 {
				pipeline = append(pipeline, bson.M{
					"$limit": *limitValue,
				})
			}

			// Add projection stage if select fields are specified
			if len(projection) > 0 {
				pipeline = append(pipeline, bson.M{
					"$project": projection,
				})
			}

			return bson.M{
				"$lookup": bson.M{
					"from":     relatedCollection,
					"let":      bson.M{"localField": "$" + localColumn},
					"pipeline": pipeline,
					"as":       relationName,
				},
			}, nil
		} else {
			// Use simple $lookup without filtering
			lookupStage := bson.M{
				"$lookup": bson.M{
					"from":         relatedCollection,
					"localField":   localColumn,
					"foreignField": foreignColumn,
					"as":           relationName,
				},
			}
			return lookupStage, nil
		}

	case schema.RelationManyToOne:
		// For many-to-one, we look up from the related collection
		// where our foreign key matches the primary key in the related collection

		// Get the foreign key column in our collection
		foreignColumn, err := q.fieldMapper.SchemaToColumn(q.modelName, relation.ForeignKey)
		if err != nil {
			foreignColumn = relation.ForeignKey
		}

		// Get the referenced field in the related collection (usually primary key)
		referencedField := relation.References
		if referencedField == "" {
			referencedField = "id"
		}
		// Map to column name
		referencedColumn, err := q.fieldMapper.SchemaToColumn(relation.Model, referencedField)
		if err != nil {
			referencedColumn = referencedField
		}
		// MongoDB uses _id for primary key
		if referencedColumn == "id" {
			referencedColumn = "_id"
		}

		// Check if we need pipeline style lookup (for filtering, ordering, pagination, or projection)
		needsPipeline := len(whereFilter) > 0 || len(orderBy) > 0 || limitValue != nil || skipValue != nil || len(projection) > 0

		if needsPipeline {
			// Build pipeline with stages
			pipeline := []bson.M{}

			// Add match stage for the join condition
			pipeline = append(pipeline, bson.M{
				"$match": bson.M{
					"$expr": bson.M{
						"$eq": []any{"$" + referencedColumn, "$$localField"},
					},
				},
			})

			// Add match stage for where conditions
			if len(whereFilter) > 0 {
				pipeline = append(pipeline, bson.M{
					"$match": whereFilter,
				})
			}

			// Add sort stage
			if len(orderBy) > 0 {
				pipeline = append(pipeline, bson.M{
					"$sort": orderBy,
				})
			}

			// Add skip stage
			if skipValue != nil && *skipValue > 0 {
				pipeline = append(pipeline, bson.M{
					"$skip": *skipValue,
				})
			}

			// Add limit stage
			if limitValue != nil && *limitValue > 0 {
				pipeline = append(pipeline, bson.M{
					"$limit": *limitValue,
				})
			}

			// Add projection stage if select fields are specified
			if len(projection) > 0 {
				pipeline = append(pipeline, bson.M{
					"$project": projection,
				})
			}

			return bson.M{
				"$lookup": bson.M{
					"from":     relatedCollection,
					"let":      bson.M{"localField": "$" + foreignColumn},
					"pipeline": pipeline,
					"as":       relationName,
				},
			}, nil
		} else {
			// Use simple $lookup without filtering
			return bson.M{
				"$lookup": bson.M{
					"from":         relatedCollection,
					"localField":   foreignColumn,
					"foreignField": referencedColumn,
					"as":           relationName,
				},
			}, nil
		}

	case schema.RelationOneToOne:
		// One-to-one is similar to many-to-one but we expect single result
		// The implementation is the same as many-to-one

		// Get the foreign key column
		foreignColumn, err := q.fieldMapper.SchemaToColumn(q.modelName, relation.ForeignKey)
		if err != nil {
			foreignColumn = relation.ForeignKey
		}

		// Get the referenced field in the related collection
		referencedField := relation.References
		if referencedField == "" {
			referencedField = "id"
		}
		referencedColumn, err := q.fieldMapper.SchemaToColumn(relation.Model, referencedField)
		if err != nil {
			referencedColumn = referencedField
		}
		if referencedColumn == "id" {
			referencedColumn = "_id"
		}

		// Check if we need pipeline style lookup (for filtering, ordering, pagination, or projection)
		needsPipeline := len(whereFilter) > 0 || len(orderBy) > 0 || limitValue != nil || skipValue != nil || len(projection) > 0

		if needsPipeline {
			// Build pipeline with stages
			pipeline := []bson.M{}

			// Add match stage for the join condition
			pipeline = append(pipeline, bson.M{
				"$match": bson.M{
					"$expr": bson.M{
						"$eq": []any{"$" + referencedColumn, "$$localField"},
					},
				},
			})

			// Add match stage for where conditions
			if len(whereFilter) > 0 {
				pipeline = append(pipeline, bson.M{
					"$match": whereFilter,
				})
			}

			// Add sort stage
			if len(orderBy) > 0 {
				pipeline = append(pipeline, bson.M{
					"$sort": orderBy,
				})
			}

			// Add skip stage
			if skipValue != nil && *skipValue > 0 {
				pipeline = append(pipeline, bson.M{
					"$skip": *skipValue,
				})
			}

			// Add limit stage
			if limitValue != nil && *limitValue > 0 {
				pipeline = append(pipeline, bson.M{
					"$limit": *limitValue,
				})
			}

			// Add projection stage if select fields are specified
			if len(projection) > 0 {
				pipeline = append(pipeline, bson.M{
					"$project": projection,
				})
			}

			return bson.M{
				"$lookup": bson.M{
					"from":     relatedCollection,
					"let":      bson.M{"localField": "$" + foreignColumn},
					"pipeline": pipeline,
					"as":       relationName,
				},
			}, nil
		} else {
			// Use simple $lookup without filtering
			return bson.M{
				"$lookup": bson.M{
					"from":         relatedCollection,
					"localField":   foreignColumn,
					"foreignField": referencedColumn,
					"as":           relationName,
				},
			}, nil
		}

	case schema.RelationManyToMany:
		// Many-to-many requires a join table, which is more complex
		// For now, return an error
		return nil, fmt.Errorf("many-to-many relations are not yet supported in MongoDB includes")

	default:
		return nil, fmt.Errorf("unsupported relation type: %v", relation.Type)
	}
}

// buildNestedLookupStages builds $lookup stages for nested includes like "comments.author"
func (q *MongoDBSelectQuery) buildNestedLookupStages(nestedPath string) ([]bson.M, error) {
	parts := strings.Split(nestedPath, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid nested path, expected at least 2 parts, got: %s", nestedPath)
	}

	// For now, we handle the first two parts and let the field mapper handle deeper nesting
	parentRelation := parts[0]
	childRelation := parts[1]

	// The strategy is to modify the parent relation's $lookup to include
	// a nested $lookup in its pipeline for the child relation

	// Get parent relation info
	currentSchema, err := q.db.GetSchema(q.modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for model %s: %w", q.modelName, err)
	}

	parentRel, exists := currentSchema.Relations[parentRelation]
	if !exists {
		return nil, fmt.Errorf("parent relation %s not found in model %s", parentRelation, q.modelName)
	}

	// Get child relation info
	parentSchema, err := q.db.GetSchema(parentRel.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema for parent model %s: %w", parentRel.Model, err)
	}

	childRel, exists := parentSchema.Relations[childRelation]
	if !exists {
		return nil, fmt.Errorf("child relation %s not found in parent model %s", childRelation, parentRel.Model)
	}

	// Check if there are include options for the parent relation
	includeOptions := q.SelectQueryImpl.GetIncludeOptions()
	parentIncludeOpt, _ := includeOptions[parentRelation]

	// Build $lookup with nested pipeline and parent options
	lookupStage, err := q.buildLookupWithNestedRelation(parentRelation, parentRel, childRelation, childRel, parentIncludeOpt)
	if err != nil {
		return nil, err
	}

	stages := []bson.M{lookupStage}

	// For many-to-one parent relations, add unwind stage
	if parentRel.Type == schema.RelationManyToOne {
		unwindStage := bson.M{
			"$unwind": bson.M{
				"path":                       "$" + parentRelation,
				"preserveNullAndEmptyArrays": true,
			},
		}
		stages = append(stages, unwindStage)
	}

	return stages, nil
}

// buildLookupWithNestedRelation builds a $lookup stage that includes a nested $lookup for child relations
func (q *MongoDBSelectQuery) buildLookupWithNestedRelation(parentRelation string, parentRel schema.Relation, childRelation string, childRel schema.Relation, parentIncludeOpt *types.IncludeOption) (bson.M, error) {
	// Get collection names
	parentCollection, err := q.fieldMapper.ModelToTable(parentRel.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection name for parent model %s: %w", parentRel.Model, err)
	}

	childCollection, err := q.fieldMapper.ModelToTable(childRel.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection name for child model %s: %w", childRel.Model, err)
	}

	// Build the nested $lookup pipeline
	var nestedPipeline []bson.M

	// Add the child relation lookup
	switch childRel.Type {
	case schema.RelationManyToOne:
		// For many-to-one (e.g., comment.author), we lookup from the child collection
		// where the child's foreign key matches the parent's primary key

		// Get the foreign key column in the parent collection
		foreignColumn, err := q.fieldMapper.SchemaToColumn(parentRel.Model, childRel.ForeignKey)
		if err != nil {
			foreignColumn = childRel.ForeignKey
		}

		// Get the referenced column in the child collection
		referencedColumn := childRel.References
		if referencedColumn == "" {
			referencedColumn = "id"
		}
		referencedColumn, err = q.fieldMapper.SchemaToColumn(childRel.Model, referencedColumn)
		if err != nil {
			referencedColumn = childRel.References
		}
		// MongoDB uses _id for primary key
		if referencedColumn == "id" {
			referencedColumn = "_id"
		}

		// Add nested $lookup for child relation
		nestedPipeline = append(nestedPipeline, bson.M{
			"$lookup": bson.M{
				"from":         childCollection,
				"localField":   foreignColumn,
				"foreignField": referencedColumn,
				"as":           childRelation,
			},
		})

		// Convert array to single object for many-to-one relations
		nestedPipeline = append(nestedPipeline, bson.M{
			"$unwind": bson.M{
				"path":                       "$" + childRelation,
				"preserveNullAndEmptyArrays": true,
			},
		})

	case schema.RelationOneToMany:
		// For one-to-many (e.g., user.posts from comments), we lookup from the child collection
		// where the child's foreign key matches our id

		// Get the referenced column (primary key) in the parent collection
		referencedColumn := childRel.References
		if referencedColumn == "" {
			referencedColumn = "id"
		}
		referencedColumn, err = q.fieldMapper.SchemaToColumn(parentRel.Model, referencedColumn)
		if err != nil {
			referencedColumn = childRel.References
		}
		// MongoDB uses _id for primary key
		if referencedColumn == "id" {
			referencedColumn = "_id"
		}

		// Get the foreign key column in the child collection
		foreignColumn, err := q.fieldMapper.SchemaToColumn(childRel.Model, childRel.ForeignKey)
		if err != nil {
			foreignColumn = childRel.ForeignKey
		}

		// Add nested $lookup for child relation
		nestedPipeline = append(nestedPipeline, bson.M{
			"$lookup": bson.M{
				"from":         childCollection,
				"localField":   referencedColumn,
				"foreignField": foreignColumn,
				"as":           childRelation,
			},
		})

	default:
		return nil, fmt.Errorf("unsupported child relation type: %v", childRel.Type)
	}

	// Now build the main $lookup for the parent relation with the nested pipeline
	switch parentRel.Type {
	case schema.RelationOneToMany:
		// Get the local field (usually the primary key)
		localField := parentRel.References
		if localField == "" {
			localField = "id"
		}
		localColumn, err := q.fieldMapper.SchemaToColumn(q.modelName, localField)
		if err != nil {
			localColumn = localField
		}
		// MongoDB uses _id for primary key
		if localColumn == "id" {
			localColumn = "_id"
		}

		// Get the foreign key column in the parent collection
		foreignColumn, err := q.fieldMapper.SchemaToColumn(parentRel.Model, parentRel.ForeignKey)
		if err != nil {
			foreignColumn = parentRel.ForeignKey
		}

		// Build parent pipeline with optional filters
		parentPipeline := []bson.M{
			{
				"$match": bson.M{
					"$expr": bson.M{
						"$eq": []any{"$" + foreignColumn, "$$localField"},
					},
				},
			},
		}

		// Add parent relation options if specified
		if parentIncludeOpt != nil {
			// Add where filter
			if parentIncludeOpt.Where != nil {
				qb := NewMongoDBQueryBuilder(q.db)
				whereFilter, err := qb.ConditionToFilter(parentIncludeOpt.Where, parentRel.Model)
				if err == nil && len(whereFilter) > 0 {
					parentPipeline = append(parentPipeline, bson.M{
						"$match": whereFilter,
					})
				}
			}

			// Add order by
			if len(parentIncludeOpt.OrderBy) > 0 {
				orderBy := bson.D{}
				for _, order := range parentIncludeOpt.OrderBy {
					columnName, err := q.fieldMapper.SchemaToColumn(parentRel.Model, order.Field)
					if err != nil {
						columnName = order.Field
					}
					direction := 1
					if order.Direction == types.DESC {
						direction = -1
					}
					orderBy = append(orderBy, bson.E{Key: columnName, Value: direction})
				}
				parentPipeline = append(parentPipeline, bson.M{
					"$sort": orderBy,
				})
			}

			// Add skip
			if parentIncludeOpt.Offset != nil && *parentIncludeOpt.Offset > 0 {
				parentPipeline = append(parentPipeline, bson.M{
					"$skip": *parentIncludeOpt.Offset,
				})
			}

			// Add limit
			if parentIncludeOpt.Limit != nil && *parentIncludeOpt.Limit > 0 {
				parentPipeline = append(parentPipeline, bson.M{
					"$limit": *parentIncludeOpt.Limit,
				})
			}

			// Add projection
			if len(parentIncludeOpt.Select) > 0 {
				projection := bson.M{}
				for _, field := range parentIncludeOpt.Select {
					columnName, err := q.fieldMapper.SchemaToColumn(parentRel.Model, field)
					if err != nil {
						columnName = field
					}
					projection[columnName] = 1
				}
				parentPipeline = append(parentPipeline, bson.M{
					"$project": projection,
				})
			}
		}

		// Use pipeline-style $lookup to include nested relations
		return bson.M{
			"$lookup": bson.M{
				"from":     parentCollection,
				"let":      bson.M{"localField": "$" + localColumn},
				"pipeline": append(parentPipeline, nestedPipeline...),
				"as":       parentRelation,
			},
		}, nil

	case schema.RelationManyToOne:
		// Get the foreign key column in the current collection
		foreignColumn, err := q.fieldMapper.SchemaToColumn(q.modelName, parentRel.ForeignKey)
		if err != nil {
			foreignColumn = parentRel.ForeignKey
		}

		// Get the referenced column in the parent collection
		referencedColumn := parentRel.References
		if referencedColumn == "" {
			referencedColumn = "id"
		}
		referencedColumn, err = q.fieldMapper.SchemaToColumn(parentRel.Model, referencedColumn)
		if err != nil {
			referencedColumn = parentRel.References
		}
		// MongoDB uses _id for primary key
		if referencedColumn == "id" {
			referencedColumn = "_id"
		}

		// Build parent pipeline with optional filters
		parentPipeline := []bson.M{
			{
				"$match": bson.M{
					"$expr": bson.M{
						"$eq": []any{"$" + referencedColumn, "$$localField"},
					},
				},
			},
		}

		// Add parent relation options if specified
		if parentIncludeOpt != nil {
			// Add where filter
			if parentIncludeOpt.Where != nil {
				qb := NewMongoDBQueryBuilder(q.db)
				whereFilter, err := qb.ConditionToFilter(parentIncludeOpt.Where, parentRel.Model)
				if err == nil && len(whereFilter) > 0 {
					parentPipeline = append(parentPipeline, bson.M{
						"$match": whereFilter,
					})
				}
			}

			// Add order by
			if len(parentIncludeOpt.OrderBy) > 0 {
				orderBy := bson.D{}
				for _, order := range parentIncludeOpt.OrderBy {
					columnName, err := q.fieldMapper.SchemaToColumn(parentRel.Model, order.Field)
					if err != nil {
						columnName = order.Field
					}
					direction := 1
					if order.Direction == types.DESC {
						direction = -1
					}
					orderBy = append(orderBy, bson.E{Key: columnName, Value: direction})
				}
				parentPipeline = append(parentPipeline, bson.M{
					"$sort": orderBy,
				})
			}

			// Add skip
			if parentIncludeOpt.Offset != nil && *parentIncludeOpt.Offset > 0 {
				parentPipeline = append(parentPipeline, bson.M{
					"$skip": *parentIncludeOpt.Offset,
				})
			}

			// Add limit
			if parentIncludeOpt.Limit != nil && *parentIncludeOpt.Limit > 0 {
				parentPipeline = append(parentPipeline, bson.M{
					"$limit": *parentIncludeOpt.Limit,
				})
			}

			// Add projection
			if len(parentIncludeOpt.Select) > 0 {
				projection := bson.M{}
				for _, field := range parentIncludeOpt.Select {
					columnName, err := q.fieldMapper.SchemaToColumn(parentRel.Model, field)
					if err != nil {
						columnName = field
					}
					projection[columnName] = 1
				}
				parentPipeline = append(parentPipeline, bson.M{
					"$project": projection,
				})
			}
		}

		// Use pipeline-style $lookup to include nested relations
		lookupStage := bson.M{
			"$lookup": bson.M{
				"from":     parentCollection,
				"let":      bson.M{"localField": "$" + foreignColumn},
				"pipeline": append(parentPipeline, nestedPipeline...),
				"as":       parentRelation,
			},
		}

		// For many-to-one relations, we need to unwind after the lookup
		// Return both the lookup and unwind stages
		return lookupStage, nil

	default:
		return nil, fmt.Errorf("unsupported parent relation type: %v", parentRel.Type)
	}
}

// FindMany executes the query and returns multiple results
func (q *MongoDBSelectQuery) FindMany(ctx context.Context, dest any) error {
	sql, args, err := q.BuildSQL()
	if err != nil {
		return fmt.Errorf("failed to build MongoDB command: %w", err)
	}

	rawQuery := q.GetDatabase().Raw(sql, args...)
	err = rawQuery.Find(ctx, dest)
	if err != nil {
		return fmt.Errorf("failed to execute find: %w", err)
	}

	// For ORM queries, map column names back to schema field names
	err = q.mapColumnNamesToSchemaFields(dest)
	if err != nil {
		return fmt.Errorf("failed to map field names: %w", err)
	}

	return nil
}

// FindFirst executes the query and returns the first result
func (q *MongoDBSelectQuery) FindFirst(ctx context.Context, dest any) error {
	// Add limit 1 for efficiency
	v := reflect.ValueOf(q.SelectQueryImpl).Elem()
	limitField := v.FieldByName("limit")
	if limitField.IsValid() && limitField.CanSet() {
		limitField.Set(reflect.ValueOf(1))
	}

	sql, args, err := q.BuildSQL()
	if err != nil {
		return fmt.Errorf("failed to build MongoDB command: %w", err)
	}

	rawQuery := q.GetDatabase().Raw(sql, args...)
	err = rawQuery.FindOne(ctx, dest)
	if err != nil {
		return fmt.Errorf("failed to execute find one: %w", err)
	}

	// For ORM queries, map column names back to schema field names
	err = q.mapSingleColumnNamesToSchemaFields(dest)
	if err != nil {
		return fmt.Errorf("failed to map field names: %w", err)
	}

	return nil
}

// mapColumnNamesToSchemaFields maps column names to schema field names in query results
func (q *MongoDBSelectQuery) mapColumnNamesToSchemaFields(dest any) error {
	// Only process []map[string]any destinations
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return nil // Not a pointer, skip mapping
	}

	destElem := destValue.Elem()
	if destElem.Kind() != reflect.Slice {
		return nil // Not a slice, skip mapping
	}

	sliceElemType := destElem.Type().Elem()
	if sliceElemType != reflect.TypeOf(map[string]any{}) {
		return nil // Not []map[string]any, skip mapping
	}

	// Process each map in the slice
	for i := 0; i < destElem.Len(); i++ {
		mapValue := destElem.Index(i)
		mapInterface := mapValue.Interface().(map[string]any)

		// Map the main model data
		err := q.mapSingleDocumentFields(q.modelName, mapInterface)
		if err != nil {
			return fmt.Errorf("failed to map main model fields: %w", err)
		}
	}

	return nil
}

// mapSingleDocumentFields maps field names for a single document including nested relations
func (q *MongoDBSelectQuery) mapSingleDocumentFields(modelName string, document map[string]any) error {
	// Map the main document fields
	mappedData, err := q.fieldMapper.MapColumnToSchemaData(modelName, document)
	if err != nil {
		return fmt.Errorf("failed to map column to schema data for %s: %w", modelName, err)
	}

	// Replace the document contents with mapped data
	for key := range document {
		delete(document, key)
	}
	for key, value := range mappedData {
		document[key] = value
	}

	// Process included relations (including nested ones)
	includes := q.GetIncludes()

	// First, identify which simple includes have nested versions
	// e.g., if we have both "comments" and "comments.user", skip the simple "comments"
	nestedRoots := make(map[string]bool)
	for _, includePath := range includes {
		if strings.Contains(includePath, ".") {
			root := strings.Split(includePath, ".")[0]
			nestedRoots[root] = true
		}
	}

	for _, includePath := range includes {
		if strings.Contains(includePath, ".") {
			// Handle nested includes like "comments.user"
			err := q.mapNestedIncludedFields(document, includePath)
			if err != nil {
				return fmt.Errorf("failed to map nested included fields for %s: %w", includePath, err)
			}
		} else {
			// Skip simple includes that have nested versions
			if nestedRoots[includePath] {
				continue
			}

			// Handle simple includes like "comments"
			if includedData, exists := document[includePath]; exists {
				// Get the target model name for this include
				targetModelName, err := q.getIncludeTargetModel(includePath)
				if err != nil {
					continue // Skip if we can't determine the target model
				}

				// Map fields for included data
				err = q.mapIncludedFields(targetModelName, includedData)
				if err != nil {
					return fmt.Errorf("failed to map included fields for %s: %w", includePath, err)
				}
			}
		}
	}

	return nil
}

// mapNestedIncludedFields maps field names for nested includes like "comments.user"
func (q *MongoDBSelectQuery) mapNestedIncludedFields(document map[string]any, nestedPath string) error {
	parts := strings.Split(nestedPath, ".")
	if len(parts) < 1 {
		return fmt.Errorf("invalid nested path, got empty path")
	}

	// For a path like "departments.teams.members", we process only the first relation
	// "departments" and then recursively handle "teams.members" in the Department context
	parentRelation := parts[0]

	// The remaining path is everything after the first part
	remainingPath := ""
	if len(parts) > 1 {
		remainingPath = strings.Join(parts[1:], ".")
	}

	// Check if the parent relation exists in the document
	parentData, exists := document[parentRelation]
	if !exists {
		return nil // Parent relation not present, nothing to map
	}

	// Get the parent relation's target model
	parentModelName, err := q.getIncludeTargetModel(parentRelation)
	if err != nil {
		return fmt.Errorf("failed to get parent model for %s: %w", parentRelation, err)
	}
	fmt.Printf("[DEBUG] Processing nested path %s: parentRelation=%s, parentModel=%s, currentQueryModel=%s\n",
		nestedPath, parentRelation, parentModelName, q.modelName)

	// Map the parent relation data first
	err = q.mapIncludedFields(parentModelName, parentData)
	if err != nil {
		return fmt.Errorf("failed to map parent relation fields: %w", err)
	}

	// If there's a remaining path, recursively handle it in the context of the parent model
	if remainingPath != "" {
		// Create a new query context for the parent model
		parentQuery := &MongoDBSelectQuery{
			db:          q.db,
			fieldMapper: q.fieldMapper,
			modelName:   parentModelName,
		}
		fmt.Printf("[DEBUG] Recursing to parent context: remainingPath=%s, newContext=%s\n", remainingPath, parentModelName)

		// Check if the remaining path has multiple parts (nested) or just one part (simple relation)
		remainingParts := strings.Split(remainingPath, ".")
		if len(remainingParts) == 1 {
			// This is the final level - handle as a simple include
			err = parentQuery.mapSimpleNestedRelation(parentData, remainingPath)
			if err != nil {
				return fmt.Errorf("failed to map final nested relation %s: %w", remainingPath, err)
			}
		} else {
			// This has more nesting - recurse with mapNestedIncludedFields
			// Handle both single documents and arrays
			switch data := parentData.(type) {
			case map[string]any:
				// Single parent document
				err = parentQuery.mapNestedIncludedFields(data, remainingPath)
				if err != nil {
					return fmt.Errorf("failed to map deeper nested fields: %w", err)
				}
			case []any:
				// Array of parent documents
				for _, item := range data {
					if itemMap, ok := item.(map[string]any); ok {
						err = parentQuery.mapNestedIncludedFields(itemMap, remainingPath)
						if err != nil {
							return fmt.Errorf("failed to map deeper nested fields: %w", err)
						}
					}
				}
			}
		}
	}

	return nil
}

// mapSimpleNestedRelation handles the final level of a nested relation (like "user" in "comments.user")
func (q *MongoDBSelectQuery) mapSimpleNestedRelation(parentData any, relationName string) error {
	// Handle both single documents and arrays
	switch data := parentData.(type) {
	case map[string]any:
		// Single parent document - check if the relation exists
		if relationData, exists := data[relationName]; exists {
			// Get the target model for this relation
			targetModelName, err := q.getIncludeTargetModel(relationName)
			if err != nil {
				return fmt.Errorf("failed to get target model for %s: %w", relationName, err)
			}

			// Map the relation data
			err = q.mapIncludedFields(targetModelName, relationData)
			if err != nil {
				return fmt.Errorf("failed to map relation fields for %s: %w", relationName, err)
			}
		}

	case []any:
		// Array of parent documents
		for _, item := range data {
			if itemMap, ok := item.(map[string]any); ok {
				if relationData, exists := itemMap[relationName]; exists {
					// Get the target model for this relation
					targetModelName, err := q.getIncludeTargetModel(relationName)
					if err != nil {
						return fmt.Errorf("failed to get target model for %s: %w", relationName, err)
					}

					// Map the relation data
					err = q.mapIncludedFields(targetModelName, relationData)
					if err != nil {
						return fmt.Errorf("failed to map relation fields for %s: %w", relationName, err)
					}
				}
			}
		}
	}

	return nil
}

// mapIncludedFields maps field names for included relation data
func (q *MongoDBSelectQuery) mapIncludedFields(targetModelName string, includedData any) error {
	// Handle both single included document and arrays of included documents
	switch data := includedData.(type) {
	case map[string]any:
		// Single included document
		mappedData, err := q.fieldMapper.MapColumnToSchemaData(targetModelName, data)
		if err != nil {
			return err
		}

		// Replace contents
		for key := range data {
			delete(data, key)
		}
		for key, value := range mappedData {
			data[key] = value
		}

	case []any:
		// Array of included documents
		for _, item := range data {
			if itemMap, ok := item.(map[string]any); ok {
				mappedData, err := q.fieldMapper.MapColumnToSchemaData(targetModelName, itemMap)
				if err != nil {
					return err
				}

				// Replace contents
				for key := range itemMap {
					delete(itemMap, key)
				}
				for key, value := range mappedData {
					itemMap[key] = value
				}
			}
		}
	}

	return nil
}

// getIncludeTargetModel determines the target model name for an include path
func (q *MongoDBSelectQuery) getIncludeTargetModel(includePath string) (string, error) {
	// Get the schema for the main model through the database
	schema, err := q.db.GetSchema(q.modelName)
	if err != nil {
		return "", err
	}

	// Look for the relation with this name
	for relationName, relation := range schema.Relations {
		if relationName == includePath {
			return relation.Model, nil
		}
	}

	return "", fmt.Errorf("relation %s not found in model %s", includePath, q.modelName)
}

// mapSingleColumnNamesToSchemaFields maps column names to schema field names for single result
func (q *MongoDBSelectQuery) mapSingleColumnNamesToSchemaFields(dest any) error {
	// Only process map[string]any destinations
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return nil // Not a pointer, skip mapping
	}

	destElem := destValue.Elem()
	if destElem.Type() != reflect.TypeOf(map[string]any{}) {
		return nil // Not map[string]any, skip mapping
	}

	mapInterface := destElem.Interface().(map[string]any)

	// Map the single document including nested relations
	err := q.mapSingleDocumentFields(q.modelName, mapInterface)
	if err != nil {
		return fmt.Errorf("failed to map single document fields: %w", err)
	}

	return nil
}

// Count executes a count query
func (q *MongoDBSelectQuery) Count(ctx context.Context) (int64, error) {
	// Get collection name
	tableName, err := q.fieldMapper.ModelToTable(q.modelName)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve collection name: %w", err)
	}

	// Build filter from conditions
	filter, err := q.buildFilter()
	if err != nil {
		return 0, fmt.Errorf("failed to build filter: %w", err)
	}

	// For count, we use aggregation pipeline with $count stage
	pipeline := []bson.M{}

	// Add $match stage if there's a filter
	if len(filter) > 0 {
		pipeline = append(pipeline, bson.M{"$match": filter})
	}

	// Add $count stage
	pipeline = append(pipeline, bson.M{"$count": "count"})

	// Create MongoDB command
	cmd := MongoDBCommand{
		Operation:  "aggregate",
		Collection: tableName,
		Pipeline:   pipeline,
	}

	// Convert to JSON
	jsonCmd, err := cmd.ToJSON()
	if err != nil {
		return 0, err
	}

	// Execute the aggregation
	rawQuery := q.GetDatabase().Raw(jsonCmd)

	// MongoDB $count returns documents like {"count": 5}
	var result struct {
		Count int64 `bson:"count"`
	}

	err = rawQuery.FindOne(ctx, &result)
	if err != nil {
		if err.Error() == "no documents found" {
			// No documents means count is 0
			return 0, nil
		}
		return 0, fmt.Errorf("failed to execute count query: %w", err)
	}

	return result.Count, nil
}

// Helper methods to access base query fields
func (q *MongoDBSelectQuery) GetDatabase() types.Database {
	return q.db
}

func (q *MongoDBSelectQuery) GetFieldMapper() types.FieldMapper {
	return q.fieldMapper
}

func (q *MongoDBSelectQuery) GetModelName() string {
	return q.modelName
}

// Override SelectQuery methods to preserve MongoDB-specific type
func (q *MongoDBSelectQuery) WhereCondition(condition types.Condition) types.SelectQuery {
	newBase := q.SelectQueryImpl.WhereCondition(condition).(*query.SelectQueryImpl)
	return &MongoDBSelectQuery{
		SelectQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBSelectQuery) Include(relations ...string) types.SelectQuery {
	newBase := q.SelectQueryImpl.Include(relations...).(*query.SelectQueryImpl)
	return &MongoDBSelectQuery{
		SelectQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBSelectQuery) IncludeWithOptions(path string, opt *types.IncludeOption) types.SelectQuery {
	newBase := q.SelectQueryImpl.IncludeWithOptions(path, opt).(*query.SelectQueryImpl)
	return &MongoDBSelectQuery{
		SelectQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBSelectQuery) OrderBy(fieldName string, direction types.Order) types.SelectQuery {
	newBase := q.SelectQueryImpl.OrderBy(fieldName, direction).(*query.SelectQueryImpl)
	return &MongoDBSelectQuery{
		SelectQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBSelectQuery) GroupBy(fieldNames ...string) types.SelectQuery {
	newBase := q.SelectQueryImpl.GroupBy(fieldNames...).(*query.SelectQueryImpl)
	return &MongoDBSelectQuery{
		SelectQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBSelectQuery) Having(condition types.Condition) types.SelectQuery {
	newBase := q.SelectQueryImpl.Having(condition).(*query.SelectQueryImpl)
	return &MongoDBSelectQuery{
		SelectQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBSelectQuery) Limit(limit int) types.SelectQuery {
	newBase := q.SelectQueryImpl.Limit(limit).(*query.SelectQueryImpl)
	return &MongoDBSelectQuery{
		SelectQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBSelectQuery) Offset(offset int) types.SelectQuery {
	newBase := q.SelectQueryImpl.Offset(offset).(*query.SelectQueryImpl)
	return &MongoDBSelectQuery{
		SelectQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBSelectQuery) Distinct() types.SelectQuery {
	newBase := q.SelectQueryImpl.Distinct().(*query.SelectQueryImpl)
	return &MongoDBSelectQuery{
		SelectQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

func (q *MongoDBSelectQuery) DistinctOn(fieldNames ...string) types.SelectQuery {
	newBase := q.SelectQueryImpl.DistinctOn(fieldNames...).(*query.SelectQueryImpl)
	return &MongoDBSelectQuery{
		SelectQueryImpl: newBase,
		db:              q.db,
		fieldMapper:     q.fieldMapper,
		modelName:       q.modelName,
	}
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// shouldUnwindRelation determines if a relation should be unwound (converted from array to single object)
func (q *MongoDBSelectQuery) shouldUnwindRelation(relationName string) (bool, error) {
	// Get the current model's schema
	currentSchema, err := q.db.GetSchema(q.modelName)
	if err != nil {
		return false, fmt.Errorf("failed to get schema for model %s: %w", q.modelName, err)
	}

	// Get the relation definition
	relation, exists := currentSchema.Relations[relationName]
	if !exists {
		return false, fmt.Errorf("relation %s not found in model %s", relationName, q.modelName)
	}

	// Unwind for many-to-one and one-to-one relations (they should return single objects)
	return relation.Type == schema.RelationManyToOne || relation.Type == schema.RelationOneToOne, nil
}
