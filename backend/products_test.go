package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
)

type productStub struct {
	insert func(context.Context, domain.Product) error
	get    func(context.Context, string) (domain.Product, error)
	list   func(context.Context, int) ([]domain.Product, error)
}

func (s productStub) InsertProduct(c context.Context, p domain.Product) error { return s.insert(c, p) }
func (s productStub) GetProduct(c context.Context, id string) (domain.Product, error) {
	return s.get(c, id)
}
func (s productStub) ListProducts(c context.Context, n int) ([]domain.Product, error) {
	return s.list(c, n)
}

func productRequest(t *testing.T, s productStub, method, path, body string, status int) map[string]json.RawMessage {
	t.Helper()
	w := httptest.NewRecorder()
	newHandler(func(context.Context) error { return nil }, s, nil).ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	if w.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, status, w.Body.String())
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Fatal("expected JSON content type")
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response) != 1 {
		t.Fatalf("unexpected envelope: %s", w.Body.String())
	}
	if status >= 400 {
		var failure struct{ Code, Message string }
		if err := json.Unmarshal(response["error"], &failure); err != nil || failure.Code == "" || failure.Message == "" {
			t.Fatalf("invalid error envelope: %s", w.Body.String())
		}
		if strings.Contains(w.Body.String(), "secret") {
			t.Fatal("internal error leaked")
		}
	} else if response["data"] == nil {
		t.Fatal("missing data envelope")
	}
	return response
}

func TestCreateProduct(t *testing.T) {
	want := domain.Product{ID: "p1", Name: "Item", Brand: "Brand", Model: "Model"}
	called := false
	s := productStub{insert: func(ctx context.Context, p domain.Product) error {
		called = true
		if p != want {
			t.Fatalf("product=%+v, want %+v", p, want)
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 2*time.Second {
			t.Fatal("missing bounded DB context")
		}
		return nil
	}}
	response := productRequest(t, s, "POST", "/products", `{"id":"p1","name":"Item","brand":"Brand","model":"Model"}`, 201)
	var got productJSON
	if err := json.Unmarshal(response["data"], &got); err != nil || got != productResponse(want) || !called {
		t.Fatalf("response=%+v, called=%v, err=%v", got, called, err)
	}
	// Descriptive fields are optional according to the domain model.
	productRequest(t, productStub{insert: func(context.Context, domain.Product) error { return nil }}, "POST", "/products", `{"id":"minimal"}`, 201)
}

func TestInvalidProductRequests(t *testing.T) {
	for _, body := range []string{"", `{`, `null`, `[]`, `{"id":1}`, `{}`, `{"id":" "}`, `{"id":"p","price":1}`, `{"id":"p"} {}`, `{"id":"p"} trailing`, `{"id":"p","name":"` + strings.Repeat("x", 65536) + `"}`} {
		t.Run(fmt.Sprintf("body-length-%d", len(body)), func(t *testing.T) {
			// Nil callbacks panic if malformed requests reach persistence.
			productRequest(t, productStub{}, "POST", "/products", body, 400)
		})
	}
}

func TestProductDatabaseErrors(t *testing.T) {
	for _, test := range []struct {
		name, method, path, body string
		err                      error
		status                   int
	}{
		{"duplicate", "POST", "/products", `{"id":"p"}`, fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: "23505", ConstraintName: "products_pkey", Detail: "secret"}), 409},
		{"insert failure", "POST", "/products", `{"id":"p"}`, errors.New("secret database failure"), 500},
		{"other constraint", "POST", "/products", `{"id":"p"}`, &pgconn.PgError{Code: "23514", Detail: "secret"}, 500},
		{"missing", "GET", "/products/missing", "", fmt.Errorf("wrapped: %w", sql.ErrNoRows), 404},
		{"get failure", "GET", "/products/p", "", errors.New("secret database failure"), 500},
		{"list failure", "GET", "/products", "", errors.New("secret database failure"), 500},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := productStub{
				insert: func(context.Context, domain.Product) error { return test.err },
				get:    func(context.Context, string) (domain.Product, error) { return domain.Product{}, test.err },
				list:   func(context.Context, int) ([]domain.Product, error) { return nil, test.err },
			}
			productRequest(t, s, test.method, test.path, test.body, test.status)
		})
	}
}

func TestGetProduct(t *testing.T) {
	want := domain.Product{ID: "a/b", Name: "Item"}
	s := productStub{get: func(ctx context.Context, id string) (domain.Product, error) {
		if id != want.ID {
			t.Fatalf("id=%q", id)
		}
		return want, nil
	}}
	r := productRequest(t, s, "GET", "/products/a%2Fb", "", 200)
	var got productJSON
	if err := json.Unmarshal(r["data"], &got); err != nil || got != productResponse(want) {
		t.Fatalf("got=%+v, err=%v", got, err)
	}
}

func TestListProducts(t *testing.T) {
	for _, test := range []struct {
		query string
		limit int
	}{{"", 50}, {"?limit=1", 1}, {"?limit=100", 100}} {
		t.Run(test.query, func(t *testing.T) {
			called := false
			s := productStub{list: func(ctx context.Context, limit int) ([]domain.Product, error) {
				called = true
				if limit != test.limit {
					t.Fatalf("limit=%d, want %d", limit, test.limit)
				}
				return []domain.Product{{ID: "p1"}}, nil
			}}
			r := productRequest(t, s, "GET", "/products"+test.query, "", 200)
			var got []productJSON
			if err := json.Unmarshal(r["data"], &got); err != nil || len(got) != 1 || got[0].ID != "p1" || !called {
				t.Fatalf("list=%v, err=%v", got, err)
			}
		})
	}
	r := productRequest(t, productStub{list: func(context.Context, int) ([]domain.Product, error) { return nil, nil }}, "GET", "/products", "", 200)
	if string(r["data"]) != "[]" {
		t.Fatalf("empty list=%s", r["data"])
	}
	for _, query := range []string{"0", "-1", "101", "abc", "", "1.5", "9999999999999999999999", "1&limit=2"} {
		productRequest(t, productStub{}, "GET", "/products?limit="+query, "", 400)
	}
}

func TestProductCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := productStub{get: func(ctx context.Context, _ string) (domain.Product, error) {
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Fatal("cancellation not propagated")
		}
		return domain.Product{}, ctx.Err()
	}}
	w := httptest.NewRecorder()
	newHandler(nil, s, nil).ServeHTTP(w, httptest.NewRequest("GET", "/products/p", nil).WithContext(ctx))
	if w.Code != 500 {
		t.Fatalf("status=%d", w.Code)
	}
}
