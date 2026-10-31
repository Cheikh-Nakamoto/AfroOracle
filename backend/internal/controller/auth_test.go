package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"afrioracle/internal/model"
)

const testNodeKey = "controller-auth-test-node-A-key-0001"

func authController(t *testing.T) (*Controllers, *model.InMemoryStore) {
	t.Helper()
	store := model.NewInMemoryStore()
	registry, err := NewNodeRegistry(map[string]NodeCredential{
		"A":   {APIKey: testNodeKey, Tier: 2, RepScore: 88},
		"FAO": {APIKey: "controller-auth-test-node-FAO-key-0001", Tier: 3, RepScore: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	c := NewControllers(store)
	c.Nodes = registry
	return c, store
}

func TestSubmitRejectsMissingCredentialWithoutChangingStore(t *testing.T) {
	c, store := authController(t)
	r := httptest.NewRequest(http.MethodPost, "/api/submit", strings.NewReader(`{"node_id":"A","tier":2,"rep_score":88,"value":187}`))
	w := httptest.NewRecorder()
	c.HandleSubmit(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := len(store.Submissions()); got != 0 {
		t.Fatalf("unauthorized request stored %d submissions", got)
	}
}

func TestSubmitRejectsForgedTier(t *testing.T) {
	c, store := authController(t)
	r := httptest.NewRequest(http.MethodPost, "/api/submit", strings.NewReader(`{"node_id":"A","tier":3,"rep_score":88,"value":187,"transcript":"mil 187"}`))
	r.Header.Set("Authorization", "Bearer "+testNodeKey)
	w := httptest.NewRecorder()
	c.HandleSubmit(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := len(store.Submissions()); got != 0 {
		t.Fatalf("forged tier stored %d submissions", got)
	}
}

func TestSubmitUsesRegistryTierAndReputation(t *testing.T) {
	c, store := authController(t)
	r := httptest.NewRequest(http.MethodPost, "/api/submit", strings.NewReader(`{"node_id":"A","tier":2,"rep_score":0,"value":187,"transcript":"mil 187"}`))
	r.Header.Set("Authorization", "Bearer "+testNodeKey)
	w := httptest.NewRecorder()
	c.HandleSubmit(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	subs := store.Submissions()
	if len(subs) != 1 || subs[0].Tier != 2 || subs[0].RepScore != 88 {
		t.Fatalf("stored submission=%+v", subs)
	}
}

func TestConsensusRequiresRegisteredCredential(t *testing.T) {
	c, _ := authController(t)
	r := httptest.NewRequest(http.MethodPost, "/api/consensus", nil)
	w := httptest.NewRecorder()
	c.HandleConsensus(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestDemoRegistryIncludesAllowListedAutomaticNode(t *testing.T) {
	registry := DemoNodeRegistry()
	id, credential, ok := registry.Authenticate("Bearer " + demoKey("FAO"))
	if !ok || id != "FAO" || credential.Tier != 3 {
		t.Fatalf("identity=%q credential=%+v ok=%t", id, credential, ok)
	}
}

func TestEmptySubmitReturnsBadRequestWithoutChangingStore(t *testing.T) {
	c, store := authController(t)
	r := httptest.NewRequest(http.MethodPost, "/api/submit", strings.NewReader(`{"node_id":"A"}`))
	r.Header.Set("Authorization", "Bearer "+testNodeKey)
	w := httptest.NewRecorder()
	c.HandleSubmit(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := len(store.Submissions()); got != 0 {
		t.Fatalf("empty request stored %d submissions", got)
	}
}

func TestWhitespaceTranscriptWithoutTypedValueIsRejected(t *testing.T) {
	c, store := authController(t)
	r := httptest.NewRequest(http.MethodPost, "/api/submit", strings.NewReader(`{"node_id":"A","transcript":"  \n\t"}`))
	r.Header.Set("Authorization", "Bearer "+testNodeKey)
	w := httptest.NewRecorder()
	c.HandleSubmit(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := len(store.Submissions()); got != 0 {
		t.Fatalf("blank transcript stored %d submissions", got)
	}
}
