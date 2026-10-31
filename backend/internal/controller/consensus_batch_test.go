package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"afrioracle/internal/model"
)

func addAcceptedNode(store *model.InMemoryStore, node string, value float64) {
	proof := 94
	store.AddSubmission(model.StateSub{NodeID: node, Tier: 2, RepScore: 65, Value: value, ProofScore: &proof, Channel: "test", Status: "pending"})
}

func callConsensus(t *testing.T, c *Controllers, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/consensus", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+testNodeKey)
	w := httptest.NewRecorder()
	c.HandleConsensus(w, r)
	var payload map[string]any
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &payload)
	}
	return w, payload
}

func TestConsensusRequiresThreeDistinctNodeIDs(t *testing.T) {
	c, store := authController(t)
	addAcceptedNode(store, "A", 187)
	addAcceptedNode(store, "A", 188)
	addAcceptedNode(store, "A", 189)
	w, _ := callConsensus(t, c, `{"batch_id":"same-node"}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := len(store.Publications()); got != 0 {
		t.Fatalf("same-node sources produced %d publications", got)
	}
	if got := len(store.Submissions()); got != 3 {
		t.Fatalf("insufficient batch consumed submissions: %d remain", got)
	}
}

func TestConsensusBatchIsIdempotentAndConsumed(t *testing.T) {
	c, store := authController(t)
	addAcceptedNode(store, "A", 187)
	addAcceptedNode(store, "B", 188)
	addAcceptedNode(store, "C", 186)
	body := `{"batch_id":"retry-1"}`
	first, firstPayload := callConsensus(t, c, body)
	if first.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	second, secondPayload := callConsensus(t, c, body)
	if second.Code != http.StatusOK {
		t.Fatalf("retry status=%d body=%s", second.Code, second.Body.String())
	}
	if secondPayload["replayed"] != true {
		t.Fatalf("retry not marked replayed: %v", secondPayload)
	}
	if firstPayload["batch_id"] != secondPayload["batch_id"] {
		t.Fatalf("first/replay batch IDs differ: %v / %v", firstPayload, secondPayload)
	}
	third, _ := callConsensus(t, c, `{"batch_id":"new-batch"}`)
	if third.Code != http.StatusUnprocessableEntity {
		t.Fatalf("new empty batch status=%d body=%s", third.Code, third.Body.String())
	}
	if got := len(store.Publications()); got != 1 {
		t.Fatalf("publications=%d, want exactly one", got)
	}
}

func TestConcurrentConsensusRetryPublishesOnce(t *testing.T) {
	c, store := authController(t)
	addAcceptedNode(store, "A", 187)
	addAcceptedNode(store, "B", 188)
	addAcceptedNode(store, "C", 186)
	const attempts = 8
	statuses := make(chan int, attempts)
	var workers sync.WaitGroup
	for i := 0; i < attempts; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			r := httptest.NewRequest(http.MethodPost, "/api/consensus", strings.NewReader(`{"batch_id":"concurrent-retry"}`))
			r.Header.Set("Authorization", "Bearer "+testNodeKey)
			w := httptest.NewRecorder()
			c.HandleConsensus(w, r)
			statuses <- w.Code
		}()
	}
	workers.Wait()
	close(statuses)
	for status := range statuses {
		if status != http.StatusOK && status != http.StatusConflict {
			t.Fatalf("concurrent retry returned HTTP %d", status)
		}
	}
	if got := len(store.Publications()); got != 1 {
		t.Fatalf("concurrent retries created %d publications, want 1", got)
	}
}

func TestConsensusCountsDuplicateNodeOnceWhenOtherSourcesReachQuorum(t *testing.T) {
	c, store := authController(t)
	addAcceptedNode(store, "A", 187)
	addAcceptedNode(store, "A", 240)
	addAcceptedNode(store, "B", 188)
	addAcceptedNode(store, "C", 186)
	w, payload := callConsensus(t, c, `{"batch_id":"duplicate-source"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if payload["median"] != float64(187) {
		t.Fatalf("median=%v, want 187", payload["median"])
	}
	var aOutcomes int
	var duplicateExcluded bool
	for _, raw := range payload["outcomes"].([]any) {
		outcome := raw.(map[string]any)
		if outcome["NodeID"] == "A" {
			aOutcomes++
			if outcome["Status"] == "excluded" && strings.Contains(outcome["Note"].(string), "déjà comptée") {
				duplicateExcluded = true
			}
		}
	}
	if aOutcomes != 2 || !duplicateExcluded {
		t.Fatalf("duplicate node outcomes=%v", payload["outcomes"])
	}
}

func TestUnvalidatedConsensusDoesNotPublishOrReplaceAskValue(t *testing.T) {
	c, store := authController(t)
	addAcceptedNode(store, "previous-A", 187)
	addAcceptedNode(store, "previous-B", 187)
	addAcceptedNode(store, "previous-C", 187)
	previousBatch, previousStatus := store.TakeConsensusBatch("previous-batch")
	if previousStatus != model.BatchStarted {
		t.Fatalf("previous batch status=%v", previousStatus)
	}
	previousOutcomes := make([]model.Outcome, 0, len(previousBatch.Submissions))
	for _, sub := range previousBatch.Submissions {
		previousOutcomes = append(previousOutcomes, model.Outcome{NodeID: sub.NodeID, Status: "accepted", SubmissionID: sub.ID})
	}
	previousPublication := model.Publication{Median: 187, Confidence: 100, Validated: true, Tx: "previous", Chain: "simulated", Timestamp: time.Now().Unix()}
	previousRun := model.ConsensusRun{Result: model.ConsensusResult{Median: 187, Confidence: 100, Validated: true, Outcomes: previousOutcomes}, Tx: "previous", Chain: "simulated", Publication: &previousPublication}
	if !store.CompleteConsensusBatch("previous-batch", previousRun) {
		t.Fatal("could not seed prior validated batch")
	}
	addAcceptedNode(store, "A", 100)
	addAcceptedNode(store, "B", 100)
	addAcceptedNode(store, "C", 1000)
	addAcceptedNode(store, "D", 1000)
	w, payload := callConsensus(t, c, `{"batch_id":"low-confidence"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if payload["validated"] != false || payload["chain"] != "not-published" || payload["tx"] != "" {
		t.Fatalf("unvalidated response=%v", payload)
	}
	if got := len(store.Publications()); got != 1 {
		t.Fatalf("unvalidated run added publication; total=%d", got)
	}
	var checked int
	for _, sub := range store.Submissions() {
		if sub.NodeID == "A" || sub.NodeID == "B" || sub.NodeID == "C" || sub.NodeID == "D" {
			checked++
			if sub.Status != "unvalidated" || sub.Reward != 0 {
				t.Fatalf("unvalidated node state=%+v", sub)
			}
		}
	}
	if checked != 4 {
		t.Fatalf("checked %d low-confidence submissions, want 4", checked)
	}
	askResponse := httptest.NewRecorder()
	c.HandleAsk(askResponse, httptest.NewRequest(http.MethodGet, "/api/ask?q=mil", nil))
	if !strings.Contains(askResponse.Body.String(), "187") {
		t.Fatalf("ask did not retain previous validated value: %s", askResponse.Body.String())
	}
}
