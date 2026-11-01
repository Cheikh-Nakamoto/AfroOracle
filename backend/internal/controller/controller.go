// Package controller holds the HTTP handlers. Each handler decodes and
// validates the request, calls model functions and the store, and renders via
// view. processSubmit (transcribe → extract → proofScore → store) is shared by
// handleSubmit and handleWhatsapp.
package controller

import (
	"sync"

	"afrioracle/internal/model"
)

// Controllers bundles the handlers with their store.
type Controllers struct {
	Store    model.Store
	Registry *model.Registry

	// consensusMu serialises consensus runs so snapshot, publish and commit
	// are atomic: concurrent calls cannot double-publish or double-reward.
	consensusMu sync.Mutex
}

func NewControllers(s model.Store, reg *model.Registry) *Controllers {
	return &Controllers{Store: s, Registry: reg}
}
