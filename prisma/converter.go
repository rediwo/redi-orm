package prisma

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rediwo/redi-orm/schema"
)

// Converter converts Prisma AST to ReORM Schema objects
type Converter struct {
	schemas    map[string]*schema.Schema
	enums      map[string][]*EnumValue
	datasource *DatasourceStatement
	generator  *GeneratorStatement
}

// NewConverter creates a new converter instance
func NewConverter() *Converter {
	return &Converter{
		schemas: make(map[string]*schema.Schema),
		enums:   make(map[string][]*EnumValue),
	}
}

// Convert converts a Prisma schema AST to ReORM schemas
func (c *Converter) Convert(prismaSchema *PrismaSchema) (map[string]*schema.Schema, error) {
	// First pass: collect datasource, generator, and enums
	for _, stmt := range prismaSchema.Statements {
		switch s := stmt.(type) {
		case *DatasourceStatement:
			c.datasource = s
		case *GeneratorStatement:
			c.generator = s
		case *EnumStatement:
			c.enums[s.Name] = s.Values
		}
	}

	// Second pass: convert models
	for _, stmt := range prismaSchema.Statements {
		if modelStmt, ok := stmt.(*ModelStatement); ok {
			s, err := c.convertModel(modelStmt)
			if err != nil {
				return nil, fmt.Errorf("failed to convert model %s: %v", modelStmt.Name, err)
			}
			c.schemas[modelStmt.Name] = s
		}
	}

	// Third pass: add relations
	for _, stmt := range prismaSchema.Statements {
		if modelStmt, ok := stmt.(*ModelStatement); ok {
			if err := c.addRelations(modelStmt); err != nil {
				return nil, fmt.Errorf("failed to add relations for model %s: %v", modelStmt.Name, err)
			}
		}
	}

	return c.schemas, nil
}

// convertModel converts a Prisma model to a ReORM schema
func (c *Converter) convertModel(modelStmt *ModelStatement) (*schema.Schema, error) {
	s := schema.New(modelStmt.Name)

	// Extract table name from block attributes if specified
	tableName := c.extractTableName(modelStmt.BlockAttributes)
	if tableName != "" {
		s.WithTableName(tableName)
	}

	// Convert fields
	for _, field := range modelStmt.Fields {
		if c.isRelationField(field) {
			// Skip relation fields in this pass, they'll be handled separately
			continue
		}

		f, err := c.convertField(field)
		if err != nil {
			return nil, fmt.Errorf("failed to convert field %s: %v", field.Name, err)
		}
		s.AddField(f)
	}

	// Add indexes from block attributes
	indexes := c.extractIndexes(modelStmt.BlockAttributes)
	for _, idx := range indexes {
		s.AddIndex(idx)
	}

	// Check for composite primary key
	compositeKey := c.extractCompositeKey(modelStmt.BlockAttributes)
	if len(compositeKey) > 0 {
		s.WithCompositeKey(compositeKey)
	}

	return s, nil
}

// convertField converts a Prisma field to a ReORM field
func (c *Converter) convertField(field *Field) (schema.Field, error) {
	f := schema.Field{
		Name:     field.Name,
		Nullable: field.Optional,
	}

	// Convert type
	fieldType, err := c.convertType(field.Type.Name, field.List)
	if err != nil {
		return f, fmt.Errorf("failed to convert field %s type %s: %v", field.Name, field.Type.Name, err)
	}
	f.Type = fieldType

	// Process attributes
	for _, attr := range field.Attributes {
		switch attr.Name {
		case "id":
			f.PrimaryKey = true
		case "unique":
			f.Unique = true
		case "default":
			if len(attr.Args) > 0 {
				// Check if the default value is autoincrement()
				if fc, ok := attr.Args[0].(*FunctionCall); ok && fc.Name == "autoincrement" {
					f.AutoIncrement = true
					f.Default = "AUTO_INCREMENT"
				} else {
					defaultValue, err := c.convertExpression(attr.Args[0])
					if err != nil {
						return f, fmt.Errorf("failed to convert default value: %v", err)
					}
					f.Default = defaultValue
				}
			}
		case "autoincrement":
			f.AutoIncrement = true
		default:
			// Handle @db.* attributes (e.g., @db.VarChar(255), @db.Money)
			if strings.HasPrefix(attr.Name, "db.") {
				if len(attr.Args) > 0 {
					// Convert arguments to string representation
					var args []string
					for _, arg := range attr.Args {
						if argStr, err := c.convertExpression(arg); err == nil {
							args = append(args, fmt.Sprintf("%v", argStr))
						}
					}
					if len(args) > 0 {
						f.DbType = "@" + attr.Name + "(" + strings.Join(args, ", ") + ")"
					} else {
						f.DbType = "@" + attr.Name
					}
				} else {
					f.DbType = "@" + attr.Name
				}
			}
		}
	}

	return f, nil
}

