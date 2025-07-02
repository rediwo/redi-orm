package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// SQLiteRawQuery implements the RawQuery interface for SQLite
type SQLiteRawQuery struct {
	db   *sql.DB
	sql  string
	args []any
}

// NewSQLiteRawQuery creates a new SQLite raw query
func NewSQLiteRawQuery(db *sql.DB, sql string, args ...any) types.RawQuery {
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
func (q *SQLiteRawQuery) Find(ctx context.Context, dest any) error {
	rows, err := q.db.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return utils.ScanRows(rows, dest)
}

// FindOne executes the raw query and returns a single result
func (q *SQLiteRawQuery) FindOne(ctx context.Context, dest any) error {
	return utils.ScanRowContext(q.db, ctx, q.sql, q.args, dest)
}
