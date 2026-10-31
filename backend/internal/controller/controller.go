// Package controller holds the HTTP handlers. Each handler decodes and
// validates the request, calls model functions and the store, and renders via
// view. processSubmit (transcribe → extract → proofScore → store) is shared by
// handleSubmit and handleWhatsapp.
package controller

import "afrioracle/internal/model"

// Controllers bundles the handlers with their store.
type Controllers struct {
	Store model.Store
	Nodes *NodeRegistry
}

func NewControllers(s model.Store) *Controllers {
	return &Controllers{Store: s, Nodes: EmptyNodeRegistry()}
}
