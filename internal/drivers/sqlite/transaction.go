package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

type SQLiteTransaction struct {
	tx *sql.Tx
}

func (t *SQLiteTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *SQLiteTransaction) Rollback() error {
	return t.tx.Rollback()
}

func (t *SQLiteTransaction) Insert(tableName string, data map[string]interface{}) (int64, error) {
	var columns []string
	var placeholders []string
	var values []interface{}

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	result, err := t.tx.Exec(query, values...)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (t *SQLiteTransaction) Update(tableName string, id interface{}, data map[string]interface{}) error {
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

	_, err := t.tx.Exec(query, values...)
	return err
}

func (t *SQLiteTransaction) Delete(tableName string, id interface{}) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", tableName)
	_, err := t.tx.Exec(query, id)
	return err
}

var _ types.Transaction = (*SQLiteTransaction)(nil)
