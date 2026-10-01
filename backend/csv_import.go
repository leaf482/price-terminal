package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var csvColumns = []string{"observed_at", "currency", "offer_price_minor", "sale_price_minor", "list_price_minor", "msrp_minor", "msrp_source", "stock"}
var importIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

type csvIssue struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}
type csvStore interface {
	GetListing(context.Context, string) (domain.Listing, error)
	ImportObservations(context.Context, string, string, []domain.PriceObservation) error
}

func parseObservationCSV(input io.Reader, listingID, importID string) ([]domain.PriceObservation, []csvIssue) {
	reader := csv.NewReader(input)
	header, err := reader.Read()
	if err != nil || !slices.Equal(header, csvColumns) {
		return nil, []csvIssue{{1, "expected header: " + strings.Join(csvColumns, ",")}}
	}
	observations := []domain.PriceObservation{}
	issues := []csvIssue{}
	for count := 0; ; count++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		line := count + 2
		if err != nil {
			var parseErr *csv.ParseError
			if errors.As(err, &parseErr) {
				line = parseErr.StartLine
			}
			issues = append(issues, csvIssue{line, "malformed CSV row"})
			break
		}
		line, _ = reader.FieldPos(0)
		if count >= persistence.MaxImportRows {
			issues = append(issues, csvIssue{line, "maximum 500 observation rows"})
			break
		}
		o, err := csvObservation(row, listingID, importID, line)
		if err != nil {
			issues = append(issues, csvIssue{line, err.Error()})
		} else {
			observations = append(observations, o)
		}
	}
	if len(observations) == 0 && len(issues) == 0 {
		issues = append(issues, csvIssue{2, "at least one observation row is required"})
	}
	if len(issues) > 0 {
		return nil, issues
	}
	return observations, nil
}

func csvObservation(row []string, listingID, importID string, line int) (domain.PriceObservation, error) {
	var at time.Time
	// Use the same time.Time JSON/RFC3339 parser as manual entry. Never pass
	// through JavaScript Date or round fractional seconds. Keep original evidence.
	encoded, _ := json.Marshal(row[0])
	if err := at.UnmarshalJSON(encoded); err != nil {
		return domain.PriceObservation{}, fmt.Errorf("observed_at must be a valid RFC3339 timestamp")
	}
	currency := domain.Currency(strings.TrimSpace(row[1]))
	if currency != "" {
		if err := currency.Validate(); err != nil {
			return domain.PriceObservation{}, err
		}
	}
	in := domain.PriceObservationInput{ListingID: listingID, ObservedAt: at, Stock: domain.StockState(strings.TrimSpace(row[7])), MSRPSource: row[6], Source: fmt.Sprintf("manual: CSV import %s row %d; observed_at=%s", importID, line, row[0])}
	targets := []**domain.Money{&in.OfferPrice, &in.SalePrice, &in.RetailerListPrice, &in.MSRP}
	for i, target := range targets {
		text := strings.TrimSpace(row[i+2])
		if text == "" {
			continue
		}
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return domain.PriceObservation{}, fmt.Errorf("%s must be an integer minor-unit amount", csvColumns[i+2])
		}
		money, err := domain.NewMoney(n, currency)
		if err != nil {
			return domain.PriceObservation{}, fmt.Errorf("%s: %w", csvColumns[i+2], err)
		}
		*target = &money
	}
	return domain.NewPriceObservation(in)
}

func csvImportHandler(store csvStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Import-ID")
		if !importIDPattern.MatchString(id) {
			productError(w, 400, "invalid_import_id", "X-Import-ID must contain 1..100 letters, digits, underscores or hyphens")
			return
		}
		// Bound bytes before CSV parsing, including a single oversized field.
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			productError(w, 413, "import_too_large", "CSV must be at most 1 MiB")
			return
		}
		listingID := r.PathValue("id")
		observations, issues := parseObservationCSV(strings.NewReader(string(body)), listingID, id)
		if len(issues) > 0 {
			writeJSON(w, 400, map[string]any{"error": map[string]any{"code": "invalid_csv", "message": "No rows imported", "rows": issues}})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		if _, err := store.GetListing(ctx, listingID); err != nil {
			catalogReadError(w, err, "listing")
			return
		}
		if err := store.ImportObservations(ctx, listingID, id, observations); err != nil {
			var pg *pgconn.PgError
			if errors.As(err, &pg) && pg.Code == "23505" {
				productError(w, 409, "duplicate_import", "Import ID already exists. Check history before retrying with a new ID.")
			} else {
				productError(w, 500, "internal_error", "Import failed; no partial import was committed. Check history before retrying after an uncertain connection error.")
			}
			return
		}
		writeJSON(w, 201, map[string]any{"data": map[string]any{"imported": len(observations), "import_id": id}})
	}
}
