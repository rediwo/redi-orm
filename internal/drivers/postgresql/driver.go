package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/registry"
	"github.com/rediwo/redi-orm/schema"
	"github.com/rediwo/redi-orm/types"
	_ "github.com/lib/pq"
)

func init() {
	// Register PostgreSQL driver
	registry.Register(types.PostgreSQL, func(config types.Config) (types.Database, error) {
		return NewPostgreSQLDB(config)
	})
}

type PostgreSQLDB struct {
	db     *sql.DB
	config types.Config
}

func NewPostgreSQLDB(config types.Config) (*PostgreSQLDB, error) {
	return &PostgreSQLDB{config: config}, nil
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

func (p *PostgreSQLDB) CreateTable(sch interface{}) error {
	schema, ok := sch.(*schema.Schema)
	if !ok {
		return fmt.Errorf("expected *schema.Schema, got %T", sch)
	}
	
	var columns []string
	var primaryKeyCol string
	
	for _, field := range schema.Fields {
		col := fmt.Sprintf("\"%s\" %s", field.Name, fieldTypeToSQL(field.Type))

		if field.PrimaryKey {
			primaryKeyCol = field.Name
			if field.AutoIncrement {
				if field.Type == "int64" {
					col = fmt.Sprintf("\"%s\" BIGSERIAL", field.Name)
				} else {
					col = fmt.Sprintf("\"%s\" SERIAL", field.Name)
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

func (p *PostgreSQLDB) Insert(tableName string, data map[string]interface{}) (int64, error) {
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

func (p *PostgreSQLDB) FindByID(tableName string, id interface{}) (map[string]interface{}, error) {
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

func (p *PostgreSQLDB) Find(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
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

func (p *PostgreSQLDB) Update(tableName string, id interface{}, data map[string]interface{}) error {
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

func (p *PostgreSQLDB) Delete(tableName string, id interface{}) error {
	query := fmt.Sprintf("DELETE FROM \"%s\" WHERE id = $1", tableName)
	_, err := p.db.Exec(query, id)
	return err
}

func (p *PostgreSQLDB) Select(tableName string, columns []string) types.QueryBuilder {
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