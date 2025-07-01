package schema

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeInt      FieldType = "int"
	FieldTypeInt64    FieldType = "int64"
	FieldTypeFloat    FieldType = "float"
	FieldTypeBool     FieldType = "bool"
	FieldTypeDateTime FieldType = "datetime"
	FieldTypeJSON     FieldType = "json"
	FieldTypeDecimal  FieldType = "decimal"

	// Array types
	FieldTypeStringArray   FieldType = "string[]"
	FieldTypeIntArray      FieldType = "int[]"
	FieldTypeInt64Array    FieldType = "int64[]"
	FieldTypeFloatArray    FieldType = "float[]"
	FieldTypeBoolArray     FieldType = "bool[]"
	FieldTypeDecimalArray  FieldType = "decimal[]"
	FieldTypeDateTimeArray FieldType = "datetime[]"
)

type Field struct {
	Name          string
	Type          FieldType
	PrimaryKey    bool
	AutoIncrement bool
	Nullable      bool
	Unique        bool
	Default       interface{}
	Index         bool
	DbType        string   // Database-specific type (e.g., "@db.VarChar(255)", "@db.Money")
	DbAttributes  []string // Additional database attributes
	Map           string   // Column name mapping (@map("column_name"))
}

// GetColumnName returns the actual database column name for this field
func (f Field) GetColumnName() string {
	if f.Map != "" {
		return f.Map
	}
	return f.Name
}

type Relation struct {
	Type       RelationType
	Model      string
	ForeignKey string
	References string
	OnDelete   string
	OnUpdate   string
}

type RelationType string

const (
	RelationOneToOne   RelationType = "oneToOne"
	RelationOneToMany  RelationType = "oneToMany"
	RelationManyToOne  RelationType = "manyToOne"
	RelationManyToMany RelationType = "manyToMany"
)

type Schema struct {
	Name         string
	TableName    string
	Fields       []Field
	Relations    map[string]Relation
	Indexes      []Index
	CompositeKey []string // Fields that form the composite primary key
}

type Index struct {
	Name   string
	Fields []string
	Unique bool
}

func New(name string) *Schema {
	return &Schema{
		Name:      name,
		TableName: ModelNameToTableName(name),
		Fields:    []Field{},
		Relations: make(map[string]Relation),
		Indexes:   []Index{},
	}
}

// ModelNameToTableName converts model name to default table name (pluralized, snake_case)
func ModelNameToTableName(modelName string) string {
	snakeCase := CamelToSnakeCase(modelName)
	return Pluralize(snakeCase)
}

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

// Pluralize adds 's' to make a word plural (simple implementation)
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

func (s *Schema) WithTableName(name string) *Schema {
	s.TableName = name
	return s
}

func (s *Schema) AddField(field Field) *Schema {
	s.Fields = append(s.Fields, field)
	return s
}

func (s *Schema) AddRelation(name string, relation Relation) *Schema {
	s.Relations[name] = relation
	return s
}

func (s *Schema) AddIndex(index Index) *Schema {
	s.Indexes = append(s.Indexes, index)
	return s
}

func (s *Schema) WithCompositeKey(fields []string) *Schema {
	s.CompositeKey = fields
	return s
}

func (s *Schema) GetField(name string) (*Field, error) {
	for i := range s.Fields {
		if s.Fields[i].Name == name {
			return &s.Fields[i], nil
		}
	}
	return nil, fmt.Errorf("field %s not found", name)
}

func (s *Schema) GetPrimaryKey() (*Field, error) {
	for i := range s.Fields {
		if s.Fields[i].PrimaryKey {
			return &s.Fields[i], nil
		}
	}
	return nil, fmt.Errorf("no primary key found")
}

func (s *Schema) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("schema name cannot be empty")
	}
	if s.TableName == "" {
		return fmt.Errorf("table name cannot be empty")
	}
	if len(s.Fields) == 0 {
		return fmt.Errorf("schema must have at least one field")
	}

	// Check primary key constraints
	hasSinglePrimaryKey := false
	hasCompositePrimaryKey := len(s.CompositeKey) > 0

	for _, field := range s.Fields {
		if field.PrimaryKey {
			if hasSinglePrimaryKey {
				return fmt.Errorf("schema can only have one single-field primary key")
			}
			if hasCompositePrimaryKey {
				return fmt.Errorf("schema cannot have both single and composite primary keys")
			}
			hasSinglePrimaryKey = true
		}
	}

	// Validate composite key fields exist
	if hasCompositePrimaryKey {
		for _, keyField := range s.CompositeKey {
			if _, err := s.GetField(keyField); err != nil {
				return fmt.Errorf("composite key field %s not found", keyField)
			}
		}
	}

	if !hasSinglePrimaryKey && !hasCompositePrimaryKey {
		return fmt.Errorf("schema must have a primary key (single field or composite)")
	}

	return nil
}

