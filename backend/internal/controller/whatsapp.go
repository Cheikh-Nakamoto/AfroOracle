package controller

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"
)

const maxWhatsAppWebhookBytes = 1 << 20
const whatsappDedupRetention = 24 * time.Hour

type whatsappWebhook struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []whatsappMessage `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

type whatsappMessage struct {
	ID   string `json:"id"`
	From string `json:"from"`
	Type string `json:"type"`
	Text struct {
		Body string `json:"body"`
	} `json:"text"`
	Audio struct {
		ID       string `json:"id"`
		MIMEType string `json:"mime_type"`
	} `json:"audio"`
}

func (c *Controllers) HandleWhatsapp(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		c.handleWhatsAppVerify(w, r)
	case http.MethodPost:
		c.handleWhatsAppEvent(w, r)
	default:
		http.Error(w, "GET/POST only", http.StatusMethodNotAllowed)
	}
}

func (c *Controllers) handleWhatsAppVerify(w http.ResponseWriter, r *http.Request) {
	if c.WhatsAppVerifyToken == "" {
		http.Error(w, "webhook verification is not configured", http.StatusServiceUnavailable)
		return
	}
	provided := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")
	if provided == "" || challenge == "" || !hmac.Equal([]byte(provided), []byte(c.WhatsAppVerifyToken)) {
		http.Error(w, "verification failed", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, challenge)
}

func (c *Controllers) handleWhatsAppEvent(w http.ResponseWriter, r *http.Request) {
	if c.WhatsAppAppSecret == "" {
		log.Printf("whatsapp webhook dropped: app secret is not configured")
		http.Error(w, "webhook signature verification is not configured", http.StatusServiceUnavailable)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxWhatsAppWebhookBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			log.Printf("whatsapp webhook dropped: body exceeds size limit")
			http.Error(w, "webhook body too large", http.StatusRequestEntityTooLarge)
			return
		}
		log.Printf("whatsapp webhook dropped: could not read body")
		http.Error(w, "could not read webhook body", http.StatusBadRequest)
		return
	}
	if !validWhatsAppSignature(c.WhatsAppAppSecret, body, r.Header.Get("X-Hub-Signature-256")) {
		log.Printf("whatsapp webhook dropped: invalid signature")
		http.Error(w, "invalid webhook signature", http.StatusUnauthorized)
		return
	}

	var event whatsappWebhook
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("whatsapp webhook dropped: invalid JSON")
		http.Error(w, "invalid webhook JSON", http.StatusBadRequest)
		return
	}
	for _, entry := range event.Entry {
		for _, change := range entry.Changes {
			for _, message := range change.Value.Messages {
				if strings.TrimSpace(message.ID) == "" {
					log.Printf("whatsapp webhook dropped: message has no wamid")
					http.Error(w, "message id is required", http.StatusBadRequest)
					return
				}
				if !c.claimWhatsAppMessage(message.ID) {
					log.Printf("whatsapp webhook ignored: duplicate wamid")
					continue
				}
				transcript := message.Text.Body
				if message.Type == "audio" || message.Audio.ID != "" {
					if strings.TrimSpace(message.Audio.ID) == "" {
						log.Printf("whatsapp webhook dropped: audio message has no media ID")
						continue
					}
					var err error
					transcript, err = c.transcribeWhatsAppAudio(r.Context(), message.Audio.ID)
					if err != nil {
						c.releaseWhatsAppMessage(message.ID)
						log.Printf("whatsapp webhook audio processing failed")
						http.Error(w, "audio processing failed", http.StatusBadGateway)
						return
					}
				}
				if strings.TrimSpace(transcript) == "" {
					log.Printf("whatsapp webhook dropped: unsupported or empty message")
					continue
				}
				status, _ := c.processSubmit(SubmitReq{
					NodeID:     "whatsapp:" + strings.TrimSpace(message.From),
					Transcript: transcript,
					Channel:    "whatsapp",
				})
				if status >= http.StatusInternalServerError {
					c.releaseWhatsAppMessage(message.ID)
					log.Printf("whatsapp webhook processing failed: status=%d", status)
					http.Error(w, "webhook processing failed", http.StatusBadGateway)
					return
				}
				if status >= http.StatusBadRequest {
					log.Printf("whatsapp webhook message rejected: status=%d", status)
				}
			}
		}
	}
	// Meta retries deliveries unless it receives a quick success response. Each
	// message's wamid is reserved before processing and retained after success.
	w.WriteHeader(http.StatusOK)
}

func (c *Controllers) transcribeWhatsAppAudio(parent context.Context, mediaID string) (string, error) {
	if c.WhatsAppMedia == nil || c.WhatsAppTranscriber == nil {
		return "", fmt.Errorf("WhatsApp media download or transcription is not configured")
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	audio, contentType, err := c.WhatsAppMedia.Download(ctx, mediaID)
	if err != nil {
		return "", err
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "audio/") {
		return "", fmt.Errorf("unsupported WhatsApp audio type")
	}
	format := strings.TrimPrefix(mediaType, "audio/")
	return c.WhatsAppTranscriber.Transcribe(ctx, audio, format)
}

func validWhatsAppSignature(secret string, body []byte, signature string) bool {
	if secret == "" || !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil || len(provided) != sha256.Size {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

func (c *Controllers) claimWhatsAppMessage(wamid string) bool {
	c.whatsappMu.Lock()
	defer c.whatsappMu.Unlock()
	if c.whatsappSeen == nil {
		c.whatsappSeen = make(map[string]time.Time)
	}
	now := time.Now()
	for id, seenAt := range c.whatsappSeen {
		if now.Sub(seenAt) > whatsappDedupRetention {
			delete(c.whatsappSeen, id)
		}
	}
	if seenAt, exists := c.whatsappSeen[wamid]; exists && now.Sub(seenAt) <= whatsappDedupRetention {
		return false
	}
	c.whatsappSeen[wamid] = now
	return true
}

func (c *Controllers) releaseWhatsAppMessage(wamid string) {
	c.whatsappMu.Lock()
	defer c.whatsappMu.Unlock()
	delete(c.whatsappSeen, wamid)
}
