package migration

import (
	"errors"
	"testing"

	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

// mockDifferMigrator implements types.DatabaseMigrator for testing differ
type mockDifferMigrator struct {
	tables          []string
	tableInfoMap    map[string]*types.TableInfo
	compareSchemaFn func(*types.TableInfo, any) (*types.MigrationPlan, error)
	generateSQLFn   func(*types.MigrationPlan) ([]string, error)
	shouldError     map[string]bool // Map of method names that should return errors
}

func newMockDifferMigrator() *mockDifferMigrator {
	return &mockDifferMigrator{
		tables:       []string{},
		tableInfoMap: make(map[string]*types.TableInfo),
		shouldError:  make(map[string]bool),
	}
}

func (m *mockDifferMigrator) GetTables() ([]string, error) {
	if m.shouldError["GetTables"] {
		return nil, errors.New("GetTables error")
	}
	return m.tables, nil
}

func (m *mockDifferMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
	if m.shouldError["GetTableInfo"] {
		return nil, errors.New("GetTableInfo error")
	}
	if info, ok := m.tableInfoMap[tableName]; ok {
		return info, nil
	}
	return &types.TableInfo{Name: tableName}, nil
}

func (m *mockDifferMigrator) GenerateCreateTableSQL(s any) (string, error) {
	if m.shouldError["GenerateCreateTableSQL"] {
		return "", errors.New("GenerateCreateTableSQL error")
	}
	if sch, ok := s.(*schema.Schema); ok {
		return "CREATE TABLE " + sch.TableName, nil
	}
	return "CREATE TABLE test", nil
}

func (m *mockDifferMigrator) GenerateDropTableSQL(tableName string) string {
	return "DROP TABLE " + tableName
}

func (m *mockDifferMigrator) GenerateAddColumnSQL(tableName string, field any) (string, error) {
	return "ALTER TABLE " + tableName + " ADD COLUMN test", nil
}

func (m *mockDifferMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	return []string{"ALTER TABLE " + change.TableName + " MODIFY COLUMN " + change.ColumnName}, nil
}

func (m *mockDifferMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	return []string{"ALTER TABLE " + tableName + " DROP COLUMN " + columnName}, nil
}

func (m *mockDifferMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	return "CREATE INDEX " + indexName + " ON " + tableName
}

func (m *mockDifferMigrator) GenerateDropIndexSQL(indexName string) string {
	return "DROP INDEX " + indexName
}

func (m *mockDifferMigrator) ApplyMigration(sql string) error {
	return nil
}

func (m *mockDifferMigrator) GetDatabaseType() string {
	return "test"
}

func (m *mockDifferMigrator) IsSystemTable(tableName string) bool {
	return false
}

func (m *mockDifferMigrator) CompareSchema(existingTable *types.TableInfo, desiredSchema any) (*types.MigrationPlan, error) {
	if m.shouldError["CompareSchema"] {
		return nil, errors.New("CompareSchema error")
	}
	if m.compareSchemaFn != nil {
		return m.compareSchemaFn(existingTable, desiredSchema)
	}
	return &types.MigrationPlan{}, nil
}

func (m *mockDifferMigrator) GenerateMigrationSQL(plan *types.MigrationPlan) ([]string, error) {
	if m.shouldError["GenerateMigrationSQL"] {
		return nil, errors.New("GenerateMigrationSQL error")
	}
	if m.generateSQLFn != nil {
		return m.generateSQLFn(plan)
	}

	var sqls []string
	for range plan.AddColumns {
		sqls = append(sqls, "ALTER TABLE ADD COLUMN")
	}
	return sqls, nil
}

func TestNewDiffer(t *testing.T) {
	migrator := newMockDifferMigrator()
	differ := NewDiffer(migrator)

	if differ == nil {
		t.Error("NewDiffer returned nil")
	}
	if differ.migrator != migrator {
		t.Error("NewDiffer did not set migrator correctly")
	}
}

func TestDiffer_ComputeDiff(t *testing.T) {
	tests := []struct {
		name           string
		existingTables []string
		schemas        map[string]*schema.Schema
		shouldError    map[string]bool
		wantChanges    []types.ChangeType
		wantErr        bool
	}{
		{
			name:           "create new table",
			existingTables: []string{},
			schemas: map[string]*schema.Schema{
				"User": {Name: "User", TableName: "users"},
			},
			wantChanges: []types.ChangeType{types.ChangeTypeCreateTable},
			wantErr:     false,
		},
		{
			name:           "drop table",
			existingTables: []string{"users", "posts"},
			schemas: map[string]*schema.Schema{
				"User": {Name: "User", TableName: "users"},
			},
			wantChanges: []types.ChangeType{types.ChangeTypeDropTable},
			wantErr:     false,
		},
		{
			name:           "no changes",
			existingTables: []string{"users"},
			schemas: map[string]*schema.Schema{
				"User": {Name: "User", TableName: "users"},
			},
			wantChanges: []types.ChangeType{},
			wantErr:     false,
		},
		{
			name:           "skip migrations table",
			existingTables: []string{"users", MigrationsTableName}, // Use the constant
			schemas:        map[string]*schema.Schema{},
			wantChanges:    []types.ChangeType{types.ChangeTypeDropTable}, // Only drop users, not _migrations
			wantErr:        false,
		},
		{
			name:           "error getting tables",
			existingTables: []string{},
			schemas:        map[string]*schema.Schema{},
			shouldError:    map[string]bool{"GetTables": true},
			wantErr:        true,
		},
		{
			name:           "error generating create table SQL",
			existingTables: []string{},
			schemas: map[string]*schema.Schema{
				"User": {Name: "User", TableName: "users"},
			},
			shouldError: map[string]bool{"GenerateCreateTableSQL": true},
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrator := newMockDifferMigrator()
			migrator.tables = tt.existingTables
			migrator.shouldError = tt.shouldError

			differ := NewDiffer(migrator)
			changes, err := differ.ComputeDiff(tt.schemas)

			if (err != nil) != tt.wantErr {
				t.Errorf("ComputeDiff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(changes) != len(tt.wantChanges) {
					t.Errorf("ComputeDiff() returned %d changes, want %d", len(changes), len(tt.wantChanges))
					return
				}

				for i, change := range changes {
					if i < len(tt.wantChanges) && change.Type != tt.wantChanges[i] {
						t.Errorf("ComputeDiff() change[%d].Type = %v, want %v", i, change.Type, tt.wantChanges[i])
					}
				}
			}
		})
	}
}

