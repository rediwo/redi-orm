package base

import (
	"testing"

	"github.com/rediwo/redi-orm/schema"
)

func TestAnalyzeSchemasDependencies(t *testing.T) {
	tests := []struct {
		name        string
		schemas     map[string]*schema.Schema
		want        []string
		wantErr     bool
		errContains string
	}{
		{
			name: "simple linear dependency",
			schemas: map[string]*schema.Schema{
				"User": schema.New("User"),
				"Post": schema.New("Post").AddRelation("author", schema.Relation{
					Type:       "manyToOne",
					Model:      "User",
					ForeignKey: "userId",
				}),
			},
			want:    []string{"User", "Post"},
			wantErr: false,
		},
		{
			name: "multiple dependencies",
			schemas: map[string]*schema.Schema{
				"User":     schema.New("User"),
				"Category": schema.New("Category"),
				"Post": schema.New("Post").
					AddRelation("author", schema.Relation{
						Type:       "manyToOne",
						Model:      "User",
						ForeignKey: "userId",
					}).
					AddRelation("category", schema.Relation{
						Type:       "manyToOne",
						Model:      "Category",
						ForeignKey: "categoryId",
					}),
			},
			// Order between User and Category doesn't matter as they have no dependencies
			// So we accept either order
			wantErr: false,
		},
		{
			name: "chain dependencies",
			schemas: map[string]*schema.Schema{
				"Country": schema.New("Country"),
				"City": schema.New("City").AddRelation("country", schema.Relation{
					Type:       "manyToOne",
					Model:      "Country",
					ForeignKey: "countryId",
				}),
				"Address": schema.New("Address").AddRelation("city", schema.Relation{
					Type:       "manyToOne",
					Model:      "City",
					ForeignKey: "cityId",
				}),
			},
			want:    []string{"Country", "City", "Address"},
			wantErr: false,
		},
		{
			name: "circular dependency",
			schemas: map[string]*schema.Schema{
				"User": schema.New("User").AddRelation("profile", schema.Relation{
					Type:       "oneToOne",
					Model:      "Profile",
					ForeignKey: "profileId",
				}),
				"Profile": schema.New("Profile").AddRelation("user", schema.Relation{
					Type:       "manyToOne",
					Model:      "User",
					ForeignKey: "userId",
				}),
			},
			wantErr:     true,
			errContains: "circular dependency",
		},
		{
			name: "self-reference",
			schemas: map[string]*schema.Schema{
				"Category": schema.New("Category").AddRelation("parent", schema.Relation{
					Type:       "manyToOne",
					Model:      "Category",
					ForeignKey: "parentId",
				}),
			},
			want:    []string{"Category"},
			wantErr: false,
		},
		{
			name: "reference to missing model",
			schemas: map[string]*schema.Schema{
				"Post": schema.New("Post").AddRelation("author", schema.Relation{
					Type:       "manyToOne",
					Model:      "User", // User schema not provided
					ForeignKey: "userId",
				}),
			},
			want:    []string{"Post"},
			wantErr: false,
		},
		{
			name: "oneToMany relation (no dependency)",
			schemas: map[string]*schema.Schema{
				"User": schema.New("User").AddRelation("posts", schema.Relation{
					Type:  "oneToMany",
					Model: "Post",
				}),
				"Post": schema.New("Post"),
			},
			// oneToMany doesn't create dependency from User to Post
			// Order doesn't matter, just check both are present
			wantErr: false,
		},
		{
			name: "complex circular dependency",
			schemas: map[string]*schema.Schema{
				"A": schema.New("A").AddRelation("b", schema.Relation{
					Type:       "manyToOne",
					Model:      "B",
					ForeignKey: "bId",
				}),
				"B": schema.New("B").AddRelation("c", schema.Relation{
					Type:       "manyToOne",
					Model:      "C",
					ForeignKey: "cId",
				}),
				"C": schema.New("C").AddRelation("a", schema.Relation{
					Type:       "manyToOne",
					Model:      "A",
					ForeignKey: "aId",
				}),
			},
			wantErr:     true,
			errContains: "circular dependency",
		},
		{
			name:    "empty schemas",
			schemas: map[string]*schema.Schema{},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "independent schemas",
			schemas: map[string]*schema.Schema{
				"User":     schema.New("User"),
				"Product":  schema.New("Product"),
				"Category": schema.New("Category"),
			},
			// Order doesn't matter for independent schemas
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AnalyzeSchemasDependencies(tt.schemas)

			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyzeSchemasDependencies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("AnalyzeSchemasDependencies() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			// For schemas without specific order requirements, just check that all are present
			if tt.name == "independent schemas" || tt.name == "oneToMany relation (no dependency)" || tt.name == "multiple dependencies" {
				if len(got) != len(tt.schemas) {
					t.Errorf("AnalyzeSchemasDependencies() returned %d schemas, want %d", len(got), len(tt.schemas))
				}
				// Check all schemas are present
				gotMap := make(map[string]bool)
				for _, name := range got {
					gotMap[name] = true
				}
				for name := range tt.schemas {
					if !gotMap[name] {
						t.Errorf("AnalyzeSchemasDependencies() missing schema %s", name)
					}
				}
				return
			}

			// For specific order tests, check exact order
			if tt.want != nil {
				if !equalStringSlices(got, tt.want) {
					t.Errorf("AnalyzeSchemasDependencies() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestSplitSchemasByDependency(t *testing.T) {
	tests := []struct {
		name             string
		schemas          map[string]*schema.Schema
		wantWithFKLen    int
		wantWithoutFKLen int
		hasCircular      bool
	}{
		{
			name: "no circular dependencies",
			schemas: map[string]*schema.Schema{
				"User": schema.New("User"),
				"Post": schema.New("Post").AddRelation("author", schema.Relation{
					Type:       "manyToOne",
					Model:      "User",
					ForeignKey: "userId",
				}),
			},
			wantWithFKLen:    2,
			wantWithoutFKLen: 0,
			hasCircular:      false,
		},
		{
			name: "circular dependencies",
			schemas: map[string]*schema.Schema{
				"User": schema.New("User").AddRelation("profile", schema.Relation{
					Type:       "oneToOne",
					Model:      "Profile",
					ForeignKey: "profileId",
				}),
				"Profile": schema.New("Profile").AddRelation("user", schema.Relation{
					Type:       "manyToOne",
					Model:      "User",
					ForeignKey: "userId",
				}),
			},
			wantWithFKLen:    0,
			wantWithoutFKLen: 2,
			hasCircular:      true,
		},
		{
			name:             "empty schemas",
			schemas:          map[string]*schema.Schema{},
			wantWithFKLen:    0,
			wantWithoutFKLen: 0,
			hasCircular:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withFK, withoutFK, err := splitSchemasByDependency(tt.schemas)

			// splitSchemasByDependency should not return error
			if err != nil {
				t.Errorf("splitSchemasByDependency() error = %v", err)
				return
			}

			if len(withFK) != tt.wantWithFKLen {
				t.Errorf("splitSchemasByDependency() withFK length = %d, want %d", len(withFK), tt.wantWithFKLen)
			}

			if len(withoutFK) != tt.wantWithoutFKLen {
				t.Errorf("splitSchemasByDependency() withoutFK length = %d, want %d", len(withoutFK), tt.wantWithoutFKLen)
			}

			// Verify all schemas are accounted for
			totalSchemas := len(withFK) + len(withoutFK)
			if totalSchemas != len(tt.schemas) {
				t.Errorf("splitSchemasByDependency() total schemas = %d, want %d", totalSchemas, len(tt.schemas))
			}
		})
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexString(s, substr) >= 0))
}

func indexString(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
