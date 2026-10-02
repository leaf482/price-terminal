package persistence

import (
	"context"
	"database/sql"
	"github.com/leaf482/price-terminal/backend/domain"
	"time"
)

type AlertContext struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	RetailerID   string `json:"retailer_id"`
	RetailerName string `json:"retailer_name"`
}
type OverviewAlert struct {
	Alert domain.PriceAlert `json:"alert"`
	AlertContext
	LastTriggeredAt *time.Time `json:"last_triggered_at"`
}
type OverviewEvent struct {
	AlertID       string          `json:"alert_id"`
	ListingID     string          `json:"listing_id"`
	ObservationID string          `json:"observation_id"`
	Kind          string          `json:"kind"`
	MinorUnits    int64           `json:"minor_units"`
	Currency      domain.Currency `json:"currency"`
	TriggeredAt   time.Time       `json:"triggered_at"`
	AlertContext
}
type AlertOverview struct {
	Alerts          []OverviewAlert `json:"alerts"`
	Events          []OverviewEvent `json:"events"`
	AlertsTruncated bool            `json:"alerts_truncated"`
	EventsTruncated bool            `json:"events_truncated"`
}
type overviewAlertScanner struct {
	row     interface{ Scan(...any) error }
	context *AlertContext
	last    *sql.NullTime
}

func (s overviewAlertScanner) Scan(dest ...any) error {
	c := s.context
	return s.row.Scan(append(dest, &c.ProductID, &c.ProductName, &c.RetailerID, &c.RetailerName, s.last)...)
}

// Two bounded reads independent of alert count. These read stored events only;
// no evaluation or modification of historical facts happens during the overview.
func (s *Store) AlertOverview(ctx context.Context) (AlertOverview, error) {
	out := AlertOverview{Alerts: []OverviewAlert{}, Events: []OverviewEvent{}}
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,a.listing_id,a.kind,a.currency,a.threshold_minor_units,a.drop_basis_points,a.enabled,a.require_in_stock,p.id,p.name,r.id,r.name,(SELECT max(e.triggered_at) FROM price_alert_events e WHERE e.alert_id=a.id)
 FROM (SELECT * FROM price_alerts ORDER BY id COLLATE "C" LIMIT 101) a JOIN listings l ON l.id=a.listing_id JOIN products p ON p.id=l.product_id JOIN retailers r ON r.id=l.retailer_id ORDER BY a.id COLLATE "C"`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var value OverviewAlert
		var last sql.NullTime
		value.Alert, err = scanAlert(overviewAlertScanner{rows, &value.AlertContext, &last})
		if err != nil {
			rows.Close()
			return out, err
		}
		if last.Valid {
			at := last.Time.UTC()
			value.LastTriggeredAt = &at
		}
		out.Alerts = append(out.Alerts, value)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	if len(out.Alerts) > 100 {
		out.AlertsTruncated = true
		out.Alerts = out.Alerts[:100]
	}
	rows, err = s.db.QueryContext(ctx, `SELECT e.alert_id,e.listing_id,e.observation_id,e.kind,e.minor_units,e.currency,e.triggered_at,p.id,p.name,r.id,r.name
 FROM (SELECT * FROM price_alert_events ORDER BY triggered_at DESC,alert_id COLLATE "C",observation_id COLLATE "C" LIMIT 21) e JOIN listings l ON l.id=e.listing_id JOIN products p ON p.id=l.product_id JOIN retailers r ON r.id=l.retailer_id ORDER BY e.triggered_at DESC,e.alert_id COLLATE "C",e.observation_id COLLATE "C"`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var v OverviewEvent
		if err := rows.Scan(&v.AlertID, &v.ListingID, &v.ObservationID, &v.Kind, &v.MinorUnits, &v.Currency, &v.TriggeredAt, &v.ProductID, &v.ProductName, &v.RetailerID, &v.RetailerName); err != nil {
			return out, err
		}
		v.TriggeredAt = v.TriggeredAt.UTC()
		out.Events = append(out.Events, v)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Events) > 20 {
		out.EventsTruncated = true
		out.Events = out.Events[:20]
	}
	return out, nil
}
