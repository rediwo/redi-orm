package query

import (
	"context"
	"fmt"

	"github.com/rediwo/redi-orm/types"
)

// RawQueryImpl implements the RawQuery interface
type RawQueryImpl struct {
	database types.Database
	sql      string
	args     []interface{}
}

// NewRawQuery creates a new raw query
func NewRawQuery(database types.Database, sql string, args ...interface{}) *RawQueryImpl {
	return &RawQueryImpl{
		database: database,
		sql:      sql,
		args:     args,
	}
}

// Exec executes the raw query and returns the result
func (q *RawQueryImpl) Exec(ctx context.Context) (types.Result, error) {
	sqlResult, err := q.database.Exec(q.sql, q.args...)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute raw query: %w", err)
	}

	lastInsertID, _ := sqlResult.LastInsertId()
	rowsAffected, _ := sqlResult.RowsAffected()

	return types.Result{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
	}, nil
}

// Find executes the raw query and returns multiple results
func (q *RawQueryImpl) Find(ctx context.Context, dest interface{}) error {
	rows, err := q.database.Query(q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute raw query: %w", err)
	}
	defer rows.Close()

	// This is a simplified implementation
	// In a real implementation, you'd want to properly scan the rows into dest
	// For now, we'll return an error indicating it needs implementation
	return fmt.Errorf("raw query result scanning not yet implemented")
}

// FindOne executes the raw query and returns a single result
func (q *RawQueryImpl) FindOne(ctx context.Context, dest interface{}) error {
	_ = q.database.QueryRow(q.sql, q.args...)

	// This is a simplified implementation
	// In a real implementation, you'd want to properly scan the row into dest
	// For now, we'll return an error indicating it needs implementation
	return fmt.Errorf("raw query result scanning not yet implemented")
}

// GetSQL returns the SQL and arguments
func (q *RawQueryImpl) GetSQL() (string, []interface{}) {
	return q.sql, q.args
}
