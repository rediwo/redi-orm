package types

import (
	"context"
	"database/sql"

	"github.com/rediwo/redi-orm/schema"
)

// Order represents sorting direction
type Order int

const (
	ASC Order = iota
	DESC
)

// OrderByClause represents an ORDER BY clause
type OrderByClause struct {
	Field     string
	Direction Order
}

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
	Raw(sql string, args ...any) RawQuery

	// Transaction management
	Begin(ctx context.Context) (Transaction, error)
	Transaction(ctx context.Context, fn func(tx Transaction) error) error

	// Metadata
	GetModels() []string
	GetModelSchema(modelName string) (*schema.Schema, error)
	GetDriverType() string
	GetCapabilities() DriverCapabilities

	// Internal field mapping (used by driver implementations)
	ResolveTableName(modelName string) (string, error)
	ResolveFieldName(modelName, fieldName string) (string, error)
	ResolveFieldNames(modelName string, fieldNames []string) ([]string, error)

	// Raw query support
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row

	// Migration support
	GetMigrator() DatabaseMigrator
}

// ModelQuery interface for model-based queries
type ModelQuery interface {
	// Query building (uses schema field names)
	Select(fields ...string) SelectQuery
	Insert(data any) InsertQuery
	Update(data any) UpdateQuery
	Delete() DeleteQuery
	Aggregate() AggregationQuery

	// Condition building (uses schema field names)
	Where(fieldName string) FieldCondition
	WhereCondition(condition Condition) ModelQuery
	WhereRaw(sql string, args ...any) ModelQuery

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
	FindMany(ctx context.Context, dest any) error
	FindUnique(ctx context.Context, dest any) error
	FindFirst(ctx context.Context, dest any) error
	Count(ctx context.Context) (int64, error)
	Exists(ctx context.Context) (bool, error)

	// Aggregation (uses schema field names)
	Sum(ctx context.Context, fieldName string) (float64, error)
	Avg(ctx context.Context, fieldName string) (float64, error)
	Max(ctx context.Context, fieldName string) (any, error)
	Min(ctx context.Context, fieldName string) (any, error)

	// Internal
	GetModelName() string
}

// SelectQuery interface for select operations
type SelectQuery interface {
	// Condition building
	Where(fieldName string) FieldCondition
	WhereCondition(condition Condition) SelectQuery
	Include(relations ...string) SelectQuery
	IncludeWithOptions(path string, opt *IncludeOption) SelectQuery
	OrderBy(fieldName string, direction Order) SelectQuery
	GroupBy(fieldNames ...string) SelectQuery
	Having(condition Condition) SelectQuery
	Limit(limit int) SelectQuery
	Offset(offset int) SelectQuery
	Distinct() SelectQuery
	DistinctOn(fieldNames ...string) SelectQuery

	// Execution
	FindMany(ctx context.Context, dest any) error
	FindFirst(ctx context.Context, dest any) error
	Count(ctx context.Context) (int64, error)

	// Internal methods (for driver implementation)
	BuildSQL() (string, []any, error)
	GetModelName() string
}

// AggregationQuery interface for aggregation operations
type AggregationQuery interface {
	// Grouping
	GroupBy(fieldNames ...string) AggregationQuery
	Having(condition Condition) AggregationQuery

	// Aggregation functions
	Count(fieldName string, alias string) AggregationQuery
	CountAll(alias string) AggregationQuery
	Sum(fieldName string, alias string) AggregationQuery
	Avg(fieldName string, alias string) AggregationQuery
	Min(fieldName string, alias string) AggregationQuery
	Max(fieldName string, alias string) AggregationQuery

	// Selection of grouped fields
	Select(fieldNames ...string) AggregationQuery

	// Conditions and ordering
	Where(fieldName string) FieldCondition
	WhereCondition(condition Condition) AggregationQuery
	OrderBy(fieldName string, direction Order) AggregationQuery
	OrderByAggregation(aggregationType string, fieldName string, direction Order) AggregationQuery
	Limit(limit int) AggregationQuery
	Offset(offset int) AggregationQuery

	// Execution
	Exec(ctx context.Context, dest any) error

	// Internal methods
	BuildSQL() (string, []any, error)
	GetModelName() string
}

// InsertQuery interface for insert operations
type InsertQuery interface {
	// Data uses schema field names
	Values(data ...any) InsertQuery
	OnConflict(action ConflictAction) InsertQuery
	Returning(fieldNames ...string) InsertQuery

	// Execution
	Exec(ctx context.Context) (Result, error)
	ExecAndReturn(ctx context.Context, dest any) error

	// Internal methods
	BuildSQL() (string, []any, error)
	GetModelName() string
}

