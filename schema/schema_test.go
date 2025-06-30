package schema

import (
	"reflect"
	"testing"
)

func TestNewSchema(t *testing.T) {
	s := New("User")

	if s.Name != "User" {
		t.Errorf("Expected schema name to be 'User', got '%s'", s.Name)
	}

	if s.TableName != "users" {
		t.Errorf("Expected table name to be 'users', got '%s'", s.TableName)
	}

	if len(s.Fields) != 0 {
		t.Errorf("Expected no fields initially, got %d", len(s.Fields))
	}

	if s.Relations == nil {
		t.Error("Expected Relations map to be initialized")
	}

	if s.Indexes == nil {
		t.Error("Expected Indexes slice to be initialized")
	}
}

func TestWithTableName(t *testing.T) {
	s := New("User").WithTableName("custom_users")

	if s.TableName != "custom_users" {
		t.Errorf("Expected table name to be 'custom_users', got '%s'", s.TableName)
	}
}

func TestAddField(t *testing.T) {
	s := New("User")
	field := Field{
		Name:       "id",
		Type:       FieldTypeInt,
		PrimaryKey: true,
	}

	s.AddField(field)

	if len(s.Fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(s.Fields))
	}

	if s.Fields[0].Name != "id" {
		t.Errorf("Expected field name to be 'id', got '%s'", s.Fields[0].Name)
	}
}

func TestAddRelation(t *testing.T) {
	s := New("Post")
	relation := Relation{
		Type:       RelationManyToOne,
		Model:      "User",
		ForeignKey: "user_id",
		References: "id",
	}

	s.AddRelation("author", relation)

	if len(s.Relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(s.Relations))
	}

	if rel, exists := s.Relations["author"]; !exists {
		t.Error("Expected 'author' relation to exist")
	} else if rel.Model != "User" {
		t.Errorf("Expected relation model to be 'User', got '%s'", rel.Model)
	}
}

func TestAddIndex(t *testing.T) {
	s := New("User")
	index := Index{
		Name:   "idx_email",
		Fields: []string{"email"},
		Unique: true,
	}

	s.AddIndex(index)

	if len(s.Indexes) != 1 {
		t.Errorf("Expected 1 index, got %d", len(s.Indexes))
	}

	if s.Indexes[0].Name != "idx_email" {
		t.Errorf("Expected index name to be 'idx_email', got '%s'", s.Indexes[0].Name)
	}
}

func TestGetField(t *testing.T) {
	s := New("User").
		AddField(Field{Name: "id", Type: FieldTypeInt}).
		AddField(Field{Name: "name", Type: FieldTypeString})

	// Test existing field
	field, err := s.GetField("name")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if field.Type != FieldTypeString {
		t.Errorf("Expected field type to be FieldTypeString, got %v", field.Type)
	}

	// Test non-existing field
	_, err = s.GetField("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existing field")
	}
}

func TestGetPrimaryKey(t *testing.T) {
	// Test with primary key
	s1 := New("User").
		AddField(Field{Name: "id", Type: FieldTypeInt, PrimaryKey: true}).
		AddField(Field{Name: "name", Type: FieldTypeString})

	pk, err := s1.GetPrimaryKey()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if pk.Name != "id" {
		t.Errorf("Expected primary key name to be 'id', got '%s'", pk.Name)
	}

	// Test without primary key
	s2 := New("User").
		AddField(Field{Name: "name", Type: FieldTypeString})

	_, err = s2.GetPrimaryKey()
	if err == nil {
		t.Error("Expected error when no primary key exists")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		schema  *Schema
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Empty schema name",
			schema:  &Schema{Name: "", TableName: "users", Fields: []Field{{Name: "id", PrimaryKey: true}}},
			wantErr: true,
			errMsg:  "schema name cannot be empty",
		},
		{
			name:    "Empty table name",
			schema:  &Schema{Name: "User", TableName: "", Fields: []Field{{Name: "id", PrimaryKey: true}}},
			wantErr: true,
			errMsg:  "table name cannot be empty",
		},
		{
			name:    "No fields",
			schema:  &Schema{Name: "User", TableName: "users", Fields: []Field{}},
			wantErr: true,
			errMsg:  "schema must have at least one field",
		},
		{
			name: "No primary key",
			schema: &Schema{
				Name:      "User",
				TableName: "users",
				Fields:    []Field{{Name: "name", Type: FieldTypeString}},
			},
			wantErr: true,
			errMsg:  "schema must have a primary key (single field or composite)",
		},
		{
			name: "Multiple primary keys",
			schema: &Schema{
				Name:      "User",
				TableName: "users",
				Fields: []Field{
					{Name: "id1", Type: FieldTypeInt, PrimaryKey: true},
					{Name: "id2", Type: FieldTypeInt, PrimaryKey: true},
				},
			},
			wantErr: true,
			errMsg:  "schema can only have one single-field primary key",
		},
		{
			name: "Valid schema",
			schema: &Schema{
				Name:      "User",
				TableName: "users",
				Fields: []Field{
					{Name: "id", Type: FieldTypeInt, PrimaryKey: true},
					{Name: "name", Type: FieldTypeString},
				},
				Relations: make(map[string]Relation),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestFieldTypeFromGo(t *testing.T) {
	tests := []struct {
		goType   reflect.Type
		expected FieldType
	}{
		{reflect.TypeOf(""), FieldTypeString},
		{reflect.TypeOf(int(0)), FieldTypeInt},
		{reflect.TypeOf(int32(0)), FieldTypeInt},
		{reflect.TypeOf(int64(0)), FieldTypeInt64},
		{reflect.TypeOf(float32(0)), FieldTypeFloat},
		{reflect.TypeOf(float64(0)), FieldTypeFloat},
		{reflect.TypeOf(true), FieldTypeBool},
		{reflect.TypeOf(struct{}{}), FieldTypeString}, // Default case
	}

	for _, tt := range tests {
		result := FieldTypeFromGo(tt.goType)
		if result != tt.expected {
			t.Errorf("FieldTypeFromGo(%v) = %v, want %v", tt.goType, result, tt.expected)
		}
	}
}
