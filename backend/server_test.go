package main

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestHTTPServerBounds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := newHTTPServer(ctx, "127.0.0.1:0", http.HandlerFunc(healthHandler))
	if s.ReadHeaderTimeout != 5*time.Second || s.ReadTimeout != 30*time.Second || s.IdleTimeout != time.Minute {
		t.Fatal("missing connection bounds")
	}
	if s.WriteTimeout != 0 {
		t.Fatal("fixed write timeout can truncate configurable bulk collection")
	}
	cancel()
	if s.BaseContext(nil).Err() != context.Canceled {
		t.Fatal("shutdown cancellation lost")
	}
}
