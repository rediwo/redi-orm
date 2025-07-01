package mysql

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rediwo/redi-orm/query"
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func init() {
	// Register MySQL driver
	registry.Register("mysql", func(config types.Config) (types.Database, error) {
		return NewMySQLDB(config)
	})
}

// MySQLDB implements the Database interface for MySQL
type MySQLDB struct {
	db          *sql.DB
	config      types.Config
	fieldMapper types.FieldMapper
	schemas     map[string]*schema.Schema
}

// NewMySQLDB creates a new MySQL database instance
func NewMySQLDB(config types.Config) (*MySQLDB, error) {
	return &MySQLDB{
		config:      config,
		fieldMapper: types.NewDefaultFieldMapper(),
		schemas:     make(map[string]*schema.Schema),
	}, nil
}

// Connect establishes connection to MySQL database
func (m *MySQLDB) Connect(ctx context.Context) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		m.config.User,
		m.config.Password,
		m.config.Host,
		m.config.Port,
		m.config.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open MySQL database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	m.db = db
	return nil
}

// Close closes the database connection
func (m *MySQLDB) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (m *MySQLDB) Ping(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}
	return m.db.PingContext(ctx)
}

// RegisterSchema registers a schema with the database
func (m *MySQLDB) RegisterSchema(modelName string, schema *schema.Schema) error {
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if err := schema.Validate(); err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	m.schemas[modelName] = schema

	// Register with field mapper
	if mapper, ok := m.fieldMapper.(*types.DefaultFieldMapper); ok {
		mapper.RegisterSchema(modelName, schema)
	}

	return nil
}

// GetSchema returns a registered schema
func (m *MySQLDB) GetSchema(modelName string) (*schema.Schema, error) {
	schema, exists := m.schemas[modelName]
	if !exists {
		return nil, fmt.Errorf("schema for model '%s' not registered", modelName)
	}
	return schema, nil
}

// GetModels returns all registered model names
func (m *MySQLDB) GetModels() []string {
	models := make([]string, 0, len(m.schemas))
	for modelName := range m.schemas {
		models = append(models, modelName)
	}
	return models
}

// GetModelSchema returns schema for a model (alias for GetSchema)
func (m *MySQLDB) GetModelSchema(modelName string) (*schema.Schema, error) {
	return m.GetSchema(modelName)
}

// ResolveTableName resolves model name to table name
func (m *MySQLDB) ResolveTableName(modelName string) (string, error) {
	return m.fieldMapper.ModelToTable(modelName)
}

// ResolveFieldName resolves schema field name to column name
func (m *MySQLDB) ResolveFieldName(modelName, fieldName string) (string, error) {
	return m.fieldMapper.SchemaToColumn(modelName, fieldName)
}

// ResolveFieldNames resolves multiple schema field names to column names
func (m *MySQLDB) ResolveFieldNames(modelName string, fieldNames []string) ([]string, error) {
	return m.fieldMapper.SchemaFieldsToColumns(modelName, fieldNames)
}

// GetFieldMapper returns the field mapper
func (m *MySQLDB) GetFieldMapper() types.FieldMapper {
	return m.fieldMapper
}

