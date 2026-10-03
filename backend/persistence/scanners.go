package persistence

import (
	"database/sql"
	"github.com/leaf482/price-terminal/backend/domain"
)

// rowScanner is satisfied by both sql.Row and sql.Rows.
type rowScanner interface{ Scan(...any) error }

func scanProduct(row rowScanner) (domain.Product, error) {
	var p domain.Product
	if err := row.Scan(&p.ID, &p.Name, &p.Brand, &p.Model, &p.Archived); err != nil {
		return p, err
	}
	return p, p.Validate()
}
func scanRetailer(row rowScanner) (domain.Retailer, error) {
	var r domain.Retailer
	if err := row.Scan(&r.ID, &r.Name); err != nil {
		return r, err
	}
	return r, r.Validate()
}

// The last selected column must be NOT tracking_enabled.
func scanListing(row rowScanner) (domain.Listing, error) {
	var l domain.Listing
	if err := row.Scan(&l.ID, &l.ProductID, &l.RetailerID, &l.URL, &l.RetailerProductID, &l.TrackingDisabled); err != nil {
		return l, err
	}
	return l, l.Validate()
}

// observationFromAmounts reconstructs the same four price roles in storage
// order: MSRP, retailer list, sale, offer. NULL remains absent; zero is present.
// Callers reconstruct their timestamp before passing it here. The domain
// constructor remains responsible for validating the complete observation.
func observationFromAmounts(input domain.PriceObservationInput, currency sql.NullString, amounts [4]sql.NullInt64) (domain.PriceObservation, error) {
	fields := []**domain.Money{&input.MSRP, &input.RetailerListPrice, &input.SalePrice, &input.OfferPrice}
	for i, amount := range amounts {
		if amount.Valid {
			*fields[i] = &domain.Money{MinorUnits: amount.Int64, Currency: domain.Currency(currency.String)}
		}
	}
	return domain.NewPriceObservation(input)
}