// GetFieldByColumnName returns a field by its database column name
func (s *Schema) GetFieldByColumnName(columnName string) (*Field, error) {
	for i := range s.Fields {
		if s.Fields[i].GetColumnName() == columnName {
			return &s.Fields[i], nil
		}
	}
	return nil, fmt.Errorf("field with column name %s not found", columnName)
}

// GetFieldNameByColumnName returns the schema field name for a given column name
func (s *Schema) GetFieldNameByColumnName(columnName string) (string, error) {
	field, err := s.GetFieldByColumnName(columnName)
	if err != nil {
		return "", err
	}
	return field.Name, nil
}

// GetColumnNameByFieldName returns the database column name for a given schema field name
func (s *Schema) GetColumnNameByFieldName(fieldName string) (string, error) {
	field, err := s.GetField(fieldName)
	if err != nil {
		return "", err
	}
	return field.GetColumnName(), nil
}

// MapFieldNamesToColumns converts a slice of schema field names to database column names
func (s *Schema) MapFieldNamesToColumns(fieldNames []string) ([]string, error) {
	columnNames := make([]string, len(fieldNames))
	for i, fieldName := range fieldNames {
		columnName, err := s.GetColumnNameByFieldName(fieldName)
		if err != nil {
			return nil, fmt.Errorf("failed to map field %s: %w", fieldName, err)
		}
		columnNames[i] = columnName
	}
	return columnNames, nil
}

// MapColumnNamesToFields converts a slice of database column names to schema field names
func (s *Schema) MapColumnNamesToFields(columnNames []string) ([]string, error) {
	fieldNames := make([]string, len(columnNames))
	for i, columnName := range columnNames {
		fieldName, err := s.GetFieldNameByColumnName(columnName)
		if err != nil {
			return nil, fmt.Errorf("failed to map column %s: %w", columnName, err)
		}
		fieldNames[i] = fieldName
	}
	return fieldNames, nil
}

// MapSchemaDataToColumns converts data with schema field names to data with database column names
func (s *Schema) MapSchemaDataToColumns(data map[string]interface{}) (map[string]interface{}, error) {
	mapped := make(map[string]interface{})
	for fieldName, value := range data {
		columnName, err := s.GetColumnNameByFieldName(fieldName)
		if err != nil {
			return nil, fmt.Errorf("failed to map field %s: %w", fieldName, err)
		}
		mapped[columnName] = value
	}
	return mapped, nil
}

// MapColumnDataToSchema converts data with database column names to data with schema field names
func (s *Schema) MapColumnDataToSchema(data map[string]interface{}) (map[string]interface{}, error) {
	mapped := make(map[string]interface{})
	for columnName, value := range data {
		fieldName, err := s.GetFieldNameByColumnName(columnName)
		if err != nil {
			// If column not found in schema, keep original column name (for raw queries)
			mapped[columnName] = value
			continue
		}
		mapped[fieldName] = value
	}
	return mapped, nil
}

// GetTableName returns the database table name (same as TableName field, but method for consistency)
func (s *Schema) GetTableName() string {
	return s.TableName
}

// HasRelation checks if a relation exists
func (s *Schema) HasRelation(relationName string) bool {
	_, exists := s.Relations[relationName]
	return exists
}

// GetRelation returns a relation by name
func (s *Schema) GetRelation(relationName string) (Relation, error) {
	relation, exists := s.Relations[relationName]
	if !exists {
		return Relation{}, fmt.Errorf("relation %s not found", relationName)
	}
	return relation, nil
}

func FieldTypeFromGo(t reflect.Type) FieldType {
	switch t.Kind() {
	case reflect.String:
		return FieldTypeString
	case reflect.Int, reflect.Int32:
		return FieldTypeInt
	case reflect.Int64:
		return FieldTypeInt64
	case reflect.Float32, reflect.Float64:
		return FieldTypeFloat
	case reflect.Bool:
		return FieldTypeBool
	default:
		return FieldTypeString
	}
}
