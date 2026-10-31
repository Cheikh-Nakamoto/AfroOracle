package model

import "sync"

// Store is the backend's operational read/aggregation store (PostgreSQL in
// production; InMemoryStore when DATABASE_URL is unset).
type Store interface {
	AddSubmission(StateSub)
	Submissions() []StateSub
	ScorePhoto(photoPrice *float64, submitted float64, photoBytes []byte) ProofResult
	AddPublication(pub Publication, outcomes []Outcome)
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

func (s *InMemoryStore) AddSubmission(sub StateSub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.submissions = append(s.submissions, sub)
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
func (s *InMemoryStore) ScorePhoto(photoPrice *float64, submitted float64, photoBytes []byte) ProofResult {
	if len(photoBytes) == 0 {
		// An absent image has no content fingerprint and must not make every
		// subsequent submission look like a reused photo.
		return proofScore(photoPrice, submitted, nil, map[string]bool{})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return proofScore(photoPrice, submitted, photoBytes, s.fingerprints)
}

func (s *InMemoryStore) AddPublication(pub Publication, outcomes []Outcome) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.publications = append(s.publications, pub)
	s.lastOutcomes = outcomes
	byNode := map[string]Outcome{}
	for _, o := range outcomes {
		byNode[o.NodeID] = o
	}
	for i := range s.submissions {
		if o, ok := byNode[s.submissions[i].NodeID]; ok {
			s.submissions[i].Status = o.Status
			s.submissions[i].Reward = o.Reward
			s.submissions[i].ScoreDelta = o.ScoreDelta
		}
	}
}

func (s *InMemoryStore) Publications() []Publication {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Publication, len(s.publications))
	copy(out, s.publications)
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
