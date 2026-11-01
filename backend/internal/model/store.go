package model

import (
	"sort"
	"sync"
)

// Store is the backend's operational read/aggregation store (PostgreSQL in
// production; InMemoryStore when DATABASE_URL is unset).
type Store interface {
	AddSubmission(StateSub) error
	Submissions() []StateSub
	ScorePhoto(photoPrice *float64, submitted float64, photoBytes []byte) (ProofResult, error)
	// PendingBatch returns the submissions not yet consumed by a consensus run.
	PendingBatch() []StateSub
	// CommitBatch records the publication and consumes exactly the given
	// batch (matched by Seq), applying outcomes only to those submissions.
	CommitBatch(pub Publication, outcomes []Outcome, batch []StateSub) error
	Publications() []Publication
	LastOutcomes() []Outcome
	Fingerprints() []string
	AcceptedCount() int
}

// InMemoryStore holds submissions, publications, photo fingerprints and the
// last consensus outcomes behind a mutex.
type InMemoryStore struct {
	mu           sync.Mutex
	submissions  []StateSub
	publications []Publication
	fingerprints map[string]bool
	lastOutcomes []Outcome
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{fingerprints: map[string]bool{}}
}

func (s *InMemoryStore) AddSubmission(sub StateSub) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.submissions = append(s.submissions, sub)
	return nil
}

func (s *InMemoryStore) Submissions() []StateSub {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]StateSub, len(s.submissions))
	copy(out, s.submissions)
	return out
}

// ScorePhoto runs proofScore against the store's fingerprint set so duplicate
// photos are detected across submissions.
func (s *InMemoryStore) ScorePhoto(photoPrice *float64, submitted float64, photoBytes []byte) (ProofResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return proofScore(photoPrice, submitted, photoBytes, s.fingerprints), nil
}

func (s *InMemoryStore) PendingBatch() []StateSub {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []StateSub
	for i, sub := range s.submissions {
		if !sub.Consumed {
			sub.Seq = i
			out = append(out, sub)
		}
	}
	return out
}

func (s *InMemoryStore) CommitBatch(pub Publication, outcomes []Outcome, batch []StateSub) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publications = append(s.publications, pub)
	s.lastOutcomes = outcomes
	byNode := map[string]Outcome{}
	for _, o := range outcomes {
		byNode[o.NodeID] = o
	}
	for _, b := range batch {
		if b.Seq < 0 || b.Seq >= len(s.submissions) {
			continue
		}
		sub := &s.submissions[b.Seq]
		sub.Consumed = true
		if o, ok := byNode[sub.NodeID]; ok {
			sub.Status = o.Status
			sub.Reward = o.Reward
			sub.ScoreDelta = o.ScoreDelta
		}
	}
	return nil
}

func (s *InMemoryStore) Publications() []Publication {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Publication, len(s.publications))
	copy(out, s.publications)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Timestamp < out[j].Timestamp })
	return out
}

func (s *InMemoryStore) LastOutcomes() []Outcome {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Outcome, len(s.lastOutcomes))
	copy(out, s.lastOutcomes)
	return out
}

func (s *InMemoryStore) Fingerprints() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	fps := make([]string, 0, len(s.fingerprints))
	for f := range s.fingerprints {
		fps = append(fps, f)
	}
	return fps
}

func (s *InMemoryStore) AcceptedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, sub := range s.submissions {
		if sub.Status == "accepted" {
			n++
		}
	}
	return n
}
