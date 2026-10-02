package persistence

import (
	"math"
	"testing"
)

func TestPriceChangeValues(t *testing.T) {
	for _, c := range []struct {
		before, after             int64
		delta, percent, direction string
	}{
		{100, 75, "-25", "-25.00", "decreased"}, {100, 125, "25", "25.00", "increased"}, {100, 100, "0", "0.00", "unchanged"},
		{100, 0, "-100", "-100.00", "decreased"}, {0, 100, "100", "", "increased"}, {0, 0, "0", "", "unchanged"}, {3, 2, "-1", "-33.33", "decreased"}, {math.MaxInt64, 0, "-9223372036854775807", "-100.00", "decreased"},
	} {
		delta, p, direction := changeValues(c.before, c.after)
		if delta != c.delta || direction != c.direction || (c.percent == "" && p != nil) || (c.percent != "" && (p == nil || *p != c.percent)) {
			t.Fatal(c, delta, p, direction)
		}
	}
}
