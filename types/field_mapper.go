package types

import (
	"fmt"

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
func (m *DefaultFieldMapper) MapSchemaToColumnData(modelName string, data map[string]any) (map[string]any, error) {
	s, err := m.GetSchema(modelName)
	if err != nil {
		return nil, err
	}

	return s.MapSchemaDataToColumns(data)
}

// MapColumnToSchemaData converts data map from column names to schema field names
func (m *DefaultFieldMapper) MapColumnToSchemaData(modelName string, data map[string]any) (map[string]any, error) {
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
