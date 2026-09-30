package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/ingestion"
)

type manualMoney struct {
	MinorUnits *int64          `json:"minor_units"`
	Currency   domain.Currency `json:"currency"`
}

func (m *manualMoney) money() (*domain.Money, error) {
	if m == nil {
		return nil, nil
	}
	if m.MinorUnits == nil {
		return nil, fmt.Errorf("minor_units must be an explicit integer")
	}
	value, err := domain.NewMoney(*m.MinorUnits, m.Currency)
	return &value, err
}

func manualObservationHandler(store ingestion.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body *struct {
			ID                string            `json:"id"`
			ObservedAt        time.Time         `json:"observed_at"`
			Source            string            `json:"source"`
			Stock             domain.StockState `json:"stock"`
			MSRP              *manualMoney      `json:"msrp"`
			MSRPSource        string            `json:"msrp_source"`
			RetailerListPrice *manualMoney      `json:"retailer_list_price"`
			SalePrice         *manualMoney      `json:"sale_price"`
			OfferPrice        *manualMoney      `json:"offer_price"`
		}
		if err := decodeCatalogBody(w, r, &body); err != nil || body == nil {
			productError(w, 400, "invalid_request", "expected manual observation JSON")
			return
		}
		if strings.TrimSpace(body.ID) == "" || strings.TrimSpace(body.Source) == "" {
			productError(w, 400, "invalid_observation", "id and source evidence are required")
			return
		}
		input := domain.PriceObservationInput{ListingID: r.PathValue("id"), ObservedAt: body.ObservedAt, Source: "manual: " + body.Source, Stock: body.Stock, MSRPSource: body.MSRPSource}
		for _, field := range []struct {
			from *manualMoney
			to   **domain.Money
		}{
			{body.MSRP, &input.MSRP}, {body.RetailerListPrice, &input.RetailerListPrice}, {body.SalePrice, &input.SalePrice}, {body.OfferPrice, &input.OfferPrice},
		} {
			value, err := field.from.money()
			if err != nil {
				productError(w, 400, "invalid_observation", err.Error())
				return
			}
			*field.to = value
		}
		observation, err := domain.NewPriceObservation(input)
		if err != nil {
			productError(w, 400, "invalid_observation", err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		listing, err := store.GetListing(ctx, input.ListingID)
		if err != nil {
			catalogReadError(w, err, "listing")
			return
		}
		_, err = ingestion.New(nil, store).Record(ctx, body.ID, listing, observation)
		if err != nil {
			var pg *pgconn.PgError
			switch {
			case errors.As(err, &pg) && pg.Code == "23505":
				productError(w, 409, "duplicate_observation", "observation ID already exists; check history before recording again")
			case errors.As(err, &pg) && pg.Code == "23503":
				productError(w, 404, "not_found", "listing not found")
			default:
				productError(w, 500, "internal_error", "unable to record observation")
			}
			return
		}
		writeJSON(w, 201, map[string]any{"data": map[string]any{"id": body.ID, "observation": observationResponse(observation)}})
	}
}
