package query

import (
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// JoinType represents the type of SQL join
type JoinType string

const (
	InnerJoin JoinType = "INNER JOIN"
	LeftJoin  JoinType = "LEFT JOIN"
	RightJoin JoinType = "RIGHT JOIN"
	FullJoin  JoinType = "FULL OUTER JOIN"
)

// JoinClause represents a single join operation
type JoinClause struct {
	Type         JoinType
	Table        string               // Table to join
	Alias        string               // Table alias
	Condition    string               // Join condition
	Schema       *schema.Schema       // Schema of joined table
	Relation     *schema.Relation     // Relation definition
	RelationName string               // Name of the relation field (e.g., "posts")
	ParentAlias  string               // Alias of the parent table
	RelationPath string               // Full relation path (e.g., "posts.comments")
	NestedJoins  []JoinClause         // Nested joins for this table
	IncludeOpt   *types.IncludeOption // Include options for filtering
}

// JoinBuilder handles building SQL joins from relations
type JoinBuilder struct {
	database       types.Database
	joins          []JoinClause
	tableAliases   map[string]int // Track alias counters
	schemaCache    map[string]*schema.Schema
	joinedPaths    map[string]string               // Track joined relation paths to their aliases
	includeOptions map[string]*types.IncludeOption // Include options by relation path
}

// NewJoinBuilder creates a new join builder
func NewJoinBuilder(database types.Database) *JoinBuilder {
	return &JoinBuilder{
		database:       database,
		joins:          []JoinClause{},
		tableAliases:   make(map[string]int),
		schemaCache:    make(map[string]*schema.Schema),
		joinedPaths:    make(map[string]string),
		includeOptions: make(map[string]*types.IncludeOption),
	}
}

// NewJoinBuilderWithReservedAliases creates a new join builder with reserved aliases
func NewJoinBuilderWithReservedAliases(database types.Database, reservedAliases ...string) *JoinBuilder {
	jb := &JoinBuilder{
		database:       database,
		joins:          []JoinClause{},
		tableAliases:   make(map[string]int),
		schemaCache:    make(map[string]*schema.Schema),
		joinedPaths:    make(map[string]string),
		includeOptions: make(map[string]*types.IncludeOption),
	}

	// Mark reserved aliases as used
	for _, alias := range reservedAliases {
		jb.tableAliases[alias] = 1
	}

	return jb
}

// AddRelationJoin adds a join based on a relation
func (b *JoinBuilder) AddRelationJoin(
	fromModel string,
	fromAlias string,
	relationName string,
	joinType JoinType,
) error {
	// Get schemas
	fromSchema, err := b.getSchema(fromModel)
	if err != nil {
		return fmt.Errorf("failed to get schema for %s: %w", fromModel, err)
	}

	// Get relation
	relation, err := fromSchema.GetRelation(relationName)
	if err != nil {
		return fmt.Errorf("failed to get relation %s: %w", relationName, err)
	}

	// Get related schema
	relatedSchema, err := b.getSchema(relation.Model)
	if err != nil {
		return fmt.Errorf("failed to get schema for related model %s: %w", relation.Model, err)
	}

	// Generate table name and alias
	relatedTable := relatedSchema.GetTableName()
	alias := b.generateAlias(relatedTable)

	// Build join condition
	condition, err := b.buildJoinCondition(&relation, fromAlias, alias, fromSchema, relatedSchema)
	if err != nil {
		return fmt.Errorf("failed to build join condition: %w", err)
	}

	// Create join clause
	join := JoinClause{
		Type:         joinType,
		Table:        relatedTable,
		Alias:        alias,
		Condition:    condition,
		Schema:       relatedSchema,
		Relation:     &relation,
		RelationName: relationName,
		ParentAlias:  fromAlias,
		RelationPath: relationName, // Will be updated in AddNestedRelationJoin for nested paths
	}

	b.joins = append(b.joins, join)
	return nil
}

// AddNestedRelationJoin adds a nested relation join
func (b *JoinBuilder) AddNestedRelationJoin(
	parentAlias string,
	parentModel string,
	relationPath []string,
	joinType JoinType,
) error {
	if len(relationPath) == 0 {
		return nil
	}

	currentModel := parentModel
	currentAlias := parentAlias
	currentPath := ""

	for _, relationName := range relationPath {
		// Build the path for this level
		if currentPath == "" {
			currentPath = relationName
		} else {
			currentPath = currentPath + "." + relationName
		}

		// Check if this path is already joined
		if existingAlias, exists := b.joinedPaths[currentPath]; exists {
			// This relation is already joined, use its alias for the next level
			currentAlias = existingAlias

			// Get the relation to update currentModel for next iteration
			currentSchema, err := b.getSchema(currentModel)
			if err != nil {
				return fmt.Errorf("failed to get schema for %s: %w", currentModel, err)
			}
			relation, err := currentSchema.GetRelation(relationName)
			if err != nil {
				return fmt.Errorf("failed to get relation %s in model %s: %w", relationName, currentModel, err)
			}
			currentModel = relation.Model
			continue
		}

		// Get current schema
		currentSchema, err := b.getSchema(currentModel)
		if err != nil {
			return fmt.Errorf("failed to get schema for %s: %w", currentModel, err)
		}

		// Get relation
		relation, err := currentSchema.GetRelation(relationName)
		if err != nil {
			return fmt.Errorf("failed to get relation %s in model %s: %w", relationName, currentModel, err)
		}

		// Add join for this relation
		err = b.AddRelationJoin(currentModel, currentAlias, relationName, joinType)
		if err != nil {
			return err
		}

		// Update the relation path for the join we just added
		if len(b.joins) > 0 {
			b.joins[len(b.joins)-1].RelationPath = currentPath
		}

		// Update current model and alias for next iteration
		currentModel = relation.Model
		// Get the alias of the join we just added
		if len(b.joins) > 0 {
			newAlias := b.joins[len(b.joins)-1].Alias
			b.joinedPaths[currentPath] = newAlias
			currentAlias = newAlias
		}
	}

	return nil
}

// SetIncludeOptions sets include options for specific relation paths
func (b *JoinBuilder) SetIncludeOptions(includeOptions map[string]*types.IncludeOption) {
	b.includeOptions = includeOptions
}

// BuildSQL generates the JOIN SQL clauses
func (b *JoinBuilder) BuildSQL() string {
	if len(b.joins) == 0 {
		return ""
	}

	var parts []string
	for _, join := range b.joins {
		// Get include options for this join if they exist
		includeOpt, hasOpt := b.includeOptions[join.RelationPath]

		// Build the base join condition
		condition := join.Condition

		// SQL-level filtering: add WHERE conditions to the JOIN ON clause
		if hasOpt && includeOpt.Where != nil && join.Schema != nil {
			// Create a condition context for the joined table
			// Create a temporary field mapper with the schema
			fieldMapper := types.NewDefaultFieldMapper()
			fieldMapper.RegisterSchema(join.Relation.Model, join.Schema)
			ctx := types.NewConditionContext(fieldMapper, join.Relation.Model, join.Alias)

			// Build the WHERE condition SQL
			whereSql, whereArgs := includeOpt.Where.ToSQL(ctx)
			if whereSql != "" {
				// For now, we'll embed the args as literals for the JOIN clause
				// This is a simplified implementation - in production, we'd need better handling
				for _, arg := range whereArgs {
					// Replace the first placeholder with the literal value
					switch v := arg.(type) {
					case string:
						whereSql = strings.Replace(whereSql, "?", fmt.Sprintf("'%s'", v), 1)
					case bool:
						// Use database-specific boolean literal
						whereSql = strings.Replace(whereSql, "?", b.database.GetBooleanLiteral(v), 1)
					default:
						whereSql = strings.Replace(whereSql, "?", fmt.Sprintf("%v", v), 1)
					}
				}
				condition = fmt.Sprintf("%s AND %s", condition, whereSql)
			}
		}

		parts = append(parts, fmt.Sprintf("%s %s AS %s ON %s",
			join.Type,
			join.Table,
			join.Alias,
			condition,
		))
	}

	return strings.Join(parts, " ")
}

// GetJoinedTables returns information about all joined tables
func (b *JoinBuilder) GetJoinedTables() []JoinClause {
	return b.joins
}

// generateAlias generates a unique alias for a table
func (b *JoinBuilder) generateAlias(tableName string) string {
	// Get base alias (first letter of each word)
	parts := strings.Split(tableName, "_")
	var alias string
	for _, part := range parts {
		if len(part) > 0 {
			alias += string(part[0])
		}
	}

	// Make it unique
	count, exists := b.tableAliases[alias]
	if !exists {
		b.tableAliases[alias] = 1
		return alias
	}

	b.tableAliases[alias] = count + 1
	return fmt.Sprintf("%s%d", alias, count+1)
}

// getSchema retrieves a schema from cache or database
func (b *JoinBuilder) getSchema(modelName string) (*schema.Schema, error) {
	if cached, exists := b.schemaCache[modelName]; exists {
		return cached, nil
	}

	schema, err := b.database.GetModelSchema(modelName)
	if err != nil {
		return nil, err
	}

	b.schemaCache[modelName] = schema
	return schema, nil
}

// buildJoinCondition builds the SQL join condition
func (b *JoinBuilder) buildJoinCondition(
	relation *schema.Relation,
	fromAlias string,
	toAlias string,
	fromSchema *schema.Schema,
	toSchema *schema.Schema,
) (string, error) {
	switch relation.Type {
	case schema.RelationManyToOne:
		// FROM.foreign_key = TO.references
		fromCol, err := fromSchema.GetColumnNameByFieldName(relation.ForeignKey)
		if err != nil {
			return "", err
		}
		toCol, err := toSchema.GetColumnNameByFieldName(relation.References)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s = %s.%s", fromAlias, fromCol, toAlias, toCol), nil

	case schema.RelationOneToMany:
		// TO.foreign_key = FROM.id (or references field)
		toCol, err := toSchema.GetColumnNameByFieldName(relation.ForeignKey)
		if err != nil {
			return "", err
		}

		fromField := relation.References
		if fromField == "" {
			fromField = "id"
		}
		fromCol, err := fromSchema.GetColumnNameByFieldName(fromField)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s = %s.%s", toAlias, toCol, fromAlias, fromCol), nil

	case schema.RelationOneToOne:
		// Check which side has the foreign key
		if _, err := fromSchema.GetField(relation.ForeignKey); err == nil {
			// Foreign key in from table
			fromCol, err := fromSchema.GetColumnNameByFieldName(relation.ForeignKey)
			if err != nil {
				return "", err
			}
			toCol, err := toSchema.GetColumnNameByFieldName(relation.References)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s.%s = %s.%s", fromAlias, fromCol, toAlias, toCol), nil
		} else {
			// Foreign key in to table
			toCol, err := toSchema.GetColumnNameByFieldName(relation.ForeignKey)
			if err != nil {
				return "", err
			}
			fromField := relation.References
			if fromField == "" {
				fromField = "id"
			}
			fromCol, err := fromSchema.GetColumnNameByFieldName(fromField)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s.%s = %s.%s", toAlias, toCol, fromAlias, fromCol), nil
		}

	case schema.RelationManyToMany:
		// Many-to-many requires special handling with junction table
		return "", fmt.Errorf("many-to-many relations require junction table joins")

	default:
		return "", fmt.Errorf("unknown relation type: %s", relation.Type)
	}
}

// AddManyToManyJoin adds joins for a many-to-many relation
func (b *JoinBuilder) AddManyToManyJoin(
	fromModel string,
	fromAlias string,
	relationName string,
	joinType JoinType,
) error {
	// Get schemas
	fromSchema, err := b.getSchema(fromModel)
	if err != nil {
		return fmt.Errorf("failed to get schema for %s: %w", fromModel, err)
	}

	// Get relation
	relation, err := fromSchema.GetRelation(relationName)
	if err != nil {
		return fmt.Errorf("failed to get relation %s: %w", relationName, err)
	}

	if relation.Type != schema.RelationManyToMany {
		return fmt.Errorf("relation %s is not many-to-many", relationName)
	}

	// Get related schema
	relatedSchema, err := b.getSchema(relation.Model)
	if err != nil {
		return fmt.Errorf("failed to get schema for related model %s: %w", relation.Model, err)
	}

	// Generate junction table name
	junctionTable := schema.GetJunctionTableName(fromModel, relation.Model)
	junctionAlias := b.generateAlias(junctionTable)

	// First join: from table to junction table
	fromField := "id"
	fromCol, _ := fromSchema.GetColumnNameByFieldName(fromField)
	junctionFromCol := strings.ToLower(fromSchema.GetTableName()) + "_id"

	join1 := JoinClause{
		Type:      joinType,
		Table:     junctionTable,
		Alias:     junctionAlias,
		Condition: fmt.Sprintf("%s.%s = %s.%s", fromAlias, fromCol, junctionAlias, junctionFromCol),
	}
	b.joins = append(b.joins, join1)

	// Second join: junction table to related table
	relatedAlias := b.generateAlias(relatedSchema.GetTableName())
	relatedField := relation.References
	if relatedField == "" {
		relatedField = "id"
	}
	relatedCol, _ := relatedSchema.GetColumnNameByFieldName(relatedField)
	junctionToCol := strings.ToLower(relatedSchema.GetTableName()) + "_id"

	join2 := JoinClause{
		Type:      joinType,
		Table:     relatedSchema.GetTableName(),
		Alias:     relatedAlias,
		Condition: fmt.Sprintf("%s.%s = %s.%s", junctionAlias, junctionToCol, relatedAlias, relatedCol),
		Schema:    relatedSchema,
		Relation:  &relation,
	}
	b.joins = append(b.joins, join2)

	return nil
}
