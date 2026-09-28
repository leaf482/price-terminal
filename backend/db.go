package main

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func openDatabase() (*sql.DB, error) {
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
