# Agent 09 — Backend MVC refactor

You are worker agent 09 of the AfriOracle pipeline. Read `AGENTS.md`, `PIPELINE.md`, `spec.md`, and this file first, then the current backend source (`backend/main.go`, `backend/ai.go`, `backend/consensus.go`, `backend/consensus_test.go`, `backend/go.mod`). Refactor the flat `package main` backend into a Model / Controller / View layout.

## Goal

Reorganize the backend into three layers (Model, Controller, View) without changing the HTTP API, the JSON on the wire, or the demo behavior. No new dependencies. The golden case (median 187, confidence 100, D excluded) and `./scripts/demo.sh` must still pass.

## Target layout (all under `backend/`)

```
go.mod                        # unchanged (module afrioracle)
main.go                       # package main — composition root only
internal/model/
  model.go                    # domain types: Submission, Publication, Outcome, ConsensusResult, RewardEntry, StateSub
  consensus.go                # tierWeight, reputation, Weight/Accepted, weightedMedian, runConsensus, deviation, contextAgent
  ai.go                       # mock flag, ExtractResult, ProofResult, transcribe, extract, proofScore, reportHash, sha256Hex, modelName
  chain.go                    # chainPublish(result, report) (tx, chain)
  store.go                    # Store interface + InMemoryStore
  consensus_test.go           # moved from backend/consensus_test.go, package model (internal test)
internal/controller/
  controller.go               # Controllers struct holding the Store
  submit.go                   # SubmitReq DTO + handleSubmit + processSubmit
  consensus.go                # handleConsensus
  state.go                    # handleState
  ask.go                      # handleAsk
  whatsapp.go                 # handleWhatsapp (GET verify + POST message)
  root.go                     # handleRoot
  middleware.go               # cors
internal/view/
  view.go                     # writeJSON + response DTOs for the stable public shapes
```

Delete after moving their contents: `backend/ai.go`, `backend/consensus.go`, `backend/consensus_test.go`.

## Layer responsibilities

- **Model** (`internal/model`): pure domain types + business logic + storage. No `net/http`, no request-body parsing. `Store` is an interface with an in-memory implementation (`InMemoryStore`) holding `submissions`, `publications`, `fingerprints`, `lastOutcomes` behind a mutex. `SubmitReq` is NOT model — it is the controller's request DTO.
- **Controller** (`internal/controller`): HTTP handlers. Decode/validate the request, call model functions and the store, render via `view`. `processSubmit` (transcribe → extract → proofScore → build `Submission` → store) lives here and is shared by `handleSubmit` and `handleWhatsapp`.
- **View** (`internal/view`): response rendering. `writeJSON(w, status, v)` and any named response structs. Keep the exact JSON field names and shapes.
- **main** (`backend/main.go`): composition root. Build the `InMemoryStore`, the `Controllers`, register routes on an `http.ServeMux`, apply CORS, `ListenAndServe`.

## Invariants (must not change)

- Endpoints, methods, status codes: `POST /api/submit` (201 / 200 clarify), `POST /api/consensus` (200 / 422), `GET|POST /webhook/whatsapp`, `GET /api/state`, `GET /api/ask`, `GET /`.
- JSON field names: `node_id`, `tier`, `rep_score`, `value`, `photo_price`, `channel`, `transcript`, `photo_b64`, `proof_score`, `status`, `reward`, `score_delta`, `median`, `confidence`, `validated`, `outcomes`, `tx`, `chain`, `value_x100`, `aiReportHash`, `submissions`, `publications`, `rewards`, `fingerprints`, `answer`.
- Consensus behavior: tier weights `{1:3.0, 2:1.0, 3:0.7}`; reputation grid; proof<50 excluded (reward 0, delta −2); automatic node (`ProofScore` nil) accepted at 0.7; collecte rule min 3 → 422.
- Mock AI path unchanged (`META_API_KEY` unset → mock). `chainPublish` still labels `simulated` vs `polygon-amoy`.
- `consensus_test.go` uses `package model` (internal test) so it can call the unexported consensus functions.

## Done criteria

Run from repo root:

```bash
cd backend && go build ./... && go vet ./... && go test ./...
./scripts/demo.sh
```

- `go build ./...`, `go vet ./...`, `go test ./...` all pass from `backend/`.
- `./scripts/demo.sh` prints `DEMO OK: median 187 FCFA/kg (confidence 100 ...)`, D excluded.
- `curl -s localhost:8000/api/state` returns the same JSON shape as before.

## Ownership

Own: `backend/main.go`, `backend/internal/**`. Move-and-delete: `backend/ai.go`, `backend/consensus.go`, `backend/consensus_test.go`. Do NOT edit `contracts/`, `frontend/`, `scripts/`, `demo/`, `docs/`, `spec.md`, `README.md`, `AGENTS.md`, `PIPELINE.md`, or any file not listed. Exception: append your handoff block to `state/PROJECT_STATE.md`.

## Handoff

Append one block to `state/PROJECT_STATE.md` (under ~1k tokens): `done`, `decisions_made`, `files_changed`, `open_questions`, `next_task`. Reference files by path; never paste code.
