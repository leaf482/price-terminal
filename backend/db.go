package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func openDatabase() (*sql.DB, error) {
	// Require an explicit database target instead of silently using OS-user defaults.
	for _, key := range []string{"PGHOST", "PGDATABASE", "PGUSER"} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			return nil, fmt.Errorf("missing required PostgreSQL configuration: %s", key)
		}
	}
	// An empty connection string uses the standard PG* environment variables.
	config, err := pgx.ParseConfig("")
	if err != nil {
		// Driver errors may contain credentials; do not return them to logs or HTTP.
		return nil, errors.New("invalid PostgreSQL configuration")
	}

	// Opening the pool is lazy so liveness remains available during DB outages.
	db := stdlib.OpenDB(*config)
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	return db, nil
}
