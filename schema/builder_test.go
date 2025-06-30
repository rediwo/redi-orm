package schema

import (
	"testing"
)

func TestFieldBuilder(t *testing.T) {
	t.Run("String field", func(t *testing.T) {
		field := NewField("name").String().Build()
		if field.Name != "name" {
			t.Errorf("Expected field name to be 'name', got '%s'", field.Name)
		}
		if field.Type != FieldTypeString {
			t.Errorf("Expected field type to be FieldTypeString, got %v", field.Type)
		}
	})

	t.Run("Int field with primary key", func(t *testing.T) {
		field := NewField("id").Int().PrimaryKey().AutoIncrement().Build()
		if field.Type != FieldTypeInt {
			t.Errorf("Expected field type to be FieldTypeInt, got %v", field.Type)
		}
		if !field.PrimaryKey {
			t.Error("Expected field to be primary key")
		}
		if !field.AutoIncrement {
			t.Error("Expected field to be auto increment")
		}
		if field.Nullable {
			t.Error("Expected primary key field to not be nullable")
		}
	})

	t.Run("Nullable field", func(t *testing.T) {
		field := NewField("age").Int().Nullable().Build()
		if !field.Nullable {
			t.Error("Expected field to be nullable")
		}
	})

	t.Run("Unique field", func(t *testing.T) {
		field := NewField("email").String().Unique().Build()
		if !field.Unique {
			t.Error("Expected field to be unique")
		}
	})

	t.Run("Field with default value", func(t *testing.T) {
		field := NewField("active").Bool().Default(true).Build()
		if field.Default != true {
			t.Errorf("Expected default value to be true, got %v", field.Default)
		}
	})

	t.Run("Indexed field", func(t *testing.T) {
		field := NewField("username").String().Index().Build()
		if !field.Index {
			t.Error("Expected field to have index")
		}
	})

	t.Run("All field types", func(t *testing.T) {
		tests := []struct {
			name     string
			builder  func(*FieldBuilder) *FieldBuilder
			expected FieldType
		}{
			{"String", func(fb *FieldBuilder) *FieldBuilder { return fb.String() }, FieldTypeString},
			{"Int", func(fb *FieldBuilder) *FieldBuilder { return fb.Int() }, FieldTypeInt},
			{"Int64", func(fb *FieldBuilder) *FieldBuilder { return fb.Int64() }, FieldTypeInt64},
			{"Float", func(fb *FieldBuilder) *FieldBuilder { return fb.Float() }, FieldTypeFloat},
			{"Bool", func(fb *FieldBuilder) *FieldBuilder { return fb.Bool() }, FieldTypeBool},
			{"DateTime", func(fb *FieldBuilder) *FieldBuilder { return fb.DateTime() }, FieldTypeDateTime},
			{"JSON", func(fb *FieldBuilder) *FieldBuilder { return fb.JSON() }, FieldTypeJSON},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fb := NewField("test")
				field := tt.builder(fb).Build()
				if field.Type != tt.expected {
					t.Errorf("Expected field type to be %v, got %v", tt.expected, field.Type)
				}
			})
		}
	})

	t.Run("Chaining multiple modifiers", func(t *testing.T) {
		field := NewField("email").
			String().
			Unique().
			Index().
			Default("").
			Build()

		if field.Type != FieldTypeString {
			t.Errorf("Expected field type to be FieldTypeString, got %v", field.Type)
		}
		if !field.Unique {
			t.Error("Expected field to be unique")
		}
		if !field.Index {
			t.Error("Expected field to have index")
		}
		if field.Default != "" {
			t.Errorf("Expected default value to be empty string, got %v", field.Default)
		}
	})
}
