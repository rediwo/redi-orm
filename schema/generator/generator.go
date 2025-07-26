package generator

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rediwo/redi-orm/prisma"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// SchemaGenerator converts internal schema to Prisma format
type SchemaGenerator struct {
	migrator types.DatabaseSpecificMigrator
}

// NewSchemaGenerator creates a new schema generator
func NewSchemaGenerator(migrator types.DatabaseSpecificMigrator) *SchemaGenerator {
	return &SchemaGenerator{
		migrator: migrator,
	}
}

// GeneratePrismaSchema converts a Schema to Prisma schema string
func (g *SchemaGenerator) GeneratePrismaSchema(s *schema.Schema) (string, error) {
	ast, err := g.SchemaToPrismaAST(s)
	if err != nil {
		return "", err
	}
	return ast.String(), nil
}

// SchemaToPrismaAST converts a Schema to Prisma AST
func (g *SchemaGenerator) SchemaToPrismaAST(s *schema.Schema) (*prisma.ModelStatement, error) {
	model := &prisma.ModelStatement{
		Name:            s.Name,
		Fields:          []*prisma.Field{},
		BlockAttributes: []*prisma.BlockAttribute{},
	}

	// Add table name mapping if different from model name
	if s.TableName != "" && s.TableName != schema.ModelNameToTableName(s.Name) {
		model.BlockAttributes = append(model.BlockAttributes, &prisma.BlockAttribute{
			Name: "map",
			Args: []prisma.Expression{
				&prisma.StringLiteral{Value: s.TableName},
			},
		})
	}

	// Convert fields
	for _, field := range s.Fields {
		prismaField, err := g.fieldToPrismaField(field)
		if err != nil {
			return nil, err
		}
		model.Fields = append(model.Fields, prismaField)
	}

	// Convert relations to fields
	for relationName, relation := range s.Relations {
		relationField := g.relationToPrismaField(relationName, relation)
		model.Fields = append(model.Fields, relationField)
	}

	// Add composite primary key if exists
	if len(s.CompositeKey) > 0 {
		var keyExprs []prisma.Expression
		for _, key := range s.CompositeKey {
			keyExprs = append(keyExprs, &prisma.Identifier{Value: key})
		}
		model.BlockAttributes = append(model.BlockAttributes, &prisma.BlockAttribute{
			Name: "id",
			Args: []prisma.Expression{
				&prisma.ArrayExpression{Elements: keyExprs},
			},
		})
	}

	// Add indexes
	for _, index := range s.Indexes {
		var fieldExprs []prisma.Expression
		for _, field := range index.Fields {
			fieldExprs = append(fieldExprs, &prisma.Identifier{Value: field})
		}

		attrName := "index"
		if index.Unique {
			attrName = "unique"
		}

		model.BlockAttributes = append(model.BlockAttributes, &prisma.BlockAttribute{
			Name: attrName,
			Args: []prisma.Expression{
				&prisma.ArrayExpression{Elements: fieldExprs},
			},
		})
	}

	return model, nil
}

// fieldToPrismaField converts a Field to prisma.Field
func (g *SchemaGenerator) fieldToPrismaField(f schema.Field) (*prisma.Field, error) {
	prismaType := g.fieldTypeToPrismaType(f.Type)

	field := &prisma.Field{
		Name:       f.Name,
		Type:       &prisma.FieldType{Name: prismaType},
		Optional:   f.Nullable,
		List:       g.isArrayType(f.Type),
		Attributes: []*prisma.Attribute{},
	}

	// Add attributes
	if f.PrimaryKey {
		field.Attributes = append(field.Attributes, &prisma.Attribute{Name: "id"})
	}

	if f.Unique {
		field.Attributes = append(field.Attributes, &prisma.Attribute{Name: "unique"})
	}

	if f.Default != nil {
		// Always use convertDefaultValue with migrator
		defaultExpr := g.convertDefaultValue(f.Default, f.Type, f.AutoIncrement, g.migrator)
		if defaultExpr != nil {
			field.Attributes = append(field.Attributes, &prisma.Attribute{
				Name: "default",
				Args: []prisma.Expression{defaultExpr},
			})
		}
	} else if f.AutoIncrement {
		field.Attributes = append(field.Attributes, &prisma.Attribute{
			Name: "default",
			Args: []prisma.Expression{
				&prisma.FunctionCall{Name: "autoincrement"},
			},
		})
	}

	// Add @map attribute if field has custom column mapping
	if f.Map != "" {
		field.Attributes = append(field.Attributes, &prisma.Attribute{
			Name: "map",
			Args: []prisma.Expression{
				&prisma.StringLiteral{Value: f.Map},
			},
		})
	}

	// Add database-specific type if specified
	if f.DbType != "" && strings.HasPrefix(f.DbType, "@") {
		// Parse @db.VarChar(255) format
		dbType := strings.TrimPrefix(f.DbType, "@")
		if strings.Contains(dbType, "(") {
			// Has parameters
			parts := strings.SplitN(dbType, "(", 2)
			attrName := parts[0]
			params := strings.TrimSuffix(parts[1], ")")

			field.Attributes = append(field.Attributes, &prisma.Attribute{
				Name: attrName,
				Args: g.parseDbTypeArgs(params),
			})
		} else {
			// No parameters
			field.Attributes = append(field.Attributes, &prisma.Attribute{Name: dbType})
		}
	}

	return field, nil
}

