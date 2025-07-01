package types

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/rediwo/redi-orm/schema"
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

// CamelToSnakeCase converts camelCase to snake_case
func CamelToSnakeCase(input string) string {
	if input == "" {
		return ""
	}

	// Add underscore before uppercase letters that follow lowercase letters or numbers
	re1 := regexp.MustCompile("([a-z0-9])([A-Z])")
	result := re1.ReplaceAllString(input, "${1}_${2}")

	// Add underscore before uppercase letters that are followed by lowercase letters
	// and preceded by uppercase letters (for cases like XMLHttpRequest -> xml_http_request)
	re2 := regexp.MustCompile("([A-Z])([A-Z][a-z])")
	result = re2.ReplaceAllString(result, "${1}_${2}")

	return strings.ToLower(result)
}

// SnakeToCamelCase converts snake_case to camelCase
func SnakeToCamelCase(input string) string {
	if input == "" {
		return ""
	}

	parts := strings.Split(input, "_")
	if len(parts) == 1 {
		return input
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(string(parts[i][0])) + parts[i][1:]
		}
	}

	return result
}

// ToPascalCase converts string to PascalCase
func ToPascalCase(input string) string {
	if input == "" {
		return ""
	}

	// If it's snake_case, convert first
	if strings.Contains(input, "_") {
		parts := strings.Split(input, "_")
		result := ""
		for _, part := range parts {
			if len(part) > 0 {
				result += strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
			}
		}
		return result
	}

	// If it's already camelCase, just capitalize first letter
	if len(input) > 0 {
		return strings.ToUpper(string(input[0])) + input[1:]
	}

	return input
}

// ModelNameToTableName converts model name to default table name (pluralized, snake_case)
func ModelNameToTableName(modelName string) string {
	snakeCase := CamelToSnakeCase(modelName)
	return Pluralize(snakeCase)
}

// Pluralize adds 's' to make a word plural (simple implementation)
// For more complex pluralization, consider using a dedicated library
func Pluralize(word string) string {
	if word == "" {
		return word
	}

	word = strings.ToLower(word)

	// Simple pluralization rules
	if strings.HasSuffix(word, "s") || strings.HasSuffix(word, "x") ||
		strings.HasSuffix(word, "z") || strings.HasSuffix(word, "ch") ||
		strings.HasSuffix(word, "sh") {
		return word + "es"
	}

	if strings.HasSuffix(word, "y") && len(word) > 1 {
		prev := rune(word[len(word)-2])
		if !isVowel(prev) {
			return word[:len(word)-1] + "ies"
		}
	}

	if strings.HasSuffix(word, "f") {
		return word[:len(word)-1] + "ves"
	}

	if strings.HasSuffix(word, "fe") {
		return word[:len(word)-2] + "ves"
	}

	return word + "s"
}

// isVowel checks if a character is a vowel
func isVowel(r rune) bool {
	switch unicode.ToLower(r) {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
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
