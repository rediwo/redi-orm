package types

import (
	"database/sql"
	"github.com/rediwo/redi-orm/schema"
)

// Config holds database connection configuration
type Config struct {
	Type     string // Database type (e.g., "sqlite", "mysql", "postgresql")
	Host     string
	Port     int
	Database string
	User     string
	Password string
	FilePath string // for SQLite
}

// QueryBuilder interface for building database queries
type QueryBuilder interface {
	Where(field string, operator string, value interface{}) QueryBuilder
	WhereIn(field string, values []interface{}) QueryBuilder
	OrderBy(field string, direction string) QueryBuilder
	Limit(limit int) QueryBuilder
	Offset(offset int) QueryBuilder
	Execute() ([]map[string]interface{}, error)
	First() (map[string]interface{}, error)
	Count() (int64, error)
}

// Transaction interface for database transactions
type Transaction interface {
	Commit() error
	Rollback() error
	Insert(tableName string, data map[string]interface{}) (int64, error)
	Update(tableName string, id interface{}, data map[string]interface{}) error
	Delete(tableName string, id interface{}) error
}

// Database interface defines all database operations
type Database interface {
	Connect() error
	Close() error
	CreateTable(schema *schema.Schema) error
	DropTable(tableName string) error

	// Schema management
	RegisterSchema(modelName string, schema interface{}) error
	GetRegisteredSchemas() map[string]interface{}

	// CRUD operations (accepts model names and converts to table names)
	Insert(modelName string, data map[string]interface{}) (int64, error)
	FindByID(modelName string, id interface{}) (map[string]interface{}, error)
	Find(modelName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error)
	Update(modelName string, id interface{}, data map[string]interface{}) error
	Delete(modelName string, id interface{}) error

	// Query builder (accepts model names)
	Select(modelName string, columns []string) QueryBuilder

	// Raw operations (uses actual table names)
	RawInsert(tableName string, data map[string]interface{}) (int64, error)
	RawFindByID(tableName string, id interface{}) (map[string]interface{}, error)
	RawFind(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error)
	RawUpdate(tableName string, id interface{}, data map[string]interface{}) error
	RawDelete(tableName string, id interface{}) error
	RawSelect(tableName string, columns []string) QueryBuilder

	// Transaction
	Begin() (Transaction, error)

	// Raw query
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row

	// Migration support
	GetMigrator() DatabaseMigrator
	EnsureSchema() error // Auto-migrate all registered schemas
}

// DatabaseMigrator defines database-specific migration operations
type DatabaseMigrator interface {
	// Introspection
	GetTables() ([]string, error)
	GetTableInfo(tableName string) (*TableInfo, error)

	// SQL Generation
	GenerateCreateTableSQL(schema *schema.Schema) (string, error)
	GenerateDropTableSQL(tableName string) string
	GenerateAddColumnSQL(tableName string, field interface{}) (string, error)
	GenerateDropColumnSQL(tableName, columnName string) string
	GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string
	GenerateDropIndexSQL(indexName string) string

	// Migration execution
	ApplyMigration(sql string) error
	GetDatabaseType() string
	
	// Schema comparison and migration planning
	CompareSchema(existingTable *TableInfo, desiredSchema *schema.Schema) (*MigrationPlan, error)
	GenerateMigrationSQL(plan *MigrationPlan) ([]string, error)
}

// TableInfo represents database table information
type TableInfo struct {
	Name        string
	Columns     []ColumnInfo
	Indexes     []IndexInfo
	ForeignKeys []ForeignKeyInfo
}

// ColumnInfo represents database column information
type ColumnInfo struct {
	Name          string
	Type          string
	Nullable      bool
	Default       interface{}
	PrimaryKey    bool
	AutoIncrement bool
	Unique        bool
}

// IndexInfo represents database index information
type IndexInfo struct {
	Name    string
	Columns []string
	Unique  bool
}

// ForeignKeyInfo represents foreign key information
type ForeignKeyInfo struct {
	Name             string
	Column           string
	ReferencedTable  string
	ReferencedColumn string
	OnDelete         string
	OnUpdate         string
}

// MigrationPlan represents a plan for database schema migration
type MigrationPlan struct {
	CreateTables  []string        // Tables to be created
	AddColumns    []ColumnChange  // Columns to be added
	ModifyColumns []ColumnChange  // Columns to be modified
	DropColumns   []ColumnChange  // Columns to be dropped
	AddIndexes    []IndexChange   // Indexes to be added
	DropIndexes   []IndexChange   // Indexes to be dropped
}

// ColumnChange represents a column modification
type ColumnChange struct {
	TableName  string
	ColumnName string
	OldColumn  *ColumnInfo // nil for additions
	NewColumn  *ColumnInfo // nil for deletions
}

// IndexChange represents an index modification
type IndexChange struct {
	TableName string
	IndexName string
	OldIndex  *IndexInfo // nil for additions
	NewIndex  *IndexInfo // nil for deletions
}