// CreateModel creates a table for the given model
func (m *MySQLDB) CreateModel(ctx context.Context, modelName string) error {
	schema, err := m.GetSchema(modelName)
	if err != nil {
		return fmt.Errorf("failed to get schema for model %s: %w", modelName, err)
	}

	sql, err := m.generateCreateTableSQL(schema)
	if err != nil {
		return fmt.Errorf("failed to generate CREATE TABLE SQL: %w", err)
	}

	_, err = m.db.ExecContext(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// DropModel drops the table for the given model
func (m *MySQLDB) DropModel(ctx context.Context, modelName string) error {
	tableName, err := m.ResolveTableName(modelName)
	if err != nil {
		return fmt.Errorf("failed to resolve table name: %w", err)
	}

	sql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)
	_, err = m.db.ExecContext(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to drop table: %w", err)
	}

	return nil
}

// Model creates a new model query
func (m *MySQLDB) Model(modelName string) types.ModelQuery {
	return query.NewModelQuery(modelName, m, m.GetFieldMapper())
}

// Raw creates a new raw query
func (m *MySQLDB) Raw(sql string, args ...interface{}) types.RawQuery {
	return NewMySQLRawQuery(m.db, sql, args...)
}

// Begin starts a new transaction
func (m *MySQLDB) Begin(ctx context.Context) (types.Transaction, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return NewMySQLTransaction(tx, m), nil
}

// Transaction executes a function within a transaction
func (m *MySQLDB) Transaction(ctx context.Context, fn func(tx types.Transaction) error) error {
	tx, err := m.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %w", err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Exec executes a raw SQL statement
func (m *MySQLDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return m.db.Exec(query, args...)
}

// Query executes a raw SQL query that returns rows
func (m *MySQLDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return m.db.Query(query, args...)
}

// QueryRow executes a raw SQL query that returns a single row
func (m *MySQLDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return m.db.QueryRow(query, args...)
}

// generateCreateTableSQL generates CREATE TABLE SQL for MySQL
func (m *MySQLDB) generateCreateTableSQL(schema *schema.Schema) (string, error) {
	// Simplified implementation - would need full MySQL-specific logic
	return "", fmt.Errorf("MySQL CREATE TABLE generation not yet implemented")
}

// MySQLRawQuery implements RawQuery for MySQL (simplified implementation)
type MySQLRawQuery struct {
	db   *sql.DB
	sql  string
	args []interface{}
}

func NewMySQLRawQuery(db *sql.DB, sql string, args ...interface{}) types.RawQuery {
	return &MySQLRawQuery{
		db:   db,
		sql:  sql,
		args: args,
	}
}

func (q *MySQLRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.db.ExecContext(ctx, q.sql, q.args...)
	if err != nil {
		return types.Result{}, err
	}

	lastInsertID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	return types.Result{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
	}, nil
}

func (q *MySQLRawQuery) Find(ctx context.Context, dest interface{}) error {
	return fmt.Errorf("MySQL raw query result scanning not yet implemented")
}

func (q *MySQLRawQuery) FindOne(ctx context.Context, dest interface{}) error {
	return fmt.Errorf("MySQL raw query result scanning not yet implemented")
}

// MySQLTransaction implements Transaction for MySQL (simplified implementation)
type MySQLTransaction struct {
	tx *sql.Tx
	db *MySQLDB
}

func NewMySQLTransaction(tx *sql.Tx, database *MySQLDB) types.Transaction {
	return &MySQLTransaction{
		tx: tx,
		db: database,
	}
}

func (t *MySQLTransaction) Model(modelName string) types.ModelQuery {
	// For now, return a placeholder
	return nil
}

func (t *MySQLTransaction) Raw(sql string, args ...interface{}) types.RawQuery {
	return &MySQLTransactionRawQuery{tx: t.tx, sql: sql, args: args}
}

func (t *MySQLTransaction) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

func (t *MySQLTransaction) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}

func (t *MySQLTransaction) Savepoint(ctx context.Context, name string) error {
	return fmt.Errorf("savepoints not yet implemented for MySQL")
}

func (t *MySQLTransaction) RollbackTo(ctx context.Context, name string) error {
	return fmt.Errorf("savepoints not yet implemented for MySQL")
}

func (t *MySQLTransaction) CreateMany(ctx context.Context, modelName string, data []interface{}) (types.Result, error) {
	return types.Result{}, fmt.Errorf("batch operations not yet implemented for MySQL")
}

func (t *MySQLTransaction) UpdateMany(ctx context.Context, modelName string, condition types.Condition, data interface{}) (types.Result, error) {
	return types.Result{}, fmt.Errorf("batch operations not yet implemented for MySQL")
}

func (t *MySQLTransaction) DeleteMany(ctx context.Context, modelName string, condition types.Condition) (types.Result, error) {
	return types.Result{}, fmt.Errorf("batch operations not yet implemented for MySQL")
}

// MySQLTransactionRawQuery implements RawQuery for MySQL transactions
type MySQLTransactionRawQuery struct {
	tx   *sql.Tx
	sql  string
	args []interface{}
}

func (q *MySQLTransactionRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.tx.ExecContext(ctx, q.sql, q.args...)
	if err != nil {
		return types.Result{}, err
	}

	lastInsertID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	return types.Result{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
	}, nil
}

func (q *MySQLTransactionRawQuery) Find(ctx context.Context, dest interface{}) error {
	return fmt.Errorf("MySQL transaction raw query result scanning not yet implemented")
}

func (q *MySQLTransactionRawQuery) FindOne(ctx context.Context, dest interface{}) error {
	return fmt.Errorf("MySQL transaction raw query result scanning not yet implemented")
}

// GetMigrator returns a migrator for MySQL (placeholder implementation)
func (m *MySQLDB) GetMigrator() types.DatabaseMigrator {
	// Return a placeholder migrator for now
	return &MySQLMigrator{db: m.db}
}

// MySQLMigrator implements DatabaseMigrator for MySQL (placeholder implementation)
type MySQLMigrator struct {
	db *sql.DB
}

// NewMySQLMigrator creates a new MySQL migrator
func NewMySQLMigrator(db *sql.DB) *MySQLMigrator {
	return &MySQLMigrator{db: db}
}

// GetTables returns all table names
func (m *MySQLMigrator) GetTables() ([]string, error) {
	return nil, fmt.Errorf("GetTables not yet implemented")
}

// GetTableInfo returns table information
func (m *MySQLMigrator) GetTableInfo(tableName string) (*types.TableInfo, error) {
	return nil, fmt.Errorf("GetTableInfo not yet implemented")
}

// GenerateCreateTableSQL generates CREATE TABLE SQL
func (m *MySQLMigrator) GenerateCreateTableSQL(schema interface{}) (string, error) {
	return "", fmt.Errorf("GenerateCreateTableSQL not yet implemented")
}

// GenerateDropTableSQL generates DROP TABLE SQL
func (m *MySQLMigrator) GenerateDropTableSQL(tableName string) string {
	return fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)
}

