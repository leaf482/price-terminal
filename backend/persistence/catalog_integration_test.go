//go:build integration

package persistence_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

// Only a newly created random database is migrated and dropped. The configured
// database is used solely as the administrative connection for CREATE/DROP.
func freshDatabase(t *testing.T, ctx context.Context) *sql.DB {
	t.Helper()
	goose, err := exec.LookPath("goose")
	if err != nil {
		t.Fatal("integration tests require Goose v3.28.0 on PATH")
	}
	config, err := pgx.ParseConfig("")
	if err != nil {
		t.Fatal("invalid PostgreSQL environment configuration")
	}
	admin := stdlib.OpenDB(*config)
	t.Cleanup(func() { admin.Close() })
	name := "price_terminal_test_" + strings.ToLower(rand.Text())
	quoted := pgx.Identifier{name}.Sanitize()
	if _, err := admin.ExecContext(ctx, "CREATE DATABASE "+quoted+" TEMPLATE template0"); err != nil {
		t.Fatalf("create isolated test database (PostgreSQL and CREATEDB permission required): %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(cleanupCtx, "DROP DATABASE "+quoted+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database %s: %v", name, err)
		}
	})
	config.Database = name
	db := stdlib.OpenDB(*config)
	t.Cleanup(func() { db.Close() })
	for _, action := range []string{"up", "up", "down", "up"} {
		command := exec.CommandContext(ctx, goose, "-env", "none", "-dir", "../migrations", "-timeout", "30s",
			"postgres", "dbname="+name+" connect_timeout=5", action)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("goose %s: %v\n%s", action, err, output)
		}
		t.Logf("goose %s: %s", action, output)
	}
	return db
}

func TestCatalogIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	store := persistence.New(db)
	t.Run("fresh schema", func(t *testing.T) {
		rows, err := db.QueryContext(ctx, `SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname = 'public' ORDER BY tablename`)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var tables []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatal(err)
			}
			tables = append(tables, name)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		want := []string{"goose_db_version", "listings", "products", "retailers"}
		if !reflect.DeepEqual(tables, want) {
			t.Fatalf("tables = %v, want %v", tables, want)
		}
		t.Logf("tables: %v", tables)
		var version int
		if err := db.QueryRowContext(ctx, `SELECT version_id FROM goose_db_version WHERE is_applied ORDER BY id DESC LIMIT 1`).Scan(&version); err != nil {
			t.Fatal(err)
		}
		if version != 2 {
			t.Fatalf("version = %d, want 2", version)
		}
	})

	for _, product := range []domain.Product{
		{ID: "p1", Name: "Maker's item", Brand: " Brand ", Model: "Model A"}, {ID: "p2"},
	} {
		if err := store.InsertProduct(ctx, product); err != nil {
			t.Fatal(err)
		}
		got, err := store.GetProduct(ctx, product.ID)
		if err != nil || got != product {
			t.Fatalf("product round trip = %+v, %v; want %+v", got, err, product)
		}
	}
	for _, retailer := range []domain.Retailer{{ID: "r1", Name: "Retailer's store"}, {ID: "r2"}} {
		if err := store.InsertRetailer(ctx, retailer); err != nil {
			t.Fatal(err)
		}
		got, err := store.GetRetailer(ctx, retailer.ID)
		if err != nil || got != retailer {
			t.Fatalf("retailer round trip = %+v, %v; want %+v", got, err, retailer)
		}
	}
	base := domain.Listing{ID: "l1", ProductID: "p1", RetailerID: "r1", URL: "https://shop.example/item", RetailerProductID: "sku1"}
	missingID := base
	missingID.ID, missingID.RetailerProductID = "l2", ""
	for _, listing := range []domain.Listing{base, missingID} {
		if err := store.InsertListing(ctx, listing); err != nil {
			t.Fatal(err)
		}
		got, err := store.GetListing(ctx, listing.ID)
		if err != nil || got != listing {
			t.Fatalf("listing round trip = %+v, %v; want %+v", got, err, listing)
		}
	}

	assertPGError := func(t *testing.T, err error, code, constraint string) {
		t.Helper()
		var pgError *pgconn.PgError
		if !errors.As(err, &pgError) || pgError.Code != code || (constraint != "" && pgError.ConstraintName != constraint) {
			t.Fatalf("error = %v; want SQLSTATE %s constraint %s", err, code, constraint)
		}
	}
	t.Run("duplicate primary keys", func(t *testing.T) {
		assertPGError(t, store.InsertProduct(ctx, domain.Product{ID: "p1"}), "23505", "products_pkey")
		assertPGError(t, store.InsertRetailer(ctx, domain.Retailer{ID: "r1"}), "23505", "retailers_pkey")
		listing := base
		listing.URL += "/other"
		assertPGError(t, store.InsertListing(ctx, listing), "23505", "listings_pkey")
	})
	t.Run("exact source identity", func(t *testing.T) {
		for _, listing := range []domain.Listing{base, missingID} {
			listing.ID += "-duplicate"
			listing.ProductID = "p2"
			assertPGError(t, store.InsertListing(ctx, listing), "23505", "listings_source_identity_key")
		}
		for _, test := range []struct {
			name   string
			change func(*domain.Listing)
		}{
			{"variant", func(l *domain.Listing) { l.RetailerProductID = "sku2" }},
			{"retailer", func(l *domain.Listing) { l.RetailerID = "r2" }},
			{"URL", func(l *domain.Listing) { l.URL += "?alias=1" }},
		} {
			listing := base
			listing.ID = test.name
			test.change(&listing)
			if err := store.InsertListing(ctx, listing); err != nil {
				t.Fatalf("distinct %s: %v", test.name, err)
			}
		}
	})
	counts := func(t *testing.T) [3]int {
		t.Helper()
		var result [3]int
		if err := db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM products), (SELECT count(*) FROM retailers), (SELECT count(*) FROM listings)`).Scan(&result[0], &result[1], &result[2]); err != nil {
			t.Fatal(err)
		}
		return result
	}
	t.Run("foreign keys without partial writes", func(t *testing.T) {
		before := counts(t)
		for _, refs := range [][2]string{{"absent", "r1"}, {"p1", "absent"}, {"absent", "absent"}} {
			listing := base
			listing.ID, listing.URL = "invalid-fk", "https://shop.example/invalid"
			listing.ProductID, listing.RetailerID = refs[0], refs[1]
			assertPGError(t, store.InsertListing(ctx, listing), "23503", "")
		}
		if after := counts(t); after != before {
			t.Fatalf("failed writes changed counts: %v -> %v", before, after)
		}
	})
	t.Run("invalid domain values are not persisted", func(t *testing.T) {
		before := counts(t)
		for _, err := range []error{
			store.InsertProduct(ctx, domain.Product{}),
			store.InsertRetailer(ctx, domain.Retailer{ID: " "}),
			store.InsertListing(ctx, domain.Listing{ID: "bad-url", ProductID: "p1", RetailerID: "r1", URL: "/relative"}),
			store.InsertListing(ctx, domain.Listing{ID: "bad-ref", URL: base.URL}),
		} {
			var pgError *pgconn.PgError
			if err == nil || errors.As(err, &pgError) {
				t.Fatalf("want domain validation error before SQL, got %v", err)
			}
		}
		if after := counts(t); after != before {
			t.Fatalf("invalid inputs changed counts: %v -> %v", before, after)
		}
	})
	t.Run("missing records", func(t *testing.T) {
		_, productErr := store.GetProduct(ctx, "absent")
		_, retailerErr := store.GetRetailer(ctx, "absent")
		_, listingErr := store.GetListing(ctx, "invalid-fk")
		for _, err := range []error{productErr, retailerErr, listingErr} {
			if !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("want sql.ErrNoRows, got %v", err)
			}
		}
	})
}
