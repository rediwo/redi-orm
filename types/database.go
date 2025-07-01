package types

import (
	"context"
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

// Order represents sorting direction
type Order int

const (
	ASC Order = iota
	DESC
)

// ConflictAction represents action to take on insert conflicts
type ConflictAction int

const (
	ConflictIgnore ConflictAction = iota
	ConflictReplace
	ConflictUpdate
)

// Result represents operation result
type Result struct {
	LastInsertID int64
	RowsAffected int64
}

// JoinCondition represents a join condition
type JoinCondition struct {
	Left  string
	Op    string
	Right string
}

// BatchUpdate represents a batch update operation
type BatchUpdate struct {
	Condition Condition
	Data      interface{}
}

// DBStats represents database connection statistics
type DBStats struct {
	OpenConnections int
	InUse           int
	Idle            int
}

// TableSchema represents table schema information
type TableSchema struct {
	Name    string
	Columns []ColumnSchema
	Indexes []IndexSchema
}

// ColumnSchema represents column schema information
type ColumnSchema struct {
	Name          string
	Type          string
	Nullable      bool
	Default       interface{}
	PrimaryKey    bool
	AutoIncrement bool
	Unique        bool
}

// IndexSchema represents index schema information
type IndexSchema struct {
	Name    string
	Columns []string
	Unique  bool
}

// Database interface defines all database operations
type Database interface {
	// Connection management
	Connect(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error

	// Schema management
	RegisterSchema(modelName string, schema *schema.Schema) error
	GetSchema(modelName string) (*schema.Schema, error)
	CreateModel(ctx context.Context, modelName string) error
	DropModel(ctx context.Context, modelName string) error
	
	// Schema loading with auto-migration
	LoadSchema(ctx context.Context, schemaContent string) error
	LoadSchemaFrom(ctx context.Context, filename string) error
	SyncSchemas(ctx context.Context) error

	// Model query builder (uses modelName)
	Model(modelName string) ModelQuery
	Raw(sql string, args ...interface{}) RawQuery

	// Transaction management
	Begin(ctx context.Context) (Transaction, error)
	Transaction(ctx context.Context, fn func(tx Transaction) error) error

	// Metadata
	GetModels() []string
	GetModelSchema(modelName string) (*schema.Schema, error)

	// Internal field mapping (used by driver implementations)
	ResolveTableName(modelName string) (string, error)
	ResolveFieldName(modelName, fieldName string) (string, error)
	ResolveFieldNames(modelName string, fieldNames []string) ([]string, error)

	// Legacy raw query support
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row

	// Migration support (legacy)
	GetMigrator() DatabaseMigrator
}

// ModelQuery interface for model-based queries
type ModelQuery interface {
	// Query building (uses schema field names)
	Select(fields ...string) SelectQuery
	Insert(data interface{}) InsertQuery
	Update(data interface{}) UpdateQuery
	Delete() DeleteQuery

	// Condition building (uses schema field names)
	Where(fieldName string) FieldCondition
	WhereCondition(condition Condition) ModelQuery
	WhereRaw(sql string, args ...interface{}) ModelQuery

	// Relation queries (uses relation names)
	Include(relations ...string) ModelQuery
	With(relations ...string) ModelQuery

	// Sorting and pagination (uses schema field names)
	OrderBy(fieldName string, direction Order) ModelQuery
	GroupBy(fieldNames ...string) ModelQuery
	Having(condition Condition) ModelQuery
	Limit(limit int) ModelQuery
	Offset(offset int) ModelQuery

	// Execution
	FindMany(ctx context.Context, dest interface{}) error
	FindUnique(ctx context.Context, dest interface{}) error
	FindFirst(ctx context.Context, dest interface{}) error
	Count(ctx context.Context) (int64, error)
	Exists(ctx context.Context) (bool, error)

	// Aggregation (uses schema field names)
	Sum(ctx context.Context, fieldName string) (float64, error)
	Avg(ctx context.Context, fieldName string) (float64, error)
	Max(ctx context.Context, fieldName string) (interface{}, error)
	Min(ctx context.Context, fieldName string) (interface{}, error)

	// Internal
	GetModelName() string
}

// SelectQuery interface for select operations
type SelectQuery interface {
	// Condition building
	Where(fieldName string) FieldCondition
	WhereCondition(condition Condition) SelectQuery
	Include(relations ...string) SelectQuery
	OrderBy(fieldName string, direction Order) SelectQuery
	GroupBy(fieldNames ...string) SelectQuery
	Having(condition Condition) SelectQuery
	Limit(limit int) SelectQuery
	Offset(offset int) SelectQuery
	Distinct() SelectQuery

	// Execution
	FindMany(ctx context.Context, dest interface{}) error
	FindFirst(ctx context.Context, dest interface{}) error
	Count(ctx context.Context) (int64, error)

	// Internal methods (for driver implementation)
	BuildSQL() (string, []interface{}, error)
	GetModelName() string
}

// InsertQuery interface for insert operations
type InsertQuery interface {
	// Data uses schema field names
	Values(data ...interface{}) InsertQuery
	OnConflict(action ConflictAction) InsertQuery
	Returning(fieldNames ...string) InsertQuery

	// Execution
	Exec(ctx context.Context) (Result, error)
	ExecAndReturn(ctx context.Context, dest interface{}) error

	// Internal methods
	BuildSQL() (string, []interface{}, error)
	GetModelName() string
}

// UpdateQuery interface for update operations
type UpdateQuery interface {
	// Data uses schema field names
	Set(data interface{}) UpdateQuery
	Where(fieldName string) FieldCondition
	WhereCondition(condition Condition) UpdateQuery
	Returning(fieldNames ...string) UpdateQuery

	// Atomic operations (uses schema field names)
	Increment(fieldName string, value int64) UpdateQuery
	Decrement(fieldName string, value int64) UpdateQuery

	// Execution
	Exec(ctx context.Context) (Result, error)
	ExecAndReturn(ctx context.Context, dest interface{}) error

	// Internal methods
	BuildSQL() (string, []interface{}, error)
	GetModelName() string
}

// DeleteQuery interface for delete operations
type DeleteQuery interface {
	Where(fieldName string) FieldCondition
	WhereCondition(condition Condition) DeleteQuery
	Returning(fieldNames ...string) DeleteQuery

	// Execution
	Exec(ctx context.Context) (Result, error)

	// Internal methods
	BuildSQL() (string, []interface{}, error)
	GetModelName() string
}

// RawQuery interface for raw SQL queries
type RawQuery interface {
	Exec(ctx context.Context) (Result, error)
	Find(ctx context.Context, dest interface{}) error
	FindOne(ctx context.Context, dest interface{}) error
}

// Condition interface for query conditions
type Condition interface {
	ToSQL() (string, []interface{})
	And(condition Condition) Condition
	Or(condition Condition) Condition
	Not() Condition
}

// FieldCondition interface for field-specific conditions
type FieldCondition interface {
	// Basic comparisons
	Equals(value interface{}) Condition
	NotEquals(value interface{}) Condition
	GreaterThan(value interface{}) Condition
	GreaterThanOrEqual(value interface{}) Condition
	LessThan(value interface{}) Condition
	LessThanOrEqual(value interface{}) Condition

	// Collection operations
	In(values ...interface{}) Condition
	NotIn(values ...interface{}) Condition

	// String operations
	Contains(value string) Condition
	StartsWith(value string) Condition
	EndsWith(value string) Condition
	Like(pattern string) Condition

	// Null checks
	IsNull() Condition
	IsNotNull() Condition

	// Range operations
	Between(min, max interface{}) Condition

	// Internal methods (for driver implementation)
	GetFieldName() string
	GetModelName() string
}

// Transaction interface for database transactions
type Transaction interface {
	// Inherit all model query capabilities
	Model(modelName string) ModelQuery
	Raw(sql string, args ...interface{}) RawQuery

	// Transaction control
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	// Savepoints
	Savepoint(ctx context.Context, name string) error
	RollbackTo(ctx context.Context, name string) error

	// Batch operations (uses schema field names)
	CreateMany(ctx context.Context, modelName string, data []interface{}) (Result, error)
	UpdateMany(ctx context.Context, modelName string, condition Condition, data interface{}) (Result, error)
	DeleteMany(ctx context.Context, modelName string, condition Condition) (Result, error)
}

// FieldMapper interface for field name mapping between schema and database
type FieldMapper interface {
	// Field name mapping
	SchemaToColumn(modelName, fieldName string) (string, error)
	ColumnToSchema(modelName, columnName string) (string, error)

	// Batch mapping
	SchemaFieldsToColumns(modelName string, fieldNames []string) ([]string, error)
	ColumnFieldsToSchema(modelName string, columnNames []string) ([]string, error)

	// Data mapping
	MapSchemaToColumnData(modelName string, data map[string]interface{}) (map[string]interface{}, error)
	MapColumnToSchemaData(modelName string, data map[string]interface{}) (map[string]interface{}, error)

	// Table name mapping
	ModelToTable(modelName string) (string, error)
}

// Legacy migration types for backward compatibility
type TableInfo struct {
	Name        string
	Columns     []ColumnInfo
	Indexes     []IndexInfo
	ForeignKeys []ForeignKeyInfo
}

type ColumnInfo struct {
	Name          string
	Type          string
	Nullable      bool
	Default       interface{}
	PrimaryKey    bool
	AutoIncrement bool
	Unique        bool
}

type IndexInfo struct {
	Name    string
	Columns []string
	Unique  bool
}

type ForeignKeyInfo struct {
	Name             string
	Column           string
	ReferencedTable  string
	ReferencedColumn string
	OnDelete         string
	OnUpdate         string
}

type MigrationPlan struct {
	CreateTables  []string       // Tables to be created
	AddColumns    []ColumnChange // Columns to be added
	ModifyColumns []ColumnChange // Columns to be modified
	DropColumns   []ColumnChange // Columns to be dropped
	AddIndexes    []IndexChange  // Indexes to be added
	DropIndexes   []IndexChange  // Indexes to be dropped
}

type ColumnChange struct {
	TableName  string
	ColumnName string
	OldColumn  *ColumnInfo // nil for additions
	NewColumn  *ColumnInfo // nil for deletions
}

type IndexChange struct {
	TableName string
	IndexName string
	OldIndex  *IndexInfo // nil for additions
	NewIndex  *IndexInfo // nil for deletions
}

type DatabaseMigrator interface {
	// Introspection
	GetTables() ([]string, error)
	GetTableInfo(tableName string) (*TableInfo, error)

	// SQL Generation
	GenerateCreateTableSQL(schema interface{}) (string, error)
	GenerateDropTableSQL(tableName string) string
	GenerateAddColumnSQL(tableName string, field interface{}) (string, error)
	GenerateModifyColumnSQL(change ColumnChange) ([]string, error)
	GenerateDropColumnSQL(tableName, columnName string) ([]string, error)
	GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string
	GenerateDropIndexSQL(indexName string) string

	// Migration execution
	ApplyMigration(sql string) error
	GetDatabaseType() string

	// Schema comparison and migration planning
	CompareSchema(existingTable *TableInfo, desiredSchema interface{}) (*MigrationPlan, error)
	GenerateMigrationSQL(plan *MigrationPlan) ([]string, error)
}
