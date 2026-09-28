package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
		{http.MethodGet, "/missing", http.StatusNotFound},
	} {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			newHandler().ServeHTTP(recorder, httptest.NewRequest(test.method, test.path, nil))
			if recorder.Code != test.status {
				t.Errorf("status = %d, want %d", recorder.Code, test.status)
			}
		})
	}
}
