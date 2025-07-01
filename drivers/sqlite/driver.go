package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func init() {
	// Register SQLite driver
	registry.Register("sqlite", func(config types.Config) (types.Database, error) {
		return NewSQLiteDB(config)
	})

	// Register SQLite URI parser
	registry.RegisterURIParser("sqlite", &SQLiteURIParser{})
}

type SQLiteDB struct {
	db       *sql.DB
	config   types.Config
	migrator *SQLiteMigrator
	schemas  map[string]interface{}
}

func NewSQLiteDB(config types.Config) (*SQLiteDB, error) {
	return &SQLiteDB{
		config:  config,
		schemas: make(map[string]interface{}),
	}, nil
}

func (s *SQLiteDB) Connect() error {
	db, err := sql.Open("sqlite3", s.config.FilePath)
	if err != nil {
		return err
	}
	s.db = db
	s.migrator = NewSQLiteMigrator(db)
	return nil
}

func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteDB) CreateTable(schema *schema.Schema) error {
	var columns []string
	for _, field := range schema.Fields {
		col := fmt.Sprintf("%s %s", field.GetColumnName(), fieldTypeToSQL(field.Type))

		if field.PrimaryKey {
			col += " PRIMARY KEY"
			if field.AutoIncrement {
				col += " AUTOINCREMENT"
			}
		}

		if !field.Nullable && !field.PrimaryKey {
			col += " NOT NULL"
		}

		if field.Unique {
			col += " UNIQUE"
		}

		if field.Default != nil {
			col += fmt.Sprintf(" DEFAULT %v", field.Default)
		}

		columns = append(columns, col)
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", schema.TableName, strings.Join(columns, ", "))
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteDB) DropTable(tableName string) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteDB) RawInsert(tableName string, data map[string]interface{}) (int64, error) {
	var columns []string
	var placeholders []string
	var values []interface{}

	i := 1
	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
		i++
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	result, err := s.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (s *SQLiteDB) RawFindByID(tableName string, id interface{}) (map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE id = ? LIMIT 1", tableName)
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := s.scanRows(rows)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}

	return results[0], nil
}

func (s *SQLiteDB) RawFind(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	var where []string
	var values []interface{}

	for col, val := range conditions {
		where = append(where, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}

	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}

	rows, err := s.db.Query(query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRows(rows)
}

func (s *SQLiteDB) RawUpdate(tableName string, id interface{}, data map[string]interface{}) error {
	var sets []string
	var values []interface{}

	for col, val := range data {
		sets = append(sets, fmt.Sprintf("%s = ?", col))
		values = append(values, val)
	}
	values = append(values, id)

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = ?",
		tableName,
		strings.Join(sets, ", "))

	_, err := s.db.Exec(query, values...)
	return err
}

func (s *SQLiteDB) RawDelete(tableName string, id interface{}) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)
	_, err := s.db.Exec(query, id)
	return err
}

func (s *SQLiteDB) RawSelect(tableName string, columns []string) types.QueryBuilder {
	return &SQLiteQueryBuilder{
		db:        s.db,
		tableName: tableName,
		columns:   columns,
	}
}

func (s *SQLiteDB) Begin() (types.Transaction, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	return &SQLiteTransaction{tx: tx}, nil
}

func (s *SQLiteDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return s.db.Exec(query, args...)
}

func (s *SQLiteDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}

func (s *SQLiteDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.db.QueryRow(query, args...)
}

// GetDB returns the underlying sql.DB instance
func (s *SQLiteDB) GetDB() *sql.DB {
	return s.db
}

// GetMigrator returns the database migrator for SQLite
func (s *SQLiteDB) GetMigrator() types.DatabaseMigrator {
	return s.migrator
}

// EnsureSchema performs auto-migration for all registered schemas
func (s *SQLiteDB) EnsureSchema() error {
	migrator := s.GetMigrator()
	if migrator == nil {
		return fmt.Errorf("migrator not available")
	}

	// Get existing tables
	existingTables, err := migrator.GetTables()
	if err != nil {
		return fmt.Errorf("failed to get existing tables: %w", err)
	}

	// Create a map for quick lookup
	existingTableMap := make(map[string]bool)
	for _, table := range existingTables {
		existingTableMap[table] = true
	}

	// Process each registered schema
	for _, schemaInterface := range s.schemas {
		schema, ok := schemaInterface.(*schema.Schema)
		if !ok {
			continue
		}

		tableName := schema.TableName
		
		if !existingTableMap[tableName] {
			// Table doesn't exist, create it
			if err := s.CreateTable(schema); err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
		} else {
			// Table exists, check for schema changes
			// For now, we'll just skip existing tables
			// In the future, we can implement column additions/modifications
			continue
		}
	}

	return nil
}

// RegisterSchema registers a schema for model name resolution
func (s *SQLiteDB) RegisterSchema(modelName string, schema interface{}) error {
	s.schemas[modelName] = schema
	return nil
}

