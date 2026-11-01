package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"afrioracle/internal/model"
)

// fakeWA is an offline stand-in for the WhatsApp Cloud API: media id -> URL
// -> bytes, plus the outbound messages endpoint.
type fakeWA struct {
	mu      sync.Mutex
	media   map[string][]byte
	mime    map[string]string
	auths   []string
	replies []map[string]any
	failMsg bool
	srv     *httptest.Server
}

func newFakeWA(t *testing.T) *fakeWA {
	f := &fakeWA{media: map[string][]byte{}, mime: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/blob/", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.auths = append(f.auths, r.Header.Get("Authorization"))
		w.Write(f.media[strings.TrimPrefix(r.URL.Path, "/blob/")])
	})
	mux.HandleFunc("/PHONEID/messages", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.auths = append(f.auths, r.Header.Get("Authorization"))
		if f.failMsg {
			http.Error(w, "nope", 500)
			return
		}
		var m map[string]any
		json.NewDecoder(r.Body).Decode(&m)
		f.replies = append(f.replies, m)
		w.Write([]byte(`{"messages":[{"id":"out"}]}`))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/")
		f.mu.Lock()
		defer f.mu.Unlock()
		f.auths = append(f.auths, r.Header.Get("Authorization"))
		if _, ok := f.media[id]; !ok {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"url": f.srv.URL + "/blob/" + id, "mime_type": f.mime[id]})
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeWA) replyTexts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, r := range f.replies {
		out = append(out, r["to"].(string)+"|"+r["text"].(map[string]any)["body"].(string))
	}
	return out
}

func mediaBody(wamid, from, typ, id, mime, caption string) string {
	key := typ
	extra := ""
	if caption != "" {
		extra = `,"caption":"` + caption + `"`
	}
	return `{"entry":[{"changes":[{"value":{"messages":[{"from":"` + from + `","id":"` + wamid + `","type":"` + typ +
		`","` + key + `":{"id":"` + id + `","mime_type":"` + mime + `"` + extra + `}}]}}]}]}`
}

func waMediaServer(t *testing.T, f *fakeWA, maxBytes int64) (*httptest.Server, model.Store) {
	t.Helper()
	t.Setenv("WHATSAPP_APP_SECRET", waSecret)
	reg := model.NewRegistry()
	reg.Register("A", "tok-A", 2, 88)
	reg.BindPhone("221770000001", "A")
	st := model.NewInMemoryStore()
	c := NewControllers(st, reg)
	c.WA = NewWAClient(f.srv.URL, "wa-token", "PHONEID", maxBytes)
	ts := httptest.NewServer(NewRouter(c, 1<<20))
	t.Cleanup(ts.Close)
	return ts, st
}

func waSigned(t *testing.T, ts *httptest.Server, body string) (int, string) {
	return waPost(t, ts.URL, body, sign(body, waSecret))
}

func TestWhatsappVoiceThenPhotoStoresSubmissionWithProof(t *testing.T) {
	f := newFakeWA(t)
	f.media["IMG1"], f.mime["IMG1"] = []byte("slate-photo-bytes"), "image/jpeg"
	f.media["AUD1"], f.mime["AUD1"] = []byte("OggSvoice"), "audio/ogg; codecs=opus"
	ts, st := waMediaServer(t, f, 1<<20)

	if code, out := waSigned(t, ts, mediaBody("w1", "221770000001", "image", "IMG1", "image/jpeg", "")); code != 200 {
		t.Fatalf("photo: %d %s", code, out)
	}
	if len(st.Submissions()) != 0 {
		t.Fatal("photo alone must not create a submission")
	}
	code, out := waSigned(t, ts, mediaBody("w2", "221770000001", "audio", "AUD1", "audio/ogg; codecs=opus", ""))
	if code != 201 {
		t.Fatalf("voice: %d %s", code, out)
	}
	subs := st.Submissions()
	if len(subs) != 1 || subs[0].NodeID != "A" || subs[0].Value != 187 || subs[0].ProofScore == nil || subs[0].Channel != "whatsapp" {
		t.Fatalf("bad submission %+v", subs)
	}
	for _, a := range f.auths {
		if a != "Bearer wa-token" {
			t.Fatalf("download/reply not authenticated with WHATSAPP_TOKEN: %q", a)
		}
	}
	rs := f.replyTexts()
	if len(rs) != 2 || !strings.HasPrefix(rs[1], "221770000001|") || !strings.Contains(rs[1], "187") {
		t.Fatalf("replies %v", rs)
	}
	// photo is consumed: a later voice has no photo attached
	f.media["AUD2"] = []byte("x")
	waSigned(t, ts, mediaBody("w3", "221770000001", "audio", "AUD2", "audio/ogg", ""))
	if s := st.Submissions(); len(s) != 2 || s[1].ProofScore == nil {
		t.Fatalf("second submission %+v", s)
	}
}

