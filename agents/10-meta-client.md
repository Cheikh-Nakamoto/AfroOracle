# Agent 10 — Meta Model API client

You are worker agent 10 of the AfriOracle pipeline. Read `AGENTS.md`, `PIPELINE.md`, `spec.md`, and this file. Build a Go client for the Meta Model API in a new package `backend/internal/meta/`.

## Goal

A thin, dependency-free (stdlib `net/http` + `encoding/json`) client for the four capabilities AfriOracle needs: chat completion (Muse Spark 1.3), audio transcription (Muse Voice Transcribe), image segmentation (SAM 3.1), and image generation (Muse Image). No new module dependencies. Do **not** make live API calls during development — build the client and unit-test request construction offline.

## Verified API facts (use these)

- Base URL: `https://api.meta.ai/v1`
- Auth: `Authorization: Bearer $META_API_KEY`
- Model IDs: `muse-spark-1.3`, `muse-voice-transcribe-1.0`, `sam-3.1`, `muse-image-1.0`.
- Endpoints:
  - Chat / structured output: OpenAI-compatible chat completions (`/v1/chat/completions`) and Responses API (`/v1/responses`). Use chat completions for simple JSON; use `/v1/responses` when you need image+text (vision) input.
  - Transcription (file): `POST /v1/asr/transcribe`.
  - Segmentation: `POST /v1/responses` with model `sam-3.1` (text prompt + image → boxes/masks). Files via `POST /v1/files`.
  - Image: `POST /v1/images/generations`, `POST /v1/images/edits`.
- Docs to confirm exact JSON field names (WebFetch): `https://dev.meta.ai/docs/overview`, `https://dev.meta.ai/docs/models`.

## Required Go surface

```go
package meta

type Client struct{ /* baseURL, apiKey, http.Client */ }
func New(apiKey string) *Client

type Message struct{ Role, Content string }
type ChatRequest struct {
    Model      string
    Messages   []Message
    MaxTokens  int
    JSONObject bool // set to request response_format json_object for structured extraction
}
type ChatResponse struct {
    Text string
    Raw  []byte
}
func (c *Client) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)

// Transcribe sends raw audio to /v1/asr/transcribe and returns the transcript text.
func (c *Client) Transcribe(ctx context.Context, audio []byte, format string) (string, error)

// Segment sends a prompt + image to SAM 3.1 and returns detected objects (label + box/mask).
// Adjust the struct fields to the real response shape after reading the docs.
type Segment struct{ Label string; Box [4]float64 }
func (c *Client) Segment(ctx context.Context, prompt string, image []byte) ([]Segment, error)

// GenerateImage sends a text prompt to Muse Image and returns decoded image bytes.
func (c *Client) GenerateImage(ctx context.Context, prompt string) ([]byte, error)
```

Confirm the exact request/response JSON against the docs (WebFetch) and adapt the structs/fields to match. If a field is unclear, choose the OpenAI-compatible shape and note it in a comment. `New(apiKey string)` takes the key as a parameter — do **not** read env vars inside the package; the caller passes it.

## Rules

- Own ONLY `backend/internal/meta/*`. Do not edit `model/`, `controller/`, `view/`, `main.go`, `contracts/`, `frontend/`, `mobile/`, `scripts/`, `demo/`, `docs/`, `spec.md`, `README.md`, `AGENTS.md`, `PIPELINE.md`.
- No live API calls in dev (do not burn the $150). Unit-test with `httptest.Server` or by asserting the built `http.Request` (URL path, `Authorization` header, JSON body).

## Done criteria

```bash
cd backend && go build ./... && go vet ./... && go test ./...
```

- All three pass from `backend/`.
- `meta` has at least one unit test (`meta_test.go`) that verifies request construction (URL, `Authorization` header, body fields) for `Chat` and `Transcribe` without network.

## Handoff

Append one block to `state/PROJECT_STATE.md` (under ~1k tokens): `done`, `decisions_made`, `files_changed`, `open_questions`, `next_task`. Reference files by path; never paste code.
