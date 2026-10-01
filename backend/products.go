package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
)

// This small consumer interface permits handler tests without a live database.
type productStore interface {
	InsertProduct(context.Context, domain.Product) error
	GetProduct(context.Context, string) (domain.Product, error)
	ListProducts(context.Context, int) ([]domain.Product, error)
}

type productAPI struct{ store productStore }

type productJSON struct {
	Archived bool   `json:"archived"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Brand    string `json:"brand"`
	Model    string `json:"model"`
}

func productResponse(p domain.Product) productJSON {
	return productJSON{ID: p.ID, Name: p.Name, Brand: p.Brand, Model: p.Model, Archived: p.Archived}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func productError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func (api productAPI) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var body *struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Brand string `json:"brand"`
		Model string `json:"model"`
	}
	if err := decoder.Decode(&body); err != nil || body == nil {
		productError(w, 400, "invalid_request", "expected a product JSON object")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		productError(w, 400, "invalid_request", "expected exactly one JSON object")
		return
	}
	p := domain.Product{ID: body.ID, Name: body.Name, Brand: body.Brand, Model: body.Model}
	if err := p.Validate(); err != nil {
		productError(w, 400, "invalid_product", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := api.store.InsertProduct(ctx, p); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" && pgError.ConstraintName == "products_pkey" {
			productError(w, 409, "duplicate_product", "product ID already exists")
		} else {
			productError(w, 500, "internal_error", "unable to create product")
		}
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": productResponse(p)})
}

func (api productAPI) get(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	p, err := api.store.GetProduct(ctx, r.PathValue("id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			productError(w, 404, "product_not_found", "product not found")
		} else {
			productError(w, 500, "internal_error", "unable to read product")
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": productResponse(p)})
}

func (api productAPI) list(w http.ResponseWriter, r *http.Request) {
	limit := 50
	query := r.URL.Query()["limit"]
	var err error
	if len(query) > 0 {
		limit, err = strconv.Atoi(query[0])
	}
	if err != nil || len(query) > 1 || limit < 1 || limit > persistence.MaxProductListLimit {
		productError(w, 400, "invalid_limit", "limit must be an integer from 1 to 100")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	products, err := api.store.ListProducts(ctx, limit)
	if err != nil {
		productError(w, 500, "internal_error", "unable to list products")
		return
	}
	data := make([]productJSON, 0, len(products))
	for _, p := range products {
		data = append(data, productResponse(p))
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}
