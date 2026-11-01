package controller

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"afrioracle/internal/view"
)

// DefaultMaxBody caps request bodies (photos arrive base64-encoded in JSON).
const DefaultMaxBody int64 = 8 << 20

// NewRouter wires every route with CORS, a body size cap and consistent
// JSON error shapes ({"error": "..."}) for wrong methods and unknown routes.
func NewRouter(c *Controllers, maxBody int64) http.Handler {
	mux := http.NewServeMux()
	api := func(path string, h http.HandlerFunc, methods ...string) {
		mux.HandleFunc(path, Cors(allow(methods, limitBody(maxBody, h))))
	}
	api("/api/submit", c.authed(c.HandleSubmit), http.MethodPost)
	api("/api/consensus", c.authed(c.HandleConsensus), http.MethodPost)
	api("/api/state", c.HandleState, http.MethodGet)
	api("/api/mode", c.HandleMode, http.MethodGet)
	api("/api/ask", c.HandleAsk, http.MethodGet)
	mux.HandleFunc("/webhook/whatsapp", allow([]string{http.MethodGet, http.MethodPost}, limitBody(maxBody, c.HandleWhatsapp)))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			view.WriteError(w, http.StatusNotFound, "not found")
			return
		}
		c.HandleRoot(w, r)
	})
	return mux
}

// allow rejects methods outside the list with 405 and an Allow header.
func allow(methods []string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, m := range methods {
			if r.Method == m || (m == http.MethodGet && r.Method == http.MethodHead) {
				h(w, r)
				return
			}
		}
		w.Header().Set("Allow", strings.Join(methods, ", "))
		view.WriteError(w, http.StatusMethodNotAllowed, strings.Join(methods, "/")+" only")
	}
}

// limitBody reads at most max bytes; anything larger gets 413 before the
// handler runs. The body is replaced so handlers decode it as usual.
func limitBody(max int64, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			data, err := io.ReadAll(io.LimitReader(r.Body, max+1))
			if err != nil {
				view.WriteError(w, http.StatusBadRequest, "unreadable body")
				return
			}
			if int64(len(data)) > max {
				view.WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(data))
		}
		h(w, r)
	}
}
