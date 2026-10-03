package main

import (
	"strings"
	"testing"
)

func TestDatabaseConfiguration(t *testing.T) {
	for _, key := range []string{"PGHOST", "PGDATABASE", "PGUSER"} {
		t.Run(key, func(t *testing.T) {
			for _, k := range []string{"PGHOST", "PGDATABASE", "PGUSER"} {
				t.Setenv(k, "example")
			}
			t.Setenv(key, "")
			db, err := openDatabase()
			if db != nil || err == nil || !strings.Contains(err.Error(), key) {
				t.Fatal(db, err)
			}
		})
	}
	t.Setenv("PGHOST", "127.0.0.1")
	t.Setenv("PGDATABASE", "example")
	t.Setenv("PGUSER", "example")
	t.Setenv("PGPASSWORD", "secret-test-value")
	t.Setenv("PGPORT", "invalid")
	if _, err := openDatabase(); err == nil || err.Error() != "invalid PostgreSQL configuration" {
		t.Fatal(err)
	}
	t.Setenv("PGPORT", "5432")
	t.Setenv("PGSSLMODE", "disable")
	db, err := openDatabase()
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
}