// convertType converts Prisma type to ReORM FieldType
func (c *Converter) convertType(typeName string, isList bool) (schema.FieldType, error) {
	// Handle array types
	if isList {
		switch typeName {
		case "String":
			return schema.FieldTypeStringArray, nil
		case "Int":
			return schema.FieldTypeIntArray, nil
		case "BigInt":
			return schema.FieldTypeInt64Array, nil
		case "Float":
			return schema.FieldTypeFloatArray, nil
		case "Boolean":
			return schema.FieldTypeBoolArray, nil
		case "Decimal":
			return schema.FieldTypeDecimalArray, nil
		case "DateTime":
			return schema.FieldTypeDateTimeArray, nil
		default:
			// Check if it's an enum array
			if _, exists := c.enums[typeName]; exists {
				return schema.FieldTypeStringArray, nil // Enum arrays are stored as string arrays
			}
			// Unknown array type, default to JSON
			return schema.FieldTypeJSON, nil
		}
	}

	// Handle scalar types
	switch typeName {
	case "String":
		return schema.FieldTypeString, nil
	case "Int":
		return schema.FieldTypeInt, nil
	case "BigInt":
		return schema.FieldTypeInt64, nil
	case "Float":
		return schema.FieldTypeFloat, nil
	case "Boolean":
		return schema.FieldTypeBool, nil
	case "DateTime":
		return schema.FieldTypeDateTime, nil
	case "Json":
		return schema.FieldTypeJSON, nil
	case "Decimal":
		return schema.FieldTypeDecimal, nil
	default:
		// Check if it's an enum
		if _, exists := c.enums[typeName]; exists {
			return schema.FieldTypeString, nil // Enums are stored as strings
		}
		// Unknown type, default to string
		return schema.FieldTypeString, nil
	}
}