// relationToPrismaField converts a relation to a Prisma field
func (g *SchemaGenerator) relationToPrismaField(name string, r schema.Relation) *prisma.Field {
	field := &prisma.Field{
		Name:       name,
		Type:       &prisma.FieldType{Name: r.Model},
		Optional:   r.Type == schema.RelationManyToOne || r.Type == schema.RelationOneToOne,
		List:       r.Type == schema.RelationOneToMany || r.Type == schema.RelationManyToMany,
		Attributes: []*prisma.Attribute{},
	}

	// Add @relation attribute if we have foreign key info
	if r.ForeignKey != "" || r.References != "" {
		var args []prisma.Expression

		if r.ForeignKey != "" {
			args = append(args, &prisma.NamedArgument{
				Name: "fields",
				Value: &prisma.ArrayExpression{
					Elements: []prisma.Expression{
						&prisma.Identifier{Value: r.ForeignKey},
					},
				},
			})
		}

		if r.References != "" {
			args = append(args, &prisma.NamedArgument{
				Name: "references",
				Value: &prisma.ArrayExpression{
					Elements: []prisma.Expression{
						&prisma.Identifier{Value: r.References},
					},
				},
			})
		}

		if len(args) > 0 {
			field.Attributes = append(field.Attributes, &prisma.Attribute{
				Name: "relation",
				Args: args,
			})
		}
	}

	return field
}

// fieldTypeToPrismaType converts schema.FieldType to Prisma type string
func (g *SchemaGenerator) fieldTypeToPrismaType(ft schema.FieldType) string {
	switch ft {
	case schema.FieldTypeString, schema.FieldTypeStringArray:
		return "String"
	case schema.FieldTypeInt, schema.FieldTypeIntArray:
		return "Int"
	case schema.FieldTypeInt64, schema.FieldTypeInt64Array:
		return "BigInt"
	case schema.FieldTypeFloat, schema.FieldTypeFloatArray:
		return "Float"
	case schema.FieldTypeBool, schema.FieldTypeBoolArray:
		return "Boolean"
	case schema.FieldTypeDateTime, schema.FieldTypeDateTimeArray:
		return "DateTime"
	case schema.FieldTypeJSON:
		return "Json"
	case schema.FieldTypeDecimal, schema.FieldTypeDecimalArray:
		return "Decimal"
	case schema.FieldTypeObjectId:
		return "String" // ObjectId is represented as String in Prisma
	case schema.FieldTypeBinary:
		return "Bytes"
	case schema.FieldTypeDecimal128:
		return "Decimal"
	case schema.FieldTypeTimestamp:
		return "DateTime"
	case schema.FieldTypeDocument, schema.FieldTypeArray:
		return "Json"
	default:
		return "String"
	}
}

// isArrayType checks if a field type is an array
func (g *SchemaGenerator) isArrayType(ft schema.FieldType) bool {
	switch ft {
	case schema.FieldTypeStringArray, schema.FieldTypeIntArray, schema.FieldTypeInt64Array,
		schema.FieldTypeFloatArray, schema.FieldTypeBoolArray, schema.FieldTypeDecimalArray,
		schema.FieldTypeDateTimeArray, schema.FieldTypeArray:
		return true
	default:
		return false
	}
}

