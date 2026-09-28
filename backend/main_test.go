package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	healthHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); got != "ok\n" {
		t.Errorf("body = %q, want %q", got, "ok\n")
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Errorf("content type = %q, want text/plain; charset=utf-8", got)
	}
}

func TestRoutes(t *testing.T) {
	for _, test := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/healthz", http.StatusOK},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
		{http.MethodGet, "/readyz", http.StatusOK},
		{http.MethodPost, "/readyz", http.StatusMethodNotAllowed},
		{http.MethodGet, "/missing", http.StatusNotFound},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			newHandler(func(context.Context) error { return nil }).ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
			if recorder.Code != test.status {
				t.Errorf("status = %d, want %d", recorder.Code, test.status)
			}
		})
	}
}

func TestReadiness(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"available", nil, http.StatusOK, "ready\n"},
		{"unavailable", errors.New("internal DB error with secret credentials"), http.StatusServiceUnavailable, "not ready\n"},
		{"timeout", context.DeadlineExceeded, http.StatusServiceUnavailable, "not ready\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			ping := func(ctx context.Context) error {
				called = true
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 2*time.Second {
					t.Error("database ping must have a deadline within two seconds")
				}
				return test.err
			}
			recorder := httptest.NewRecorder()
			newHandler(ping).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if !called || recorder.Code != test.status || recorder.Body.String() != test.body {
				t.Errorf("called=%v status=%d body=%q; want true, %d, %q", called, recorder.Code, recorder.Body.String(), test.status, test.body)
			}
		})
	}
}

func TestHealthDoesNotPingDatabase(t *testing.T) {
	ping := func(context.Context) error {
		t.Fatal("liveness must not call the database")
		return nil
	}
	recorder := httptest.NewRecorder()
	newHandler(ping).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "ok\n" {
		t.Errorf("status=%d body=%q; want 200 and ok", recorder.Code, recorder.Body.String())
	}
}

func TestReadinessPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ping := func(ctx context.Context) error {
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Error("request cancellation was not propagated to the database")
		}
		return ctx.Err()
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil).WithContext(ctx)
	newHandler(ping).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Errorf("status=%d, want 503", recorder.Code)
	}
}
