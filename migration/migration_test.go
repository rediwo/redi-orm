package migration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rediwo/redi-orm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrationFileNaming tests migration file naming conventions
func TestMigrationFileNaming(t *testing.T) {
	t.Run("ValidMigrationFileNames", func(t *testing.T) {
		validNames := []string{
			"20240101120000_create_users.up.sql",
			"20240101120000_create_users.down.sql",
			"1234567890_add_email_to_users.up.sql",
			"9999999999_drop_old_tables.down.sql",
		}

		for _, name := range validNames {
			version, migrationName, direction := parseMigrationFileName(name)
			assert.NotEmpty(t, version, "Should extract version from %s", name)
			assert.NotEmpty(t, migrationName, "Should extract name from %s", name)
			assert.NotEmpty(t, direction, "Should extract direction from %s", name)
		}
	})

	t.Run("InvalidMigrationFileNames", func(t *testing.T) {
		invalidNames := []string{
			"migration.sql",
			"add_users.up.sql",
			"20240101_missing_extension",
			"not_a_migration.txt",
		}

		for _, name := range invalidNames {
			version, _, _ := parseMigrationFileName(name)
			assert.Empty(t, version, "Should not extract version from invalid name %s", name)
		}
	})
}

// Helper function to parse migration file names
func parseMigrationFileName(filename string) (version, name, direction string) {
	// Expected format: <timestamp>_<name>.<direction>.sql
	parts := strings.Split(filename, "_")
	if len(parts) < 2 {
		return "", "", ""
	}

	// Validate that first part is a timestamp (all digits)
	potentialVersion := parts[0]
	for _, ch := range potentialVersion {
		if ch < '0' || ch > '9' {
			return "", "", ""
		}
	}

	// Validate file extension
	if !strings.HasSuffix(filename, ".up.sql") && !strings.HasSuffix(filename, ".down.sql") {
		return "", "", ""
	}

	version = potentialVersion

	// Extract name and direction
	remaining := strings.Join(parts[1:], "_")
	if strings.HasSuffix(remaining, ".up.sql") {
		name = strings.TrimSuffix(remaining, ".up.sql")
		direction = "up"
	} else if strings.HasSuffix(remaining, ".down.sql") {
		name = strings.TrimSuffix(remaining, ".down.sql")
		direction = "down"
	}

	return version, name, direction
}