// convertDefaultValue converts a default value to Prisma expression using database-specific parsing
func (g *SchemaGenerator) convertDefaultValue(value any, fieldType schema.FieldType, isAutoIncrement bool, migrator types.DatabaseSpecificMigrator) prisma.Expression {
	if isAutoIncrement {
		return &prisma.FunctionCall{Name: "autoincrement"}
	}

	// Let the database driver normalize the value first
	normalized := migrator.ParseDefaultValue(value, fieldType)

	// Check if it's a special function
	if funcName, isFunc := migrator.NormalizeDefaultToPrismaFunction(normalized, fieldType); isFunc {
		return &prisma.FunctionCall{Name: funcName}
	}

	// Handle regular values
	switch v := normalized.(type) {
	case string:
		return &prisma.StringLiteral{Value: v}
	case int, int32, int64:
		return &prisma.NumberLiteral{Value: fmt.Sprintf("%d", v)}
	case float32, float64:
		return &prisma.NumberLiteral{Value: fmt.Sprintf("%f", v)}
	case bool:
		if v {
			return &prisma.Identifier{Value: "true"}
		}
		return &prisma.Identifier{Value: "false"}
	case nil:
		return nil
	default:
		// Fallback to string representation
		return &prisma.StringLiteral{Value: fmt.Sprintf("%v", v)}
	}
}

// parseDbTypeArgs parses database type arguments
func (g *SchemaGenerator) parseDbTypeArgs(params string) []prisma.Expression {
	var args []prisma.Expression

	// Simple parsing for now - just handle comma-separated values
	parts := strings.Split(params, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Try to parse as number
		if _, err := fmt.Sscanf(part, "%d", new(int)); err == nil {
			args = append(args, &prisma.NumberLiteral{Value: part})
		} else {
			// Treat as string
			args = append(args, &prisma.StringLiteral{Value: strings.Trim(part, `"'`)})
		}
	}

	return args
}

