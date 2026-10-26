package controller

import "net/http"

func (c *Controllers) HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<!doctype html><title>AfriOracle</title><h1>AfriOracle dashboard</h1><p>See <code>frontend/</code> for the Next.js dashboard.</p>"))
}
