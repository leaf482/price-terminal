package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/leaf482/price-terminal/backend/ingestion"
	"github.com/leaf482/price-terminal/backend/persistence"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "ok\n")
}

func readinessHandler(ping func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := ping(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, "ready\n")
	}
}

func newHandler(ping func(context.Context) error, products productStore, catalog catalogStore, prices ...currentAPI) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", readinessHandler(ping))
	api := productAPI{store: products}
	mux.HandleFunc("POST /products", api.create)
	mux.HandleFunc("GET /products/{id}", api.get)
	mux.HandleFunc("GET /products", api.list)
	registerCatalogRoutes(mux, catalog)
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
		registerCurrentRoutes(mux, prices[0])
		if dashboard, ok := catalog.(dashboardStore); ok {
			mux.HandleFunc("GET /dashboard", dashboardHandler(dashboard, prices[0]))
		}
	}
	return mux
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	address, err := listenAddress(os.Getenv("LISTEN_ADDR"))
	if err != nil {
		return err
	}
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	store := persistence.New(db)
	interval, err := durationEnv("COLLECTION_INTERVAL", time.Minute)
	if err != nil {
		return err
	}
	timeout, err := durationEnv("COLLECTION_TIMEOUT", 10*time.Second)
	if err != nil {
		return err
	}
	maxAge, err := durationEnv("PRICE_MAX_AGE", 15*time.Minute)
	if err != nil {
		return err
	}
	runtime, err := loadCollector(os.Getenv("COLLECTOR_CONFIG"), store, interval, timeout)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	collectorDone := make(chan error, 1)
	go func() { collectorDone <- runtime.Run(ctx) }()
	server := &http.Server{
		Addr:              address,
		Handler:           newHandler(db.PingContext, store, store, currentAPI{store: store, status: runtime.Status, collect: runtime.Collect, maxAge: maxAge, now: time.Now}),
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on http://%s", server.Addr)
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.ListenAndServe() }()
	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-serverDone:
	}
	stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		_ = server.Close()
	}
	<-collectorDone // Join collection before the deferred database Close.
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	return shutdownErr
}
