package controller

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"afrioracle/internal/meta"
	"afrioracle/internal/model"
)

// fakeMeta is an offline stand-in for the Meta Model API.
type fakeMeta struct {
	mu    sync.Mutex
	paths []string
	auth  string
	audio []byte
	chats []map[string]any
	slate string // JSON content returned for vision (slate) chats
}

func (f *fakeMeta) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.paths = append(f.paths, r.URL.Path)
	f.auth = r.Header.Get("Authorization")
	body, _ := io.ReadAll(r.Body)
	switch r.URL.Path {
	case "/asr/transcribe":
		f.audio = body
		json.NewEncoder(w).Encode(map[string]string{"text": "le mil est a 187 francs"})
	case "/chat/completions":
		var m map[string]any
		json.Unmarshal(body, &m)
		f.chats = append(f.chats, m)
		content := `{"produit":"mil","prix":187,"unite":"kilo","marche":"Kaolack","needs_confirmation":false,"question":""}`
		if strings.Contains(string(body), "image_url") {
			content = f.slate
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}})
	case "/responses":
		json.NewEncoder(w).Encode(map[string]any{"segments": []any{
			map[string]any{"label": "price slate", "box": []float64{0, 0, 1, 1}},
		}})
	default:
		http.NotFound(w, r)
	}
}

func useFake(t *testing.T) *fakeMeta {
	t.Helper()
	f := &fakeMeta{slate: `{"price":187}`}
	srv := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(srv.Close)
	t.Cleanup(model.UseMeta(meta.NewWithBaseURL("test-key", srv.URL)))
	return f
}

func TestRealModeAudioSubmitReturnsTranscriptPrice(t *testing.T) {
	f := useFake(t)
	ts := newTestServer(1 << 20)
	defer ts.Close()
	tok := enroll(ts.URL, "A", 2, 80)
	audio := []byte("RIFFfakewav")
	body, _ := json.Marshal(map[string]any{
		"node_id": "A", "audio_b64": base64.StdEncoding.EncodeToString(audio), "audio_format": "wav",
		"photo_b64": base64.StdEncoding.EncodeToString([]byte("img")),
	})
	code, out := postTok(t, ts.URL+"/api/submit", tok, string(body))
	if code != http.StatusCreated {
		t.Fatalf("status %d %v", code, out)
	}
	sub := out["submission"].(map[string]any)
	if sub["Value"].(float64) != 187 {
		t.Fatalf("value = %v, want 187", sub["Value"])
	}
	if string(f.audio) != string(audio) || f.auth != "Bearer test-key" {
		t.Fatalf("audio/auth not forwarded: %q %q", f.audio, f.auth)
	}
	if len(f.chats) == 0 || f.chats[0]["model"] != meta.ModelChat {
		t.Fatalf("extraction did not go through Muse Spark: %v", f.chats)
	}
}

func TestRealModeProofUsesSlateReadingAndSAM(t *testing.T) {
	f := useFake(t)
	ts := newTestServer(1 << 20)
	defer ts.Close()
	tok := enroll(ts.URL, "A", 2, 80)
	photo := base64.StdEncoding.EncodeToString([]byte("img"))
	// Forged photo_price must be ignored in real mode: slate says 187.
	body := `{"node_id":"A","value":187,"photo_price":999,"photo_b64":"` + photo + `"}`
	code, out := postTok(t, ts.URL+"/api/submit", tok, body)
	if code != http.StatusCreated {
		t.Fatalf("status %d %v", code, out)
	}
	proof := out["proof"].(map[string]any)
	if proof["Score"].(float64) != 94 || proof["SlatePrice"].(float64) != 187 {
		t.Fatalf("proof = %v", proof)
	}
	if proof["Model"] != "Muse Spark 1.3 + SAM 3.1" {
		t.Fatalf("model = %v", proof["Model"])
	}
	objs, _ := proof["ObjectsFound"].([]any)
	if len(objs) != 1 || objs[0] != "price slate" {
		t.Fatalf("objects = %v", objs)
	}
	joined := strings.Join(f.paths, ",")
	if !strings.Contains(joined, "/chat/completions") || !strings.Contains(joined, "/responses") {
		t.Fatalf("paths = %v", f.paths)
	}
}

func TestRealModeUnreadableSlateAndAPIErrorsFailSafe(t *testing.T) {
	f := useFake(t)
	f.slate = `{"price":null}`
	ts := newTestServer(1 << 20)
	defer ts.Close()
	tok := enroll(ts.URL, "A", 2, 80)
	photo := base64.StdEncoding.EncodeToString([]byte("img"))
	_, out := postTok(t, ts.URL+"/api/submit", tok, `{"node_id":"A","value":187,"photo_b64":"`+photo+`"}`)
	if out["proof"].(map[string]any)["Score"].(float64) != 40 {
		t.Fatalf("unreadable slate must score 40: %v", out)
	}
}

func TestRealModeTranscribeFailureIs502(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", 500) }))
	defer bad.Close()
	defer model.UseMeta(meta.NewWithBaseURL("k", bad.URL))()
	ts := newTestServer(1 << 20)
	defer ts.Close()
	tok := enroll(ts.URL, "A", 2, 80)
	code, _ := postTok(t, ts.URL+"/api/submit", tok, `{"node_id":"A","audio_b64":"`+base64.StdEncoding.EncodeToString([]byte("x"))+`"}`)
	if code != http.StatusBadGateway {
		t.Fatalf("status %d, want 502", code)
	}
	if code, _ := postTok(t, ts.URL+"/api/submit", tok, `{"node_id":"A","audio_b64":"!!"}`); code != http.StatusBadRequest {
		t.Fatalf("bad audio base64 status %d, want 400", code)
	}
}

func TestMockModeNeverCallsMetaAndReportsMode(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	tok := enroll(ts.URL, "A", 2, 80)
	code, out := postTok(t, ts.URL+"/api/submit", tok, `{"node_id":"A","audio_b64":"`+base64.StdEncoding.EncodeToString([]byte("x"))+`"}`)
	if code != http.StatusCreated || out["submission"].(map[string]any)["Value"].(float64) != 187 {
		t.Fatalf("mock audio: %d %v", code, out)
	}
	_, mode := get(t, ts.URL+"/api/mode")
	if mode["ai"] != "mock" {
		t.Fatalf("mode = %v", mode)
	}
	f := useFake(t)
	_ = f
	_, mode = get(t, ts.URL+"/api/mode")
	if mode["ai"] != "real" {
		t.Fatalf("mode = %v", mode)
	}
}
