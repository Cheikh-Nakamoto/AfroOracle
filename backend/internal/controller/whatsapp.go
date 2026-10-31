package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"afrioracle/internal/view"
)

func (c *Controllers) HandleWhatsapp(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if r.URL.Query().Get("hub.verify_token") == os.Getenv("WHATSAPP_VERIFY_TOKEN") {
			w.Write([]byte(r.URL.Query().Get("hub.challenge")))
			return
		}
		http.Error(w, "verification failed", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost {
		var msg struct {
			Entry []struct {
				Changes []struct {
					Value struct {
						Messages []struct {
							Text struct {
								Body string `json:"body"`
							} `json:"text"`
						} `json:"messages"`
					} `json:"value"`
				} `json:"changes"`
			} `json:"entry"`
		}
		json.NewDecoder(r.Body).Decode(&msg)
		body := ""
		for _, e := range msg.Entry {
			for _, ch := range e.Changes {
				for _, m := range ch.Value.Messages {
					body = m.Text.Body
				}
			}
		}
		if body != "" {
			status, resp := c.processSubmit(context.Background(), SubmitReq{Transcript: body, Channel: "whatsapp"})
			view.WriteJSON(w, status, resp)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "GET/POST only", http.StatusMethodNotAllowed)
}
