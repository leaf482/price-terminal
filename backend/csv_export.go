package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

type csvExportStore interface {
	GetListing(context.Context, string) (domain.Listing, error)
	ExportObservationAudit(context.Context, string) ([]persistence.ObservationAudit, error)
}

func csvMoney(m domain.Money, present bool) string {
	if !present {
		return ""
	}
	return strconv.FormatInt(m.MinorUnits, 10)
}

func csvExportHandler(store csvExportStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		id := r.PathValue("id")
		if _, err := store.GetListing(ctx, id); err != nil {
			catalogReadError(w, err, "listing")
			return
		}
		rows, err := store.ExportObservationAudit(ctx, id)
		if errors.Is(err, persistence.ErrExportLimit) || len(rows) > persistence.MaxExportRows {
			productError(w, 422, "export_limit", "Export exceeds the 10000 observation limit; no CSV was generated.")
			return
		}
		if err != nil {
			productError(w, 500, "internal_error", "Unable to export observations")
			return
		}
		// Buffer before sending download headers: query/encoding failures must never
		// produce a successful but incomplete CSV response.
		var body bytes.Buffer
		csvWriter := csv.NewWriter(&body)
		_ = csvWriter.Write([]string{"observation_id", "observed_at", "source", "currency", "offer_price_minor", "sale_price_minor", "list_price_minor", "msrp_minor", "msrp_source", "stock", "valid", "invalidated_at", "invalidation_reason"})
		for _, row := range rows {
			o := row.Observation
			currency, _ := o.Currency()
			invalidated := ""
			if row.InvalidatedAt != nil {
				invalidated = row.InvalidatedAt.UTC().Format(time.RFC3339Nano)
			}
			_ = csvWriter.Write([]string{row.ID, o.ObservedAt().Format(time.RFC3339Nano), o.Source(), string(currency), csvMoney(o.OfferPrice()), csvMoney(o.SalePrice()), csvMoney(o.RetailerListPrice()), csvMoney(o.MSRP()), o.MSRPSource(), string(o.Stock()), strconv.FormatBool(row.InvalidatedAt == nil), invalidated, row.Reason})
		}
		csvWriter.Flush()
		if csvWriter.Error() != nil {
			productError(w, 500, "internal_error", "Unable to encode observations")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="listing-observations.csv"`)
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(body.Bytes())
	}
}
