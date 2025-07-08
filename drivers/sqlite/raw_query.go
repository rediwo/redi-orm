package sqlite

import (
	"context"
	"fmt"
	"strings"

	"github.com/rediwo/redi-orm/types"
	"github.com/rediwo/redi-orm/utils"
)

// SQLiteRawQuery implements the RawQuery interface for SQLite
type SQLiteRawQuery struct {
	driver *SQLiteDB
	sql    string
	args   []any
}

// NewSQLiteRawQuery creates a new SQLite raw query
func NewSQLiteRawQuery(driver *SQLiteDB, sql string, args ...any) types.RawQuery {
	return &SQLiteRawQuery{
		driver: driver,
		sql:    sql,
		args:   args,
	}
}

// Exec executes the raw query and returns the result
func (q *SQLiteRawQuery) Exec(ctx context.Context) (types.Result, error) {
	result, err := q.driver.Exec(q.sql, q.args...)
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
	rows, err := q.driver.Query(q.sql, q.args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return utils.ScanRows(rows, dest)
}

// FindOne executes the raw query and returns a single result
func (q *SQLiteRawQuery) FindOne(ctx context.Context, dest any) error {
	// Special handling for INSERT...RETURNING to catch constraint violations
	upperSQL := strings.ToUpper(strings.TrimSpace(q.sql))
	if strings.HasPrefix(upperSQL, "INSERT") && strings.Contains(upperSQL, "RETURNING") {
		// First try to execute the query
		rows, err := q.driver.Query(q.sql, q.args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		// Use ScanRow which handles rows.Next() internally
		err = utils.ScanRow(rows, dest)
		if err != nil && err.Error() == "sql: no rows in result set" {
			// No rows returned - this could be due to a constraint violation
			// Try to execute without RETURNING to get the actual error
			nonReturningSQL := q.sql
			if idx := strings.LastIndex(strings.ToUpper(q.sql), "RETURNING"); idx != -1 {
				nonReturningSQL = strings.TrimSpace(q.sql[:idx])
			}

			_, execErr := q.driver.Exec(nonReturningSQL, q.args...)
			if execErr != nil {
				return execErr // Return the actual constraint error
			}
			return err // Return the original "no rows" error
		}
		return err
	}

	// For non-INSERT...RETURNING queries, use the standard method
	return utils.ScanRowContext(q.driver.DB, ctx, q.sql, q.args, dest)
}
