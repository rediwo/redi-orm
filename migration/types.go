package migration

import (
	"time"
)

const (
	// MigrationsTableName is the name of the table that stores migration history
	MigrationsTableName = "redi_migrations"
)

// Migration represents a database migration
type Migration struct {
	ID        int
	Version   string
	Name      string
	AppliedAt time.Time
	Checksum  string
}

// MigrationOptions contains options for migration operations
type MigrationOptions struct {
	AutoMigrate bool
	DryRun      bool
	Force       bool // Force destructive changes without confirmation
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
}

// MigrationPlan represents a planned migration
type MigrationPlan struct {
	Version   string
	Changes   []SchemaChange
	Checksum  string
	Timestamp time.Time
}
