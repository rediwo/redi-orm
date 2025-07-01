package models

import (
	"fmt"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

type Model struct {
	schema *schema.Schema
	db     types.Database
}

func New(s *schema.Schema, db types.Database) *Model {
	return &Model{
		schema: s,
		db:     db,
	}
}

func (m *Model) Get(id interface{}) (map[string]interface{}, error) {
	return m.db.FindByID(m.schema.Name, id)
}

func (m *Model) Select(columns ...string) *QueryBuilder {
	return &QueryBuilder{
		db:        m.db,
		tableName: m.schema.Name,
		columns:   columns,
		qb:        m.db.Select(m.schema.Name, columns),
	}
}

func (m *Model) Add(data map[string]interface{}) (int64, error) {
	// Validate data against schema
	if err := m.validateData(data, true); err != nil {
		return 0, err
	}

	return m.db.Insert(m.schema.Name, data)
}

func (m *Model) Set(id interface{}, data map[string]interface{}) error {
	// Validate data against schema
	if err := m.validateData(data, false); err != nil {
		return err
	}

	return m.db.Update(m.schema.Name, id, data)
}

func (m *Model) Remove(id interface{}) error {
	return m.db.Delete(m.schema.Name, id)
}

func (m *Model) validateData(data map[string]interface{}, isInsert bool) error {
	// Check required fields for insert
	if isInsert {
		for _, field := range m.schema.Fields {
			if !field.Nullable && !field.AutoIncrement && field.Default == nil {
				if _, exists := data[field.Name]; !exists {
					return fmt.Errorf("required field %s is missing", field.Name)
				}
			}
		}
	}

	// Validate field types
	for key, value := range data {
		field, err := m.schema.GetField(key)
		if err != nil {
			return fmt.Errorf("unknown field: %s", key)
		}

		// Skip validation for null values if field is nullable
		if value == nil && field.Nullable {
			continue
		}

		// Basic type validation (can be expanded)
		if err := validateFieldType(field, value); err != nil {
			return fmt.Errorf("field %s: %w", key, err)
		}
	}

	return nil
}

func validateFieldType(field *schema.Field, value interface{}) error {
	if value == nil {
		if !field.Nullable {
			return fmt.Errorf("cannot be null")
		}
		return nil
	}

	// Basic type checking - can be expanded
	switch field.Type {
	case schema.FieldTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string")
		}
	case schema.FieldTypeInt, schema.FieldTypeInt64:
		switch value.(type) {
		case int, int32, int64, float64:
			// Accept numeric types
		default:
			return fmt.Errorf("expected integer")
		}
	case schema.FieldTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean")
		}
	}

	return nil
}
