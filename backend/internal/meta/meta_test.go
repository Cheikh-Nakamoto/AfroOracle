package meta

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// capture records the request a handler receives.
type capture struct {
	method      string
	path        string
	auth        string
	contentType string
	body        []byte
	jsonBody    map[string]any
}

func serveCapture(cap *capture, respBody, respCT string, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.method = r.Method
		cap.path = r.URL.Path
		cap.auth = r.Header.Get("Authorization")
		cap.contentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		cap.body = raw
		_ = json.Unmarshal(raw, &cap.jsonBody)
		w.Header().Set("Content-Type", respCT)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}))
}

func testClient(srv *httptest.Server) *Client {
	c := New("test-key")
	c.baseURL = srv.URL
	return c
}

func TestChatRequestConstruction(t *testing.T) {
	var got capture
	srv := serveCapture(&got,
		`{"choices":[{"message":{"role":"assistant","content":"{\"prix\":250}"}}]}`,
		"application/json", http.StatusOK)
	defer srv.Close()

	c := testClient(srv)
	res, err := c.Chat(context.Background(), ChatRequest{
		Model:      ModelChat,
		Messages:   []Message{{Role: "user", Content: "extract price"}},
		MaxTokens:  256,
		JSONObject: true,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if got.path != "/chat/completions" {
		t.Errorf("path = %q, want /chat/completions", got.path)
	}
	if got.auth != "Bearer test-key" {
		t.Errorf("auth = %q, want Bearer test-key", got.auth)
	}
	if got.jsonBody["model"] != ModelChat {
		t.Errorf("model = %v, want %q", got.jsonBody["model"], ModelChat)
	}
	msgs, _ := got.jsonBody["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages = %v, want 1 message", got.jsonBody["messages"])
	}
	m0, _ := msgs[0].(map[string]any)
	if m0["role"] != "user" || m0["content"] != "extract price" {
		t.Errorf("message = %v, want role=user content=extract price", m0)
	}
	if got.jsonBody["max_tokens"] != float64(256) {
		t.Errorf("max_tokens = %v, want 256", got.jsonBody["max_tokens"])
	}
	rf, _ := got.jsonBody["response_format"].(map[string]any)
	if rf["type"] != "json_object" {
		t.Errorf("response_format = %v, want {type:json_object}", got.jsonBody["response_format"])
	}
	if res.Text != `{"prix":250}` {
		t.Errorf("Text = %q, want extraction JSON", res.Text)
	}
	if len(res.Raw) == 0 {
		t.Error("Raw is empty, want raw response body")
	}
}

func TestChatDefaults(t *testing.T) {
	var got capture
	srv := serveCapture(&got, `{"choices":[]}`, "application/json", http.StatusOK)
	defer srv.Close()

	c := testClient(srv)
	res, err := c.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "system", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if got.jsonBody["model"] != ModelChat {
		t.Errorf("default model = %v, want %q", got.jsonBody["model"], ModelChat)
	}
	if _, ok := got.jsonBody["max_tokens"]; ok {
		t.Errorf("max_tokens = %v, want omitted when 0", got.jsonBody["max_tokens"])
	}
	if _, ok := got.jsonBody["response_format"]; ok {
		t.Errorf("response_format = %v, want omitted when JSONObject=false", got.jsonBody["response_format"])
	}
	if res.Text != "" {
		t.Errorf("Text = %q, want empty for no choices", res.Text)
	}
}

func TestChatServerError(t *testing.T) {
	srv := serveCapture(&capture{}, `{"error":"overloaded"}`, "application/json", http.StatusServiceUnavailable)
	defer srv.Close()

	c := testClient(srv)
	if _, err := c.Chat(context.Background(), ChatRequest{}); err == nil {
		t.Error("expected error on 503, got nil")
	} else if !strings.Contains(err.Error(), "/chat/completions") {
		t.Errorf("error = %q, want path in message", err)
	}
}

func TestTranscribeRequestConstruction(t *testing.T) {
	var got capture
	srv := serveCapture(&got, `{"text":"prix mil deux cent cinquante"}`, "application/json", http.StatusOK)
	defer srv.Close()

	c := testClient(srv)
	audio := []byte{0x52, 0x49, 0x46, 0x46, 0x00}
	text, err := c.Transcribe(context.Background(), audio, "wav")
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}

	if got.method != http.MethodPost {
		t.Errorf("method = %q, want POST", got.method)
	}
	if got.path != "/asr/transcribe" {
		t.Errorf("path = %q, want /asr/transcribe", got.path)
	}
	if got.auth != "Bearer test-key" {
		t.Errorf("auth = %q, want Bearer test-key", got.auth)
	}
	if got.contentType != "audio/wav" {
		t.Errorf("content-type = %q, want audio/wav", got.contentType)
	}
	if !reflect.DeepEqual(got.body, audio) {
		t.Errorf("body = %v, want raw audio bytes", got.body)
	}
	if text != "prix mil deux cent cinquante" {
		t.Errorf("text = %q, want transcript", text)
	}
}