// GenerateFullPrismaFile generates a complete Prisma schema file with all models
func (g *SchemaGenerator) GenerateFullPrismaFile(schemas []*schema.Schema, datasource *prisma.DatasourceStatement, generator *prisma.GeneratorStatement) (string, error) {
	var builder strings.Builder

	// Add comment for auto-generated files
	if generator != nil && len(generator.Properties) > 0 {
		for _, prop := range generator.Properties {
			if prop.Name == "provider" {
				if strLit, ok := prop.Value.(*prisma.StringLiteral); ok && strLit.Value == "RediORM Auto Generator" {
					builder.WriteString("// This schema was auto-generated by RediORM from existing database tables\n")
					builder.WriteString("// Generated at: " + time.Now().Format(time.RFC3339) + "\n\n")
					break
				}
			}
		}
	}

	// Add datasource only if explicitly provided
	if datasource != nil {
		builder.WriteString(datasource.String())
		builder.WriteString("\n\n")
	}

	// Add generator only if explicitly provided
	if generator != nil {
		builder.WriteString(generator.String())
		builder.WriteString("\n\n")
	}

	// Sort schemas by name for consistent output
	sortedSchemas := make([]*schema.Schema, len(schemas))
	copy(sortedSchemas, schemas)
	sort.Slice(sortedSchemas, func(i, j int) bool {
		return sortedSchemas[i].Name < sortedSchemas[j].Name
	})

	// Add models
	for i, s := range sortedSchemas {
		if i > 0 {
			builder.WriteString("\n")
		}

		model, err := g.SchemaToPrismaAST(s)
		if err != nil {
			return "", fmt.Errorf("failed to convert schema %s: %w", s.Name, err)
		}

		builder.WriteString(model.String())
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// GenerateSchemaFromTable creates a schema from database table information
func GenerateSchemaFromTable(tableInfo *types.TableInfo, migrator types.DatabaseSpecificMigrator) (*schema.Schema, error) {
	// Create new schema with model name from table name
	// Use PascalCase for model names (e.g., users -> User, post_tags -> PostTag)
	modelName := utils.Singularize(utils.ToPascalCase(tableInfo.Name))
	s := schema.New(modelName)
	s.WithTableName(tableInfo.Name)

	// Track primary key fields for composite key detection
	var primaryKeyFields []string

	// Add fields from columns
	for _, col := range tableInfo.Columns {
		// Convert column name to camelCase for field name
		fieldName := utils.ToCamelCase(col.Name)

		field := schema.Field{
			Name:          fieldName,
			Type:          migrator.MapDatabaseTypeToFieldType(col.Type),
			Nullable:      col.Nullable,
			PrimaryKey:    col.PrimaryKey,
			AutoIncrement: col.AutoIncrement,
			Unique:        col.Unique,
		}

		// Use migrator to parse and normalize default value
		if col.Default != nil && migrator != nil {
			field.Default = migrator.ParseDefaultValue(col.Default, field.Type)
		} else {
			field.Default = col.Default
		}

		// Only set Map if the column name doesn't match the standard conversion
		// e.g., "parent_id" -> "parentId" -> "parent_id" (no map needed)
		// but "USER_ID" -> "userId" -> "user_id" != "USER_ID" (map needed)
		expectedColumnName := utils.ToSnakeCase(fieldName)
		if col.Name != expectedColumnName {
			field.Map = col.Name
		}

		// Track primary key fields
		if col.PrimaryKey {
			primaryKeyFields = append(primaryKeyFields, field.Name)
		}

		s.AddField(field)
	}

	// Set composite key if multiple primary key fields
	if len(primaryKeyFields) > 1 {
		s.CompositeKey = primaryKeyFields
	}

	// Add indexes (excluding primary key indexes)
	for _, idx := range tableInfo.Indexes {
		// Skip primary key indexes
		if migrator.IsPrimaryKeyIndex(idx.Name) {
			continue
		}

		// Convert column names to field names
		fieldNames := make([]string, len(idx.Columns))
		for i, colName := range idx.Columns {
			fieldNames[i] = utils.ToCamelCase(colName)
		}

		s.AddIndex(schema.Index{
			Name:   idx.Name,
			Fields: fieldNames,
			Unique: idx.Unique,
		})
	}

	// Note: Foreign keys could be used to infer relations, but this requires
	// knowledge of other tables/models. This could be added in a second pass.

	return s, nil
}

// GenerateSchemasFromTablesWithRelations generates schemas with relations inferred from foreign keys
func GenerateSchemasFromTablesWithRelations(migrator types.DatabaseMigrator) ([]*schema.Schema, error) {
	// Get all tables from database
	tables, err := migrator.GetTables()
	if err != nil {
		return nil, fmt.Errorf("failed to get tables: %w", err)
	}

	// First pass: Generate all schemas without relations
	schemaMap := make(map[string]*schema.Schema)
	tableInfoMap := make(map[string]*types.TableInfo)

	for _, tableName := range tables {
		// Skip system tables
		if migrator.IsSystemTable(tableName) {
			continue
		}

		// Get table information
		tableInfo, err := migrator.GetTableInfo(tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to get info for table %s: %w", tableName, err)
		}

		// Generate schema from table
		// Try to cast to DatabaseSpecificMigrator for better default value handling
		var specificMigrator types.DatabaseSpecificMigrator
		if wrapper, ok := migrator.(interface {
			GetSpecific() types.DatabaseSpecificMigrator
		}); ok {
			specificMigrator = wrapper.GetSpecific()
		} else {
			return nil, fmt.Errorf("migrator must have GetSpecific() method that returns DatabaseSpecificMigrator")
		}

		schema, err := GenerateSchemaFromTable(tableInfo, specificMigrator)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema for table %s: %w", tableName, err)
		}

		schemaMap[tableName] = schema
		tableInfoMap[tableName] = tableInfo
	}

	// Second pass: Add relations based on foreign keys
	for tableName, tableInfo := range tableInfoMap {
		schemaObj := schemaMap[tableName]

		for _, fk := range tableInfo.ForeignKeys {
			// Skip if referenced table doesn't exist in our schemas
			referencedSchema, exists := schemaMap[fk.ReferencedTable]
			if !exists {
				continue
			}

			// Add many-to-one relation on this model
			// Use the foreign key column name without _id suffix as the relation name
			relationName := utils.ToCamelCase(strings.TrimSuffix(fk.Column, "_id"))
			if relationName == utils.ToCamelCase(fk.Column) {
				// If trimming _id didn't change anything, try to use the referenced table name
				relationName = utils.ToCamelCase(utils.Singularize(fk.ReferencedTable))
			}

			schemaObj.AddRelation(relationName, schema.Relation{
				Type:       schema.RelationManyToOne,
				Model:      referencedSchema.Name,
				ForeignKey: utils.ToCamelCase(fk.Column),
				References: utils.ToCamelCase(fk.ReferencedColumn),
				OnDelete:   fk.OnDelete,
				OnUpdate:   fk.OnUpdate,
			})

			// Add one-to-many relation on referenced model
			// Use the plural form of the model name (not table name)
			// First convert to lowercase for proper pluralization
			inverseName := utils.Pluralize(strings.ToLower(schemaObj.Name))
			referencedSchema.AddRelation(inverseName, schema.Relation{
				Type:       schema.RelationOneToMany,
				Model:      schemaObj.Name,
				ForeignKey: utils.ToCamelCase(fk.Column),
				References: utils.ToCamelCase(fk.ReferencedColumn),
			})
		}
	}

	// Convert map to slice
	var schemas []*schema.Schema
	for _, s := range schemaMap {
		schemas = append(schemas, s)
	}

	// Sort by name for consistent output
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})

	return schemas, nil
}
