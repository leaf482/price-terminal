//go:build integration

package persistence_test

import (
	"context"
	"github.com/leaf482/price-terminal/backend/domain"
	"reflect"
	"testing"
	"time"
)

func TestArchiveIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	s := seedReliability(t, ctx, db)
	p, err := s.GetProduct(ctx, "p")
	if err != nil || p.Archived {
		t.Fatal(p, err)
	}
	listing, _ := s.GetListing(ctx, "l")
	history, _ := s.ListPriceObservations(ctx, "l")
	events, _, _ := s.ListAlertEvents(ctx, "l")
	for _, archived := range []bool{true, false} {
		if err := s.SetProductArchived(ctx, "p", archived); err != nil {
			t.Fatal(err)
		}
		direct, err := s.GetProduct(ctx, "p")
		if err != nil || direct.Archived != archived || direct.Name != p.Name {
			t.Fatal(direct, err)
		}
		active, err := s.ListDashboardProducts(ctx, 21, false)
		if err != nil || (len(active) == 0) != archived {
			t.Fatal(active, err)
		}
		all, err := s.ListDashboardProducts(ctx, 21, true)
		if err != nil || len(all) != 1 || all[0].Archived != archived {
			t.Fatal(all, err)
		}
		after, err := s.GetListing(ctx, "l")
		if err != nil || after != listing {
			t.Fatal(after, err)
		}
		facts, err := s.ListPriceObservations(ctx, "l")
		if err != nil || !reflect.DeepEqual(facts, history) {
			t.Fatal("history changed", err)
		}
		es, _, err := s.ListAlertEvents(ctx, "l")
		if err != nil || !reflect.DeepEqual(es, events) {
			t.Fatal("events changed", err)
		}
		if _, err := s.GetPromotion(ctx, "l", "promo"); err != nil {
			t.Fatal(err)
		}
		current, err := s.CurrentListing(ctx, "l")
		if err != nil || current.Observation == nil {
			t.Fatal(current, err)
		}
		saved, err := s.UpdateProductMetadata(ctx, domain.Product{ID: "p", Name: p.Name, Brand: p.Brand, Model: p.Model})
		if err != nil || saved.Archived != archived {
			t.Fatal("metadata reset archive", saved, err)
		}
	}
	// Filtering occurs before LIMIT, even when archived IDs sort first.
	if err := s.InsertProduct(ctx, domain.Product{ID: "a", Archived: true}); err != nil {
		t.Fatal(err)
	}
	first, err := s.ListDashboardProducts(ctx, 1, false)
	if err != nil || len(first) != 1 || first[0].ID != "p" {
		t.Fatal(first, err)
	}
}
