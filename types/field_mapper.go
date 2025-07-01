package types

import (
	"fmt"
	"unicode"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/utils"
)

// DefaultFieldMapper provides default field mapping implementation
type DefaultFieldMapper struct {
	schemas map[string]*schema.Schema
}

// NewDefaultFieldMapper creates a new default field mapper
func NewDefaultFieldMapper() *DefaultFieldMapper {
	return &DefaultFieldMapper{
		schemas: make(map[string]*schema.Schema),
	}
}

// RegisterSchema registers a schema for field mapping
func (m *DefaultFieldMapper) RegisterSchema(modelName string, s *schema.Schema) {
	m.schemas[modelName] = s
}

// GetSchema returns a registered schema
func (m *DefaultFieldMapper) GetSchema(modelName string) (*schema.Schema, error) {
	s, exists := m.schemas[modelName]
	if !exists {
		return nil, fmt.Errorf("schema for model '%s' not registered", modelName)
	}
	return s, nil
}

// SchemaToColumn converts a schema field name to database column name
func (m *DefaultFieldMapper) SchemaToColumn(modelName, fieldName string) (string, error) {
	s, err := m.GetSchema(modelName)
	if err != nil {
		return "", err
	}

	return s.GetColumnNameByFieldName(fieldName)
}

// ColumnToSchema converts a database column name to schema field name
func (m *DefaultFieldMapper) ColumnToSchema(modelName, columnName string) (string, error) {
	s, err := m.GetSchema(modelName)
	if err != nil {
		return "", err
	}

	return s.GetFieldNameByColumnName(columnName)
}

// SchemaFieldsToColumns converts multiple schema field names to column names
func (m *DefaultFieldMapper) SchemaFieldsToColumns(modelName string, fieldNames []string) ([]string, error) {
	s, err := m.GetSchema(modelName)
	if err != nil {
		return nil, err
	}

	return s.MapFieldNamesToColumns(fieldNames)
}

// ColumnFieldsToSchema converts multiple column names to schema field names
func (m *DefaultFieldMapper) ColumnFieldsToSchema(modelName string, columnNames []string) ([]string, error) {
	s, err := m.GetSchema(modelName)
	if err != nil {
		return nil, err
	}

	return s.MapColumnNamesToFields(columnNames)
}

// MapSchemaToColumnData converts data map from schema field names to column names
func (m *DefaultFieldMapper) MapSchemaToColumnData(modelName string, data map[string]interface{}) (map[string]interface{}, error) {
	s, err := m.GetSchema(modelName)
	if err != nil {
		return nil, err
	}

	return s.MapSchemaDataToColumns(data)
}

// MapColumnToSchemaData converts data map from column names to schema field names
func (m *DefaultFieldMapper) MapColumnToSchemaData(modelName string, data map[string]interface{}) (map[string]interface{}, error) {
	s, err := m.GetSchema(modelName)
	if err != nil {
		return nil, err
	}

	return s.MapColumnDataToSchema(data)
}

// ModelToTable converts model name to table name
func (m *DefaultFieldMapper) ModelToTable(modelName string) (string, error) {
	s, err := m.GetSchema(modelName)
	if err != nil {
		return "", err
	}

	return s.GetTableName(), nil
}

// Utility functions for case conversion

// ModelNameToTableName converts model name to default table name (pluralized, snake_case)
func ModelNameToTableName(modelName string) string {
	snakeCase := utils.ToSnakeCase(modelName)
	return utils.Pluralize(snakeCase)
}


// ValidateFieldName checks if a field name is valid
func ValidateFieldName(fieldName string) error {
	if fieldName == "" {
		return fmt.Errorf("field name cannot be empty")
	}

	// Check if it starts with a letter or underscore
	first := rune(fieldName[0])
	if !unicode.IsLetter(first) && first != '_' {
		return fmt.Errorf("field name must start with a letter or underscore")
	}

	// Check remaining characters
	for _, r := range fieldName[1:] {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return fmt.Errorf("field name can only contain letters, digits, and underscores")
		}
	}

	return nil
}

// ValidateColumnName checks if a column name is valid for database
func ValidateColumnName(columnName string) error {
	if columnName == "" {
		return fmt.Errorf("column name cannot be empty")
	}

	// Basic validation - extend as needed for specific databases
	if len(columnName) > 64 {
		return fmt.Errorf("column name too long (max 64 characters)")
	}

	return ValidateFieldName(columnName)
}
