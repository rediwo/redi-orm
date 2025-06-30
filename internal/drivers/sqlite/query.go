package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

type SQLiteQueryBuilder struct {
	db         *sql.DB
	tableName  string
	columns    []string
	conditions []string
	values     []interface{}
	orderBy    string
	limit      int
	offset     int
}

func (qb *SQLiteQueryBuilder) Where(field string, operator string, value interface{}) types.QueryBuilder {
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s %s ?", field, operator))
	qb.values = append(qb.values, value)
	return qb
}

func (qb *SQLiteQueryBuilder) WhereIn(field string, values []interface{}) types.QueryBuilder {
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
		qb.values = append(qb.values, values[i])
	}
	qb.conditions = append(qb.conditions, fmt.Sprintf("%s IN (%s)", field, strings.Join(placeholders, ", ")))
	return qb
}

func (qb *SQLiteQueryBuilder) OrderBy(field string, direction string) types.QueryBuilder {
	qb.orderBy = fmt.Sprintf("%s %s", field, direction)
	return qb
}

func (qb *SQLiteQueryBuilder) Limit(limit int) types.QueryBuilder {
	qb.limit = limit
	return qb
}

func (qb *SQLiteQueryBuilder) Offset(offset int) types.QueryBuilder {
	qb.offset = offset
	return qb
}

func (qb *SQLiteQueryBuilder) Execute() ([]map[string]interface{}, error) {
	query := qb.buildQuery()
	rows, err := qb.db.Query(query, qb.values...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

func (qb *SQLiteQueryBuilder) First() (map[string]interface{}, error) {
	qb.limit = 1
	results, err := qb.Execute()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}
	return results[0], nil
}

func (qb *SQLiteQueryBuilder) Count() (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", qb.tableName)

	if len(qb.conditions) > 0 {
		query += " WHERE " + strings.Join(qb.conditions, " AND ")
	}

	var count int64
	err := qb.db.QueryRow(query, qb.values...).Scan(&count)
	return count, err
}

func (qb *SQLiteQueryBuilder) buildQuery() string {
	cols := "*"
	if len(qb.columns) > 0 {
		cols = strings.Join(qb.columns, ", ")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", cols, qb.tableName)

	if len(qb.conditions) > 0 {
		query += " WHERE " + strings.Join(qb.conditions, " AND ")
	}

	if qb.orderBy != "" {
		query += " ORDER BY " + qb.orderBy
	}

	if qb.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limit)
	}

	if qb.offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.offset)
	}

	return query
}

func scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
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
