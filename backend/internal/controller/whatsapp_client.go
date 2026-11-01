package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// WAClient is the seam to the WhatsApp Cloud API: fetch a media object by id
// and send a text reply. Tests substitute an httptest-backed implementation.
type WAClient interface {
	// Media downloads the media with the given id, failing if it exceeds the
	// client's size cap. mime is the type reported by the API.
	Media(ctx context.Context, id string) (data []byte, mime string, err error)
	SendText(ctx context.Context, to, body string) error
}

const (
	defaultWABase    = "https://graph.facebook.com/v21.0"
	defaultMaxMedia  = 8 << 20
	waRequestTimeout = 20 * time.Second
	waErrBodySnippet = 200
)

type httpWA struct {
	base, token, phoneID string
	maxBytes             int64
	hc                   *http.Client
}

// NewWAClient builds the real Cloud API client. base is the Graph API root.
func NewWAClient(base, token, phoneID string, maxBytes int64) WAClient {
	if maxBytes <= 0 {
		maxBytes = defaultMaxMedia
	}
	return &httpWA{
		base: strings.TrimRight(base, "/"), token: token, phoneID: phoneID,
		maxBytes: maxBytes, hc: &http.Client{Timeout: waRequestTimeout},
	}
}

// NewWAClientFromEnv returns nil unless WHATSAPP_TOKEN is set.
// Optional: WHATSAPP_API_BASE, WHATSAPP_PHONE_NUMBER_ID, WHATSAPP_MAX_MEDIA_BYTES.
func NewWAClientFromEnv() (WAClient, error) {
	tok := os.Getenv("WHATSAPP_TOKEN")
	if tok == "" {
		return nil, nil
	}
	base := os.Getenv("WHATSAPP_API_BASE")
	if base == "" {
		base = defaultWABase
	}
	var max int64
	if v := os.Getenv("WHATSAPP_MAX_MEDIA_BYTES"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("WHATSAPP_MAX_MEDIA_BYTES must be a positive integer, got %q", v)
		}
		max = n
	}
	return NewWAClient(base, tok, os.Getenv("WHATSAPP_PHONE_NUMBER_ID"), max), nil
}

func (w *httpWA) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+w.token)
	resp, err := w.hc.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		defer resp.Body.Close()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, waErrBodySnippet))
		return nil, fmt.Errorf("whatsapp api %s: status %d: %s", req.URL.Path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return resp, nil
}

func (w *httpWA) Media(ctx context.Context, id string) ([]byte, string, error) {
	req, err := http.NewRequest(http.MethodGet, w.base+"/"+id, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := w.do(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("media lookup: %w", err)
	}
	var meta struct {
		URL      string `json:"url"`
		MimeType string `json:"mime_type"`
		FileSize int64  `json:"file_size"`
	}
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<16)).Decode(&meta)
	resp.Body.Close()
	if err != nil || meta.URL == "" {
		return nil, "", fmt.Errorf("media lookup: no url in response (%v)", err)
	}
	if meta.FileSize > w.maxBytes {
		return nil, "", fmt.Errorf("media too large: %d > %d bytes", meta.FileSize, w.maxBytes)
	}
	req, err = http.NewRequest(http.MethodGet, meta.URL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err = w.do(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("media download: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, w.maxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("media download: %w", err)
	}
	if int64(len(data)) > w.maxBytes {
		return nil, "", fmt.Errorf("media too large: over %d bytes", w.maxBytes)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("media download: empty body")
	}
	return data, meta.MimeType, nil
}

func (w *httpWA) SendText(ctx context.Context, to, body string) error {
	if w.phoneID == "" {
		return fmt.Errorf("WHATSAPP_PHONE_NUMBER_ID not set; cannot reply")
	}
	payload, _ := json.Marshal(map[string]any{
		"messaging_product": "whatsapp", "to": to, "type": "text",
		"text": map[string]string{"body": body},
	})
	req, err := http.NewRequest(http.MethodPost, w.base+"/"+w.phoneID+"/messages", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.do(ctx, req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
