package model

import "sync"

// Store is the backend's operational read/aggregation store (PostgreSQL in
// production; InMemoryStore when DATABASE_URL is unset).
type Store interface {
	AddSubmission(StateSub)
	Submissions() []StateSub
	ScorePhoto(photoPrice *float64, submitted float64, photoBytes []byte) ProofResult
	TakeConsensusBatch(batchID string) (ConsensusBatch, BatchStatus)
	CompleteConsensusBatch(batchID string, run ConsensusRun) bool
	Publications() []Publication
	LastOutcomes() []Outcome
	Fingerprints() []string
	AcceptedCount() int
}

type BatchStatus int

const (
	BatchInsufficient BatchStatus = iota
	BatchStarted
	BatchReplay
	BatchInProgress
)

// ConsensusBatch is a one-time snapshot of all submissions assigned to one
// consensus run. A completed run is retained for idempotent retries.
type ConsensusBatch struct {
	ID          string
	Submissions []StateSub
	Run         *ConsensusRun
}

type ConsensusRun struct {
	BatchID     string
	Result      ConsensusResult
	Tx          string
	Chain       string
	Publication *Publication
}

// InMemoryStore holds submissions, publications, photo fingerprints and the
// last consensus outcomes behind a mutex.
type InMemoryStore struct {
	mu               sync.Mutex
	submissions      []StateSub
	publications     []Publication
	fingerprints     map[string]bool
	lastOutcomes     []Outcome
	batches          map[string]*ConsensusBatch
	nextSubmissionID uint64
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{fingerprints: map[string]bool{}, batches: map[string]*ConsensusBatch{}}
}

func (s *InMemoryStore) AddSubmission(sub StateSub) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextSubmissionID++
	sub.ID = s.nextSubmissionID
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
	s.mu.Lock()
	defer s.mu.Unlock()
	return proofScore(photoPrice, submitted, photoBytes, s.fingerprints)
}

// TakeConsensusBatch atomically claims the current pending submissions if
// they contain at least three distinct accepted node IDs. A claimed batch is
// unavailable to later runs; a repeated batch ID returns its cached result.
func (s *InMemoryStore) TakeConsensusBatch(batchID string) (ConsensusBatch, BatchStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.batches[batchID]; ok {
		if existing.Run != nil {
			return cloneConsensusBatch(existing), BatchReplay
		}
		return cloneConsensusBatch(existing), BatchInProgress
	}

	pending := make([]StateSub, 0)
	winners := make(map[string]uint64)
	for _, sub := range s.submissions {
		if sub.batchID != "" {
			continue
		}
		pending = append(pending, sub)
		candidate := submissionFromState(sub)
		if sub.NodeID != "" && candidate.Accepted() && candidate.Weight() > 0 {
			if _, exists := winners[sub.NodeID]; !exists {
				winners[sub.NodeID] = sub.ID
			}
		}
	}
	if len(winners) < 3 {
		return ConsensusBatch{ID: batchID, Submissions: pending}, BatchInsufficient
	}
	duplicateIDs := make(map[uint64]bool)
	for i := range pending {
		if winner, exists := winners[pending[i].NodeID]; exists && pending[i].ID != winner {
			candidate := submissionFromState(pending[i])
			if candidate.Accepted() && candidate.Weight() > 0 {
				pending[i].DuplicateSource = true
				duplicateIDs[pending[i].ID] = true
			}
		}
	}
	for i := range s.submissions {
		if s.submissions[i].batchID == "" {
			s.submissions[i].batchID = batchID
			s.submissions[i].DuplicateSource = duplicateIDs[s.submissions[i].ID]
		}
	}
	claimed := &ConsensusBatch{ID: batchID, Submissions: pending}
	s.batches[batchID] = claimed
	return cloneConsensusBatch(claimed), BatchStarted
}

// CompleteConsensusBatch commits the run once. A nil Publication records a
// consumed but unvalidated result without changing the public price feed.
func (s *InMemoryStore) CompleteConsensusBatch(batchID string, run ConsensusRun) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	batch, ok := s.batches[batchID]
	if !ok || batch.Run != nil {
		return false
	}
	run.BatchID = batchID
	batch.Run = cloneConsensusRun(&run)
	if run.Publication != nil {
		s.publications = append(s.publications, *run.Publication)
	}
	s.lastOutcomes = append([]Outcome(nil), run.Result.Outcomes...)
	byID := make(map[uint64]Outcome, len(run.Result.Outcomes))
	for _, outcome := range run.Result.Outcomes {
		byID[outcome.SubmissionID] = outcome
	}
	for i := range s.submissions {
		if s.submissions[i].batchID != batchID {
			continue
		}
		if outcome, ok := byID[s.submissions[i].ID]; ok {
			s.submissions[i].Status = outcome.Status
			s.submissions[i].Reward = outcome.Reward
			s.submissions[i].ScoreDelta = outcome.ScoreDelta
		}
	}
	return true
}

func submissionFromState(sub StateSub) Submission {
	return Submission{
		NodeID: sub.NodeID, Tier: sub.Tier, RepScore: sub.RepScore, Value: sub.Value,
		ProofScore: sub.ProofScore, Channel: sub.Channel, ID: sub.ID,
		DuplicateSource: sub.DuplicateSource,
	}
}

func cloneConsensusRun(run *ConsensusRun) *ConsensusRun {
	if run == nil {
		return nil
	}
	clone := *run
	clone.Result.Outcomes = append([]Outcome(nil), run.Result.Outcomes...)
	if run.Publication != nil {
		publication := *run.Publication
		clone.Publication = &publication
	}
	return &clone
}

func cloneConsensusBatch(batch *ConsensusBatch) ConsensusBatch {
	clone := *batch
	clone.Submissions = append([]StateSub(nil), batch.Submissions...)
	clone.Run = cloneConsensusRun(batch.Run)
	return clone
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
