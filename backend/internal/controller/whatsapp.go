package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

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
				if m.Type != "text" || strings.TrimSpace(m.Text.Body) == "" || m.ID == "" {
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
				status, last = c.processSubmit(SubmitReq{
					NodeID: node.ID, Tier: node.Tier, RepScore: node.Rep,
					Transcript: m.Text.Body, Channel: "whatsapp",
				})
				if status >= 500 {
					c.releaseWamid(m.ID) // let Meta's retry succeed
					view.WriteJSON(w, status, last)
					return
				}
			}
		}
	}
	view.WriteJSON(w, status, last)
}
