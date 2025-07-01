package postgresql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// PostgreSQLRawQuery implements the RawQuery interface for PostgreSQL
type PostgreSQLRawQuery struct {
	db   *sql.DB
	sql  string
	args []interface{}
}

// Exec executes the query and returns the result
func (q *PostgreSQLRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.db.ExecContext(ctx, q.sql, q.args...)
	if err != nil {
		return types.Result{}, fmt.Errorf("failed to execute query: %w", err)
	}

	// PostgreSQL doesn't support LastInsertId in the standard way
	// We need to use RETURNING clause for that
	lastInsertID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	return types.Result{
		LastInsertID: lastInsertID,
		RowsAffected: rowsAffected,
	}, nil
}

// Find executes the query and scans multiple rows into dest
func (q *PostgreSQLRawQuery) Find(ctx context.Context, dest interface{}) error {
	rows, err := q.db.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return utils.ScanRows(rows, dest)
}

// FindOne executes the query and scans a single row into dest
func (q *PostgreSQLRawQuery) FindOne(ctx context.Context, dest interface{}) error {
	return utils.ScanRowContext(q.db, ctx, q.sql, q.args, dest)
}