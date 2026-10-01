package main

import (
	"context"
	"database/sql"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCSVValues(t *testing.T) {
	input := strings.Join(csvColumns, ",") + "\n" +
		"2026-01-02T03:04:05.123456789+05:30,USD,0,100,200,300,manufacturer,in_stock\n" +
		"2026-01-03T00:00:00Z,,,,,,,unknown\n" +
		"2026-01-04T00:00:00Z,JPY,,10,,,,out_of_stock\n"
	rows, issues := parseObservationCSV(strings.NewReader(input), "l", "import")
	if len(issues) > 0 || len(rows) != 3 {
		t.Fatal(rows, issues)
	}
	if got := rows[0].ObservedAt().Format(time.RFC3339Nano); got != "2026-01-01T21:34:05.123456789Z" {
		t.Fatal(got)
	}
	if !strings.Contains(rows[0].Source(), "CSV import import row 2; observed_at=2026-01-02T03:04:05.123456789+05:30") {
		t.Fatal(rows[0].Source())
	}
	if m, ok := rows[0].OfferPrice(); !ok || m.MinorUnits != 0 {
		t.Fatal("zero lost")
	}
	if m, _ := rows[0].MSRP(); m.MinorUnits != 300 || rows[0].MSRPSource() != "manufacturer" {
		t.Fatal("MSRP lost")
	}
	if m, _ := rows[0].RetailerListPrice(); m.MinorUnits != 200 {
		t.Fatal("list lost")
	}
	if _, ok := rows[1].Currency(); ok {
		t.Fatal("stock currency invented")
	}
	if _, ok := rows[1].OfferPrice(); ok {
		t.Fatal("missing became zero")
	}
	if m, ok := rows[2].SalePrice(); !ok || m.Currency != domain.JPY || m.MinorUnits != 10 {
		t.Fatal("JPY lost")
	}
}

type csvStub struct {
	calls          int
	rows           []domain.PriceObservation
	err, lookupErr error
}

func (s *csvStub) GetListing(context.Context, string) (domain.Listing, error) {
	return domain.Listing{ID: "l"}, s.lookupErr
}
func (s *csvStub) ImportObservations(_ context.Context, _, _ string, rows []domain.PriceObservation) error {
	s.calls++
	s.rows = rows
	return s.err
}
func csvRequest(s *csvStub, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", "/listings/l/observations/import", strings.NewReader(body))
	r.SetPathValue("id", "l")
	r.Header.Set("X-Import-ID", "test-import")
	w := httptest.NewRecorder()
	csvImportHandler(s)(w, r)
	return w
}
func TestCSVValidationBeforePersistence(t *testing.T) {
	header := strings.Join(csvColumns, ",") + "\n"
	valid := "2026-01-02T03:04:05Z,USD,0,,,,,unknown\n"
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"valid", header + valid + valid, 201},
		{"bad calendar", header + valid + "2026-02-30T12:00:00Z,USD,1,,,,,unknown\n", 400},
		{"MSRP evidence", header + "2026-01-02T00:00:00Z,USD,,,,1,,unknown\n", 400},
		{"malformed number", header + "2026-01-02T00:00:00Z,USD,1.5,,,,,unknown\n", 400},
		{"mixed currency syntax", header + "2026-01-02T00:00:00Z,USD,1,JPY 2,,,,unknown\n", 400},
		{"missing currency", header + "2026-01-02T00:00:00Z,,1,,,,,unknown\n", 400},
		{"bad header", "observed_at,stock\n", 400}, {"header only", header, 400},
		{"bad CSV", header + valid + "\"unterminated", 400},
		{"row limit", header + strings.Repeat(valid, 501), 400},
		{"byte limit", strings.Repeat("x", (1<<20)+1), 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &csvStub{}
			w := csvRequest(s, tc.body)
			if w.Code != tc.status {
				t.Fatal(w.Code, w.Body)
			}
			if tc.status != 201 && s.calls != 0 {
				t.Fatal("invalid CSV reached writes")
			}
			if tc.name == "bad calendar" && !strings.Contains(w.Body.String(), `"row":3`) {
				t.Fatal(w.Body)
			}
		})
	}
	for _, s := range []*csvStub{{err: errors.New("secret")}, {lookupErr: sql.ErrNoRows}} {
		w := csvRequest(s, header+valid)
		if w.Code != 500 && w.Code != 404 {
			t.Fatal(w.Code)
		}
		if strings.Contains(w.Body.String(), "secret") {
			t.Fatal("internal error leaked")
		}
	}
}
