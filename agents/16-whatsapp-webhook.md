# Task 16 — secure the WhatsApp webhook

## Goal

Reject forged webhook calls, fail closed when verification secrets are missing, and deduplicate Meta message retries.

## Scope

- `backend/internal/controller/{controller,whatsapp}.go` (+ webhook tests)
- `backend/main.go`
- `.env.example`
- `state/PROJECT_STATE.md`

## Requirements

- Verify `X-Hub-Signature-256` as HMAC-SHA256 over the exact raw body with `WHATSAPP_APP_SECRET`; compare signatures in constant time.
- Fail closed for GET verification when `WHATSAPP_VERIFY_TOKEN` is unset.
- Reject malformed JSON, oversized bodies, and messages without `wamid`; log drops without logging body/phone data.
- Deduplicate by `wamid` for 24 hours so repeated Meta deliveries create at most one submission and return HTTP 200 on valid duplicate deliveries.
- For audio messages, download Meta media through the configured Graph API version, restrict the download host and size, verify the supplied checksum, then pass the audio to the injected Meta transcription client.
- Acknowledge valid processed events with HTTP 200; return a retryable 502 if media retrieval or transcription fails; reject invalid signatures with 401.

## Done criteria

- Tests cover absent/bad signature, missing verify token, signed text/audio fixtures, duplicate `wamid`, media download request/auth/checksum, and malformed payload.
- `cd backend && go test ./... && go vet ./...` and `./scripts/demo.sh` pass.
