package main

import (
	"context"
	"net/http"
	"time"
)

type archiveStore interface {
	SetProductArchived(context.Context, string, bool) error
}

func archiveHandler(store archiveStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body *struct {
			Archived *bool `json:"archived"`
		}
		if err := decodeCatalogBody(w, r, &body); err != nil || body == nil || body.Archived == nil {
			productError(w, 400, "invalid_archive", "archived must be an explicit boolean")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := store.SetProductArchived(ctx, r.PathValue("id"), *body.Archived); err != nil {
			catalogReadError(w, err, "product")
			return
		}
		writeJSON(w, 200, map[string]any{"data": map[string]any{"product_id": r.PathValue("id"), "archived": *body.Archived}})
	}
}