func TestWhatsappImageWithCaptionSubmitsDirectly(t *testing.T) {
	f := newFakeWA(t)
	f.media["IMG1"], f.mime["IMG1"] = []byte("photo"), "image/jpeg"
	ts, st := waMediaServer(t, f, 1<<20)
	code, out := waSigned(t, ts, mediaBody("w1", "221770000001", "image", "IMG1", "image/jpeg", "mil 190 FCFA kilo"))
	if code != 201 {
		t.Fatalf("%d %s", code, out)
	}
	if s := st.Submissions(); len(s) != 1 || s[0].Value != 190 || s[0].ProofScore == nil {
		t.Fatalf("%+v", s)
	}
}

func TestWhatsappClarifyQuestionSentWhenNoPrice(t *testing.T) {
	f := newFakeWA(t)
	ts, st := waMediaServer(t, f, 1<<20)
	body := waBody("w1", "221770000001", "le prix du mil")
	code, _ := waSigned(t, ts, body)
	if code != 200 || len(st.Submissions()) != 0 {
		t.Fatalf("code %d subs %d", code, len(st.Submissions()))
	}
	rs := f.replyTexts()
	if len(rs) != 1 || rs[0] != "221770000001|le kilo ou le sac ?" {
		t.Fatalf("replies %v", rs)
	}
}

func TestWhatsappMediaDownloadFailureIs502AndRetryable(t *testing.T) {
	f := newFakeWA(t)
	ts, st := waMediaServer(t, f, 1<<20)
	body := mediaBody("w1", "221770000001", "audio", "MISSING", "audio/ogg", "")
	if code, _ := waSigned(t, ts, body); code != 502 {
		t.Fatalf("got %d want 502", code)
	}
	f.mu.Lock()
	f.media["MISSING"] = []byte("v")
	f.mu.Unlock()
	if code, _ := waSigned(t, ts, body); code != 201 { // Meta retry works: wamid released
		t.Fatalf("retry got %d want 201", code)
	}
	if len(st.Submissions()) != 1 {
		t.Fatal("want one submission")
	}
}

func TestWhatsappOversizedMediaRejected(t *testing.T) {
	f := newFakeWA(t)
	f.media["BIG"] = make([]byte, 100)
	ts, st := waMediaServer(t, f, 50)
	if code, _ := waSigned(t, ts, mediaBody("w1", "221770000001", "audio", "BIG", "audio/ogg", "")); code != 502 {
		t.Fatalf("got %d want 502", code)
	}
	if len(st.Submissions()) != 0 {
		t.Fatal("stored oversized media")
	}
}

func TestWhatsappReplyFailureIsSurfacedButSubmissionKept(t *testing.T) {
	f := newFakeWA(t)
	f.failMsg = true
	ts, st := waMediaServer(t, f, 1<<20)
	code, out := waSigned(t, ts, waBody("w1", "221770000001", "mil 187 FCFA kilo"))
	if code != 201 || !strings.Contains(out, "reply_error") || len(st.Submissions()) != 1 {
		t.Fatalf("%d %s", code, out)
	}
}

func TestWhatsappMediaWithoutClientIs503(t *testing.T) {
	ts, _ := waServer(t)
	body := mediaBody("w1", "221770000001", "audio", "X", "audio/ogg", "")
	if code, _ := waPost(t, ts.URL, body, sign(body, waSecret)); code != 503 {
		t.Fatalf("got %d want 503", code)
	}
}
