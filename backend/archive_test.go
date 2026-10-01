package main

import (
	"context"
	"database/sql"
	"errors"
	"github.com/leaf482/price-terminal/backend/collector"
	"github.com/leaf482/price-terminal/backend/domain"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type archiveStub struct {
	calls    int
	id       string
	archived bool
	err      error
}

func (s *archiveStub) SetProductArchived(_ context.Context, id string, v bool) error {
	s.calls++
	s.id = id
	s.archived = v
	return s.err
}
func TestArchiveAPI(t *testing.T) {
	for _, tt := range []struct {
		body string
		err  error
		code int
	}{{`{"archived":true}`, nil, 200}, {`{"archived":false}`, nil, 200}, {`{}`, nil, 400}, {`{"archived":null}`, nil, 400}, {`{"archived":"true"}`, nil, 400}, {`{"archived":true,"id":"other"}`, nil, 400}, {`{"archived":true}`, sql.ErrNoRows, 404}, {`{"archived":true}`, errors.New("secret"), 500}} {
		s := &archiveStub{err: tt.err}
		r := httptest.NewRequest("PATCH", "/products/p/archive", strings.NewReader(tt.body))
		r.SetPathValue("id", "p")
		w := httptest.NewRecorder()
		archiveHandler(s)(w, r)
		if w.Code != tt.code || strings.Contains(w.Body.String(), "secret") {
			t.Fatal(w)
		}
		if tt.code == 400 && s.calls != 0 {
			t.Fatal("invalid write")
		}
		if tt.code == 200 && (s.id != "p" || s.archived != strings.Contains(tt.body, "true")) {
			t.Fatal(s)
		}
	}
	if (domain.Product{ID: "p"}).Archived {
		t.Fatal("default must be active")
	}
}
func TestDashboardArchiveFilter(t *testing.T) {
	for _, query := range []string{"", "?include_archived=true", "?include_archived=false", "?include_archived=bad", "?include_archived=true&include_archived=false"} {
		s := &dashboardStub{products: []domain.Product{{ID: "p"}}}
		api := currentAPI{now: time.Now, status: func(string) collector.Status { return collector.Status{State: "inactive"} }, maxAge: time.Minute}
		w := httptest.NewRecorder()
		dashboardHandler(s, api)(w, httptest.NewRequest("GET", "/dashboard"+query, nil))
		invalid := strings.Contains(query, "bad") || strings.Contains(query, "&")
		if invalid {
			if w.Code != 400 || s.calls != 0 {
				t.Fatal(w)
			}
			continue
		}
		if w.Code != 200 || s.includeArchived != (query == "?include_archived=true") {
			t.Fatal(w, s.includeArchived)
		}
	}
}
