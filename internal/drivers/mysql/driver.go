package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	_ "github.com/go-sql-driver/mysql"
)

func init() {
	// Register MySQL driver
	registry.Register(types.MySQL, func(config types.Config) (types.Database, error) {
		return NewMySQLDB(config)
	})
}

type MySQLDB struct {
	db     *sql.DB
	config types.Config
}

func NewMySQLDB(config types.Config) (*MySQLDB, error) {
	return &MySQLDB{config: config}, nil
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

func (m *MySQLDB) CreateTable(sch interface{}) error {
	schema, ok := sch.(*schema.Schema)
	if !ok {
		return fmt.Errorf("expected *schema.Schema, got %T", sch)
	}
	
	var columns []string
	for _, field := range schema.Fields {
		col := fmt.Sprintf("`%s` %s", field.Name, fieldTypeToSQL(field.Type))

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

func (m *MySQLDB) Insert(tableName string, data map[string]interface{}) (int64, error) {
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

func (m *MySQLDB) FindByID(tableName string, id interface{}) (map[string]interface{}, error) {
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

func (m *MySQLDB) Find(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
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

func (m *MySQLDB) Update(tableName string, id interface{}, data map[string]interface{}) error {
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

func (m *MySQLDB) Delete(tableName string, id interface{}) error {
	query := fmt.Sprintf("DELETE FROM `%s` WHERE id = ?", tableName)
	_, err := m.db.Exec(query, id)
	return err
}

func (m *MySQLDB) Select(tableName string, columns []string) types.QueryBuilder {
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
			result[col] = values[i]
		}
		results = append(results, result)
	}

	return results, rows.Err()
}