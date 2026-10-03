package main

import (
	"context"
	"net/http"
	"time"

	"github.com/leaf482/price-terminal/backend/ingestion"
)

// newHandler assembles the stdlib routes. Optional store capabilities and
// current-price dependencies retain their existing registration conditions.
func newHandler(ping func(context.Context) error, products productStore, catalog catalogStore, prices ...currentAPI) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", readinessHandler(ping))
	api := productAPI{store: products}
	mux.HandleFunc("POST /products", api.create)
	mux.HandleFunc("GET /products/{id}", api.get)
	mux.HandleFunc("GET /products", api.list)
	registerCatalogRoutes(mux, catalog)
	if home, ok := catalog.(homeStore); ok {
		mux.HandleFunc("GET /overview", homeHandler(home))
	}
	if changes, ok := catalog.(priceChangesStore); ok {
		mux.HandleFunc("GET /price-changes", priceChangesHandler(changes))
	}
	if health, ok := catalog.(collectionHealthStore); ok {
		mux.HandleFunc("GET /collection/overview", collectionHealthHandler(health))
	}
	if overview, ok := catalog.(alertOverviewStore); ok {
		mux.HandleFunc("GET /alerts/overview", alertOverviewHandler(overview))
	}
	if search, ok := catalog.(searchStore); ok {
		mux.HandleFunc("GET /search", searchHandler(search))
	}
	if archive, ok := catalog.(archiveStore); ok {
		mux.HandleFunc("PATCH /products/{id}/archive", archiveHandler(archive))
	}
	if metadata, ok := catalog.(metadataStore); ok {
		registerMetadataRoutes(mux, metadata)
	}
	if exports, ok := catalog.(csvExportStore); ok {
		mux.HandleFunc("GET /listings/{id}/observations/export", csvExportHandler(exports))
	}
	if imports, ok := catalog.(csvStore); ok {
		mux.HandleFunc("POST /listings/{id}/observations/import", csvImportHandler(imports))
	}
	if tracking, ok := catalog.(trackingStore); ok {
		mux.HandleFunc("PATCH /listings/{id}/tracking", trackingHandler(tracking))
	}
	if observations, ok := catalog.(ingestion.Store); ok {
		mux.HandleFunc("POST /listings/{id}/observations", manualObservationHandler(observations))
	}
	if attempts, ok := catalog.(attemptStore); ok {
		registerAttemptRoutes(mux, attempts)
	}
	if quality, ok := catalog.(qualityStore); ok {
		registerQualityRoutes(mux, quality)
	}
	if alerts, ok := catalog.(alertStore); ok {
		registerAlertRoutes(mux, alerts)
	}
	if promotions, ok := catalog.(promotionStore); ok {
		registerPromotionRoutes(mux, promotions)
	}
	if history, ok := catalog.(historyStore); ok {
		mux.HandleFunc("GET /listings/{id}/history", historyHandler(history, time.Now))
	}
	if len(prices) > 0 {
		if overview, ok := catalog.(retailerOverviewStore); ok {
			mux.HandleFunc("GET /retailers/{id}/overview", retailerOverviewHandler(overview, prices[0]))
		}
		registerCurrentRoutes(mux, prices[0])
		if dashboard, ok := catalog.(dashboardStore); ok {
			mux.HandleFunc("GET /dashboard", dashboardHandler(dashboard, prices[0]))
		}
	}
	return mux
}