func TestDiffer_computeTableDiff(t *testing.T) {
	tests := []struct {
		name          string
		schema        *schema.Schema
		migrationPlan *types.MigrationPlan
		sqlStatements []string
		shouldError   map[string]bool
		wantChanges   int
		wantErr       bool
	}{
		{
			name:   "add columns",
			schema: &schema.Schema{Name: "User", TableName: "users"},
			migrationPlan: &types.MigrationPlan{
				AddColumns: []types.ColumnChange{
					{TableName: "users", ColumnName: "email"},
					{TableName: "users", ColumnName: "age"},
				},
			},
			sqlStatements: []string{"ALTER TABLE users ADD email", "ALTER TABLE users ADD age"},
			wantChanges:   2,
			wantErr:       false,
		},
		{
			name:   "drop columns",
			schema: &schema.Schema{Name: "User", TableName: "users"},
			migrationPlan: &types.MigrationPlan{
				DropColumns: []types.ColumnChange{
					{TableName: "users", ColumnName: "old_field"},
				},
			},
			wantChanges: 1,
			wantErr:     false,
		},
		{
			name:   "add and drop indexes",
			schema: &schema.Schema{Name: "User", TableName: "users"},
			migrationPlan: &types.MigrationPlan{
				AddIndexes: []types.IndexChange{
					{
						TableName: "users",
						IndexName: "idx_email",
						NewIndex:  &types.IndexInfo{Name: "idx_email", Columns: []string{"email"}, Unique: true},
					},
				},
				DropIndexes: []types.IndexChange{
					{
						TableName: "users",
						IndexName: "idx_old",
						OldIndex:  &types.IndexInfo{Name: "idx_old", Columns: []string{"old"}, Unique: false},
					},
				},
			},
			wantChanges: 2,
			wantErr:     false,
		},
		{
			name:        "error getting table info",
			schema:      &schema.Schema{Name: "User", TableName: "users"},
			shouldError: map[string]bool{"GetTableInfo": true},
			wantErr:     true,
		},
		{
			name:        "error comparing schema",
			schema:      &schema.Schema{Name: "User", TableName: "users"},
			shouldError: map[string]bool{"CompareSchema": true},
			wantErr:     true,
		},
		{
			name:          "error generating migration SQL",
			schema:        &schema.Schema{Name: "User", TableName: "users"},
			migrationPlan: &types.MigrationPlan{},
			shouldError:   map[string]bool{"GenerateMigrationSQL": true},
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrator := newMockDifferMigrator()
			migrator.shouldError = tt.shouldError

			if tt.migrationPlan != nil {
				migrator.compareSchemaFn = func(*types.TableInfo, any) (*types.MigrationPlan, error) {
					return tt.migrationPlan, nil
				}
			}

			if tt.sqlStatements != nil {
				migrator.generateSQLFn = func(*types.MigrationPlan) ([]string, error) {
					return tt.sqlStatements, nil
				}
			}

			differ := NewDiffer(migrator)
			changes, err := differ.computeTableDiff(tt.schema)

			if (err != nil) != tt.wantErr {
				t.Errorf("computeTableDiff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(changes) != tt.wantChanges {
				t.Errorf("computeTableDiff() returned %d changes, want %d", len(changes), tt.wantChanges)
			}
		})
	}
}

func TestComputeChecksum(t *testing.T) {
	tests := []struct {
		name    string
		changes []types.SchemaChange
	}{
		{
			name:    "empty changes",
			changes: []types.SchemaChange{},
		},
		{
			name: "single change",
			changes: []types.SchemaChange{
				{
					Type:      types.ChangeTypeCreateTable,
					TableName: "users",
					SQL:       "CREATE TABLE users",
				},
			},
		},
		{
			name: "multiple changes",
			changes: []types.SchemaChange{
				{
					Type:      types.ChangeTypeCreateTable,
					TableName: "users",
					SQL:       "CREATE TABLE users",
				},
				{
					Type:      types.ChangeTypeAddColumn,
					TableName: "users",
					SQL:       "ALTER TABLE users ADD email",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeChecksum(tt.changes)

			// Checksum should be 64 characters (SHA256 hex)
			if len(got) != 64 {
				t.Errorf("ComputeChecksum() returned %d chars, want 64", len(got))
			}

			// Test that same changes produce same checksum
			got2 := ComputeChecksum(tt.changes)
			if got != got2 {
				t.Error("ComputeChecksum() not deterministic")
			}
		})
	}

	// Test that different changes produce different checksums
	changes1 := []types.SchemaChange{{Type: "CREATE", TableName: "users", SQL: "CREATE TABLE users"}}
	changes2 := []types.SchemaChange{{Type: "CREATE", TableName: "posts", SQL: "CREATE TABLE posts"}}

	checksum1 := ComputeChecksum(changes1)
	checksum2 := ComputeChecksum(changes2)

	if checksum1 == checksum2 {
		t.Error("Different changes produced same checksum")
	}
}
