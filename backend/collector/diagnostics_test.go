package collector_test

import (
	"bytes"
	"context"
	"github.com/leaf482/price-terminal/backend/collector"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestFailureCountAndSafeLogs(t *testing.T) {
	var output bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	defer slog.SetDefault(old)
	s, p := fixtures(t)
	r, _ := collector.New(s, []collector.Target{{ListingID: "a", Provider: p}, {ListingID: "bad", Provider: p}}, time.Hour, time.Second)
	s.fail = true
	for i := uint64(1); i <= 2; i++ {
		if r.Collect(context.Background(), "a") == nil || r.Status("a").ConsecutiveFailures != i {
			t.Fatal(r.Status("a"))
		}
	}
	s.fail = false
	if err := r.Collect(context.Background(), "a"); err != nil || r.Status("a").ConsecutiveFailures != 0 {
		t.Fatal(err, r.Status("a"))
	}
	_ = r.Collect(context.Background(), "bad")
	logs := output.String()
	for _, event := range []string{"collection_start", "collection_result", "ingestion_failure", "provider", "persistence"} {
		if !strings.Contains(logs, event) {
			t.Fatal("missing event", event)
		}
	}
	if strings.Contains(logs, "secret") {
		t.Fatal("sensitive error logged")
	}
}