// GenerateAddColumnSQL generates ADD COLUMN SQL
func (m *MySQLMigrator) GenerateAddColumnSQL(tableName string, field interface{}) (string, error) {
	return "", fmt.Errorf("GenerateAddColumnSQL not yet implemented")
}

// GenerateModifyColumnSQL generates MODIFY COLUMN SQL
func (m *MySQLMigrator) GenerateModifyColumnSQL(change types.ColumnChange) ([]string, error) {
	return nil, fmt.Errorf("GenerateModifyColumnSQL not yet implemented")
}

// GenerateDropColumnSQL generates DROP COLUMN SQL
func (m *MySQLMigrator) GenerateDropColumnSQL(tableName, columnName string) ([]string, error) {
	return nil, fmt.Errorf("GenerateDropColumnSQL not yet implemented")
}

// GenerateCreateIndexSQL generates CREATE INDEX SQL
func (m *MySQLMigrator) GenerateCreateIndexSQL(tableName, indexName string, columns []string, unique bool) string {
	return fmt.Sprintf("CREATE INDEX %s ON `%s` (%s)", indexName, tableName, "column_placeholder")
}

// GenerateDropIndexSQL generates DROP INDEX SQL
func (m *MySQLMigrator) GenerateDropIndexSQL(indexName string) string {
	return fmt.Sprintf("DROP INDEX %s", indexName)
}

// ApplyMigration executes a migration SQL
func (m *MySQLMigrator) ApplyMigration(sql string) error {
	_, err := m.db.Exec(sql)
	return err
}

// GetDatabaseType returns the database type
func (m *MySQLMigrator) GetDatabaseType() string {
	return "mysql"
}

// CompareSchema compares existing table with desired schema
func (m *MySQLMigrator) CompareSchema(existingTable *types.TableInfo, desiredSchema interface{}) (*types.MigrationPlan, error) {
	return nil, fmt.Errorf("CompareSchema not yet implemented")
}

// GenerateMigrationSQL generates migration SQL
func (m *MySQLMigrator) GenerateMigrationSQL(plan *types.MigrationPlan) ([]string, error) {
	return nil, fmt.Errorf("GenerateMigrationSQL not yet implemented")
}
