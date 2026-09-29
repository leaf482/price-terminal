package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/leaf482/price-terminal/backend/persistence"
)

func TestCollectionConfiguration(t *testing.T) {
	store := persistence.New(nil) // configuration performs no database calls
	valid := `[{"listing_id":"a","observation":{"observed_at":"2026-09-29T12:00:00Z","source":"fixture","stock":"unknown","currency":"JPY","offer_price":0}}]`
	for _, tt := range []struct {
		name, body string
		valid      bool
	}{
		{"price", valid, true},
		{"stock", `[{"listing_id":"a","observation":{"observed_at":"2026-09-29T12:00:00Z","source":"fixture","stock":"unknown"}}]`, true},
		{"failure", `[{"listing_id":"a","fail":true}]`, true},
		{"duplicates", `[{"listing_id":"a","fail":true},{"listing_id":"a","fail":true}]`, false},
		{"invalid price", strings.Replace(valid, `"JPY"`, `"EUR"`, 1), false},
		{"missing timestamp", strings.Replace(valid, `2026-09-29T12:00:00Z`, `bad`, 1), false},
		{"unknown field", `[{"listing_id":"a","failure":true}]`, false},
		{"missing response", `[{"listing_id":"a"}]`, false},
		{"extra JSON", valid + ` []`, false},
		{"oversized", strings.Repeat(" ", 1024*1024) + valid, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tt.body), 0600); err != nil {
				t.Fatal(err)
			}
			r, err := loadCollector(path, store, time.Second, time.Second)
			if (err == nil) != tt.valid {
				t.Fatal(err)
			}
			if tt.valid && !r.Status("a").Active {
				t.Fatal("not active")
			}
		})
	}
	if _, err := loadCollector("collection.example.json", store, time.Second, time.Second); err != nil {
		t.Fatal("example", err)
	}
	r, err := loadCollector("", store, time.Second, time.Second)
	if err != nil || r.Status("a").Active {
		t.Fatal("disabled", err)
	}
}

func TestDurationConfiguration(t *testing.T) {
	for _, tt := range []struct {
		value string
		valid bool
	}{{"", true}, {"2s", true}, {"-1s", false}, {"0", false}, {"bad", false}} {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("TEST_DURATION", tt.value)
			_, err := durationEnv("TEST_DURATION", time.Minute)
			if (err == nil) != tt.valid {
				t.Fatal(err)
			}
		})
	}
}
