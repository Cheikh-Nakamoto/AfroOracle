package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const baseReward = 5.0

var (
	mu           sync.Mutex
	submissions  []StateSub
	publications []Publication
	fingerprints = map[string]bool{}
	lastOutcomes []Outcome
)

type SubmitReq struct {
	NodeID     string   `json:"node_id"`
	Tier       int      `json:"tier"`
	RepScore   int      `json:"rep_score"`
	Value      float64  `json:"value"`
	PhotoPrice *float64 `json:"photo_price"`
	Channel    string   `json:"channel"`
	Transcript string   `json:"transcript"`
	PhotoB64   string   `json:"photo_b64"`
}

type StateSub struct {
	NodeID     string  `json:"node_id"`
	Tier       int     `json:"tier"`
	RepScore   int     `json:"rep_score"`
	Value      float64 `json:"value"`
	ProofScore *int    `json:"proof_score"`
	Channel    string  `json:"channel"`
	Status     string  `json:"status"`
	Reward     float64 `json:"reward"`
	ScoreDelta float64 `json:"score_delta"`
}

type Publication struct {
	Median       float64 `json:"median"`
	Confidence   int     `json:"confidence"`
	Validated    bool    `json:"validated"`
	Tx           string  `json:"tx"`
	Chain        string  `json:"chain"`
	ValueX100    int64   `json:"value_x100"`
	AIReportHash string  `json:"aiReportHash"`
	Timestamp    int64   `json:"timestamp"`
}

