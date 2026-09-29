//go:build integration

package persistence_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/persistence"
	"github.com/leaf482/price-terminal/backend/provider"
)

func TestBatchFailureIsolationIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	store := persistence.New(freshDatabase(t, ctx))
	if err := store.InsertProduct(ctx, domain.Product{ID: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertRetailer(ctx, domain.Retailer{ID: "r"}); err != nil {
		t.Fatal(err)
	}
	jobs := make([]ingestion.Job, 4)
	observations := make([]domain.PriceObservation, 4)
	cause := errors.New("provider unavailable")
	for index := range jobs {
		id := fmt.Sprintf("l%d", index)
		listing := domain.Listing{ID: id, ProductID: "p", RetailerID: "r", URL: "https://example.com/" + id}
		if err := store.InsertListing(ctx, listing); err != nil {
			t.Fatal(err)
		}
		o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: id, ObservedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Source: listing.URL, Stock: domain.StockInStock, OfferPrice: &domain.Money{MinorUnits: 100, Currency: domain.USD}})
		if err != nil {
			t.Fatal(err)
		}
		observations[index] = o
		response := provider.FakeResponse{Observation: o}
		if index == 1 || index == 2 {
			response.Err = cause
		}
		jobs[index] = ingestion.Job{Listing: listing, ResultID: "result-" + id, Provider: provider.NewFake(map[string]provider.FakeResponse{id: response})}
	}
	// A failed collection must also preserve an earlier accepted observation.
	if err := store.InsertPriceObservation(ctx, "prior", observations[1]); err != nil {
		t.Fatal(err)
	}
	results, err := ingestion.Run(ctx, store, jobs, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for index, result := range results {
		if result.Listing != jobs[index].Listing || result.ResultID != jobs[index].ResultID {
			t.Fatal("association lost")
		}
		if index == 1 || index == 2 {
			if !errors.Is(result.Err, cause) {
				t.Fatalf("failure lost: %v", result.Err)
			}
		} else if result.Err != nil {
			t.Fatal(result.Err)
		}
	}
	assertHistory := func() {
		for index, job := range jobs {
			got, err := store.ListPriceObservations(ctx, job.Listing.ID)
			want := []domain.PriceObservation{}
			if index != 2 {
				want = append(want, observations[index])
			}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("listing %s history=%v err=%v", job.Listing.ID, got, err)
			}
		}
	}
	assertHistory()
	// Re-delivery of these explicit IDs cannot create additional observations.
	again, err := ingestion.Run(ctx, store, jobs, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{0, 3} {
		var pgError *pgconn.PgError
		if !errors.As(again[index].Err, &pgError) || pgError.Code != "23505" {
			t.Fatalf("duplicate error lost: %v", again[index].Err)
		}
	}
	assertHistory()
}
