package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rediwo/redi-orm/types"
)

// SQLiteRawQuery implements the RawQuery interface for SQLite
type SQLiteRawQuery struct {
	db   *sql.DB
	sql  string
	args []interface{}
}

// NewSQLiteRawQuery creates a new SQLite raw query
func NewSQLiteRawQuery(db *sql.DB, sql string, args ...interface{}) *SQLiteRawQuery {
	return &SQLiteRawQuery{
		db:   db,
		sql:  sql,
		args: args,
	}
}

// Exec executes the raw query and returns the result
func (q *SQLiteRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.db.ExecContext(ctx, q.sql, q.args...)
	if err != nil {
		return types.Result{}, err
	}

	lastInsertID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	return types.Result{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
	}, nil
}

// Find executes the raw query and returns multiple results
func (q *SQLiteRawQuery) Find(ctx context.Context, dest interface{}) error {
	rows, err := q.db.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// This is a simplified implementation
	// In a real implementation, you'd want to properly scan the rows into dest
	// For now, we'll return an error indicating it needs implementation
	return fmt.Errorf("SQLite raw query result scanning not yet implemented")
}

// FindOne executes the raw query and returns a single result
func (q *SQLiteRawQuery) FindOne(ctx context.Context, dest interface{}) error {
	row := q.db.QueryRowContext(ctx, q.sql, q.args...)

	// This is a simplified implementation
	// In a real implementation, you'd want to properly scan the row into dest
	// For now, we'll return an error indicating it needs implementation
	_ = row
	return fmt.Errorf("SQLite raw query result scanning not yet implemented")
}
