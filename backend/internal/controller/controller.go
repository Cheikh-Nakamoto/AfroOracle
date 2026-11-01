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

	// WA is the WhatsApp Cloud API seam (media download, replies); nil when
	// WHATSAPP_TOKEN is unset.
	WA WAClient

	// consensusMu serialises consensus runs so snapshot, publish and commit
	// are atomic: concurrent calls cannot double-publish or double-reward.
	consensusMu sync.Mutex

	// waSeen dedupes WhatsApp redeliveries by wamid (in memory).
	waMu   sync.Mutex
	waSeen map[string]bool
	// waPhotos holds a sender's latest bare photo (base64) until their next message.
	waPhotos map[string]string
}

func NewControllers(s model.Store, reg *model.Registry) *Controllers {
	return &Controllers{Store: s, Registry: reg}
}
