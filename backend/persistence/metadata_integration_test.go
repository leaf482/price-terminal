//go:build integration

package persistence_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"reflect"
	"testing"
	"time"
)

func TestMetadataPreservesHistoryIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := seedReliability(t, ctx, db)
	listing, _ := s.GetListing(ctx, "l")
	history, err := s.ListPriceObservations(ctx, "l")
	if err != nil {
		t.Fatal(err)
	}
	p := domain.Product{ID: "p", Name: "Corrected", Brand: "New brand", Model: "New model"}
	saved, err := s.UpdateProductMetadata(ctx, p)
	if err != nil || saved != p {
		t.Fatal(saved, err)
	}
	r := domain.Retailer{ID: "r", Name: "Corrected shop"}
	rs, err := s.UpdateRetailerMetadata(ctx, r)
	if err != nil || rs != r {
		t.Fatal(rs, err)
	}
	products, err := s.ListProducts(ctx, 20)
	if err != nil || len(products) != 1 || products[0] != p {
		t.Fatal("dashboard catalog", products, err)
	}
	after, err := s.GetListing(ctx, "l")
	if err != nil || after != listing {
		t.Fatal(after, err)
	}
	facts, err := s.ListPriceObservations(ctx, "l")
	if err != nil || !reflect.DeepEqual(facts, history) {
		t.Fatal("history changed", err)
	}
	current, err := s.CurrentListing(ctx, "l")
	if err != nil || current.Observation == nil {
		t.Fatal(current, err)
	}
	for _, err := range []error{func() error { _, e := s.UpdateProductMetadata(ctx, domain.Product{ID: "missing"}); return e }(), func() error { _, e := s.UpdateRetailerMetadata(ctx, domain.Retailer{ID: "missing"}); return e }()} {
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatal(err)
		}
	}
	if _, err := s.UpdateProductMetadata(ctx, domain.Product{}); err == nil {
		t.Fatal("invalid identity accepted")
	}
	if _, err := s.UpdateRetailerMetadata(ctx, domain.Retailer{}); err == nil {
		t.Fatal("invalid identity accepted")
	}
	empty, err := s.UpdateProductMetadata(ctx, domain.Product{ID: "p"})
	if err != nil || empty.Name != "" || empty.Brand != "" || empty.Model != "" {
		t.Fatal(empty, err)
	}
}
