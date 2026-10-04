package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
)

// Shared wire-format helpers leave catalog limits and persistence error mapping
// to the handlers; they only decode requests and format responses.
var errTrailingJSON = errors.New("expected one JSON object")

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func productError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

// requestJSONDecoder preserves the shared size bound and unknown-field policy.
func requestJSONDecoder(w http.ResponseWriter, r *http.Request) *json.Decoder {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder
}

func requireJSONEnd(decoder *json.Decoder) error {
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errTrailingJSON
	}
	return nil
}

// Catalog bodies follow the Product API's size and single-object rules.
// Callers retain responsibility for null bodies and required fields.
func decodeCatalogBody(w http.ResponseWriter, r *http.Request, dst any) error {
	decoder := requestJSONDecoder(w, r)
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	return requireJSONEnd(decoder)
}

// listLimit parses only the limit values; callers retain their query-error policy.
func listLimit(values []string, maximum int) (int, bool) {
	limit := 50
	var err error
	if len(values) > 0 {
		limit, err = strconv.Atoi(values[0])
	}
	return limit, err == nil && len(values) <= 1 && limit >= 1 && limit <= maximum
}
