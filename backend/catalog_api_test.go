package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
)

type catalogStub struct {
	insertRetailer func(context.Context, domain.Retailer) error
	getRetailer    func(context.Context, string) (domain.Retailer, error)
	listRetailers  func(context.Context, int) ([]domain.Retailer, error)
	insertListing  func(context.Context, domain.Listing) error
	getListing     func(context.Context, string) (domain.Listing, error)
	listListings   func(context.Context, string, int) ([]domain.Listing, error)
	getProduct     func(context.Context, string) (domain.Product, error)
}

func (s catalogStub) InsertRetailer(c context.Context, v domain.Retailer) error {
	return s.insertRetailer(c, v)
}
func (s catalogStub) GetRetailer(c context.Context, id string) (domain.Retailer, error) {
	return s.getRetailer(c, id)
}
func (s catalogStub) ListRetailers(c context.Context, n int) ([]domain.Retailer, error) {
	return s.listRetailers(c, n)
}
func (s catalogStub) InsertListing(c context.Context, v domain.Listing) error {
	return s.insertListing(c, v)
}
func (s catalogStub) GetListing(c context.Context, id string) (domain.Listing, error) {
	return s.getListing(c, id)
}
func (s catalogStub) ListListingsByProduct(c context.Context, id string, n int) ([]domain.Listing, error) {
	return s.listListings(c, id, n)
}
func (s catalogStub) GetProduct(c context.Context, id string) (domain.Product, error) {
	return s.getProduct(c, id)
}

func catalogRequest(t *testing.T, s catalogStub, method, path, body string, status int, code string) json.RawMessage {
	t.Helper()
	w := httptest.NewRecorder()
	newHandler(nil, nil, s).ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	if w.Code != status || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d type=%s body=%s", w.Code, w.Header().Get("Content-Type"), w.Body.String())
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil || len(envelope) != 1 {
		t.Fatalf("bad envelope %s", w.Body.String())
	}
	if code != "" {
		var failure struct{ Code, Message string }
		if err := json.Unmarshal(envelope["error"], &failure); err != nil || failure.Code != code || failure.Message == "" {
			t.Fatalf("error=%s want code=%s", w.Body.String(), code)
		}
		if strings.Contains(w.Body.String(), "secret") {
			t.Fatal("internal error leaked")
		}
	} else if envelope["data"] == nil {
		t.Fatal("missing data")
	}
	return envelope["data"]
}

func TestCatalogCreateAndRead(t *testing.T) {
	retailer := domain.Retailer{ID: "r1", Name: "Retailer"}
	for _, sku := range []string{"", " SKU-1 "} {
		listing := domain.Listing{ID: "l/1", ProductID: "p1", RetailerID: "r1", URL: "https://EXAMPLE.com/a%2Fb?b=2&a=1#variant", RetailerProductID: sku}
		s := catalogStub{
			insertRetailer: func(ctx context.Context, v domain.Retailer) error {
				if v != retailer {
					t.Fatalf("retailer=%+v", v)
				}
				return nil
			},
			getRetailer: func(ctx context.Context, id string) (domain.Retailer, error) {
				if id != "r1" {
					t.Fatal(id)
				}
				return retailer, nil
			},
			insertListing: func(ctx context.Context, v domain.Listing) error {
				if v != listing {
					t.Fatalf("listing=%+v", v)
				}
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 2*time.Second {
					t.Fatal("missing database deadline")
				}
				return nil
			},
			getListing: func(ctx context.Context, id string) (domain.Listing, error) {
				if id != "l/1" {
					t.Fatal(id)
				}
				return listing, nil
			},
		}
		retailerBody, _ := json.Marshal(retailerJSON{ID: retailer.ID, Name: retailer.Name})
		listingBody, _ := json.Marshal(listingResponse(listing))
		for _, test := range []struct {
			method, path, body string
			status             int
			want               any
		}{
			{"POST", "/retailers", string(retailerBody), 201, retailerJSON{ID: retailer.ID, Name: retailer.Name}},
			{"GET", "/retailers/r1", "", 200, retailerJSON{ID: retailer.ID, Name: retailer.Name}},
			{"POST", "/listings", string(listingBody), 201, listingResponse(listing)},
			{"GET", "/listings/l%2F1", "", 200, listingResponse(listing)},
		} {
			data := catalogRequest(t, s, test.method, test.path, test.body, test.status, "")
			want, _ := json.Marshal(test.want)
			if string(data) != string(want) {
				t.Fatalf("data=%s want=%s", data, want)
			}
		}
	}
}

func TestCatalogInvalidBodies(t *testing.T) {
	for _, path := range []string{"/retailers", "/listings"} {
		for _, body := range []string{"", `{`, `null`, `[]`, `{"id":1}`, `{"unexpected":true}`, `{"id":"x"} {}`, `{"id":"x"} trailing`, `{"id":"` + strings.Repeat("x", 65536) + `"}`} {
			catalogRequest(t, catalogStub{}, "POST", path, body, 400, "invalid_request")
		}
	}
	for _, body := range []string{`{}`, `{"id":" "}`} {
		catalogRequest(t, catalogStub{}, "POST", "/retailers", body, 400, "invalid_retailer")
	}
	for _, body := range []string{`{}`, `{"id":"l","product_id":"p","retailer_id":"r","url":"/relative"}`, `{"id":"l","retailer_id":"r","url":"https://example.com"}`, `{"id":"l","product_id":"p","url":"https://example.com"}`} {
		catalogRequest(t, catalogStub{}, "POST", "/listings", body, 400, "invalid_listing")
	}
}

