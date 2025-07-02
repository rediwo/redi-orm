package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetJunctionTableName(t *testing.T) {
	tests := []struct {
		modelA   string
		modelB   string
		expected string
	}{
		{"User", "Post", "post_users"},               // Post < User alphabetically
		{"Post", "User", "post_users"},               // Same result regardless of order
		{"Category", "Product", "category_products"}, // Category < Product
		{"Tag", "Article", "article_tags"},           // Article < Tag alphabetically
		{"User", "User", "user_users"},               // Self-relation
	}

	for _, tt := range tests {
		t.Run(tt.modelA+"_"+tt.modelB, func(t *testing.T) {
			result := GetJunctionTableName(tt.modelA, tt.modelB)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Additional test to ensure consistent ordering
	t.Run("ConsistentOrdering", func(t *testing.T) {
		// Should get same result regardless of parameter order
		assert.Equal(t, GetJunctionTableName("User", "Post"), GetJunctionTableName("Post", "User"))
		assert.Equal(t, GetJunctionTableName("Tag", "Article"), GetJunctionTableName("Article", "Tag"))
		assert.Equal(t, GetJunctionTableName("Category", "Product"), GetJunctionTableName("Product", "Category"))
	})
}

func TestValidateRelation(t *testing.T) {
	// Create test schemas
	userSchema := New("User").
		AddField(Field{Name: "id", Type: FieldTypeInt, PrimaryKey: true}).
		AddField(Field{Name: "name", Type: FieldTypeString}).
		AddField(Field{Name: "email", Type: FieldTypeString})

	postSchema := New("Post").
		AddField(Field{Name: "id", Type: FieldTypeInt, PrimaryKey: true}).
		AddField(Field{Name: "title", Type: FieldTypeString}).
		AddField(Field{Name: "userId", Type: FieldTypeInt})

	tests := []struct {
		name          string
		relation      *Relation
		currentModel  *Schema
		relatedModel  *Schema
		expectError   bool
		errorContains string
	}{
		{
			name: "valid many-to-one",
			relation: &Relation{
				Type:       RelationManyToOne,
				Model:      "User",
				ForeignKey: "userId",
				References: "id",
			},
			currentModel: postSchema,
			relatedModel: userSchema,
			expectError:  false,
		},
		{
			name: "valid one-to-many",
			relation: &Relation{
				Type:       RelationOneToMany,
				Model:      "Post",
				ForeignKey: "userId",
				References: "id",
			},
			currentModel: userSchema,
			relatedModel: postSchema,
			expectError:  false,
		},
		{
			name: "missing foreign key in many-to-one",
			relation: &Relation{
				Type:       RelationManyToOne,
				Model:      "User",
				ForeignKey: "nonExistentField",
				References: "id",
			},
			currentModel:  postSchema,
			relatedModel:  userSchema,
			expectError:   true,
			errorContains: "foreign key field nonExistentField not found",
		},
		{
			name: "missing references field",
			relation: &Relation{
				Type:       RelationManyToOne,
				Model:      "User",
				ForeignKey: "userId",
				References: "nonExistentField",
			},
			currentModel:  postSchema,
			relatedModel:  userSchema,
			expectError:   true,
			errorContains: "references field nonExistentField not found",
		},
		{
			name: "related model nil",
			relation: &Relation{
				Type:       RelationManyToOne,
				Model:      "NonExistent",
				ForeignKey: "userId",
				References: "id",
			},
			currentModel:  postSchema,
			relatedModel:  nil,
			expectError:   true,
			errorContains: "related model NonExistent not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRelation(tt.relation, tt.currentModel, tt.relatedModel)
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBuildJoinCondition(t *testing.T) {
	// Create test schemas with field mappings
	userSchema := New("User").
		AddField(Field{Name: "id", Type: FieldTypeInt, PrimaryKey: true}).
		AddField(Field{Name: "firstName", Type: FieldTypeString, Map: "first_name"})

	postSchema := New("Post").
		AddField(Field{Name: "id", Type: FieldTypeInt, PrimaryKey: true}).
		AddField(Field{Name: "userId", Type: FieldTypeInt, Map: "user_id"})

	tests := []struct {
		name          string
		relation      *Relation
		currentTable  string
		relatedTable  string
		currentSchema *Schema
		relatedSchema *Schema
		expected      string
		expectError   bool
	}{
		{
			name: "many-to-one with field mapping",
			relation: &Relation{
				Type:       RelationManyToOne,
				Model:      "User",
				ForeignKey: "userId",
				References: "id",
			},
			currentTable:  "posts",
			relatedTable:  "users",
			currentSchema: postSchema,
			relatedSchema: userSchema,
			expected:      "posts.user_id = users.id",
			expectError:   false,
		},
		{
			name: "one-to-many",
			relation: &Relation{
				Type:       RelationOneToMany,
				Model:      "Post",
				ForeignKey: "userId",
				References: "id",
			},
			currentTable:  "users",
			relatedTable:  "posts",
			currentSchema: userSchema,
			relatedSchema: postSchema,
			expected:      "posts.user_id = users.id",
			expectError:   false,
		},
		{
			name: "many-to-many",
			relation: &Relation{
				Type:       RelationManyToMany,
				Model:      "Tag",
				ForeignKey: "postId",
				References: "id",
			},
			currentTable:  "posts",
			relatedTable:  "tags",
			currentSchema: postSchema,
			relatedSchema: userSchema,
			expected:      "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildJoinCondition(
				tt.relation,
				tt.currentTable,
				tt.relatedTable,
				tt.currentSchema,
				tt.relatedSchema,
			)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
