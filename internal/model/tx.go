package model

import (
	"context"
	"database/sql"
)

// Tx abstracts the database transaction for testing purposes.
type Tx interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Commit() error
	Rollback() error
}