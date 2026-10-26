// Package view renders HTTP responses. The wire shapes are frozen by
// scripts/demo.sh and the spec invariants, so responses reuse the exact field
// names and key sets the flat backend served (model types with Go field names
// for submission/proof/outcome payloads, lowercase keys for envelopes).
package view

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
