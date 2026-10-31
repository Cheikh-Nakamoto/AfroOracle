// Package controller holds the HTTP handlers. Each handler decodes and
// validates the request, calls model functions and the store, and renders via
// view. processSubmit (transcribe → extract → proofScore → store) is shared by
// handleSubmit and handleWhatsapp.
package controller

import (
	"context"
	"sync"
	"time"

	"afrioracle/internal/model"
)

type WhatsAppMediaDownloader interface {
	Download(context.Context, string) ([]byte, string, error)
}

type WhatsAppAudioTranscriber interface {
	Transcribe(context.Context, []byte, string) (string, error)
}

// Controllers bundles the handlers with their store.
type Controllers struct {
	Store               model.Store
	Nodes               *NodeRegistry
	WhatsAppAppSecret   string
	WhatsAppVerifyToken string
	WhatsAppMedia       WhatsAppMediaDownloader
	WhatsAppTranscriber WhatsAppAudioTranscriber
	whatsappMu          sync.Mutex
	whatsappSeen        map[string]time.Time
}

func NewControllers(s model.Store) *Controllers {
	return &Controllers{Store: s, Nodes: EmptyNodeRegistry(), whatsappSeen: make(map[string]time.Time)}
}
