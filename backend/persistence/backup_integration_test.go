//go:build integration

package persistence_test

import (
	"context"
	"crypto/rand"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/leaf482/price-terminal/backend/persistence"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Explicit Docker opt-in keeps ordinary PostgreSQL integration tests portable.
func TestDockerBackupRestore(t *testing.T) {
	if os.Getenv("BACKUP_VERIFY") != "1" {
		t.Skip("set BACKUP_VERIFY=1 to verify pg_dump/pg_restore via local Docker")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	source := freshDatabase(t, ctx)
	s := seedReliability(t, ctx, source)
	if _, err := s.InvalidateObservation(ctx, "bad", "backup verification"); err != nil {
		t.Fatal(err)
	}
	var sourceName string
	if err := source.QueryRowContext(ctx, `SELECT current_database()`).Scan(&sourceName); err != nil {
		t.Fatal(err)
	}
	config, err := pgx.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	restoreName := "price_terminal_restore_" + strings.ToLower(rand.Text())
	quoted := pgx.Identifier{restoreName}.Sanitize()
	if _, err := source.ExecContext(ctx, "CREATE DATABASE "+quoted+" TEMPLATE template0"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := source.ExecContext(c, "DROP DATABASE "+quoted+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
	})
	dump := "/tmp/" + restoreName + ".dump"
	docker := func(c context.Context, args ...string) error {
		cmd := exec.CommandContext(c, "docker", append([]string{"compose", "-f", "../../docker-compose.yml", "exec", "-T", "postgres"}, args...)...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Docker verification command failed: %s", output)
		}
		return err
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := docker(c, "rm", "-f", dump); err != nil {
			t.Error("remove temporary dump", err)
		}
	})
	if err := docker(ctx, "pg_dump", "-U", config.User, "-d", sourceName, "-Fc", "-f", dump); err != nil {
		t.Fatal(err)
	}
	if err := docker(ctx, "pg_restore", "--exit-on-error", "--no-owner", "--no-acl", "-U", config.User, "-d", restoreName, dump); err != nil {
		t.Fatal(err)
	}
	config.Database = restoreName
	restored := stdlib.OpenDB(*config)
	t.Cleanup(func() { restored.Close() })
	// Compare all columns of every application table and Goose history, not just counts.
	for _, table := range []string{"products", "retailers", "listings", "price_observations", "promotions", "price_alerts", "price_alert_events", "observation_invalidations", "goose_db_version"} {
		query := `SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text),'[]'::jsonb)::text FROM ` + pgx.Identifier{table}.Sanitize() + ` t`
		var original, copy string
		if err := source.QueryRowContext(ctx, query).Scan(&original); err != nil {
			t.Fatal(err)
		}
		if err := restored.QueryRowContext(ctx, query).Scan(&copy); err != nil {
			t.Fatal(err)
		}
		if original != copy {
			t.Fatalf("restored %s differs", table)
		}
		t.Logf("verified all columns: %s", table)
	}
	a, err := s.CurrentListing(ctx, "l")
	if err != nil {
		t.Fatal(err)
	}
	b, err := persistence.New(restored).CurrentListing(ctx, "l")
	if err != nil || !reflect.DeepEqual(a, b) {
		t.Fatal("restored current query differs", err)
	}
	// Version metadata must also allow a safe no-op migrate-to-latest.
	cmd := exec.CommandContext(ctx, "goose", "-env", "none", "-dir", "../migrations", "postgres", "dbname="+restoreName+" connect_timeout=5", "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("restored migration check: %v %s", err, output)
	}
}
