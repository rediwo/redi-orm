package drivers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
)

type MySQLQueryBuilder struct {
	db        *sql.DB
	tableName string
	columns   []string
	where     []string
	whereArgs []interface{}
	orderBy   []string
	limit     int
	offset    int
	err       error
}

func (qb *MySQLQueryBuilder) Where(column string, operator string, value interface{}) types.QueryBuilder {
	qb.where = append(qb.where, fmt.Sprintf("`%s` %s ?", column, operator))
	qb.whereArgs = append(qb.whereArgs, value)
	return qb
}

func (qb *MySQLQueryBuilder) WhereIn(column string, values []interface{}) types.QueryBuilder {
	placeholders := make([]string, len(values))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	qb.where = append(qb.where, fmt.Sprintf("`%s` IN (%s)", column, strings.Join(placeholders, ", ")))
	qb.whereArgs = append(qb.whereArgs, values...)
	return qb
}

func (qb *MySQLQueryBuilder) OrderBy(column string, direction string) types.QueryBuilder {
	qb.orderBy = append(qb.orderBy, fmt.Sprintf("`%s` %s", column, direction))
	return qb
}

func (qb *MySQLQueryBuilder) Limit(limit int) types.QueryBuilder {
	qb.limit = limit
	return qb
}

func (qb *MySQLQueryBuilder) Offset(offset int) types.QueryBuilder {
	qb.offset = offset
	return qb
}

func (qb *MySQLQueryBuilder) Execute() ([]map[string]interface{}, error) {
	var query strings.Builder

	if len(qb.columns) == 0 {
		query.WriteString(fmt.Sprintf("SELECT * FROM `%s`", qb.tableName))
	} else {
		quotedColumns := make([]string, len(qb.columns))
		for i, col := range qb.columns {
			quotedColumns[i] = fmt.Sprintf("`%s`", col)
		}
		query.WriteString(fmt.Sprintf("SELECT %s FROM `%s`", strings.Join(quotedColumns, ", "), qb.tableName))
	}

	if len(qb.where) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(qb.where, " AND "))
	}

	if len(qb.orderBy) > 0 {
		query.WriteString(" ORDER BY ")
		query.WriteString(strings.Join(qb.orderBy, ", "))
	}

	if qb.limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", qb.limit))
	}

	if qb.offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", qb.offset))
	}

	rows, err := qb.db.Query(query.String(), qb.whereArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return qb.scanRows(rows)
}

func (qb *MySQLQueryBuilder) First() (map[string]interface{}, error) {
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

func (qb *MySQLQueryBuilder) Count() (int64, error) {
	var query strings.Builder
	query.WriteString(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", qb.tableName))

	if len(qb.where) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(qb.where, " AND "))
	}

	var count int64
	err := qb.db.QueryRow(query.String(), qb.whereArgs...).Scan(&count)
	return count, err
}

func (qb *MySQLQueryBuilder) scanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
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
