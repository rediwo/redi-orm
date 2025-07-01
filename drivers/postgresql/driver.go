package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
)

func init() {
	// Register PostgreSQL driver
	registry.Register("postgresql", func(config types.Config) (types.Database, error) {
		return NewPostgreSQLDB(config)
	})

	// Register PostgreSQL URI parser
	registry.RegisterURIParser("postgresql", &PostgreSQLURIParser{})
}

type PostgreSQLDB struct {
	db      *sql.DB
	config  types.Config
	schemas map[string]interface{}
}

func NewPostgreSQLDB(config types.Config) (*PostgreSQLDB, error) {
	return &PostgreSQLDB{
		config:  config,
		schemas: make(map[string]interface{}),
	}, nil
}

func (p *PostgreSQLDB) Connect() error {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.config.Host,
		p.config.Port,
		p.config.User,
		p.config.Password,
		p.config.Database,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	p.db = db
	return nil
}

func (p *PostgreSQLDB) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *PostgreSQLDB) CreateTable(schema *schema.Schema) error {

	var columns []string
	var primaryKeyCol string

	for _, field := range schema.Fields {
		col := fmt.Sprintf("\"%s\" %s", field.GetColumnName(), fieldTypeToSQL(field.Type))

		if field.PrimaryKey {
			primaryKeyCol = field.GetColumnName()
			if field.AutoIncrement {
				if field.Type == "int64" {
					col = fmt.Sprintf("\"%s\" BIGSERIAL", field.GetColumnName())
				} else {
					col = fmt.Sprintf("\"%s\" SERIAL", field.GetColumnName())
				}
			}
		}

		if !field.Nullable && !field.PrimaryKey && !field.AutoIncrement {
			col += " NOT NULL"
		}

		if field.Unique && !field.PrimaryKey {
			col += " UNIQUE"
		}

		if field.Default != nil && !field.AutoIncrement {
			col += fmt.Sprintf(" DEFAULT %v", field.Default)
		}

		columns = append(columns, col)
	}

	if primaryKeyCol != "" {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (\"%s\")", primaryKeyCol))
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (%s)", schema.TableName, strings.Join(columns, ", "))
	_, err := p.db.Exec(query)
	return err
}

func (p *PostgreSQLDB) DropTable(tableName string) error {
	query := fmt.Sprintf("DROP TABLE IF EXISTS \"%s\"", tableName)
	_, err := p.db.Exec(query)
	return err
}

func (p *PostgreSQLDB) RawInsert(tableName string, data map[string]interface{}) (int64, error) {
	var columns []string
	var placeholders []string
	var values []interface{}

	i := 1
	for col, val := range data {
		columns = append(columns, fmt.Sprintf("\"%s\"", col))
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		values = append(values, val)
		i++
	}

	query := fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s) RETURNING id",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	var id int64
	err := p.db.QueryRow(query, values...).Scan(&id)
	return id, err
}

func (p *PostgreSQLDB) RawFindByID(tableName string, id interface{}) (map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM \"%s\" WHERE id = $1 LIMIT 1", tableName)
	rows, err := p.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := p.scanRows(rows)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}

	return results[0], nil
}