func TestCatalogErrorMappings(t *testing.T) {
	pg := func(code, constraint string) error {
		return fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: code, ConstraintName: constraint, Detail: "secret"})
	}
	listingBody := `{"id":"l","product_id":"p","retailer_id":"r","url":"https://example.com"}`
	for _, test := range []struct {
		method, path, body string
		err                error
		status             int
		code               string
	}{
		{"POST", "/retailers", `{"id":"r"}`, pg("23505", "retailers_pkey"), 409, "duplicate_retailer"},
		{"POST", "/listings", listingBody, pg("23505", "listings_pkey"), 409, "duplicate_listing"},
		{"POST", "/listings", listingBody, pg("23505", "listings_source_identity_key"), 409, "duplicate_listing_source"},
		{"POST", "/listings", listingBody, pg("23503", "listings_product_id_fkey"), 400, "invalid_product_reference"},
		{"POST", "/listings", listingBody, pg("23503", "listings_retailer_id_fkey"), 400, "invalid_retailer_reference"},
		{"GET", "/retailers/missing", "", fmt.Errorf("wrapped: %w", sql.ErrNoRows), 404, "retailer_not_found"},
		{"GET", "/listings/missing", "", fmt.Errorf("wrapped: %w", sql.ErrNoRows), 404, "listing_not_found"},
		{"GET", "/products/missing/listings", "", fmt.Errorf("wrapped: %w", sql.ErrNoRows), 404, "product_not_found"},
		{"POST", "/retailers", `{"id":"r"}`, errors.New("secret"), 500, "internal_error"},
		{"POST", "/listings", listingBody, pg("23503", "unknown_constraint"), 500, "internal_error"},
		{"GET", "/retailers/r", "", errors.New("secret"), 500, "internal_error"},
		{"GET", "/listings/l", "", errors.New("secret"), 500, "internal_error"},
		{"GET", "/retailers", "", errors.New("secret"), 500, "internal_error"},
		{"GET", "/products/p/listings", "", errors.New("secret"), 500, "internal_error"},
	} {
		s := catalogStub{
			insertRetailer: func(context.Context, domain.Retailer) error { return test.err },
			insertListing:  func(context.Context, domain.Listing) error { return test.err },
			getRetailer:    func(context.Context, string) (domain.Retailer, error) { return domain.Retailer{}, test.err },
			getListing:     func(context.Context, string) (domain.Listing, error) { return domain.Listing{}, test.err },
			getProduct:     func(context.Context, string) (domain.Product, error) { return domain.Product{}, test.err },
			listRetailers:  func(context.Context, int) ([]domain.Retailer, error) { return nil, test.err },
		}
		catalogRequest(t, s, test.method, test.path, test.body, test.status, test.code)
	}
	s := catalogStub{getProduct: func(context.Context, string) (domain.Product, error) { return domain.Product{ID: "p"}, nil }, listListings: func(context.Context, string, int) ([]domain.Listing, error) { return nil, errors.New("secret") }}
	catalogRequest(t, s, "GET", "/products/p/listings", "", 500, "internal_error")
}

func TestCatalogLists(t *testing.T) {
	for _, test := range []struct {
		query string
		limit int
	}{{"", 50}, {"?limit=1", 1}, {"?limit=100", 100}} {
		for _, empty := range []bool{false, true} {
			retailers := []domain.Retailer{{ID: "r", Name: "Retailer"}}
			listings := []domain.Listing{{ID: "l", ProductID: "p/1", RetailerID: "r", URL: "https://example.com/?b=2&a=1"}}
			if empty {
				retailers = nil
				listings = nil
			}
			s := catalogStub{
				listRetailers: func(ctx context.Context, n int) ([]domain.Retailer, error) {
					if n != test.limit {
						t.Fatal(n)
					}
					return retailers, nil
				},
				getProduct: func(ctx context.Context, id string) (domain.Product, error) {
					if id != "p/1" {
						t.Fatal(id)
					}
					return domain.Product{ID: id}, nil
				},
				listListings: func(ctx context.Context, id string, n int) ([]domain.Listing, error) {
					if id != "p/1" || n != test.limit {
						t.Fatalf("id=%s limit=%d", id, n)
					}
					return listings, nil
				},
			}
			r := catalogRequest(t, s, "GET", "/retailers"+test.query, "", 200, "")
			l := catalogRequest(t, s, "GET", "/products/p%2F1/listings"+test.query, "", 200, "")
			if empty {
				if string(r) != "[]" || string(l) != "[]" {
					t.Fatal("empty lists must be arrays")
				}
				continue
			}
			var gotR []retailerJSON
			var gotL []listingJSON
			if err := json.Unmarshal(r, &gotR); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(l, &gotL); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(gotR, []retailerJSON{{ID: "r", Name: "Retailer"}}) || !reflect.DeepEqual(gotL, []listingJSON{listingResponse(listings[0])}) {
				t.Fatalf("lists=%v %v", gotR, gotL)
			}
		}
	}
	for _, path := range []string{"/retailers", "/products/p/listings"} {
		for _, query := range []string{"0", "-1", "101", "abc", "", "1.5", "9999999999999999999999", "1&limit=2", "%zz"} {
			catalogRequest(t, catalogStub{}, "GET", path+"?limit="+query, "", 400, "invalid_limit")
		}
	}
}
