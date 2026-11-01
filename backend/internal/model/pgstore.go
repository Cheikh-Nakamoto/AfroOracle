package model

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
)

// PGStore is the PostgreSQL-backed Store, selected when DATABASE_URL is set.
// Submissions, publications, the last outcomes and photo fingerprints all
// live in the database, so a restart loses nothing.
type PGStore struct {
	db *sql.DB
}

const opTimeout = 10 * time.Second

// schema is applied on startup under an advisory lock, so concurrent
// instances do not race. Every statement is idempotent.
var schema = []string{
	`CREATE TABLE IF NOT EXISTS submissions (
		seq         BIGSERIAL PRIMARY KEY,
		node_id     TEXT NOT NULL,
		tier        INTEGER NOT NULL,
		rep_score   INTEGER NOT NULL,
		value       DOUBLE PRECISION NOT NULL,
		proof_score INTEGER,
		channel     TEXT NOT NULL,
		status      TEXT NOT NULL,
		reward      DOUBLE PRECISION NOT NULL DEFAULT 0,
		score_delta DOUBLE PRECISION NOT NULL DEFAULT 0,
		consumed    BOOLEAN NOT NULL DEFAULT FALSE
	)`,
	`CREATE TABLE IF NOT EXISTS publications (
		id             BIGSERIAL PRIMARY KEY,
		median         DOUBLE PRECISION NOT NULL,
		confidence     INTEGER NOT NULL,
		validated      BOOLEAN NOT NULL,
		tx             TEXT NOT NULL,
		chain          TEXT NOT NULL,
		value_x100     BIGINT NOT NULL,
		ai_report_hash TEXT NOT NULL,
		ts             BIGINT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS publications_ts ON publications (ts, id)`,
	`CREATE TABLE IF NOT EXISTS last_outcomes (
		pos         INTEGER PRIMARY KEY,
		node_id     TEXT NOT NULL,
		status      TEXT NOT NULL,
		note        TEXT NOT NULL,
		reward      DOUBLE PRECISION NOT NULL,
		score_delta DOUBLE PRECISION NOT NULL,
		deviation   DOUBLE PRECISION NOT NULL,
		weight      DOUBLE PRECISION NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS photo_fingerprints (
		fingerprint TEXT PRIMARY KEY,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
}

// NewPGStore connects to url, verifies the connection and migrates the schema.
func NewPGStore(url string) (*PGStore, error) {
	db, err := sql.Open("pgx", url)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect: %w", err)
	}
	s := &PGStore{db: db}
	if err := s.migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *PGStore) Close() error { return s.db.Close() }

func (s *PGStore) migrate(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(4902001)`); err != nil {
		return err
	}
	for _, stmt := range schema {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func ctxTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), opTimeout)
}

