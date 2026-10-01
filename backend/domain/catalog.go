package domain

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Product identifies a retailer-independent item or variant. Descriptive fields
// are optional; no retailer URL, price, or availability belongs here.
type Product struct {
	Archived bool
	ID       string
	Name     string
	Brand    string
	Model    string
}

func (p Product) Validate() error {
	return requireText("product ID", p.ID)
}

// Retailer identifies a store. Seller/marketplace modeling is deferred.
type Retailer struct {
	ID   string
	Name string
}

func (r Retailer) Validate() error {
	return requireText("retailer ID", r.ID)
}

// Listing identifies a retailer-specific page and its explicitly selected
// Product. RetailerProductID can preserve an available item/variant reference.
// Validate checks references are supplied, not whether those records exist.
// URL canonicalization, uniqueness, and matching are outside this package.
type Listing struct {
	ID                string
	ProductID         string
	RetailerID        string
	URL               string
	RetailerProductID string
	// Negative storage makes existing Go callers default to enabled as well.
	TrackingDisabled bool
}

func (l Listing) TrackingEnabled() bool { return !l.TrackingDisabled }

var ErrTrackingDisabled = errors.New("listing tracking is disabled")

func (l Listing) Validate() error {
	for _, field := range []struct{ name, value string }{
		{"listing ID", l.ID},
		{"product ID", l.ProductID},
		{"retailer ID", l.RetailerID},
		{"listing URL", l.URL},
	} {
		if err := requireText(field.name, field.value); err != nil {
			return err
		}
	}
	parsed, err := url.Parse(l.URL)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("listing URL must be an absolute HTTP or HTTPS URL")
	}
	return nil
}

func requireText(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}
