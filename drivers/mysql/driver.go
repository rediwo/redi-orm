package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func init() {
	// Register MySQL driver
	registry.Register("mysql", func(config types.Config) (types.Database, error) {
		return NewMySQLDB(config)
	})

	// Register MySQL URI parser
	registry.RegisterURIParser("mysql", &MySQLURIParser{})
}

type MySQLDB struct {
	db      *sql.DB
	config  types.Config
	schemas map[string]interface{}
}

func NewMySQLDB(config types.Config) (*MySQLDB, error) {
	return &MySQLDB{
		config:  config,
		schemas: make(map[string]interface{}),
	}, nil
}

func (m *MySQLDB) Connect() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		m.config.User,
		m.config.Password,
		m.config.Host,
		m.config.Port,
		m.config.Database,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	m.db = db
	return nil
}

func (m *MySQLDB) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *MySQLDB) CreateTable(schema *schema.Schema) error {

	var columns []string
	for _, field := range schema.Fields {
		col := fmt.Sprintf("`%s` %s", field.GetColumnName(), fieldTypeToSQL(field.Type))

		if field.PrimaryKey {
			col += " PRIMARY KEY"
			if field.AutoIncrement {
				col += " AUTO_INCREMENT"
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

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s)", schema.TableName, strings.Join(columns, ", "))
	_, err := m.db.Exec(query)
	return err
}

func (m *MySQLDB) DropTable(tableName string) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", tableName)
	_, err := m.db.Exec(query)
	return err
}

func (m *MySQLDB) RawInsert(tableName string, data map[string]interface{}) (int64, error) {
	var columns []string
	var placeholders []string
	var values []interface{}

	for col, val := range data {
		columns = append(columns, fmt.Sprintf("`%s`", col))
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	result, err := m.db.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (m *MySQLDB) RawFindByID(tableName string, id interface{}) (map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM `%s` WHERE id = ? LIMIT 1", tableName)
	rows, err := m.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := m.scanRows(rows)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}

	return results[0], nil
}

func (m *MySQLDB) RawFind(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM `%s`", tableName)
	var where []string
	var values []interface{}

	for col, val := range conditions {
		where = append(where, fmt.Sprintf("`%s` = ?", col))
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

	rows, err := m.db.Query(query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return m.scanRows(rows)
}

func (m *MySQLDB) RawUpdate(tableName string, id interface{}, data map[string]interface{}) error {
	var sets []string
	var values []interface{}

	for col, val := range data {
		sets = append(sets, fmt.Sprintf("`%s` = ?", col))
		values = append(values, val)
	}
	values = append(values, id)

	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE id = ?",
		tableName,
		strings.Join(sets, ", "))

	_, err := m.db.Exec(query, values...)
	return err
}

func (m *MySQLDB) RawDelete(tableName string, id interface{}) error {
	query := fmt.Sprintf("DELETE FROM `%s` WHERE id = ?", tableName)
	_, err := m.db.Exec(query, id)
	return err
}

func (m *MySQLDB) RawSelect(tableName string, columns []string) types.QueryBuilder {
	return &MySQLQueryBuilder{
		db:        m.db,
		tableName: tableName,
		columns:   columns,
	}
}

func (m *MySQLDB) Begin() (types.Transaction, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	return &MySQLTransaction{tx: tx}, nil
}

func (m *MySQLDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return m.db.Exec(query, args...)
}

func (m *MySQLDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return m.db.Query(query, args...)
}

func (m *MySQLDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return m.db.QueryRow(query, args...)
}

// GetMigrator returns nil for MySQL (migration not implemented yet)
func (m *MySQLDB) GetMigrator() types.DatabaseMigrator {
	return nil
}

// EnsureSchema performs auto-migration for all registered schemas
func (m *MySQLDB) EnsureSchema() error {
	// For now, just create all tables that don't exist
	// TODO: Implement proper schema migration with MySQL migrator
	
	// Get list of existing tables
	rows, err := m.db.Query("SHOW TABLES")
	if err != nil {
		return fmt.Errorf("failed to get existing tables: %w", err)
	}
	defer rows.Close()

	existingTables := make(map[string]bool)
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		existingTables[tableName] = true
	}

	// Process each registered schema
	for _, schemaInterface := range m.schemas {
		schema, ok := schemaInterface.(*schema.Schema)
		if !ok {
			continue
		}

		tableName := schema.TableName
		if !existingTables[tableName] {
			// Table doesn't exist, create it
			if err := m.CreateTable(schema); err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
		}
	}

	return nil
}

// RegisterSchema registers a schema for model name resolution
func (m *MySQLDB) RegisterSchema(modelName string, schema interface{}) error {
	m.schemas[modelName] = schema
	return nil
}

// GetRegisteredSchemas returns all registered schemas
func (m *MySQLDB) GetRegisteredSchemas() map[string]interface{} {
	result := make(map[string]interface{})
	for name, schema := range m.schemas {
		result[name] = schema
	}
	return result
}

// resolveTableName converts model name to actual table name using registered schema
func (m *MySQLDB) resolveTableName(modelName string) (string, error) {
	schemaInterface, exists := m.schemas[modelName]
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
func (m *MySQLDB) convertFieldNames(modelName string, data map[string]interface{}) (map[string]interface{}, error) {
	schemaInterface, exists := m.schemas[modelName]
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
func (m *MySQLDB) convertResultFieldNames(modelName string, data map[string]interface{}) map[string]interface{} {
	schemaInterface, exists := m.schemas[modelName]
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
func (m *MySQLDB) Insert(modelName string, data map[string]interface{}) (int64, error) {
	tableName, err := m.resolveTableName(modelName)
	if err != nil {
		return 0, err
	}

	convertedData, err := m.convertFieldNames(modelName, data)
	if err != nil {
		return 0, err
	}

	return m.RawInsert(tableName, convertedData)
}

func (m *MySQLDB) FindByID(modelName string, id interface{}) (map[string]interface{}, error) {
	tableName, err := m.resolveTableName(modelName)
	if err != nil {
		return nil, err
	}

	result, err := m.RawFindByID(tableName, id)
	if err != nil {
		return nil, err
	}

	return m.convertResultFieldNames(modelName, result), nil
}

func (m *MySQLDB) Find(modelName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	tableName, err := m.resolveTableName(modelName)
	if err != nil {
		return nil, err
	}

	convertedConditions, err := m.convertFieldNames(modelName, conditions)
	if err != nil {
		return nil, err
	}

	results, err := m.RawFind(tableName, convertedConditions, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert all results back to field names
	convertedResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		convertedResults[i] = m.convertResultFieldNames(modelName, result)
	}

	return convertedResults, nil
}

func (m *MySQLDB) Update(modelName string, id interface{}, data map[string]interface{}) error {
	tableName, err := m.resolveTableName(modelName)
	if err != nil {
		return err
	}

	convertedData, err := m.convertFieldNames(modelName, data)
	if err != nil {
		return err
	}

	return m.RawUpdate(tableName, id, convertedData)
}

func (m *MySQLDB) Delete(modelName string, id interface{}) error {
	tableName, err := m.resolveTableName(modelName)
	if err != nil {
		return err
	}
	return m.RawDelete(tableName, id)
}

func (m *MySQLDB) Select(modelName string, columns []string) types.QueryBuilder {
	tableName, err := m.resolveTableName(modelName)
	if err != nil {
		// Return a no-op query builder that will return an error
		return &MySQLQueryBuilder{
			db:        m.db,
			tableName: "",
			columns:   columns,
			err:       err,
		}
	}
	return m.RawSelect(tableName, columns)
}

func (m *MySQLDB) scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
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
			// Convert MySQL byte slices to strings for better usability
			if b, ok := values[i].([]byte); ok {
				result[col] = string(b)
			} else {
				result[col] = values[i]
			}
		}
		results = append(results, result)
	}

	return results, rows.Err()
}
