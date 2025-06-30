package models

import (
	"github.com/rediwo/redi-orm/types"
)

type QueryBuilder struct {
	db        types.Database
	tableName string
	columns   []string
	qb        types.QueryBuilder
}

func (q *QueryBuilder) Where(field string, operator string, value interface{}) *QueryBuilder {
	q.qb.Where(field, operator, value)
	return q
}

func (q *QueryBuilder) WhereIn(field string, values []interface{}) *QueryBuilder {
	q.qb.WhereIn(field, values)
	return q
}

func (q *QueryBuilder) OrderBy(field string, direction string) *QueryBuilder {
	q.qb.OrderBy(field, direction)
	return q
}

func (q *QueryBuilder) Limit(limit int) *QueryBuilder {
	q.qb.Limit(limit)
	return q
}

func (q *QueryBuilder) Offset(offset int) *QueryBuilder {
	q.qb.Offset(offset)
	return q
}

func (q *QueryBuilder) Execute() ([]map[string]interface{}, error) {
	return q.qb.Execute()
}

func (q *QueryBuilder) First() (map[string]interface{}, error) {
	return q.qb.First()
}

func (q *QueryBuilder) Count() (int64, error) {
	return q.qb.Count()
}
