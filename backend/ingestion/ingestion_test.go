package ingestion_test

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/provider"
)

type storeStub struct {
	get    func(context.Context, string) (domain.Listing, error)
	insert func(context.Context, string, domain.PriceObservation) error
}

func (s storeStub) GetListing(c context.Context, id string) (domain.Listing, error) {
	return s.get(c, id)
}
func (s storeStub) InsertPriceObservation(c context.Context, id string, o domain.PriceObservation) error {
	return s.insert(c, id, o)
}

type providerFunc func(context.Context, domain.Listing) (domain.PriceObservation, error)

func (f providerFunc) Collect(c context.Context, l domain.Listing) (domain.PriceObservation, error) {
	return f(c, l)
}

func listing() domain.Listing {
	return domain.Listing{ID: "l", ProductID: "p", RetailerID: "r", URL: "https://example.com/item", RetailerProductID: "sku"}
}
func observation(t *testing.T, price *domain.Money) domain.PriceObservation {
	t.Helper()
	o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: time.Date(2026, 9, 28, 12, 0, 0, 123, time.UTC), Source: "fixture evidence", Stock: domain.StockInStock, OfferPrice: price})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestIngestFakeResults(t *testing.T) {
	for _, test := range []struct {
		name  string
		price *domain.Money
	}{
		{"normal", &domain.Money{MinorUnits: 100, Currency: domain.USD}},
		{"stock only", nil}, {"explicit zero", &domain.Money{Currency: domain.JPY}},
	} {
		t.Run(test.name, func(t *testing.T) {
			want := observation(t, test.price)
			fake := provider.NewFake(map[string]provider.FakeResponse{"l": {Observation: want}})
			writes := 0
			store := storeStub{get: func(context.Context, string) (domain.Listing, error) { return listing(), nil }, insert: func(_ context.Context, id string, o domain.PriceObservation) error {
				writes++
				if id != "result" || !reflect.DeepEqual(o, want) {
					t.Fatal("collected facts or ID changed")
				}
				return nil
			}}
			got, err := ingestion.New(fake, store).Ingest(context.Background(), "result", listing())
			if err != nil || got.ID() != "result" || !reflect.DeepEqual(got.Observation(), want) || writes != 1 {
				t.Fatalf("result=%+v writes=%d err=%v", got, writes, err)
			}
		})
	}
}

func TestCollectionFailureAndInvalidResultDoNotPersist(t *testing.T) {
	cause := errors.New("source failed")
	want := observation(t, nil)
	for _, test := range []struct {
		name    string
		p       provider.Provider
		request domain.Listing
		id      string
		cause   error
	}{
		{"fake failure", provider.NewFake(map[string]provider.FakeResponse{"l": {Err: cause}}), listing(), "result", cause},
		{"result alongside error", providerFunc(func(context.Context, domain.Listing) (domain.PriceObservation, error) { return want, cause }), listing(), "result", cause},
		{"zero result", providerFunc(func(context.Context, domain.Listing) (domain.PriceObservation, error) {
			return domain.PriceObservation{}, nil
		}), listing(), "result", nil},
		{"wrong listing", providerFunc(func(context.Context, domain.Listing) (domain.PriceObservation, error) { return want, nil }), domain.Listing{ID: "other", ProductID: "p", RetailerID: "r", URL: "https://example.com/other"}, "result", nil},
		{"invalid listing", nil, domain.Listing{}, "result", nil},
		{"invalid ID", nil, listing(), " ", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Nil dependencies fail the test if invalid input reaches persistence.
			got, err := ingestion.New(test.p, nil).Ingest(context.Background(), test.id, test.request)
			if err == nil || got.ID() != "" {
				t.Fatalf("accepted failure: %+v %v", got, err)
			}
			if test.cause != nil && !errors.Is(err, test.cause) {
				t.Fatalf("lost cause: %v", err)
			}
		})
	}
}

func TestPersistenceRetryDoesNotRecollect(t *testing.T) {
	cause := errors.New("database unavailable")
	want := observation(t, nil)
	collections, writes := 0, 0
	p := providerFunc(func(context.Context, domain.Listing) (domain.PriceObservation, error) {
		collections++
		return want, nil
	})
	store := storeStub{get: func(context.Context, string) (domain.Listing, error) { return listing(), nil }, insert: func(_ context.Context, id string, o domain.PriceObservation) error {
		writes++
		if id != "result" || !reflect.DeepEqual(o, want) {
			t.Fatal("retry altered ID or facts")
		}
		if writes == 1 {
			return cause
		}
		return nil
	}}
	i := ingestion.New(p, store)
	got, err := i.Ingest(context.Background(), "result", listing())
	if !errors.Is(err, cause) || got.ID() != "result" {
		t.Fatalf("lost retry value/error: %+v %v", got, err)
	}
	if err := i.Persist(context.Background(), got); err != nil {
		t.Fatal(err)
	}
	if collections != 1 || writes != 2 {
		t.Fatalf("collections=%d writes=%d", collections, writes)
	}
	if err := i.Persist(context.Background(), ingestion.Collected{}); err == nil {
		t.Fatal("zero retry accepted")
	}
}

func TestListingRelationshipFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*domain.Listing)
		err    error
	}{
		{"missing", func(*domain.Listing) {}, sql.ErrNoRows},
		{"lookup failure", func(*domain.Listing) {}, errors.New("database failed")},
		{"wrong product", func(l *domain.Listing) { l.ProductID = "other" }, nil},
		{"wrong retailer", func(l *domain.Listing) { l.RetailerID = "other" }, nil},
		{"wrong URL", func(l *domain.Listing) { l.URL = "https://example.com/other" }, nil},
		{"wrong variant", func(l *domain.Listing) { l.RetailerProductID = "other" }, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			stored := listing()
			test.change(&stored)
			store := storeStub{get: func(context.Context, string) (domain.Listing, error) { return stored, test.err }}
			fake := provider.NewFake(map[string]provider.FakeResponse{"l": {Observation: observation(t, nil)}})
			_, err := ingestion.New(fake, store).Ingest(context.Background(), "result", listing())
			if err == nil || (test.err != nil && !errors.Is(err, test.err)) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestCancellation(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, stop := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer stop()
	for _, ctx := range []context.Context{canceled, expired} {
		if _, err := ingestion.New(nil, nil).Ingest(ctx, "result", listing()); !errors.Is(err, ctx.Err()) {
			t.Fatal(err)
		}
	}
	want := observation(t, nil)
	ctx, cancelDuring := context.WithCancel(context.Background())
	p := providerFunc(func(received context.Context, _ domain.Listing) (domain.PriceObservation, error) {
		if received != ctx {
			t.Fatal("context replaced")
		}
		cancelDuring()
		return want, nil
	})
	got, err := ingestion.New(p, nil).Ingest(ctx, "result", listing())
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := ingestion.New(nil, nil).Persist(ctx, got); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	// The same context also reaches both persistence calls.
	ctx2, cancelInsert := context.WithCancel(context.Background())
	defer cancelInsert()
	store := storeStub{get: func(c context.Context, _ string) (domain.Listing, error) {
		if c != ctx2 {
			t.Fatal("lookup context replaced")
		}
		return listing(), nil
	}, insert: func(c context.Context, _ string, _ domain.PriceObservation) error {
		if c != ctx2 {
			t.Fatal("insert context replaced")
		}
		cancelInsert()
		return c.Err()
	}}
	_, err = ingestion.New(provider.NewFake(map[string]provider.FakeResponse{"l": {Observation: want}}), store).Ingest(ctx2, "result", listing())
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
