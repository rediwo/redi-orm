package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	// Register SQLite driver
	registry.Register(types.SQLite, func(config types.Config) (types.Database, error) {
		return NewSQLiteDB(config)
	})
}

type SQLiteDB struct {
	db     *sql.DB
	config types.Config
}

func NewSQLiteDB(config types.Config) (*SQLiteDB, error) {
	return &SQLiteDB{config: config}, nil
}

func (s *SQLiteDB) Connect() error {
	db, err := sql.Open("sqlite3", s.config.FilePath)
	if err != nil {
		return err
	}
	s.db = db
	return nil
}

func (s *SQLiteDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteDB) CreateTable(sch interface{}) error {
	// Type assert to *schema.Schema
	schema, ok := sch.(*schema.Schema)
	if !ok {
		return fmt.Errorf("expected *schema.Schema, got %T", sch)
	}
	var columns []string
	for _, field := range schema.Fields {
		col := fmt.Sprintf("%s %s", field.Name, fieldTypeToSQL(field.Type))

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

func (s *SQLiteDB) Insert(tableName string, data map[string]interface{}) (int64, error) {
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

func (s *SQLiteDB) FindByID(tableName string, id interface{}) (map[string]interface{}, error) {
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

func (s *SQLiteDB) Find(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
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

func (s *SQLiteDB) Update(tableName string, id interface{}, data map[string]interface{}) error {
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

func (s *SQLiteDB) Delete(tableName string, id interface{}) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)
	_, err := s.db.Exec(query, id)
	return err
}

func (s *SQLiteDB) Select(tableName string, columns []string) types.QueryBuilder {
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
