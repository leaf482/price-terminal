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
	server := newHTTPServer(ctx, address, newHandler(db.PingContext, store, store, currentAPI{store: store, status: runtime.Status, collect: runtime.Collect, bulkCollect: runtime.CollectBulk, maxAge: maxAge, now: time.Now}))

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

func newHTTPServer(ctx context.Context, address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       60 * time.Second,
		// Bulk collection can take 20 configured per-Listing timeouts. A fixed global
		// WriteTimeout would truncate valid partial results; handlers retain deadlines.
	}
}
