package controller

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"afrioracle/internal/model"
	"afrioracle/internal/view"
)

// validSignature checks Meta's X-Hub-Signature-256 ("sha256=<hex>") over the
// raw body with the app secret. Fails closed when the secret is unset.
func validSignature(secret, header string, body []byte) bool {
	if secret == "" || !strings.HasPrefix(header, "sha256=") {
		return false
	}
	got, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return false
	}
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return hmac.Equal(got, m.Sum(nil))
}

type waMedia struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	Caption  string `json:"caption"`
}

type waPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []struct {
					From string `json:"from"`
					ID   string `json:"id"`
					Type string `json:"type"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
					Audio waMedia `json:"audio"`
					Image waMedia `json:"image"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// claimWamid marks a message id as seen; false means it is a redelivery.
func (c *Controllers) claimWamid(id string) bool {
	c.waMu.Lock()
	defer c.waMu.Unlock()
	if c.waSeen == nil {
		c.waSeen = map[string]bool{}
	}
	if c.waSeen[id] {
		return false
	}
	c.waSeen[id] = true
	return true
}

func (c *Controllers) releaseWamid(id string) {
	c.waMu.Lock()
	delete(c.waSeen, id)
	c.waMu.Unlock()
}

func (c *Controllers) HandleWhatsapp(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		want := os.Getenv("WHATSAPP_VERIFY_TOKEN")
		got := r.URL.Query().Get("hub.verify_token")
		if want == "" || !hmac.Equal([]byte(got), []byte(want)) {
			view.WriteError(w, http.StatusForbidden, "verification failed")
			return
		}
		w.Write([]byte(r.URL.Query().Get("hub.challenge")))
		return
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		view.WriteError(w, http.StatusBadRequest, "unreadable body")
		return
	}
	if !validSignature(os.Getenv("WHATSAPP_APP_SECRET"), r.Header.Get("X-Hub-Signature-256"), raw) {
		log.Printf("whatsapp: rejected unsigned or badly signed webhook")
		view.WriteError(w, http.StatusUnauthorized, "invalid signature")
		return
	}
	var msg waPayload
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("whatsapp: decode: %v", err)
		view.WriteError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	var last any = map[string]string{"status": "ignored"}
	status := http.StatusOK
	for _, e := range msg.Entry {
		for _, ch := range e.Changes {
			for _, m := range ch.Value.Messages {
				if m.ID == "" || !waSupported(m.Type, m.Text.Body, m.Audio, m.Image) {
					continue
				}
				node, ok := c.Registry.ByPhone(m.From)
				if !ok {
					log.Printf("whatsapp: unknown sender %q (wamid %s)", m.From, m.ID)
					view.WriteError(w, http.StatusForbidden, "sender is not a registered node")
					return
				}
				if !c.claimWamid(m.ID) {
					last = map[string]string{"status": "duplicate"}
					continue
				}
				req := SubmitReq{NodeID: node.ID, Tier: node.Tier, RepScore: node.Rep, Channel: "whatsapp"}
				if m.Type == "text" {
					req.Transcript = m.Text.Body
				} else {
					held, st, err := c.loadMedia(r.Context(), m.From, m.Type, m.Audio, m.Image, &req)
					if err != nil {
						log.Printf("whatsapp: wamid %s: %v", m.ID, err)
						c.releaseWamid(m.ID) // let Meta's retry succeed
						view.WriteError(w, st, "media unavailable: "+err.Error())
						return
					}
					if held {
						status, last = http.StatusOK, map[string]string{"status": "photo_held"}
						last = c.reply(r.Context(), m.From, status, last, "Photo recue. Envoyez maintenant le prix en message vocal.")
						continue
					}
				}
				status, last = c.processSubmit(req)
				if status >= 500 {
					c.releaseWamid(m.ID) // let Meta's retry succeed
					view.WriteJSON(w, status, last)
					return
				}
				last = c.reply(r.Context(), m.From, status, last, "")
			}
		}
	}
	view.WriteJSON(w, status, last)
}

func waSupported(typ, text string, audio, image waMedia) bool {
	switch typ {
	case "text":
		return strings.TrimSpace(text) != ""
	case "audio":
		return audio.ID != ""
	case "image":
		return image.ID != ""
	}
	return false
}

// loadMedia downloads an audio or image message through c.WA and fills req.
// A voice note becomes AudioB64 (plus the sender's held photo, if any). A photo
// with a price caption becomes a direct submission; a bare photo is held for
// the sender's next voice/text message and held=true is returned.
func (c *Controllers) loadMedia(ctx context.Context, from, typ string, audio, image waMedia, req *SubmitReq) (held bool, status int, err error) {
	if c.WA == nil {
		return false, http.StatusServiceUnavailable, errors.New("WHATSAPP_TOKEN not configured")
	}
	media := audio
	if typ == "image" {
		media = image
	}
	data, mime, err := c.WA.Media(ctx, media.ID)
	if err != nil {
		return false, http.StatusBadGateway, err
	}
	if mime == "" {
		mime = media.MimeType
	}
	if typ == "audio" {
		req.AudioB64 = base64.StdEncoding.EncodeToString(data)
		req.AudioFormat = audioFormat(mime)
		req.PhotoB64 = c.takePhoto(from)
		return false, 0, nil
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	if strings.TrimSpace(image.Caption) == "" {
		c.holdPhoto(from, b64)
		return true, 0, nil
	}
	req.Transcript, req.PhotoB64 = image.Caption, b64
	return false, 0, nil
}

// audioFormat maps "audio/ogg; codecs=opus" to "ogg".
func audioFormat(mime string) string {
	mime, _, _ = strings.Cut(mime, ";")
	_, sub, _ := strings.Cut(strings.TrimSpace(mime), "/")
	return sub
}

func (c *Controllers) holdPhoto(from, b64 string) {
	c.waMu.Lock()
	defer c.waMu.Unlock()
	if c.waPhotos == nil {
		c.waPhotos = map[string]string{}
	}
	c.waPhotos[from] = b64
}

func (c *Controllers) takePhoto(from string) string {
	c.waMu.Lock()
	defer c.waMu.Unlock()
	p := c.waPhotos[from]
	delete(c.waPhotos, from)
	return p
}

// reply tells the sender the outcome (or the clarifying question). A failed
// reply never undoes the stored submission; it is logged and surfaced in the
// webhook response as reply_error.
func (c *Controllers) reply(ctx context.Context, to string, status int, resp any, override string) any {
	if c.WA == nil {
		return resp
	}
	text := override
	if text == "" {
		text = replyText(resp)
	}
	if text == "" {
		return resp
	}
	if err := c.WA.SendText(ctx, to, text); err != nil {
		log.Printf("whatsapp: reply to %s: %v", to, err)
		switch m := resp.(type) {
		case map[string]any:
			m["reply_error"] = err.Error()
		case map[string]string:
			m["reply_error"] = err.Error()
		}
	}
	return resp
}

func replyText(resp any) string {
	switch m := resp.(type) {
	case map[string]string:
		if m["status"] == "clarify" {
			return m["question"]
		}
	case map[string]any:
		if sub, ok := m["submission"].(model.Submission); ok {
			msg := fmt.Sprintf("Prix enregistre: %g FCFA/kilo.", sub.Value)
			if sub.ProofScore != nil {
				msg += fmt.Sprintf(" Preuve: %d/100.", *sub.ProofScore)
			}
			return msg
		}
	}
	return ""
}
