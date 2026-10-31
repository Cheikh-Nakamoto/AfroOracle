package controller

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

const testWhatsAppSecret = "whatsapp-test-app-secret"

func signedWhatsAppRequest(c *Controllers, body, signature string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", strings.NewReader(body))
	if signature != "" {
		r.Header.Set("X-Hub-Signature-256", signature)
	}
	w := httptest.NewRecorder()
	c.HandleWhatsapp(w, r)
	return w
}

func signWhatsAppBody(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestWhatsAppPostRejectsMissingAndInvalidSignatures(t *testing.T) {
	c, _ := authController(t)
	c.WhatsAppAppSecret = testWhatsAppSecret
	body := `{"entry":[]}`
	if w := signedWhatsAppRequest(c, body, ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned status=%d body=%s", w.Code, w.Body.String())
	}
	if w := signedWhatsAppRequest(c, body, "sha256=00"); w.Code != http.StatusUnauthorized {
		t.Fatalf("bad-signature status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestWhatsAppPostFailsClosedWithoutAppSecret(t *testing.T) {
	c, _ := authController(t)
	w := signedWhatsAppRequest(c, `{"entry":[]}`, "")
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestWhatsAppWebhookVerificationRequiresConfiguredToken(t *testing.T) {
	c, _ := authController(t)
	r := httptest.NewRequest(http.MethodGet, "/webhook/whatsapp?hub.verify_token=&hub.challenge=challenge", nil)
	w := httptest.NewRecorder()
	c.HandleWhatsapp(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured status=%d body=%s", w.Code, w.Body.String())
	}

	c.WhatsAppVerifyToken = "configured-token"
	r = httptest.NewRequest(http.MethodGet, "/webhook/whatsapp?hub.verify_token=configured-token&hub.challenge=challenge-123", nil)
	w = httptest.NewRecorder()
	c.HandleWhatsapp(w, r)
	if w.Code != http.StatusOK || w.Body.String() != "challenge-123" {
		t.Fatalf("verification status=%d body=%q", w.Code, w.Body.String())
	}
}

func TestWhatsAppWebhookDeduplicatesWAMID(t *testing.T) {
	c, store := authController(t)
	c.WhatsAppAppSecret = testWhatsAppSecret
	body := `{"object":"whatsapp_business_account","entry":[{"changes":[{"value":{"messages":[{"from":"221770000000","id":"wamid.fixture-1","type":"text","text":{"body":"Le prix du mil à Kaolack est de 187 FCFA/kg."}}]}}]}]}`
	signature := signWhatsAppBody(testWhatsAppSecret, body)
	first := signedWhatsAppRequest(c, body, signature)
	second := signedWhatsAppRequest(c, body, signature)
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("delivery statuses=%d,%d", first.Code, second.Code)
	}
	if got := len(store.Submissions()); got != 1 {
		t.Fatalf("duplicate wamid created %d submissions, want one", got)
	}
}

func TestWhatsAppWebhookRejectsMalformedSignedPayload(t *testing.T) {
	c, _ := authController(t)
	c.WhatsAppAppSecret = testWhatsAppSecret
	body := `{"entry":`
	w := signedWhatsAppRequest(c, body, signWhatsAppBody(testWhatsAppSecret, body))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestWhatsAppWebhookRejectsMessageWithoutWAMID(t *testing.T) {
	c, store := authController(t)
	c.WhatsAppAppSecret = testWhatsAppSecret
	body := `{"entry":[{"changes":[{"value":{"messages":[{"from":"221770000000","text":{"body":"mil 187"}}]}}]}]}`
	w := signedWhatsAppRequest(c, body, signWhatsAppBody(testWhatsAppSecret, body))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := len(store.Submissions()); got != 0 {
		t.Fatalf("missing wamid created %d submissions", got)
	}
}

type fakeWhatsAppMedia struct {
	data        []byte
	contentType string
	mediaID     string
	calls       int
}

func (f *fakeWhatsAppMedia) Download(_ context.Context, id string) ([]byte, string, error) {
	f.calls++
	f.mediaID = id
	return f.data, f.contentType, nil
}

type fakeWhatsAppTranscriber struct {
	text   string
	audio  []byte
	format string
	calls  int
}

func (f *fakeWhatsAppTranscriber) Transcribe(_ context.Context, audio []byte, format string) (string, error) {
	f.calls++
	f.audio = append([]byte(nil), audio...)
	f.format = format
	return f.text, nil
}

func TestWhatsAppAudioDownloadsAndTranscribesOncePerWAMID(t *testing.T) {
	c, store := authController(t)
	c.WhatsAppAppSecret = testWhatsAppSecret
	media := &fakeWhatsAppMedia{data: []byte("audio-bytes"), contentType: "audio/ogg; codecs=opus"}
	transcriber := &fakeWhatsAppTranscriber{text: "Le prix du mil est 187 FCFA le kilo."}
	c.WhatsAppMedia = media
	c.WhatsAppTranscriber = transcriber
	body := `{"entry":[{"changes":[{"value":{"messages":[{"from":"221770000000","id":"wamid.audio-1","type":"audio","audio":{"id":"12345","mime_type":"audio/ogg"}}]}}]}]}`
	signature := signWhatsAppBody(testWhatsAppSecret, body)
	first := signedWhatsAppRequest(c, body, signature)
	second := signedWhatsAppRequest(c, body, signature)
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("delivery statuses=%d,%d", first.Code, second.Code)
	}
	if media.calls != 1 || transcriber.calls != 1 || media.mediaID != "12345" {
		t.Fatalf("media calls=%d transcribe calls=%d media id=%q", media.calls, transcriber.calls, media.mediaID)
	}
	if transcriber.format != "ogg" || !reflect.DeepEqual(transcriber.audio, media.data) {
		t.Fatalf("transcribed format=%q audio=%q", transcriber.format, transcriber.audio)
	}
	if got := len(store.Submissions()); got != 1 {
		t.Fatalf("audio message created %d submissions", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func httpResponse(req *http.Request, body []byte) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body)), Request: req}
}

func TestWhatsAppMediaDownloaderUsesAuthorizedTwoStepDownload(t *testing.T) {
	audio := []byte("audio-content")
	digest := sha256.Sum256(audio)
	metadata, _ := json.Marshal(whatsAppMediaMetadata{
		URL: "https://lookaside.fbsbx.com/media/abc", MIMEType: "audio/ogg", SHA256: hex.EncodeToString(digest[:]), FileSize: int64(len(audio)),
	})
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer media-token" {
			t.Errorf("missing auth on %s", r.URL.Host)
		}
		switch r.URL.Host {
		case "graph.facebook.com":
			if r.URL.Path != "/v23.0/12345" || r.Method != http.MethodGet {
				t.Errorf("metadata request %s %s", r.Method, r.URL)
			}
			return httpResponse(r, metadata), nil
		case "lookaside.fbsbx.com":
			if r.Method != http.MethodGet {
				t.Errorf("download method=%s", r.Method)
			}
			return httpResponse(r, audio), nil
		default:
			t.Fatalf("unexpected request host %q", r.URL.Host)
			return nil, nil
		}
	})
	downloader := &cloudWhatsAppMediaDownloader{token: "media-token", version: "v23.0", client: &http.Client{Transport: transport}}
	got, contentType, err := downloader.Download(context.Background(), "12345")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, audio) || contentType != "audio/ogg" {
		t.Fatalf("got=%q contentType=%q", got, contentType)
	}
}

func TestWhatsAppMediaDownloaderRejectsUntrustedURL(t *testing.T) {
	requests := 0
	metadata, _ := json.Marshal(whatsAppMediaMetadata{URL: "https://127.0.0.1/internal", MIMEType: "audio/ogg", FileSize: 10})
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		return httpResponse(r, metadata), nil
	})
	downloader := &cloudWhatsAppMediaDownloader{token: "media-token", version: "v23.0", client: &http.Client{Transport: transport}}
	if _, _, err := downloader.Download(context.Background(), "12345"); err == nil {
		t.Fatal("expected untrusted media URL to be rejected")
	}
	if requests != 1 {
		t.Fatalf("made %d requests; expected to stop before fetching untrusted host", requests)
	}
}
