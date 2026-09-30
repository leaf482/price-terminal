package ingestion_test

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/provider"
	"testing"
)

type alertFailureStore struct {
	storeStub
	evaluate func(context.Context, string) error
}

func (s alertFailureStore) EvaluateAlerts(c context.Context, id string) error {
	return s.evaluate(c, id)
}
func TestAlertFailureDoesNotInvalidateObservation(t *testing.T) {
	persisted := false
	evaluated := false
	s := alertFailureStore{storeStub: storeStub{
		get:    func(context.Context, string) (domain.Listing, error) { return listing(), nil },
		insert: func(context.Context, string, domain.PriceObservation) error { persisted = true; return nil },
	}, evaluate: func(_ context.Context, id string) error {
		if !persisted || id != "result" {
			t.Fatal("evaluation preceded persistence")
		}
		evaluated = true
		return errors.New("evaluation unavailable")
	}}
	fake := provider.NewFake(map[string]provider.FakeResponse{"l": {Observation: observation(t, &domain.Money{MinorUnits: 100, Currency: domain.USD})}})
	result, err := ingestion.New(fake, s).Ingest(context.Background(), "result", listing())
	if err != nil || !persisted || !evaluated || result.ID() != "result" {
		t.Fatalf("persisted=%v evaluated=%v result=%v err=%v", persisted, evaluated, result.ID(), err)
	}
}
