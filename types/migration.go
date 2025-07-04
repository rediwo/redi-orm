package types

import (
	"time"
)

// MigrationMode represents the migration execution mode
type MigrationMode string

const (
	MigrationModeAuto MigrationMode = "auto" // Auto-migrate (development)
	MigrationModeFile MigrationMode = "file" // File-based migrations (production)
)

// MigrationOptions contains options for migration operations
type MigrationOptions struct {
	AutoMigrate   bool
	DryRun        bool
	Force         bool          // Force destructive changes without confirmation
	Mode          MigrationMode // Migration mode (auto or file)
	MigrationsDir string        // Directory containing migration files
}

// ChangeType represents the type of schema change
type ChangeType string

const (
	ChangeTypeCreateTable ChangeType = "CREATE_TABLE"
	ChangeTypeDropTable   ChangeType = "DROP_TABLE"
	ChangeTypeAddColumn   ChangeType = "ADD_COLUMN"
	ChangeTypeDropColumn  ChangeType = "DROP_COLUMN"
	ChangeTypeAlterColumn ChangeType = "ALTER_COLUMN"
	ChangeTypeAddIndex    ChangeType = "ADD_INDEX"
	ChangeTypeDropIndex   ChangeType = "DROP_INDEX"
	ChangeTypeAddFK       ChangeType = "ADD_FOREIGN_KEY"
	ChangeTypeDropFK      ChangeType = "DROP_FOREIGN_KEY"
)

// SchemaChange represents a single schema change
type SchemaChange struct {
	Type       ChangeType
	TableName  string
	ColumnName string
	IndexName  string
	SQL        string
	// IndexDef stores index definition for DROP_INDEX changes
	// This allows recreating the index during rollback
	IndexDef *IndexDefinition `json:"index_def,omitempty"`
}

// IndexDefinition stores the definition of an index
type IndexDefinition struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
}

// Migration represents a database migration
type Migration struct {
	ID        int
	Version   string
	Name      string
	AppliedAt time.Time
	Checksum  string
}

// MigrationFile represents a migration file on disk
type MigrationFile struct {
	Version  string
	Name     string
	UpSQL    string
	DownSQL  string
	Metadata MigrationMetadata
}

// MigrationMetadata contains metadata for a migration file
type MigrationMetadata struct {
	Version     string            `json:"version"`
	Name        string            `json:"name"`
	Checksum    string            `json:"checksum"`
	CreatedAt   time.Time         `json:"created_at"`
	Description string            `json:"description"`
	Changes     []SchemaChange    `json:"changes"`
	Schemas     map[string]string `json:"schemas"` // Model name -> schema hash
}
