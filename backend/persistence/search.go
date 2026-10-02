package persistence

import (
	"context"
	"errors"
	"github.com/leaf482/price-terminal/backend/domain"
	"strings"
	"unicode/utf8"
)

const SearchLimit = 20

var ErrInvalidSearch = errors.New("query must be valid text of at most 200 characters without NUL")

type CatalogSearch struct {
	Products  []domain.Product
	Retailers []domain.Retailer
	Listings  []domain.Listing
}

func ValidateSearch(query string) error {
	if !utf8.ValidString(query) || utf8.RuneCountInString(query) > 200 || strings.ContainsRune(query, 0) {
		return ErrInvalidSearch
	}
	return nil
}

// strpos treats %, _, quotes and backslashes as literal text. One statement,
// three bounded entity scans, no price reads or queries per matching record.
func (s *Store) SearchCatalog(ctx context.Context, query string) (CatalogSearch, error) {
	result := CatalogSearch{Products: []domain.Product{}, Retailers: []domain.Retailer{}, Listings: []domain.Listing{}}
	query = strings.TrimSpace(query)
	if err := ValidateSearch(query); err != nil {
		return result, err
	}
	if query == "" {
		return result, nil
	}
	rows, err := s.db.QueryContext(ctx, `
 SELECT * FROM (
 (SELECT 'product' AS kind,id,name,brand,model,archived,'' AS product_id,'' AS retailer_id,'' AS url,'' AS retailer_product_id,true AS tracking_enabled FROM products
 WHERE strpos(lower(id),lower($1))>0 OR strpos(lower(name),lower($1))>0 OR strpos(lower(brand),lower($1))>0 OR strpos(lower(model),lower($1))>0 ORDER BY id COLLATE "C" LIMIT $2)
 UNION ALL
 (SELECT 'retailer',id,name,'','',false,'','','','',true FROM retailers
 WHERE strpos(lower(id),lower($1))>0 OR strpos(lower(name),lower($1))>0 ORDER BY id COLLATE "C" LIMIT $2)
 UNION ALL
 (SELECT 'listing',id,'','','',false,product_id,retailer_id,url,retailer_product_id,tracking_enabled FROM listings
 WHERE strpos(lower(id),lower($1))>0 OR strpos(lower(url),lower($1))>0 OR strpos(lower(retailer_product_id),lower($1))>0 ORDER BY id COLLATE "C" LIMIT $2)
 ) hits ORDER BY kind COLLATE "C",id COLLATE "C"`, query, SearchLimit)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, id, name, brand, model, productID, retailerID, url, sku string
		var archived, enabled bool
		if err := rows.Scan(&kind, &id, &name, &brand, &model, &archived, &productID, &retailerID, &url, &sku, &enabled); err != nil {
			return result, err
		}
		switch kind {
		case "product":
			p := domain.Product{ID: id, Name: name, Brand: brand, Model: model, Archived: archived}
			if err := p.Validate(); err != nil {
				return result, err
			}
			result.Products = append(result.Products, p)
		case "retailer":
			r := domain.Retailer{ID: id, Name: name}
			if err := r.Validate(); err != nil {
				return result, err
			}
			result.Retailers = append(result.Retailers, r)
		case "listing":
			l := domain.Listing{ID: id, ProductID: productID, RetailerID: retailerID, URL: url, RetailerProductID: sku, TrackingDisabled: !enabled}
			if err := l.Validate(); err != nil {
				return result, err
			}
			result.Listings = append(result.Listings, l)
		}
	}
	return result, rows.Err()
}