type RewardEntry struct {
	NodeID     string  `json:"node_id"`
	Reward     float64 `json:"reward"`
	ScoreDelta float64 `json:"score_delta"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/submit", cors(handleSubmit))
	mux.HandleFunc("/api/consensus", cors(handleConsensus))
	mux.HandleFunc("/api/state", cors(handleState))
	mux.HandleFunc("/api/ask", cors(handleAsk))
	mux.HandleFunc("/webhook/whatsapp", handleWhatsapp)
	mux.HandleFunc("/", handleRoot)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	fmt.Println("AfriOracle server on :" + port)
	http.ListenAndServe(":"+port, mux)
}

func cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

func handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req SubmitReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	status, resp := processSubmit(req)
	writeJSON(w, status, resp)
}

func processSubmit(req SubmitReq) (int, any) {
	transcript := req.Transcript
	if strings.TrimSpace(transcript) == "" {
		transcript = transcribe("")
	}
	var typed *float64
	if req.Value != 0 {
		typed = &req.Value
	}
	ext := extract(transcript, typed)
	if ext.NeedsConfirmation {
		return http.StatusOK, map[string]string{"status": "clarify", "question": ext.Question}
	}

	value := 0.0
	if ext.Prix != nil {
		value = *ext.Prix
	}

	var proofScorePtr *int
	proof := ProofResult{}
	if req.Tier == 3 {
		proofScorePtr = nil // automatic node (source hash) -> always accepted
	} else {
		photoBytes, _ := base64.StdEncoding.DecodeString(req.PhotoB64)
		pr := proofScore(req.PhotoPrice, value, photoBytes, fingerprints)
		proofScorePtr = &pr.Score
		proof = pr
	}

	sub := Submission{
		NodeID:     req.NodeID,
		Tier:       req.Tier,
		RepScore:   req.RepScore,
		Value:      value,
		ProofScore: proofScorePtr,
		Channel:    req.Channel,
	}

	mu.Lock()
	submissions = append(submissions, StateSub{
		NodeID:     sub.NodeID,
		Tier:       sub.Tier,
		RepScore:   sub.RepScore,
		Value:      sub.Value,
		ProofScore: sub.ProofScore,
		Channel:    sub.Channel,
		Status:     "pending",
	})
	mu.Unlock()

	return http.StatusCreated, map[string]any{
		"status":     "accepted",
		"submission": sub,
		"proof":      proof,
	}
}

func handleConsensus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	mu.Lock()
	subs := make([]Submission, 0, len(submissions))
	for _, s := range submissions {
		subs = append(subs, Submission{
			NodeID:     s.NodeID,
			Tier:       s.Tier,
			RepScore:   s.RepScore,
			Value:      s.Value,
			ProofScore: s.ProofScore,
			Channel:    s.Channel,
		})
	}
	mu.Unlock()

	accepted := 0
	for _, s := range subs {
		if s.Accepted() && s.Weight() > 0 {
			accepted++
		}
	}
	if accepted < 3 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": "collecte rule: minimum 3 independent submissions required",
		})
		return
	}

	result := runConsensus(subs, baseReward)
	report := map[string]any{"median": result.Median, "confidence": result.Confidence, "outcomes": len(result.Outcomes)}
	tx, chain := chainPublish(result, report)

	pub := Publication{
		Median:       result.Median,
		Confidence:   result.Confidence,
		Validated:    result.Validated,
		Tx:           tx,
		Chain:        chain,
		ValueX100:    int64(result.Median*100 + 0.5),
		AIReportHash: reportHash(report),
		Timestamp:    time.Now().Unix(),
	}

	mu.Lock()
	publications = append(publications, pub)
	lastOutcomes = result.Outcomes
	byNode := map[string]Outcome{}
	for _, o := range result.Outcomes {
		byNode[o.NodeID] = o
	}
	for i := range submissions {
		if o, ok := byNode[submissions[i].NodeID]; ok {
			submissions[i].Status = o.Status
			submissions[i].Reward = o.Reward
			submissions[i].ScoreDelta = o.ScoreDelta
		}
	}
	mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"median":       result.Median,
		"confidence":   result.Confidence,
		"validated":    result.Validated,
		"outcomes":     result.Outcomes,
		"tx":           tx,
		"chain":        chain,
		"value_x100":   pub.ValueX100,
		"aiReportHash": pub.AIReportHash,
	})
}

func handleState(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	fps := make([]string, 0, len(fingerprints))
	for f := range fingerprints {
		fps = append(fps, f)
	}
	rewards := make([]RewardEntry, 0, len(lastOutcomes))
	for _, o := range lastOutcomes {
		rewards = append(rewards, RewardEntry{NodeID: o.NodeID, Reward: o.Reward, ScoreDelta: o.ScoreDelta})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"submissions":  submissions,
		"publications": publications,
		"rewards":      rewards,
		"fingerprints": fps,
	})
}

func handleAsk(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	if len(publications) == 0 {
		writeJSON(w, http.StatusOK, map[string]string{
			"answer": "Aucune donnée publiée pour le moment. Je n'estime jamais un prix moi-même.",
		})
		return
	}
	last := publications[len(publications)-1]
	answer := fmt.Sprintf("Prix du mil à Kaolack : %.0f FCFA/kg, publié le %s, %d sources, confiance %d. Tx : %s",
		last.Median, time.Unix(last.Timestamp, 0).Format("15:04"), acceptedCount(), last.Confidence, last.Tx)
	writeJSON(w, http.StatusOK, map[string]string{"answer": answer})
}

func handleWhatsapp(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if r.URL.Query().Get("hub.verify_token") == os.Getenv("WHATSAPP_VERIFY_TOKEN") {
			w.Write([]byte(r.URL.Query().Get("hub.challenge")))
			return
		}
		http.Error(w, "verification failed", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost {
		var msg struct {
			Entry []struct {
				Changes []struct {
					Value struct {
						Messages []struct {
							Text struct {
								Body string `json:"body"`
							} `json:"text"`
						} `json:"messages"`
					} `json:"value"`
				} `json:"changes"`
			} `json:"entry"`
		}
		json.NewDecoder(r.Body).Decode(&msg)
		body := ""
		for _, e := range msg.Entry {
			for _, c := range e.Changes {
				for _, m := range c.Value.Messages {
					body = m.Text.Body
				}
			}
		}
		if body != "" {
			status, resp := processSubmit(SubmitReq{Transcript: body, Channel: "whatsapp"})
			writeJSON(w, status, resp)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "GET/POST only", http.StatusMethodNotAllowed)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte("<!doctype html><title>AfriOracle</title><h1>AfriOracle dashboard</h1><p>See <code>frontend/</code> for the Next.js dashboard.</p>"))
}

func acceptedCount() int {
	n := 0
	for _, s := range submissions {
		if s.Status == "accepted" {
			n++
		}
	}
	return n
}

// chainPublish returns the tx hash and chain label. Simulated ("0x"+sha256) when
// POLYGON_RPC/PRIVATE_KEY are unset; otherwise a go-ethereum call to
// AgriculturalData.publish. TODO: wire the real go-ethereum call using contracts/abi.json.
func chainPublish(result ConsensusResult, report any) (tx, chain string) {
	if os.Getenv("POLYGON_RPC") == "" || os.Getenv("PRIVATE_KEY") == "" {
		return reportHash(report), "simulated"
	}
	return reportHash(report), "polygon-amoy"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