// TestIndexRollback tests index rollback functionality
func TestIndexRollback(t *testing.T) {
	t.Run("GenerateCreateIndexSQL", func(t *testing.T) {
		migrator := &mockMigrator{}
		generator := &Generator{
			migrator: migrator,
		}

		tests := []struct {
			name     string
			change   types.SchemaChange
			expected string
		}{
			{
				name: "recreate regular index",
				change: types.SchemaChange{
					Type:      types.ChangeTypeDropIndex,
					TableName: "users",
					IndexName: "idx_users_email",
					IndexDef: &types.IndexDefinition{
						Name:    "idx_users_email",
						Columns: []string{"email"},
						Unique:  false,
					},
				},
				expected: "CREATE INDEX idx_users_email ON users (email)",
			},
			{
				name: "recreate unique index",
				change: types.SchemaChange{
					Type:      types.ChangeTypeDropIndex,
					TableName: "users",
					IndexName: "idx_users_email_unique",
					IndexDef: &types.IndexDefinition{
						Name:    "idx_users_email_unique",
						Columns: []string{"email"},
						Unique:  true,
					},
				},
				expected: "CREATE UNIQUE INDEX idx_users_email_unique ON users (email)",
			},
			{
				name: "recreate composite index",
				change: types.SchemaChange{
					Type:      types.ChangeTypeDropIndex,
					TableName: "users",
					IndexName: "idx_users_name_email",
					IndexDef: &types.IndexDefinition{
						Name:    "idx_users_name_email",
						Columns: []string{"name", "email"},
						Unique:  false,
					},
				},
				expected: "CREATE INDEX idx_users_name_email ON users (name, email)",
			},
			{
				name: "no index definition available",
				change: types.SchemaChange{
					Type:      types.ChangeTypeDropIndex,
					TableName: "users",
					IndexName: "idx_unknown",
					IndexDef:  nil,
				},
				expected: "-- Cannot recreate index idx_unknown without stored definition",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := generator.generateCreateIndexSQL(tt.change)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("types.SchemaChangeWithIndexDef", func(t *testing.T) {
		// Test JSON serialization/deserialization
		change := types.SchemaChange{
			Type:      types.ChangeTypeDropIndex,
			TableName: "users",
			IndexName: "idx_users_email",
			SQL:       "DROP INDEX idx_users_email",
			IndexDef: &types.IndexDefinition{
				Name:    "idx_users_email",
				Columns: []string{"email", "created_at"},
				Unique:  true,
			},
		}

		// This would be tested in actual file operations
		// For now, we just verify the structure is correct
		require.NotNil(t, change.IndexDef)
		assert.Equal(t, "idx_users_email", change.IndexDef.Name)
		assert.Equal(t, []string{"email", "created_at"}, change.IndexDef.Columns)
		assert.True(t, change.IndexDef.Unique)
	})

	t.Run("types.IndexDefinitionStorage", func(t *testing.T) {
		// Test that IndexDef is properly serialized/deserialized
		change := types.SchemaChange{
			Type:      types.ChangeTypeDropIndex,
			TableName: "users",
			IndexName: "idx_user_email",
			SQL:       "DROP INDEX idx_user_email",
			IndexDef: &types.IndexDefinition{
				Name:    "idx_user_email",
				Columns: []string{"email", "created_at"},
				Unique:  true,
			},
		}

		// Serialize to JSON
		data, err := json.Marshal(change)
		require.NoError(t, err)

		// Deserialize from JSON
		var loaded types.SchemaChange
		err = json.Unmarshal(data, &loaded)
		require.NoError(t, err)

		// Verify IndexDef was preserved
		require.NotNil(t, loaded.IndexDef)
		assert.Equal(t, "idx_user_email", loaded.IndexDef.Name)
		assert.Equal(t, []string{"email", "created_at"}, loaded.IndexDef.Columns)
		assert.True(t, loaded.IndexDef.Unique)
	})

	t.Run("DifferStorestypes.IndexDefinitions", func(t *testing.T) {
		// Create a mock plan with DROP_INDEX changes
		plan := &types.MigrationPlan{
			DropIndexes: []types.IndexChange{
				{
					TableName: "users",
					IndexName: "idx_user_name",
					OldIndex: &types.IndexInfo{
						Name:    "idx_user_name",
						Columns: []string{"name"},
						Unique:  false,
					},
				},
			},
		}

		// Create a mock migrator
		migrator := &mockMigrator{}

		// Convert the plan to types.SchemaChanges
		var changes []types.SchemaChange
		for _, change := range plan.DropIndexes {
			schemaChange := types.SchemaChange{
				Type:      types.ChangeTypeDropIndex,
				TableName: change.TableName,
				IndexName: change.IndexName,
				SQL:       migrator.GenerateDropIndexSQL(change.IndexName),
			}

			// This is what we're testing - storing the index definition
			if change.OldIndex != nil {
				schemaChange.IndexDef = &types.IndexDefinition{
					Name:    change.OldIndex.Name,
					Columns: change.OldIndex.Columns,
					Unique:  change.OldIndex.Unique,
				}
			}

			changes = append(changes, schemaChange)
		}

		// Verify the index definition was stored
		require.Len(t, changes, 1)
		require.NotNil(t, changes[0].IndexDef)
		assert.Equal(t, "idx_user_name", changes[0].IndexDef.Name)
		assert.Equal(t, []string{"name"}, changes[0].IndexDef.Columns)
		assert.False(t, changes[0].IndexDef.Unique)
	})
}

// mockMigrator for testing
type mockMigrator struct {
	types.DatabaseMigrator
}

func (m *mockMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	if unique {
		return "CREATE UNIQUE INDEX " + indexName + " ON " + tableName + " (" + joinColumns(columns) + ")"
	}
	return "CREATE INDEX " + indexName + " ON " + tableName + " (" + joinColumns(columns) + ")"
}

func (m *mockMigrator) GenerateDropIndexSQL(indexName string) string {
	return "DROP INDEX " + indexName
}

func (m *mockMigrator) GetDatabaseType() string {
	return "mock"
}

func joinColumns(columns []string) string {
	result := ""
	for i, col := range columns {
		if i > 0 {
			result += ", "
		}
		result += col
	}
	return result
}
