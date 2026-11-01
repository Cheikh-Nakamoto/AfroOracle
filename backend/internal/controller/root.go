package controller

import (
	"net/http"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

func (c *Controllers) HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<!doctype html><title>AfriOracle</title><h1>AfriOracle dashboard</h1><p>See <code>frontend/</code> for the Next.js dashboard.</p>"))
}

// HandleMode reports the active AI and chain modes so operators can see
// whether Muse/SAM are really being called ("real") or mocked ("mock").
func (c *Controllers) HandleMode(w http.ResponseWriter, r *http.Request) {
	ai, chain := model.Mode()
	view.WriteJSON(w, http.StatusOK, map[string]string{"ai": ai, "chain": chain})
}
