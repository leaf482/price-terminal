package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leaf482/price-terminal/backend/domain"
	"github.com/leaf482/price-terminal/backend/persistence"
	"net/http"
	"net/url"
	"time"
)

type promotionStore interface {
	GetListing(context.Context, string) (domain.Listing, error)
	CurrentListing(context.Context, string) (persistence.CurrentListing, error)
	InsertPromotion(context.Context, domain.Promotion) error
	GetPromotion(context.Context, string, string) (domain.Promotion, error)
	RelevantPromotions(context.Context, string, time.Time) ([]domain.Promotion, bool, error)
}
type promotionMoney struct {
	MinorUnits int64           `json:"minor_units"`
	Currency   domain.Currency `json:"currency"`
}

func promotionMoneyJSON(m *domain.Money) *promotionMoney {
	if m == nil {
		return nil
	}
	return &promotionMoney{m.MinorUnits, m.Currency}
}

type promotionJSON struct {
	ID          string          `json:"id"`
	ListingID   string          `json:"listing_id"`
	Source      string          `json:"source"`
	ObservedAt  time.Time       `json:"observed_at"`
	StartsAt    *time.Time      `json:"starts_at,omitempty"`
	EndsAt      *time.Time      `json:"ends_at,omitempty"`
	Kind        string          `json:"kind"`
	Amount      *promotionMoney `json:"amount,omitempty"`
	BasisPoints *int64          `json:"basis_points,omitempty"`
	Requirement string          `json:"requirement"`
	Stacking    string          `json:"stacking"`
	Terms       string          `json:"terms"`
}

func promotionResponse(p domain.Promotion) promotionJSON {
	return promotionJSON{p.ID, p.ListingID, p.Source, p.ObservedAt, p.StartsAt, p.EndsAt, p.Kind, promotionMoneyJSON(p.Amount), p.BasisPoints, p.Requirement, p.Stacking, p.Terms}
}

type promotionAPI struct {
	store promotionStore
	now   func() time.Time
}

