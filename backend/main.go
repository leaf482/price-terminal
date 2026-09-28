package main

import (
	"io"
	"log"
	"net/http"
	"time"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "ok\n")
}

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	return mux
}

func main() {
	server := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           newHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on http://%s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
