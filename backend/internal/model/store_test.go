package model

import (
	"os"
	"testing"
)

// storeContract is the behaviour every Store must satisfy. open returns a
// fresh handle onto the SAME underlying data each call (for restart tests,
// reopen=true simulates a process restart).
func storeContract(t *testing.T, open func(reopen bool) Store) {
	t.Helper()
	price := 187.0
	photo := []byte("photo-bytes-1")

	t.Run("persistence and duplicate photo across restart", func(t *testing.T) {
		s := open(false)
		pr, err := s.ScorePhoto(&price, 187, photo)
		if err != nil || pr.DuplicatePhoto || pr.Score != 94 {
			t.Fatalf("first photo: %+v %v", pr, err)
		}
		p := 90
		for _, n := range []string{"A", "B"} {
			if err := s.AddSubmission(StateSub{NodeID: n, Tier: 2, RepScore: 80, Value: 187, ProofScore: &p, Channel: "web", Status: "pending"}); err != nil {
				t.Fatal(err)
			}
		}
		if err := s.AddSubmission(StateSub{NodeID: "FAO", Tier: 3, Value: 187, Channel: "api", Status: "pending"}); err != nil {
			t.Fatal(err)
		}

		s = open(true) // restart
		subs := s.Submissions()
		if len(subs) != 3 || subs[0].NodeID != "A" || subs[2].ProofScore != nil || *subs[1].ProofScore != 90 {
			t.Fatalf("submissions after restart: %+v", subs)
		}
		if len(s.Fingerprints()) != 1 {
			t.Fatalf("fingerprints: %v", s.Fingerprints())
		}
		pr, _ = s.ScorePhoto(&price, 187, photo)
		if !pr.DuplicatePhoto || pr.Score != 12 {
			t.Fatalf("reused photo after restart not caught: %+v", pr)
		}
		if len(s.PendingBatch()) != 3 {
			t.Fatalf("pending: %d", len(s.PendingBatch()))
		}

		batch := s.PendingBatch()
		pub := Publication{Median: 187, Confidence: 100, Validated: true, Tx: "0xabc", Chain: "simulated", ValueX100: 18700, AIReportHash: "h", Timestamp: 200}
		outs := []Outcome{{NodeID: "A", Status: "accepted", Reward: 5, ScoreDelta: 1, Weight: 1.5, Note: "ok"}, {NodeID: "B", Status: "excluded", Reward: 0, ScoreDelta: -2}}
		if err := s.CommitBatch(pub, outs, batch); err != nil {
			t.Fatal(err)
		}
		// an earlier-timestamped publication must sort first
		early := pub
		early.Timestamp, early.Median = 100, 150
		extra := StateSub{NodeID: "Z", Tier: 3, Value: 150, Channel: "api", Status: "pending"}
		s.AddSubmission(extra)
		if err := s.CommitBatch(early, nil, s.PendingBatch()); err != nil {
			t.Fatal(err)
		}

		s = open(true)
		if len(s.PendingBatch()) != 0 {
			t.Fatalf("batch not consumed: %+v", s.PendingBatch())
		}
		pubs := s.Publications()
		if len(pubs) != 2 || pubs[0].Timestamp != 100 || pubs[1] != pub {
			t.Fatalf("publications order/content: %+v", pubs)
		}
		if len(s.LastOutcomes()) != 0 {
			t.Fatalf("last outcomes should be the latest run: %+v", s.LastOutcomes())
		}
		subs = s.Submissions()
		if subs[0].Status != "accepted" || subs[0].Reward != 5 || subs[1].Status != "excluded" || subs[1].ScoreDelta != -2 {
			t.Fatalf("outcomes not applied: %+v", subs)
		}
		if subs[2].Status != "pending" {
			t.Fatalf("row without outcome should stay pending: %+v", subs[2])
		}
		if s.AcceptedCount() != 1 {
			t.Fatalf("accepted count %d", s.AcceptedCount())
		}
	})
}

func TestInMemoryStoreContract(t *testing.T) {
	var cur *InMemoryStore
	storeContract(t, func(reopen bool) Store {
		if cur == nil {
			cur = NewInMemoryStore() // in-memory "restart" keeps the same object
		}
		return cur
	})
}

// TestPGStoreContract runs against a real Postgres; skipped unless
// TEST_DATABASE_URL is set.
func TestPGStoreContract(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	var cur *PGStore
	reset := func(s *PGStore) {
		if _, err := s.db.Exec(`TRUNCATE submissions, publications, last_outcomes, photo_fingerprints RESTART IDENTITY`); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		if cur != nil {
			cur.Close()
		}
	})
	storeContract(t, func(reopen bool) Store {
		if cur != nil && reopen {
			cur.Close()
			cur = nil
		}
		if cur == nil {
			s, err := NewPGStore(url)
			if err != nil {
				t.Fatal(err)
			}
			if !reopen {
				reset(s)
			}
			cur = s
		}
		return cur
	})
}

func TestPGStoreCommitBatchIsAtomic(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	s, err := NewPGStore(url)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.db.Exec(`TRUNCATE submissions, publications, last_outcomes, photo_fingerprints RESTART IDENTITY`)
	s.AddSubmission(StateSub{NodeID: "A", Tier: 3, Value: 1, Channel: "api", Status: "pending"})
	batch := s.PendingBatch()
	pub := Publication{Median: 1, Validated: true, Tx: "t", Chain: "c", Timestamp: 1}
	if err := s.CommitBatch(pub, []Outcome{{NodeID: "A", Status: "accepted", Reward: 5}}, batch); err != nil {
		t.Fatal(err)
	}
	// replaying the same batch must fail and leave no second publication
	if err := s.CommitBatch(pub, []Outcome{{NodeID: "A", Status: "accepted", Reward: 5}}, batch); err == nil {
		t.Fatal("double commit succeeded")
	}
	if n := len(s.Publications()); n != 1 {
		t.Fatalf("publications %d, want 1", n)
	}
	if len(s.LastOutcomes()) != 1 {
		t.Fatal("outcomes lost")
	}
}
