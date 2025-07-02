package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// MySQLRawQuery implements types.RawQuery for MySQL
type MySQLRawQuery struct {
	db   *sql.DB
	sql  string
	args []any
}

// NewMySQLRawQuery creates a new MySQL raw query
func NewMySQLRawQuery(db *sql.DB, sql string, args ...any) types.RawQuery {
	return &MySQLRawQuery{
		db:   db,
		sql:  sql,
		args: args,
	}
}

// Exec executes the query and returns the result
func (q *MySQLRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.db.ExecContext(ctx, q.sql, q.args...)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute query: %w", err)
	}

	lastInsertID, err := result.LastInsertId()
	if err != nil {
		// MySQL should support LastInsertId, but handle error gracefully
		lastInsertID = 0
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		rowsAffected = 0
	}

	return types.Result{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
	}, nil
}

// Find executes the query and scans results into dest
func (q *MySQLRawQuery) Find(ctx context.Context, dest any) error {
	rows, err := q.db.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return utils.ScanRows(rows, dest)
}

// FindOne executes the query and scans a single result into dest
func (q *MySQLRawQuery) FindOne(ctx context.Context, dest any) error {
	return utils.ScanRowContext(q.db, ctx, q.sql, q.args, dest)
}
