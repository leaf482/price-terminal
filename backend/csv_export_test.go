package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

type exportStub struct {
	rows                 []persistence.ObservationAudit
	lookupErr, exportErr error
}

func (s exportStub) GetListing(context.Context, string) (domain.Listing, error) {
	return domain.Listing{}, s.lookupErr
}
func (s exportStub) ExportObservationAudit(context.Context, string) ([]persistence.ObservationAudit, error) {
	return s.rows, s.exportErr
}
func TestCSVExport(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.UTC)
	text := "evidence, with \"quotes\"\nand newline"
	o, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: at, Source: text, Stock: domain.StockInStock, OfferPrice: &domain.Money{Currency: domain.USD}, SalePrice: &domain.Money{MinorUnits: 100, Currency: domain.USD}, RetailerListPrice: &domain.Money{MinorUnits: 200, Currency: domain.USD}, MSRP: &domain.Money{MinorUnits: 300, Currency: domain.USD}, MSRPSource: text})
	if err != nil {
		t.Fatal(err)
	}
	stock, err := domain.NewPriceObservation(domain.PriceObservationInput{ListingID: "l", ObservedAt: at.Add(time.Second), Source: "manual", Stock: domain.StockUnknown})
	if err != nil {
		t.Fatal(err)
	}
	s := exportStub{rows: []persistence.ObservationAudit{{ID: "a", Observation: o, InvalidatedAt: &at, Reason: text}, {ID: "b", Observation: stock}}}
	request := httptest.NewRequest("GET", "/listings/l/observations/export", nil)
	request.SetPathValue("id", "l")
	w := httptest.NewRecorder()
	csvExportHandler(s)(w, request)
	if w.Code != 200 || w.Header().Get("Content-Type") != "text/csv; charset=utf-8" || !strings.Contains(w.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatal(w)
	}
	rows, err := csv.NewReader(w.Body).ReadAll()
	if err != nil || len(rows) != 3 {
		t.Fatal(rows, err)
	}
	if strings.Join(rows[0], ",") != "observation_id,observed_at,source,currency,offer_price_minor,sale_price_minor,list_price_minor,msrp_minor,msrp_source,stock,valid,invalidated_at,invalidation_reason" {
		t.Fatal(rows[0])
	}
	want := []string{"a", at.Format(time.RFC3339Nano), text, "USD", "0", "100", "200", "300", text, "in_stock", "false", at.Format(time.RFC3339Nano), text}
	for i, v := range want {
		if rows[1][i] != v {
			t.Fatalf("column %d: %q != %q", i, rows[1][i], v)
		}
	}
	for _, i := range []int{3, 4, 5, 6, 7, 8, 11, 12} {
		if rows[2][i] != "" {
			t.Fatal("invented value", rows[2])
		}
	}
	if rows[2][10] != "true" {
		t.Fatal(rows[2])
	}
	for _, tt := range []struct {
		name string
		s    exportStub
		code int
	}{{"empty", exportStub{}, 200}, {"missing", exportStub{lookupErr: sql.ErrNoRows}, 404}, {"limit", exportStub{exportErr: persistence.ErrExportLimit}, 422}, {"database", exportStub{exportErr: errors.New("secret")}, 500}} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			csvExportHandler(tt.s)(w, request)
			if w.Code != tt.code || strings.Contains(w.Body.String(), "secret") {
				t.Fatal(w)
			}
			if tt.code != 200 && w.Header().Get("Content-Disposition") != "" {
				t.Fatal("partial download")
			}
		})
	}
}
