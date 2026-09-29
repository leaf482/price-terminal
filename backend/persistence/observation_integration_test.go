//go:build integration

package persistence_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

func TestObservationIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db := freshDatabase(t, ctx)
	store := persistence.New(db)
	if err := store.InsertProduct(ctx, domain.Product{ID: "p"}); err != nil {
		t.Fatal(err)
	}
	if err := store.InsertRetailer(ctx, domain.Retailer{ID: "r"}); err != nil {
		t.Fatal(err)
	}
	newListing := func(t *testing.T, id string) {
		t.Helper()
		if err := store.InsertListing(ctx, domain.Listing{ID: id, ProductID: "p", RetailerID: "r", URL: "https://example.com/" + id}); err != nil {
			t.Fatal(err)
		}
	}
	observedAt := time.Date(2026, 9, 28, 12, 0, 0, 123456789, time.FixedZone("source", 9*60*60))
	input := func(listing string) domain.PriceObservationInput {
		return domain.PriceObservationInput{ListingID: listing, ObservedAt: observedAt, Source: "retailer page evidence", Stock: domain.StockUnknown}
	}
	construct := func(t *testing.T, in domain.PriceObservationInput) domain.PriceObservation {
		t.Helper()
		o, err := domain.NewPriceObservation(in)
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	assertPGError := func(t *testing.T, err error, code string) {
		t.Helper()
		var pgError *pgconn.PgError
		if !errors.As(err, &pgError) || pgError.Code != code {
			t.Fatalf("error = %v; want SQLSTATE %s", err, code)
		}
	}
	for _, test := range []struct {
		name   string
		change func(*domain.PriceObservationInput)
	}{
		{"full_USD", func(i *domain.PriceObservationInput) {
			i.Stock = domain.StockInStock
			i.MSRP = &domain.Money{MinorUnits: 12000, Currency: domain.USD}
			i.RetailerListPrice = &domain.Money{MinorUnits: 11000, Currency: domain.USD}
			i.SalePrice = &domain.Money{MinorUnits: 9000, Currency: domain.USD}
			i.OfferPrice = &domain.Money{MinorUnits: 9500, Currency: domain.USD}
			i.MSRPSource = "manufacturer's explicitly labeled MSRP"
		}},
		{"stock_only", func(i *domain.PriceObservationInput) { i.Stock = domain.StockOutOfStock }},
		{"JPY", func(i *domain.PriceObservationInput) {
			i.OfferPrice = &domain.Money{MinorUnits: 500, Currency: domain.JPY}
		}},
		{"zero_offer", func(i *domain.PriceObservationInput) { i.OfferPrice = &domain.Money{Currency: domain.USD} }},
		{"zero_MSRP", func(i *domain.PriceObservationInput) {
			i.MSRP = &domain.Money{Currency: domain.JPY}
			i.MSRPSource = "explicit zero MSRP"
		}},
		{"zero_list", func(i *domain.PriceObservationInput) { i.RetailerListPrice = &domain.Money{Currency: domain.USD} }},
		{"zero_sale", func(i *domain.PriceObservationInput) { i.SalePrice = &domain.Money{Currency: domain.JPY} }},
		{"max_amount", func(i *domain.PriceObservationInput) {
			i.OfferPrice = &domain.Money{MinorUnits: math.MaxInt64, Currency: domain.USD}
		}},
		{"before_epoch", func(i *domain.PriceObservationInput) {
			i.ObservedAt = time.Date(1960, 1, 1, 0, 0, 0, 999999999, time.UTC)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			newListing(t, test.name)
			in := input(test.name)
			test.change(&in)
			want := construct(t, in)
			if err := store.InsertPriceObservation(ctx, test.name, want); err != nil {
				t.Fatal(err)
			}
			got, err := store.ListPriceObservations(ctx, test.name)
			if err != nil || !reflect.DeepEqual(got, []domain.PriceObservation{want}) {
				t.Fatalf("round trip = %+v, %v; want %+v", got, err, want)
			}
			// Check database NULL semantics independently of the read mapper.
			var currencyPresent, offerPresent bool
			if err := db.QueryRowContext(ctx, `SELECT currency IS NOT NULL, offer_price IS NOT NULL FROM price_observations WHERE result_id = $1`, test.name).Scan(&currencyPresent, &offerPresent); err != nil {
				t.Fatal(err)
			}
			_, wantCurrency := want.Currency()
			_, wantOffer := want.OfferPrice()
			if currencyPresent != wantCurrency || offerPresent != wantOffer {
				t.Fatal("database lost presence semantics")
			}
		})
	}
	t.Run("retry identity and deterministic history", func(t *testing.T) {
		newListing(t, "history")
		in := input("history")
		in.OfferPrice = &domain.Money{MinorUnits: 100, Currency: domain.USD}
		first := construct(t, in)
		in.ObservedAt = observedAt.Add(time.Second)
		later := construct(t, in)
		// Same time, different evidence, reverse ID insertion order.
		in.ObservedAt, in.Source = observedAt, "tie evidence"
		tie := construct(t, in)
		in.ObservedAt, in.Source = observedAt.Add(time.Nanosecond), first.Source()
		nextNanosecond := construct(t, in)
		for _, record := range []struct {
			id string
			o  domain.PriceObservation
		}{
			{"history-later", later}, {"history-b", tie}, {"history-nano", nextNanosecond}, {"history-a", first},
		} {
			if err := store.InsertPriceObservation(ctx, record.id, record.o); err != nil {
				t.Fatal(err)
			}
		}
		assertPGError(t, store.InsertPriceObservation(ctx, "history-a", first), "23505")
		assertPGError(t, store.InsertPriceObservation(ctx, "history-a", later), "23505")
		want := []domain.PriceObservation{first, tie, nextNanosecond, later}
		for range 2 {
			got, err := store.ListPriceObservations(ctx, "history")
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("history = %+v, %v; want %+v", got, err, want)
			}
		}
	})
	t.Run("invalid listing FK", func(t *testing.T) {
		assertPGError(t, store.InsertPriceObservation(ctx, "bad-fk", construct(t, input("missing"))), "23503")
		got, err := store.ListPriceObservations(ctx, "missing")
		if err != nil || len(got) != 0 {
			t.Fatalf("missing history = %v, %v", got, err)
		}
	})
	t.Run("mixed currencies rejected before persistence", func(t *testing.T) {
		in := input("history")
		in.SalePrice = &domain.Money{MinorUnits: 100, Currency: domain.USD}
		in.OfferPrice = &domain.Money{MinorUnits: 100, Currency: domain.JPY}
		invalid, err := domain.NewPriceObservation(in)
		if err == nil {
			t.Fatal("mixed currencies accepted")
		}
		if err := store.InsertPriceObservation(ctx, "mixed", invalid); err == nil {
			t.Fatal("invalid observation persisted")
		}
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM price_observations WHERE result_id = 'mixed'`).Scan(&count); err != nil || count != 0 {
			t.Fatalf("mixed count = %d, %v", count, err)
		}
	})
	t.Run("database price constraints", func(t *testing.T) {
		for _, test := range []struct {
			name             string
			currency, amount any
			stock            string
		}{
			{"price without currency", nil, int64(0), "unknown"},
			{"currency without price", "USD", nil, "unknown"},
			{"unsupported currency", "EUR", int64(100), "unknown"},
			{"negative amount", "USD", int64(-1), "unknown"},
			{"invalid stock", nil, nil, "invalid"},
		} {
			t.Run(test.name, func(t *testing.T) {
				_, err := db.ExecContext(ctx, `INSERT INTO price_observations
					(result_id, listing_id, observed_at, observed_at_ns_remainder, source, stock, currency, offer_price, msrp_source)
					VALUES ($1, 'history', $2, 0, 'source', $3, $4, $5, '')`, test.name, observedAt, test.stock, test.currency, test.amount)
				assertPGError(t, err, "23514")
			})
		}
	})
}
