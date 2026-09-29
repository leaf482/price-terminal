package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/provider"
)

// A bounded, explicit Fake-only development configuration. No fixture changes
// source timestamps at collection time; no network or credentials are involved.
type collectionFixture struct {
	ListingID   string           `json:"listing_id"`
	Observation *observationJSON `json:"observation"`
	Fail        bool             `json:"fail"`
}

func loadCollector(path string, store ingestion.Store, interval, timeout time.Duration) (*collector.Runtime, error) {
	targets := make([]collector.Target, 0)
	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("collector: cannot open configuration")
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || info.Size() > 1024*1024 {
			return nil, fmt.Errorf("collector: configuration exceeds 1 MiB or cannot be read")
		}
		decoder := json.NewDecoder(io.LimitReader(file, 1024*1024+1))
		decoder.DisallowUnknownFields()
		var fixtures []collectionFixture
		if err := decoder.Decode(&fixtures); err != nil {
			return nil, fmt.Errorf("collector: invalid fixture JSON")
		}
		if decoder.Decode(new(any)) != io.EOF {
			return nil, fmt.Errorf("collector: expected one JSON array")
		}
		if len(fixtures) > ingestion.MaxBatchSize {
			return nil, fmt.Errorf("collector: at most 100 active listings")
		}
		for _, fixture := range fixtures {
			response := provider.FakeResponse{}
			if fixture.Fail {
				if fixture.Observation != nil {
					return nil, fmt.Errorf("collector: choose failure or observation")
				}
				response.Err = fmt.Errorf("configured fake failure")
			} else {
				data := fixture.Observation
				if data == nil {
					return nil, fmt.Errorf("collector: observation required")
				}
				money := func(amount *int64) *domain.Money {
					if amount == nil {
						return nil
					}
					return &domain.Money{MinorUnits: *amount, Currency: data.Currency}
				}
				hasPrice := data.MSRP != nil || data.RetailerListPrice != nil || data.SalePrice != nil || data.OfferPrice != nil
				if !hasPrice && data.Currency != "" {
					return nil, fmt.Errorf("collector: stock-only fixture must omit currency")
				}
				observation, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: fixture.ListingID, ObservedAt: data.ObservedAt, Source: data.Source, Stock: data.Stock, MSRP: money(data.MSRP), MSRPSource: data.MSRPSource, RetailerListPrice: money(data.RetailerListPrice), SalePrice: money(data.SalePrice), OfferPrice: money(data.OfferPrice)})
				if err != nil {
					return nil, fmt.Errorf("collector: invalid observation fixture")
				}
				response.Observation = observation
			}
			targets = append(targets, collector.Target{ListingID: fixture.ListingID, Provider: provider.NewFake(map[string]provider.FakeResponse{fixture.ListingID: response})})
		}
	}
	return collector.New(store, targets, interval, timeout)
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return duration, nil
}
