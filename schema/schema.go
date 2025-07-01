package schema

import (
	"fmt"
	"reflect"
	"strings"
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
		TableName: strings.ToLower(name) + "s",
		Fields:    []Field{},
		Relations: make(map[string]Relation),
		Indexes:   []Index{},
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