// UpdateQuery interface for update operations
type UpdateQuery interface {
	// Data uses schema field names
	Set(data any) UpdateQuery
	Where(fieldName string) FieldCondition
	WhereCondition(condition Condition) UpdateQuery
	Returning(fieldNames ...string) UpdateQuery

	// Atomic operations (uses schema field names)
	Increment(fieldName string, value int64) UpdateQuery
	Decrement(fieldName string, value int64) UpdateQuery

	// Execution
	Exec(ctx context.Context) (Result, error)
	ExecAndReturn(ctx context.Context, dest any) error

	// Internal methods
	BuildSQL() (string, []any, error)
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
	BuildSQL() (string, []any, error)
	GetModelName() string
}

// RawQuery interface for raw SQL queries
type RawQuery interface {
	Exec(ctx context.Context) (Result, error)
	Find(ctx context.Context, dest any) error
	FindOne(ctx context.Context, dest any) error
}

// Condition interface for query conditions
type Condition interface {
	ToSQL(ctx *ConditionContext) (string, []any)
	And(condition Condition) Condition
	Or(condition Condition) Condition
	Not() Condition
}

// FieldCondition interface for field-specific conditions
type FieldCondition interface {
	// Basic comparisons
	Equals(value any) Condition
	NotEquals(value any) Condition
	GreaterThan(value any) Condition
	GreaterThanOrEqual(value any) Condition
	LessThan(value any) Condition
	LessThanOrEqual(value any) Condition

	// Collection operations
	In(values ...any) Condition
	NotIn(values ...any) Condition

	// String operations
	Contains(value string) Condition
	StartsWith(value string) Condition
	EndsWith(value string) Condition
	Like(pattern string) Condition

	// Null checks
	IsNull() Condition
	IsNotNull() Condition

	// Range operations
	Between(min, max any) Condition

	// Internal methods (for driver implementation)
	GetFieldName() string
	GetModelName() string
}

// Transaction interface for database transactions
type Transaction interface {
	// Inherit all model query capabilities
	Model(modelName string) ModelQuery
	Raw(sql string, args ...any) RawQuery

	// Transaction control
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	// Savepoints
	Savepoint(ctx context.Context, name string) error
	RollbackTo(ctx context.Context, name string) error

	// Batch operations (uses schema field names)
	CreateMany(ctx context.Context, modelName string, data []any) (Result, error)
	UpdateMany(ctx context.Context, modelName string, condition Condition, data any) (Result, error)
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
	MapSchemaToColumnData(modelName string, data map[string]any) (map[string]any, error)
	MapColumnToSchemaData(modelName string, data map[string]any) (map[string]any, error)

	// Table name mapping
	ModelToTable(modelName string) (string, error)
}

// Migration types for backward compatibility
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
	Default       any
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

// DatabaseSpecificMigrator defines database-specific operations that each driver must implement
type DatabaseSpecificMigrator interface {
	// Database introspection
	GetTables() ([]string, error)
	GetTableInfo(tableName string) (*TableInfo, error)

	// SQL generation
	GenerateCreateTableSQL(s *schema.Schema) (string, error)
	GenerateDropTableSQL(tableName string) string
	GenerateAddColumnSQL(tableName string, field any) (string, error)
	GenerateModifyColumnSQL(change ColumnChange) ([]string, error)
	GenerateDropColumnSQL(tableName, columnName string) ([]string, error)
	GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string
	GenerateDropIndexSQL(indexName string) string

	// Migration execution
	ApplyMigration(sql string) error
	GetDatabaseType() string

	// Type mapping and column generation
	MapFieldType(field schema.Field) string
	FormatDefaultValue(value any) string
	GenerateColumnDefinitionFromColumnInfo(col ColumnInfo) string
	ConvertFieldToColumnInfo(field schema.Field) *ColumnInfo

	// Index management
	IsSystemIndex(indexName string) bool

	// Table management
	IsSystemTable(tableName string) bool
}

type DatabaseMigrator interface {
	// Introspection
	GetTables() ([]string, error)
	GetTableInfo(tableName string) (*TableInfo, error)

	// SQL Generation
	GenerateCreateTableSQL(schema any) (string, error)
	GenerateDropTableSQL(tableName string) string
	GenerateAddColumnSQL(tableName string, field any) (string, error)
	GenerateModifyColumnSQL(change ColumnChange) ([]string, error)
	GenerateDropColumnSQL(tableName, columnName string) ([]string, error)
	GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string
	GenerateDropIndexSQL(indexName string) string

	// Migration execution
	ApplyMigration(sql string) error
	GetDatabaseType() string

	// Schema comparison and migration planning
	CompareSchema(existingTable *TableInfo, desiredSchema any) (*MigrationPlan, error)
	GenerateMigrationSQL(plan *MigrationPlan) ([]string, error)
}
