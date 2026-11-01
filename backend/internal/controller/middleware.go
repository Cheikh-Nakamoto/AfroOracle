package controller

import (
	"net/http"
	"strings"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

func Cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

// authed requires "Authorization: Bearer <token>" for a registered node (401
// otherwise) and passes the node to the handler.
func (c *Controllers) authed(h func(http.ResponseWriter, *http.Request, model.Node)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		var n model.Node
		if ok && c.Registry != nil {
			n, ok = c.Registry.Authenticate(strings.TrimSpace(tok))
		}
		if !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			view.WriteError(w, http.StatusUnauthorized, "valid node credentials required")
			return
		}
		h(w, r, n)
	}
}
