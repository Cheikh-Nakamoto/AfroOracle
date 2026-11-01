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

func TestSubmitRejectsInvalidInput(t *testing.T) {
	cases := []struct{ name, body string }{
		{"empty object", `{}`},
		{"empty transcript no value", `{"node_id":"A","tier":1,"rep_score":50,"transcript":"  "}`},
		{"explicit zero value", `{"node_id":"A","tier":1,"rep_score":50,"value":0}`},
		{"negative value", `{"node_id":"A","tier":1,"rep_score":50,"value":-3}`},
		{"bad base64", `{"node_id":"A","tier":1,"rep_score":50,"value":187,"photo_b64":"%%%not-base64"}`},
		{"unknown tier", `{"node_id":"A","tier":9,"rep_score":50,"value":187}`},
		{"zero tier", `{"node_id":"A","tier":0,"rep_score":50,"value":187}`},
		{"rep too high", `{"node_id":"A","tier":1,"rep_score":101,"value":187}`},
		{"rep negative", `{"node_id":"A","tier":1,"rep_score":-1,"value":187}`},
		{"photo price negative", `{"node_id":"A","tier":1,"rep_score":50,"value":187,"photo_price":-5}`},
		{"photo price zero", `{"node_id":"A","tier":1,"rep_score":50,"value":187,"photo_price":0}`},
		{"missing node id", `{"tier":1,"rep_score":50,"value":187}`},
		{"blank node id", `{"node_id":"  ","tier":1,"rep_score":50,"value":187}`},
		{"long node id", `{"node_id":"` + strings.Repeat("x", 65) + `","tier":1,"rep_score":50,"value":187}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestServer(1 << 20)
			defer ts.Close()
			code, out := post(t, ts.URL+"/api/submit", tc.body)
			if code < 400 || code > 499 || out["error"] == nil {
				t.Fatalf("want 4xx with error, got %d %v", code, out)
			}
			_, st := get(t, ts.URL+"/api/state")
			if subs, _ := st["submissions"].([]any); len(subs) != 0 {
				t.Fatalf("store changed: %v", subs)
			}
		})
	}
}

func TestSubmitTranscriptOnlyStillWorks(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	code, out := post(t, ts.URL+"/api/submit", `{"node_id":"A","tier":3,"rep_score":50,"transcript":"mil 190 le kilo"}`)
	if code != http.StatusCreated {
		t.Fatalf("got %d %v", code, out)
	}
}

func seedQuorum(t *testing.T, base string) {
	submit(t, base, "A", 1, 92, 187, 187, "QQ==")
	submit(t, base, "B", 2, 70, 185, 185, "Qg==")
	submit(t, base, "C", 2, 70, 188, 188, "Qw==")
}

func publications(t *testing.T, base string) int {
	_, st := get(t, base+"/api/state")
	return len(st["publications"].([]any))
}

func TestQuorumNeedsDistinctNodes(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	submit(t, ts.URL, "A", 1, 92, 187, 187, "QQ==")
	submit(t, ts.URL, "A", 1, 92, 186, 186, "Qg==")
	submit(t, ts.URL, "A", 1, 92, 188, 188, "Qw==")
	code, out := post(t, ts.URL+"/api/consensus", "")
	if code != http.StatusUnprocessableEntity || out["error"] == nil {
		t.Fatalf("want 422, got %d %v", code, out)
	}
	if n := publications(t, ts.URL); n != 0 {
		t.Fatalf("publications = %d, want 0", n)
	}
}

func TestSecondConsensusConsumesBatch(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	seedQuorum(t, ts.URL)
	if code, out := post(t, ts.URL+"/api/consensus", ""); code != http.StatusOK {
		t.Fatalf("first: %d %v", code, out)
	}
	code, _ := post(t, ts.URL+"/api/consensus", "")
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("second consensus status %d, want 422", code)
	}
	if n := publications(t, ts.URL); n != 1 {
		t.Fatalf("publications = %d, want 1", n)
	}
	// a fresh batch can run again
	submit(t, ts.URL, "A", 1, 92, 190, 190, "RA==")
	submit(t, ts.URL, "B", 2, 70, 191, 191, "RQ==")
	submit(t, ts.URL, "C", 2, 70, 192, 192, "Rg==")
	if code, out := post(t, ts.URL+"/api/consensus", ""); code != http.StatusOK {
		t.Fatalf("new batch: %d %v", code, out)
	}
	if n := publications(t, ts.URL); n != 2 {
		t.Fatalf("publications = %d, want 2", n)
	}
}

func TestConcurrentConsensusPublishesOnce(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	seedQuorum(t, ts.URL)
	const n = 8
	codes := make(chan int, n)
	for i := 0; i < n; i++ {
		go func() {
			code, _ := post(t, ts.URL+"/api/consensus", "")
			codes <- code
		}()
	}
	ok := 0
	for i := 0; i < n; i++ {
		if <-codes == http.StatusOK {
			ok++
		}
	}
	if ok != 1 {
		t.Fatalf("%d consensus runs succeeded, want 1", ok)
	}
	if p := publications(t, ts.URL); p != 1 {
		t.Fatalf("publications = %d, want 1", p)
	}
}

func TestLowConfidenceIsNotPublishedNorQuoted(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	seedQuorum(t, ts.URL)
	if code, out := post(t, ts.URL+"/api/consensus", ""); code != http.StatusOK || out["validated"] != true {
		t.Fatalf("first run: %d %v", code, out)
	}
	// scattered batch -> low confidence
	for i, v := range []float64{100, 150, 200, 250, 300} {
		submit(t, ts.URL, string(rune('P'+i)), 2, 70, v, v, string(rune('a'+i))+"A==")
	}
	code, out := post(t, ts.URL+"/api/consensus", "")
	if code != http.StatusOK || out["validated"] != false {
		t.Fatalf("want 200 validated:false, got %d %v", code, out)
	}
	if tx, _ := out["tx"].(string); tx != "" {
		t.Errorf("unvalidated run must not publish on chain, tx=%q", tx)
	}
	if n := publications(t, ts.URL); n != 1 {
		t.Fatalf("publications = %d, want 1", n)
	}
	_, ask := get(t, ts.URL+"/api/ask?q=prix")
	if !strings.Contains(ask["answer"].(string), "187") {
		t.Fatalf("ask must quote previous validated value: %v", ask)
	}
}

func TestLowConfidenceWithNoPriorPublicationAsksNothing(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	for i, v := range []float64{100, 150, 200, 250, 300} {
		submit(t, ts.URL, string(rune('P'+i)), 2, 70, v, v, string(rune('a'+i))+"A==")
	}
	_, out := post(t, ts.URL+"/api/consensus", "")
	if out["validated"] != false {
		t.Fatalf("want validated:false, got %v", out)
	}
	if n := publications(t, ts.URL); n != 0 {
		t.Fatalf("publications = %d, want 0", n)
	}
	_, ask := get(t, ts.URL+"/api/ask?q=prix")
	if strings.Contains(ask["answer"].(string), "FCFA") {
		t.Fatalf("ask must not estimate: %v", ask)
	}
}

func TestDuplicateNodeContributesLatestValueOnly(t *testing.T) {
	ts := newTestServer(1 << 20)
	defer ts.Close()
	submit(t, ts.URL, "A", 1, 92, 900, 900, "QQ==")
	submit(t, ts.URL, "A", 1, 92, 187, 187, "Qg==")
	submit(t, ts.URL, "B", 2, 70, 185, 185, "Qw==")
	submit(t, ts.URL, "C", 2, 70, 188, 188, "RA==")
	code, out := post(t, ts.URL+"/api/consensus", "")
	if code != http.StatusOK {
		t.Fatalf("%d %v", code, out)
	}
	if out["confidence"].(float64) != 100 {
		t.Errorf("confidence = %v, want 100 (stale 900 must not count)", out["confidence"])
	}
	if n := len(out["outcomes"].([]any)); n != 3 {
		t.Errorf("outcomes = %d, want 3 (one per node)", n)
	}
}
