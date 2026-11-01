package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"afrioracle/internal/model"
)

func newTestServer(maxBody int64) *httptest.Server {
	return httptest.NewServer(NewRouter(NewControllers(model.NewInMemoryStore()), maxBody))
}

func post(t *testing.T, url string, body string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func get(t *testing.T, url string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func submit(t *testing.T, base, node string, tier int, rep int, value float64, photoPrice float64, photo string) map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"node_id": node, "tier": tier, "rep_score": rep, "value": value,
		"photo_price": photoPrice, "photo_b64": photo, "channel": "web",
	})
	code, out := post(t, base+"/api/submit", string(body))
	if code != http.StatusCreated {
		t.Fatalf("submit %s: status %d body %v", node, code, out)
	}
	return out
}

func TestConsensusUnderQuorumIs422(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	submit(t, ts.URL, "A", 1, 80, 187, 187, "QQ==")
	code, out := post(t, ts.URL+"/api/consensus", "")
	if code != http.StatusUnprocessableEntity || out["error"] == nil {
		t.Fatalf("want 422 with error, got %d %v", code, out)
	}
}

func TestGoldenOverHTTP(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	submit(t, ts.URL, "A", 1, 92, 187, 187, "QQ==")
	submit(t, ts.URL, "B", 2, 70, 185, 185, "Qg==")
	submit(t, ts.URL, "C", 2, 70, 188, 188, "Qw==")
	submit(t, ts.URL, "FAO", 3, 70, 190, 190, "")
	// D reuses A's photo -> duplicate -> proof 12 -> excluded
	d := submit(t, ts.URL, "D", 2, 70, 187, 187, "QQ==")
	if proof := d["proof"].(map[string]any); proof["Score"].(float64) != 12 {
		t.Fatalf("D proof = %v, want 12", proof["Score"])
	}

	code, out := post(t, ts.URL+"/api/consensus", "")
	if code != http.StatusOK {
		t.Fatalf("consensus status %d %v", code, out)
	}
	if out["confidence"].(float64) != 100 {
		t.Errorf("confidence = %v, want 100", out["confidence"])
	}
	if out["median"].(float64) != 187 {
		t.Errorf("median = %v, want 187", out["median"])
	}
	for _, o := range out["outcomes"].([]any) {
		m := o.(map[string]any)
		if m["NodeID"] == "D" && m["Reward"].(float64) != 0 {
			t.Errorf("D reward = %v, want 0", m["Reward"])
		}
	}

	code, st := get(t, ts.URL+"/api/state")
	if code != 200 || len(st["publications"].([]any)) != 1 {
		t.Fatalf("state: %d %v", code, st)
	}
	code, ask := get(t, ts.URL+"/api/ask?q=prix")
	if code != 200 || !strings.Contains(ask["answer"].(string), "187") {
		t.Fatalf("ask: %d %v", code, ask)
	}
}

func TestAskWithoutDataNeverEstimates(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	_, ask := get(t, ts.URL+"/api/ask")
	if !strings.Contains(ask["answer"].(string), "Aucune donnée") {
		t.Fatalf("ask = %v", ask)
	}
}

func TestOversizedBodyIs413(t *testing.T) {
	ts := newTestServer(64)
	defer ts.Close()
	code, out := post(t, ts.URL+"/api/submit", `{"node_id":"`+strings.Repeat("x", 500)+`"}`)
	if code != http.StatusRequestEntityTooLarge || out["error"] == nil {
		t.Fatalf("want 413 with error, got %d %v", code, out)
	}
}

func TestWrongMethodAndUnknownRouteShapes(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/submit"}, {"GET", "/api/consensus"}, {"POST", "/api/state"}, {"POST", "/api/ask"},
	} {
		req, _ := http.NewRequest(tc.method, ts.URL+tc.path, nil)
		resp, _ := http.DefaultClient.Do(req)
		var out map[string]any
		json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		if resp.StatusCode != 405 || out["error"] == nil || resp.Header.Get("Allow") == "" {
			t.Errorf("%s %s: %d %v allow=%q", tc.method, tc.path, resp.StatusCode, out, resp.Header.Get("Allow"))
		}
	}
	code, out := get(t, ts.URL+"/nope")
	if code != 404 || out["error"] == nil {
		t.Errorf("unknown route: %d %v", code, out)
	}
	code, _ = post(t, ts.URL+"/api/submit", "{not json")
	if code != 400 {
		t.Errorf("bad json = %d, want 400", code)
	}
}