func (s *PGStore) AddSubmission(sub StateSub) error {
	ctx, cancel := ctxTimeout()
	defer cancel()
	_, err := s.db.ExecContext(ctx, `INSERT INTO submissions
		(node_id, tier, rep_score, value, proof_score, channel, status, reward, score_delta, consumed)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		sub.NodeID, sub.Tier, sub.RepScore, sub.Value, sub.ProofScore, sub.Channel,
		sub.Status, sub.Reward, sub.ScoreDelta, sub.Consumed)
	return err
}

const subCols = `seq, node_id, tier, rep_score, value, proof_score, channel, status, reward, score_delta, consumed`

func (s *PGStore) querySubs(where string) []StateSub {
	ctx, cancel := ctxTimeout()
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `SELECT `+subCols+` FROM submissions `+where+` ORDER BY seq`)
	if err != nil {
		log.Printf("store: query submissions: %v", err)
		return nil
	}
	defer rows.Close()
	out := []StateSub{}
	for rows.Next() {
		var sub StateSub
		var proof sql.NullInt64
		var seq int64
		if err := rows.Scan(&seq, &sub.NodeID, &sub.Tier, &sub.RepScore, &sub.Value, &proof,
			&sub.Channel, &sub.Status, &sub.Reward, &sub.ScoreDelta, &sub.Consumed); err != nil {
			log.Printf("store: scan submission: %v", err)
			return nil
		}
		sub.Seq = int(seq)
		if proof.Valid {
			p := int(proof.Int64)
			sub.ProofScore = &p
		}
		out = append(out, sub)
	}
	return out
}

func (s *PGStore) Submissions() []StateSub { return s.querySubs("") }

func (s *PGStore) PendingBatch() []StateSub { return s.querySubs("WHERE NOT consumed") }

// ScorePhoto claims the photo fingerprint atomically (INSERT ... ON CONFLICT)
// so a reused photo is caught across restarts and across instances.
func (s *PGStore) ScorePhoto(photoPrice *float64, submitted float64, photoBytes []byte) (ProofResult, error) {
	fp := sha256Hex(photoBytes)
	ctx, cancel := ctxTimeout()
	defer cancel()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO photo_fingerprints (fingerprint) VALUES ($1) ON CONFLICT DO NOTHING`, fp)
	if err != nil {
		return ProofResult{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return ProofResult{}, err
	}
	seen := map[string]bool{}
	if n == 0 {
		seen[fp] = true
	}
	return proofScore(photoPrice, submitted, photoBytes, seen), nil
}

// CommitBatch writes the publication, replaces the last outcomes and consumes
// the batch in one transaction. A row already consumed by a concurrent run
// aborts everything, so a batch is never published or rewarded twice.
func (s *PGStore) CommitBatch(pub Publication, outcomes []Outcome, batch []StateSub) error {
	ctx, cancel := ctxTimeout()
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `INSERT INTO publications
		(median, confidence, validated, tx, chain, value_x100, ai_report_hash, ts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		pub.Median, pub.Confidence, pub.Validated, pub.Tx, pub.Chain,
		pub.ValueX100, pub.AIReportHash, pub.Timestamp); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM last_outcomes`); err != nil {
		return err
	}
	byNode := map[string]Outcome{}
	for i, o := range outcomes {
		byNode[o.NodeID] = o
		if _, err := tx.ExecContext(ctx, `INSERT INTO last_outcomes
			(pos, node_id, status, note, reward, score_delta, deviation, weight)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			i, o.NodeID, o.Status, o.Note, o.Reward, o.ScoreDelta, o.Deviation, o.Weight); err != nil {
			return err
		}
	}
	for _, b := range batch {
		var res sql.Result
		if o, ok := byNode[b.NodeID]; ok {
			res, err = tx.ExecContext(ctx, `UPDATE submissions SET consumed = TRUE,
				status = $2, reward = $3, score_delta = $4 WHERE seq = $1 AND NOT consumed`,
				b.Seq, o.Status, o.Reward, o.ScoreDelta)
		} else {
			res, err = tx.ExecContext(ctx,
				`UPDATE submissions SET consumed = TRUE WHERE seq = $1 AND NOT consumed`, b.Seq)
		}
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return fmt.Errorf("submission %d already consumed or missing", b.Seq)
		}
	}
	return tx.Commit()
}

func (s *PGStore) Publications() []Publication {
	ctx, cancel := ctxTimeout()
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `SELECT median, confidence, validated, tx, chain,
		value_x100, ai_report_hash, ts FROM publications ORDER BY ts, id`)
	if err != nil {
		log.Printf("store: query publications: %v", err)
		return nil
	}
	defer rows.Close()
	out := []Publication{}
	for rows.Next() {
		var p Publication
		if err := rows.Scan(&p.Median, &p.Confidence, &p.Validated, &p.Tx, &p.Chain,
			&p.ValueX100, &p.AIReportHash, &p.Timestamp); err != nil {
			log.Printf("store: scan publication: %v", err)
			return nil
		}
		out = append(out, p)
	}
	return out
}

func (s *PGStore) LastOutcomes() []Outcome {
	ctx, cancel := ctxTimeout()
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `SELECT node_id, status, note, reward, score_delta,
		deviation, weight FROM last_outcomes ORDER BY pos`)
	if err != nil {
		log.Printf("store: query outcomes: %v", err)
		return nil
	}
	defer rows.Close()
	out := []Outcome{}
	for rows.Next() {
		var o Outcome
		if err := rows.Scan(&o.NodeID, &o.Status, &o.Note, &o.Reward, &o.ScoreDelta,
			&o.Deviation, &o.Weight); err != nil {
			log.Printf("store: scan outcome: %v", err)
			return nil
		}
		out = append(out, o)
	}
	return out
}

func (s *PGStore) Fingerprints() []string {
	ctx, cancel := ctxTimeout()
	defer cancel()
	rows, err := s.db.QueryContext(ctx, `SELECT fingerprint FROM photo_fingerprints ORDER BY created_at, fingerprint`)
	if err != nil {
		log.Printf("store: query fingerprints: %v", err)
		return nil
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil
		}
		out = append(out, f)
	}
	return out
}

func (s *PGStore) AcceptedCount() int {
	ctx, cancel := ctxTimeout()
	defer cancel()
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM submissions WHERE status = 'accepted'`).Scan(&n); err != nil {
		log.Printf("store: count accepted: %v", err)
	}
	return n
}
