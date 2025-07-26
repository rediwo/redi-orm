package utils

import (
	"testing"
)

func TestGenerateIndexName(t *testing.T) {
	tests := []struct {
		name         string
		tableName    string
		fields       []string
		unique       bool
		existingName string
		expected     string
	}{
		{
			name:         "uses existing name when provided",
			tableName:    "users",
			fields:       []string{"email"},
			unique:       false,
			existingName: "custom_index_name",
			expected:     "custom_index_name",
		},
		{
			name:         "generates regular index name",
			tableName:    "users",
			fields:       []string{"email"},
			unique:       false,
			existingName: "",
			expected:     "idx_users_email",
		},
		{
			name:         "generates unique index name",
			tableName:    "users",
			fields:       []string{"email"},
			unique:       true,
			existingName: "",
			expected:     "uniq_users_email",
		},
		{
			name:         "generates composite index name",
			tableName:    "posts",
			fields:       []string{"user_id", "created_at"},
			unique:       false,
			existingName: "",
			expected:     "idx_posts_user_id_created_at",
		},
		{
			name:         "generates unique composite index name",
			tableName:    "posts",
			fields:       []string{"user_id", "created_at"},
			unique:       true,
			existingName: "",
			expected:     "uniq_posts_user_id_created_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateIndexName(tt.tableName, tt.fields, tt.unique, tt.existingName)
			if result != tt.expected {
				t.Errorf("GenerateIndexName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateFieldIndexName(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		fieldName string
		expected  string
	}{
		{
			name:      "generates single field index name",
			tableName: "users",
			fieldName: "email",
			expected:  "idx_users_email",
		},
		{
			name:      "handles snake_case field names",
			tableName: "posts",
			fieldName: "created_at",
			expected:  "idx_posts_created_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateFieldIndexName(tt.tableName, tt.fieldName)
			if result != tt.expected {
				t.Errorf("GenerateFieldIndexName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNormalizeIndexName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes idx prefix",
			input:    "idx_users_email",
			expected: "users_email",
		},
		{
			name:     "removes index prefix",
			input:    "index_users_email",
			expected: "users_email",
		},
		{
			name:     "removes uniq prefix",
			input:    "uniq_users_email",
			expected: "users_email",
		},
		{
			name:     "removes unique prefix",
			input:    "unique_users_email",
			expected: "users_email",
		},
		{
			name:     "removes idx suffix",
			input:    "users_email_idx",
			expected: "users_email",
		},
		{
			name:     "removes index suffix",
			input:    "users_email_index",
			expected: "users_email",
		},
		{
			name:     "handles uppercase",
			input:    "IDX_USERS_EMAIL",
			expected: "users_email",
		},
		{
			name:     "handles mixed case",
			input:    "Idx_Users_Email",
			expected: "users_email",
		},
		{
			name:     "handles no prefix or suffix",
			input:    "users_email",
			expected: "users_email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeIndexName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeIndexName() = %v, want %v", result, tt.expected)
			}
		})
	}
}
