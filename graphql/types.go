package graphql

import (
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/rediwo/redi-orm/schema"
)

// MapFieldTypeToGraphQL converts RediORM field types to GraphQL types
func MapFieldTypeToGraphQL(fieldType schema.FieldType) graphql.Type {
	switch fieldType {
	case schema.FieldTypeString:
		return graphql.String
	case schema.FieldTypeInt, schema.FieldTypeInt64:
		return graphql.Int
	case schema.FieldTypeFloat:
		return graphql.Float
	case schema.FieldTypeBool:
		return graphql.Boolean
	case schema.FieldTypeDateTime:
		return GraphQLDateTime // Custom scalar
	case schema.FieldTypeJSON:
		return GraphQLJSON // Custom scalar
	case schema.FieldTypeDecimal:
		return graphql.Float // Map decimal to float for simplicity
	default:
		return graphql.String
	}
}

// GraphQLDateTime is a custom scalar for DateTime fields
var GraphQLDateTime = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "DateTime",
	Description: "DateTime scalar type represents a date and time in ISO 8601 format",
	Serialize: func(value any) any {
		// Convert to string in ISO format
		return value
	},
	ParseValue: func(value any) any {
		// Parse from variable value
		return value
	},
	ParseLiteral: func(valueAST ast.Value) any {
		// Parse from AST
		if stringValue, ok := valueAST.(*ast.StringValue); ok {
			return stringValue.Value
		}
		return nil
	},
})

// GraphQLJSON is a custom scalar for JSON fields
var GraphQLJSON = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "JSON",
	Description: "JSON scalar type represents JSON data",
	Serialize: func(value any) any {
		return value
	},
	ParseValue: func(value any) any {
		return value
	},
	ParseLiteral: func(valueAST ast.Value) any {
		// Parse from AST
		if stringValue, ok := valueAST.(*ast.StringValue); ok {
			return stringValue.Value
		}
		return nil
	},
})

// Global filter types cache to avoid duplicate type definitions
var filterTypesCache = make(map[string]*graphql.InputObject)

// WhereInputField creates a GraphQL input field for where conditions
func WhereInputField(fieldType schema.FieldType) *graphql.InputObjectFieldConfig {
	// Create a unique filter name based on the field type
	filterTypeName := fmt.Sprintf("%sFilter", getFieldTypeName(fieldType))

	// Check if we already created this filter type
	if filterType, exists := filterTypesCache[filterTypeName]; exists {
		return &graphql.InputObjectFieldConfig{
			Type: filterType,
		}
	}

	baseType := MapFieldTypeToGraphQL(fieldType)

	// Create filter input with operators
	filterFields := graphql.InputObjectConfigFieldMap{
		"equals": &graphql.InputObjectFieldConfig{Type: baseType},
		"not":    &graphql.InputObjectFieldConfig{Type: baseType},
		"in":     &graphql.InputObjectFieldConfig{Type: graphql.NewList(baseType)},
		"notIn":  &graphql.InputObjectFieldConfig{Type: graphql.NewList(baseType)},
		"lt":     &graphql.InputObjectFieldConfig{Type: baseType},
		"lte":    &graphql.InputObjectFieldConfig{Type: baseType},
		"gt":     &graphql.InputObjectFieldConfig{Type: baseType},
		"gte":    &graphql.InputObjectFieldConfig{Type: baseType},
	}

	// Add string-specific operators
	if fieldType == schema.FieldTypeString {
		filterFields["contains"] = &graphql.InputObjectFieldConfig{Type: graphql.String}
		filterFields["startsWith"] = &graphql.InputObjectFieldConfig{Type: graphql.String}
		filterFields["endsWith"] = &graphql.InputObjectFieldConfig{Type: graphql.String}
	}

	filterInput := graphql.NewInputObject(graphql.InputObjectConfig{
		Name:   filterTypeName,
		Fields: filterFields,
	})

	// Cache the filter type
	filterTypesCache[filterTypeName] = filterInput

	return &graphql.InputObjectFieldConfig{
		Type: filterInput,
	}
}

// getFieldTypeName returns a string representation of the field type for naming
func getFieldTypeName(fieldType schema.FieldType) string {
	switch fieldType {
	case schema.FieldTypeString:
		return "String"
	case schema.FieldTypeInt, schema.FieldTypeInt64:
		return "Int"
	case schema.FieldTypeFloat:
		return "Float"
	case schema.FieldTypeBool:
		return "Boolean"
	case schema.FieldTypeDateTime:
		return "DateTime"
	case schema.FieldTypeJSON:
		return "JSON"
	default:
		return "String"
	}
}

// OrderByEnum creates an enum for ordering
var OrderByEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "OrderBy",
	Description: "Order by direction",
	Values: graphql.EnumValueConfigMap{
		"ASC": &graphql.EnumValueConfig{
			Value: "ASC",
		},
		"DESC": &graphql.EnumValueConfig{
			Value: "DESC",
		},
	},
})