func registerPromotionRoutes(mux *http.ServeMux, store promotionStore) {
	a := promotionAPI{store, time.Now}
	mux.HandleFunc("POST /listings/{id}/promotions", a.create)
	mux.HandleFunc("GET /listings/{id}/promotions", a.list)
	mux.HandleFunc("GET /listings/{id}/effective-price", a.effective)
}
func (a promotionAPI) create(w http.ResponseWriter, r *http.Request) {
	// Override only the request amount: nil minor_units means omitted or null,
	// while a non-nil pointer to zero is an intentionally supplied amount.
	// Keep the response DTO and its numeric representation unchanged.
	var body *struct {
		promotionJSON
		Amount *struct {
			MinorUnits *int64          `json:"minor_units"`
			Currency   domain.Currency `json:"currency"`
		} `json:"amount"`
	}
	if err := decodeCatalogBody(w, r, &body); err != nil || body == nil {
		productError(w, 400, "invalid_request", "expected promotion evidence")
		return
	}
	id := r.PathValue("id")
	if body.ListingID != "" && body.ListingID != id {
		productError(w, 400, "invalid_promotion", "listing mismatch")
		return
	}
	p := domain.Promotion{ID: body.ID, ListingID: id, Source: body.Source, ObservedAt: body.ObservedAt, StartsAt: body.StartsAt, EndsAt: body.EndsAt, Kind: body.Kind, BasisPoints: body.BasisPoints, Requirement: body.Requirement, Stacking: body.Stacking, Terms: body.Terms}
	if body.Amount != nil {
		if body.Amount.MinorUnits == nil {
			productError(w, 400, "invalid_promotion", "amount.minor_units must be an explicit integer")
			return
		}
		p.Amount = &domain.Money{MinorUnits: *body.Amount.MinorUnits, Currency: body.Amount.Currency}
	}
	if err := p.Validate(); err != nil {
		productError(w, 400, "invalid_promotion", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.InsertPromotion(ctx, p); err != nil {
		var pg *pgconn.PgError
		switch {
		case errors.As(err, &pg) && pg.Code == "23505":
			productError(w, 409, "duplicate_promotion", "evidence ID already exists")
		case errors.As(err, &pg) && pg.Code == "23503":
			productError(w, 404, "listing_not_found", "listing not found")
		default:
			productError(w, 500, "internal_error", "unable to store promotion")
		}
		return
	}
	writeJSON(w, 201, map[string]any{"data": promotionResponse(p)})
}
func (a promotionAPI) list(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	id := r.PathValue("id")
	if _, err := a.store.GetListing(ctx, id); err != nil {
		catalogReadError(w, err, "listing")
		return
	}
	promotions, more, err := a.store.RelevantPromotions(ctx, id, a.now().UTC())
	if err != nil {
		productError(w, 500, "internal_error", "unable to read promotions")
		return
	}
	result := make([]promotionJSON, 0, len(promotions))
	for _, p := range promotions {
		result = append(result, promotionResponse(p))
	}
	writeJSON(w, 200, map[string]any{"data": map[string]any{"promotions": result, "truncated": more}})
}
func (a promotionAPI) effective(w http.ResponseWriter, r *http.Request) {
	q, err := url.ParseQuery(r.URL.RawQuery)
	ids := q["promotion_id"]
	valid := err == nil && len(ids) > 0 && len(ids) <= 2 && q.Get("scenario") == "selected" && len(q["scenario"]) == 1
	for key, values := range q {
		if key != "promotion_id" && (key != "scenario" && key != "member" && key != "eligible" || len(values) != 1) {
			valid = false
		}
	}
	if len(ids) == 2 && ids[0] == ids[1] {
		valid = false
	}
	for _, id := range ids {
		if id == "" {
			valid = false
		}
	}
	scenario := domain.PromotionScenario{Member: "unknown", Eligible: "unknown"}
	if v, ok := q["member"]; ok && len(v) == 1 {
		scenario.Member = v[0]
	}
	if v, ok := q["eligible"]; ok && len(v) == 1 {
		scenario.Eligible = v[0]
	}
	if !valid || scenario.Validate() != nil {
		productError(w, 400, "invalid_scenario", "select scenario=selected and one or two promotion_id values; member/eligible are yes, no, or unknown")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	id := r.PathValue("id")
	now := a.now().UTC()
	current, err := a.store.CurrentListing(ctx, id)
	if err != nil {
		catalogReadError(w, err, "listing")
		return
	}
	selected := make([]domain.Promotion, 0, len(ids))
	evidence := make([]promotionJSON, 0, len(ids))
	for _, pid := range ids {
		p, err := a.store.GetPromotion(ctx, id, pid)
		if err != nil {
			catalogReadError(w, err, "promotion")
			return
		}
		selected = append(selected, p)
		evidence = append(evidence, promotionResponse(p))
	}
	result := domain.EffectivePrice{Status: "unavailable", Reason: "missing_observation"}
	var observation *observationJSON
	if current.Observation != nil {
		observation = observationResponse(*current.Observation)
		result, err = domain.CalculateEffectivePrice(*current.Observation, selected, scenario, now)
		if err != nil {
			productError(w, 500, "internal_error", "unable to calculate effective price")
			return
		}
	}
	writeJSON(w, 200, map[string]any{"data": map[string]any{
		"listing_id": id, "status": result.Status, "reason": result.Reason, "rule": "selected-promotions-v1", "calculated_at": now, "observation": observation, "price_basis": result.Basis, "scenario": scenario, "promotions": evidence,
		"base": promotionMoneyJSON(result.Base), "immediate_discount": promotionMoneyJSON(result.ImmediateDiscount), "immediate_payable": promotionMoneyJSON(result.Payable), "potential_cashback": promotionMoneyJSON(result.Cashback), "potential_net": promotionMoneyJSON(result.Net),
		"exclusions": []string{"tax", "shipping"}, "assumptions": []string{"Selected discounts are additional to the observed price; no live price or promotion verification.", "Eligibility is an explicit caller assumption, not a verified user profile.", "Validity uses only recorded bounds; missing bounds are not proof of continued availability.", "Cashback is potential, never guaranteed; observed stock and timestamp remain relevant."},
	}})
}