// convertExpression converts a Prisma expression to a Go value
func (c *Converter) convertExpression(expr Expression) (interface{}, error) {
	switch e := expr.(type) {
	case *StringLiteral:
		return e.Value, nil
	case *NumberLiteral:
		// Try to parse as int first, then float
		if val, err := strconv.Atoi(e.Value); err == nil {
			return val, nil
		}
		if val, err := strconv.ParseFloat(e.Value, 64); err == nil {
			return val, nil
		}
		return e.Value, nil
	case *Identifier:
		switch e.Value {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return e.Value, nil
		}
	case *FunctionCall:
		switch e.Name {
		case "autoincrement":
			return "AUTO_INCREMENT", nil
		case "now":
			return "CURRENT_TIMESTAMP", nil
		case "uuid":
			return "UUID()", nil
		case "cuid":
			return "CUID()", nil
		case "dbgenerated":
			// Return the generated SQL as is
			if len(e.Args) > 0 {
				if str, ok := e.Args[0].(*StringLiteral); ok {
					return str.Value, nil
				}
			}
			return "DBGENERATED", nil
		default:
			// Return function call as string
			return e.String(), nil
		}
	case *ArrayExpression:
		values := make([]interface{}, len(e.Elements))
		for i, elem := range e.Elements {
			val, err := c.convertExpression(elem)
			if err != nil {
				return nil, err
			}
			values[i] = val
		}
		return values, nil
	case *NamedArgument:
		// For named arguments, return as a map
		value, err := c.convertExpression(e.Value)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{e.Name: value}, nil
	case *DotExpression:
		// For dot expressions like db.VarChar, return as string
		return e.String(), nil
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// isRelationField checks if a field represents a relation
func (c *Converter) isRelationField(field *Field) bool {
	// Check if field type is not a primitive type
	switch field.Type.Name {
	case "String", "Int", "BigInt", "Float", "Boolean", "DateTime", "Json", "Decimal":
		return false
	default:
		// Check if it's an enum
		if _, exists := c.enums[field.Type.Name]; exists {
			return false
		}
		// If it's an array of a model, it's a relation
		if field.List {
			return true
		}
		// If it's a single reference to another model, it's a relation
		return true
	}
}

// addRelations adds relations to schemas based on Prisma field definitions
func (c *Converter) addRelations(modelStmt *ModelStatement) error {
	currentSchema, exists := c.schemas[modelStmt.Name]
	if !exists {
		return fmt.Errorf("schema for model %s not found", modelStmt.Name)
	}

	for _, field := range modelStmt.Fields {
		if !c.isRelationField(field) {
			continue
		}

		relationName := field.Name
		relatedModel := field.Type.Name

		// Determine relation type and foreign key
		var relationType schema.RelationType
		var foreignKey string
		var references string = "id" // Default reference field

		if field.List {
			// Array field indicates one-to-many relation
			relationType = schema.RelationOneToMany
			// Foreign key is in the related model
			foreignKey = strings.ToLower(modelStmt.Name) + "_id"
		} else {
			// Single field indicates many-to-one relation
			relationType = schema.RelationManyToOne
			// Look for a corresponding foreign key field
			foreignKey = c.findForeignKeyField(modelStmt, relatedModel)
			if foreignKey == "" {
				foreignKey = strings.ToLower(relatedModel) + "_id"
			}
		}

		// Check for relation attributes that specify foreign key
		for _, attr := range field.Attributes {
			if attr.Name == "relation" && len(attr.Args) > 0 {
				// Parse relation arguments - handle both old function style and new named argument style
				for _, arg := range attr.Args {
					// Handle named arguments (fields: [userId], references: [id])
					if na, ok := arg.(*NamedArgument); ok {
						switch na.Name {
						case "fields":
							if arr, ok := na.Value.(*ArrayExpression); ok && len(arr.Elements) > 0 {
								if ident, ok := arr.Elements[0].(*Identifier); ok {
									foreignKey = ident.Value
								}
							}
						case "references":
							if arr, ok := na.Value.(*ArrayExpression); ok && len(arr.Elements) > 0 {
								if ident, ok := arr.Elements[0].(*Identifier); ok {
									references = ident.Value
								}
							}
						}
					}
					// Handle legacy function call style
					if fc, ok := arg.(*FunctionCall); ok {
						switch fc.Name {
						case "fields":
							if len(fc.Args) > 0 {
								if arr, ok := fc.Args[0].(*ArrayExpression); ok && len(arr.Elements) > 0 {
									if ident, ok := arr.Elements[0].(*Identifier); ok {
										foreignKey = ident.Value
									}
								}
							}
						case "references":
							if len(fc.Args) > 0 {
								if arr, ok := fc.Args[0].(*ArrayExpression); ok && len(arr.Elements) > 0 {
									if ident, ok := arr.Elements[0].(*Identifier); ok {
										references = ident.Value
									}
								}
							}
						}
					}
				}
			}
		}

		relation := schema.Relation{
			Type:       relationType,
			Model:      relatedModel,
			ForeignKey: foreignKey,
			References: references,
		}

		currentSchema.AddRelation(relationName, relation)
	}

	return nil
}

// findForeignKeyField looks for a foreign key field in the model
func (c *Converter) findForeignKeyField(modelStmt *ModelStatement, relatedModel string) string {
	expectedFK := strings.ToLower(relatedModel) + "_id"
	for _, field := range modelStmt.Fields {
		if field.Name == expectedFK {
			return expectedFK
		}
	}
	return ""
}

// extractTableName extracts table name from block attributes
func (c *Converter) extractTableName(attrs []*BlockAttribute) string {
	for _, attr := range attrs {
		if attr.Name == "map" && len(attr.Args) > 0 {
			if str, ok := attr.Args[0].(*StringLiteral); ok {
				return str.Value
			}
		}
	}
	return ""
}

// extractIndexes extracts indexes from block attributes
func (c *Converter) extractIndexes(attrs []*BlockAttribute) []schema.Index {
	var indexes []schema.Index

	for _, attr := range attrs {
		switch attr.Name {
		case "index":
			if len(attr.Args) > 0 {
				if arr, ok := attr.Args[0].(*ArrayExpression); ok {
					var fields []string
					for _, elem := range arr.Elements {
						if ident, ok := elem.(*Identifier); ok {
							fields = append(fields, ident.Value)
						}
					}
					if len(fields) > 0 {
						indexes = append(indexes, schema.Index{
							Name:   fmt.Sprintf("idx_%s", strings.Join(fields, "_")),
							Fields: fields,
							Unique: false,
						})
					}
				}
			}
		case "unique":
			if len(attr.Args) > 0 {
				if arr, ok := attr.Args[0].(*ArrayExpression); ok {
					var fields []string
					for _, elem := range arr.Elements {
						if ident, ok := elem.(*Identifier); ok {
							fields = append(fields, ident.Value)
						}
					}
					if len(fields) > 0 {
						indexes = append(indexes, schema.Index{
							Name:   fmt.Sprintf("uniq_%s", strings.Join(fields, "_")),
							Fields: fields,
							Unique: true,
						})
					}
				}
			}
		}
	}

	return indexes
}

// GetDatasource returns the datasource configuration
func (c *Converter) GetDatasource() *DatasourceStatement {
	return c.datasource
}

// GetGenerator returns the generator configuration
func (c *Converter) GetGenerator() *GeneratorStatement {
	return c.generator
}

// GetDatabaseProvider extracts the database provider from datasource
func (c *Converter) GetDatabaseProvider() string {
	if c.datasource == nil {
		return ""
	}
	
	for _, prop := range c.datasource.Properties {
		if prop.Name == "provider" {
			if str, ok := prop.Value.(*StringLiteral); ok {
				return str.Value
			}
		}
	}
	return ""
}

// GetDatabaseURL extracts the database URL from datasource
func (c *Converter) GetDatabaseURL() string {
	if c.datasource == nil {
		return ""
	}
	
	for _, prop := range c.datasource.Properties {
		if prop.Name == "url" {
			// Handle env() function calls
			if fc, ok := prop.Value.(*FunctionCall); ok && fc.Name == "env" {
				if len(fc.Args) > 0 {
					if str, ok := fc.Args[0].(*StringLiteral); ok {
						return "${" + str.Value + "}" // Mark as environment variable
					}
				}
			}
			// Handle direct string values
			if str, ok := prop.Value.(*StringLiteral); ok {
				return str.Value
			}
		}
	}
	return ""
}

// GetGeneratorProvider extracts the generator provider
func (c *Converter) GetGeneratorProvider() string {
	if c.generator == nil {
		return ""
	}
	
	for _, prop := range c.generator.Properties {
		if prop.Name == "provider" {
			if str, ok := prop.Value.(*StringLiteral); ok {
				return str.Value
			}
		}
	}
	return ""
}

// extractCompositeKey extracts composite primary key from block attributes
func (c *Converter) extractCompositeKey(attrs []*BlockAttribute) []string {
	for _, attr := range attrs {
		if attr.Name == "id" && len(attr.Args) > 0 {
			if arr, ok := attr.Args[0].(*ArrayExpression); ok {
				var fields []string
				for _, elem := range arr.Elements {
					if ident, ok := elem.(*Identifier); ok {
						fields = append(fields, ident.Value)
					}
				}
				return fields
			}
		}
	}
	return nil
}