// GetRegisteredSchemas returns all registered schemas
func (s *SQLiteDB) GetRegisteredSchemas() map[string]interface{} {
	result := make(map[string]interface{})
	for name, schema := range s.schemas {
		result[name] = schema
	}
	return result
}

// resolveTableName converts model name to actual table name using registered schema
func (s *SQLiteDB) resolveTableName(modelName string) (string, error) {
	schemaInterface, exists := s.schemas[modelName]
	if !exists {
		return "", fmt.Errorf("schema for model '%s' not registered", modelName)
	}

	schema, ok := schemaInterface.(*schema.Schema)
	if !ok {
		return "", fmt.Errorf("invalid schema type for model '%s'", modelName)
	}

	return schema.TableName, nil
}

// convertFieldNames converts schema field names to database column names
func (s *SQLiteDB) convertFieldNames(modelName string, data map[string]interface{}) (map[string]interface{}, error) {
	schemaInterface, exists := s.schemas[modelName]
	if !exists {
		// If schema not registered, return data as-is
		return data, nil
	}

	schema, ok := schemaInterface.(*schema.Schema)
	if !ok {
		return data, nil
	}

	// Create field name mapping
	fieldMap := make(map[string]string)
	for _, field := range schema.Fields {
		fieldMap[field.Name] = field.GetColumnName()
	}

	// Convert field names
	converted := make(map[string]interface{})
	for key, value := range data {
		if columnName, exists := fieldMap[key]; exists {
			converted[columnName] = value
		} else {
			// Keep unknown fields as-is (for raw queries)
			converted[key] = value
		}
	}

	return converted, nil
}

// convertResultFieldNames converts database column names back to schema field names
func (s *SQLiteDB) convertResultFieldNames(modelName string, data map[string]interface{}) map[string]interface{} {
	schemaInterface, exists := s.schemas[modelName]
	if !exists {
		// If schema not registered, return data as-is
		return data
	}

	schema, ok := schemaInterface.(*schema.Schema)
	if !ok {
		return data
	}

	// Create reverse field name mapping (column name -> field name)
	reverseFieldMap := make(map[string]string)
	for _, field := range schema.Fields {
		reverseFieldMap[field.GetColumnName()] = field.Name
	}

	// Convert column names back to field names
	converted := make(map[string]interface{})
	for key, value := range data {
		if fieldName, exists := reverseFieldMap[key]; exists {
			converted[fieldName] = value
		} else {
			// Keep unknown columns as-is
			converted[key] = value
		}
	}

	return converted
}

// Schema-aware CRUD operations
func (s *SQLiteDB) Insert(modelName string, data map[string]interface{}) (int64, error) {
	tableName, err := s.resolveTableName(modelName)
	if err != nil {
		return 0, err
	}

	convertedData, err := s.convertFieldNames(modelName, data)
	if err != nil {
		return 0, err
	}

	return s.RawInsert(tableName, convertedData)
}

func (s *SQLiteDB) FindByID(modelName string, id interface{}) (map[string]interface{}, error) {
	tableName, err := s.resolveTableName(modelName)
	if err != nil {
		return nil, err
	}

	result, err := s.RawFindByID(tableName, id)
	if err != nil {
		return nil, err
	}

	return s.convertResultFieldNames(modelName, result), nil
}

func (s *SQLiteDB) Find(modelName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	tableName, err := s.resolveTableName(modelName)
	if err != nil {
		return nil, err
	}

	convertedConditions, err := s.convertFieldNames(modelName, conditions)
	if err != nil {
		return nil, err
	}

	results, err := s.RawFind(tableName, convertedConditions, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert all results back to field names
	convertedResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		convertedResults[i] = s.convertResultFieldNames(modelName, result)
	}

	return convertedResults, nil
}

func (s *SQLiteDB) Update(modelName string, id interface{}, data map[string]interface{}) error {
	tableName, err := s.resolveTableName(modelName)
	if err != nil {
		return err
	}

	convertedData, err := s.convertFieldNames(modelName, data)
	if err != nil {
		return err
	}

	return s.RawUpdate(tableName, id, convertedData)
}

func (s *SQLiteDB) Delete(modelName string, id interface{}) error {
	tableName, err := s.resolveTableName(modelName)
	if err != nil {
		return err
	}
	return s.RawDelete(tableName, id)
}

func (s *SQLiteDB) Select(modelName string, columns []string) types.QueryBuilder {
	tableName, err := s.resolveTableName(modelName)
	if err != nil {
		// Return a no-op query builder that will return an error
		return &SQLiteQueryBuilder{
			db:        s.db,
			tableName: "",
			columns:   columns,
			err:       err,
		}
	}
	return s.RawSelect(tableName, columns)
}

func (s *SQLiteDB) scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		result := make(map[string]interface{})
		for i, col := range cols {
			result[col] = values[i]
		}
		results = append(results, result)
	}

	return results, rows.Err()
}