func (p *PostgreSQLDB) RawFind(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM \"%s\"", tableName)
	var where []string
	var values []interface{}

	i := 1
	for col, val := range conditions {
		where = append(where, fmt.Sprintf("\"%s\" = $%d", col, i))
		values = append(values, val)
		i++
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

	rows, err := p.db.Query(query, values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return p.scanRows(rows)
}

func (p *PostgreSQLDB) RawUpdate(tableName string, id interface{}, data map[string]interface{}) error {
	var sets []string
	var values []interface{}

	i := 1
	for col, val := range data {
		sets = append(sets, fmt.Sprintf("\"%s\" = $%d", col, i))
		values = append(values, val)
		i++
	}
	values = append(values, id)

	query := fmt.Sprintf("UPDATE \"%s\" SET %s WHERE id = $%d",
		tableName,
		strings.Join(sets, ", "),
		i)

	_, err := p.db.Exec(query, values...)
	return err
}

func (p *PostgreSQLDB) RawDelete(tableName string, id interface{}) error {
	query := fmt.Sprintf("DELETE FROM \"%s\" WHERE id = $1", tableName)
	_, err := p.db.Exec(query, id)
	return err
}

func (p *PostgreSQLDB) RawSelect(tableName string, columns []string) types.QueryBuilder {
	return &PostgreSQLQueryBuilder{
		db:        p.db,
		tableName: tableName,
		columns:   columns,
	}
}

func (p *PostgreSQLDB) Begin() (types.Transaction, error) {
	tx, err := p.db.Begin()
	if err != nil {
		return nil, err
	}
	return &PostgreSQLTransaction{tx: tx}, nil
}

func (p *PostgreSQLDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return p.db.Exec(query, args...)
}

func (p *PostgreSQLDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.Query(query, args...)
}

func (p *PostgreSQLDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return p.db.QueryRow(query, args...)
}

// GetMigrator returns nil for PostgreSQL (migration not implemented yet)
func (p *PostgreSQLDB) GetMigrator() types.DatabaseMigrator {
	return nil
}

// EnsureSchema performs auto-migration for all registered schemas
func (p *PostgreSQLDB) EnsureSchema() error {
	// For now, just create all tables that don't exist
	// TODO: Implement proper schema migration with PostgreSQL migrator
	
	// Get list of existing tables
	rows, err := p.db.Query("SELECT tablename FROM pg_tables WHERE schemaname = 'public'")
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
	for _, schemaInterface := range p.schemas {
		schema, ok := schemaInterface.(*schema.Schema)
		if !ok {
			continue
		}

		tableName := schema.TableName
		if !existingTables[tableName] {
			// Table doesn't exist, create it
			if err := p.CreateTable(schema); err != nil {
				return fmt.Errorf("failed to create table %s: %w", tableName, err)
			}
		}
	}

	return nil
}

// RegisterSchema registers a schema for model name resolution
func (p *PostgreSQLDB) RegisterSchema(modelName string, schema interface{}) error {
	p.schemas[modelName] = schema
	return nil
}

// GetRegisteredSchemas returns all registered schemas
func (p *PostgreSQLDB) GetRegisteredSchemas() map[string]interface{} {
	result := make(map[string]interface{})
	for name, schema := range p.schemas {
		result[name] = schema
	}
	return result
}

// resolveTableName converts model name to actual table name using registered schema
func (p *PostgreSQLDB) resolveTableName(modelName string) (string, error) {
	schemaInterface, exists := p.schemas[modelName]
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
func (p *PostgreSQLDB) convertFieldNames(modelName string, data map[string]interface{}) (map[string]interface{}, error) {
	schemaInterface, exists := p.schemas[modelName]
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
func (p *PostgreSQLDB) convertResultFieldNames(modelName string, data map[string]interface{}) map[string]interface{} {
	schemaInterface, exists := p.schemas[modelName]
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
func (p *PostgreSQLDB) Insert(modelName string, data map[string]interface{}) (int64, error) {
	tableName, err := p.resolveTableName(modelName)
	if err != nil {
		return 0, err
	}

	convertedData, err := p.convertFieldNames(modelName, data)
	if err != nil {
		return 0, err
	}

	return p.RawInsert(tableName, convertedData)
}

func (p *PostgreSQLDB) FindByID(modelName string, id interface{}) (map[string]interface{}, error) {
	tableName, err := p.resolveTableName(modelName)
	if err != nil {
		return nil, err
	}

	result, err := p.RawFindByID(tableName, id)
	if err != nil {
		return nil, err
	}

	return p.convertResultFieldNames(modelName, result), nil
}

func (p *PostgreSQLDB) Find(modelName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	tableName, err := p.resolveTableName(modelName)
	if err != nil {
		return nil, err
	}

	convertedConditions, err := p.convertFieldNames(modelName, conditions)
	if err != nil {
		return nil, err
	}

	results, err := p.RawFind(tableName, convertedConditions, limit, offset)
	if err != nil {
		return nil, err
	}

	// Convert all results back to field names
	convertedResults := make([]map[string]interface{}, len(results))
	for i, result := range results {
		convertedResults[i] = p.convertResultFieldNames(modelName, result)
	}

	return convertedResults, nil
}

func (p *PostgreSQLDB) Update(modelName string, id interface{}, data map[string]interface{}) error {
	tableName, err := p.resolveTableName(modelName)
	if err != nil {
		return err
	}

	convertedData, err := p.convertFieldNames(modelName, data)
	if err != nil {
		return err
	}

	return p.RawUpdate(tableName, id, convertedData)
}

func (p *PostgreSQLDB) Delete(modelName string, id interface{}) error {
	tableName, err := p.resolveTableName(modelName)
	if err != nil {
		return err
	}
	return p.RawDelete(tableName, id)
}

func (p *PostgreSQLDB) Select(modelName string, columns []string) types.QueryBuilder {
	tableName, err := p.resolveTableName(modelName)
	if err != nil {
		// Return a no-op query builder that will return an error
		return &PostgreSQLQueryBuilder{
			db:        p.db,
			tableName: "",
			columns:   columns,
			err:       err,
		}
	}
	return p.RawSelect(tableName, columns)
}

func (p *PostgreSQLDB) scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
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
