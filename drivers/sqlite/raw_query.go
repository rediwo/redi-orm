package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
	// For INSERT with RETURNING, we need to handle the case where the INSERT fails
	// but the error is masked by "no rows in result set"
	err := utils.ScanRowContext(q.db, ctx, q.sql, q.args, dest)
	if err != nil {
		// If this is an INSERT ... RETURNING query that failed with "no rows in result set",
		// try to execute it without scanning to get the actual error
		if err.Error() == "sql: no rows in result set" && strings.Contains(strings.ToUpper(q.sql), "INSERT") && strings.Contains(strings.ToUpper(q.sql), "RETURNING") {
			// Execute the query to get the actual error
			_, execErr := q.db.ExecContext(ctx, q.sql, q.args...)
			if execErr != nil {
				return execErr // Return the actual error (e.g., unique constraint violation)
			}
		}
	}
	return err
}
