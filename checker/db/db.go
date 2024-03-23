package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/catalystgo/healthcheck"
)

// DoSimpleSelect this is a check that is performed using
// a simple SELECT query to the specified DB
func DoSimpleSelect(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "SELECT 1 as healthcheck;")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var f int
		err = rows.Scan(&f)
		if err != nil {
			return err
		}
	}

	return rows.Err()
}

// DatabasePingCheck returns a Check that checks the connection to
// database/sql.DB using Ping().
func DatabasePingCheck(database *sql.DB, timeout time.Duration) healthcheck.Check {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if database == nil {
			return fmt.Errorf("database is nil")
		}
		return database.PingContext(ctx)
	}
}