func TestTranscribeDefaultFormat(t *testing.T) {
	var got capture
	srv := serveCapture(&got, `{"text":"x"}`, "application/json", http.StatusOK)
	defer srv.Close()

	c := testClient(srv)
	if _, err := c.Transcribe(context.Background(), []byte{0x01}, ""); err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if got.contentType != "audio/wav" {
		t.Errorf("content-type = %q, want default audio/wav", got.contentType)
	}
}

func TestSegmentRequestConstruction(t *testing.T) {
	var got capture
	srv := serveCapture(&got,
		`{"segments":[{"label":"price slate","box":[0.1,0.2,0.8,0.9]}]}`,
		"application/json", http.StatusOK)
	defer srv.Close()

	c := testClient(srv)
	segs, err := c.Segment(context.Background(), "price slate", []byte{0xff, 0xd8, 0xff})
	if err != nil {
		t.Fatalf("Segment: %v", err)
	}

	if got.path != "/responses" {
		t.Errorf("path = %q, want /responses", got.path)
	}
	if got.auth != "Bearer test-key" {
		t.Errorf("auth = %q, want Bearer test-key", got.auth)
	}
	if got.jsonBody["model"] != ModelSegment {
		t.Errorf("model = %v, want %q", got.jsonBody["model"], ModelSegment)
	}
	input, _ := got.jsonBody["input"].([]any)
	if len(input) != 1 {
		t.Fatalf("input = %v, want 1 item", got.jsonBody["input"])
	}
	if len(segs) != 1 || segs[0].Label != "price slate" ||
		segs[0].Box != [4]float64{0.1, 0.2, 0.8, 0.9} {
		t.Errorf("segments = %+v, want 1 price-slate box", segs)
	}
}

func TestGenerateImageRequestConstruction(t *testing.T) {
	var got capture
	want := []byte{0x89, 0x50, 0x4e, 0x47}
	srv := serveCapture(&got,
		`{"data":[{"b64_json":"`+base64.StdEncoding.EncodeToString(want)+`"}]}`,
		"application/json", http.StatusOK)
	defer srv.Close()

	c := testClient(srv)
	img, err := c.GenerateImage(context.Background(), "millet market in Kaolack")
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}

	if got.path != "/images/generations" {
		t.Errorf("path = %q, want /images/generations", got.path)
	}
	if got.auth != "Bearer test-key" {
		t.Errorf("auth = %q, want Bearer test-key", got.auth)
	}
	if got.jsonBody["model"] != ModelImage {
		t.Errorf("model = %v, want %q", got.jsonBody["model"], ModelImage)
	}
	if got.jsonBody["prompt"] != "millet market in Kaolack" {
		t.Errorf("prompt = %v, want the text prompt", got.jsonBody["prompt"])
	}
	if !reflect.DeepEqual(img, want) {
		t.Errorf("image bytes = %v, want decoded png header", img)
	}
}
