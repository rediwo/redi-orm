package drivers

import (
	"database/sql"
	"fmt"
	"strings"
)

type PostgreSQLTransaction struct {
	tx *sql.Tx
}

func (t *PostgreSQLTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *PostgreSQLTransaction) Rollback() error {
	return t.tx.Rollback()
}

func (t *PostgreSQLTransaction) Insert(tableName string, data map[string]interface{}) (int64, error) {
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
	err := t.tx.QueryRow(query, values...).Scan(&id)
	return id, err
}

func (t *PostgreSQLTransaction) Update(tableName string, id interface{}, data map[string]interface{}) error {
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

	_, err := t.tx.Exec(query, values...)
	return err
}

func (t *PostgreSQLTransaction) Delete(tableName string, id interface{}) error {
	query := fmt.Sprintf("DELETE FROM \"%s\" WHERE id = $1", tableName)
	_, err := t.tx.Exec(query, id)
	return err
}

func (t *PostgreSQLTransaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	return t.tx.Exec(query, args...)
}

func (t *PostgreSQLTransaction) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.Query(query, args...)
}

func (t *PostgreSQLTransaction) QueryRow(query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRow(query, args...)
}
