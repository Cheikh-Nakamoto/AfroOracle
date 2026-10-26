// Package meta is a thin, dependency-free (stdlib net/http + encoding/json
// only) client for the Meta Model API used by AfriOracle.
//
// Verified API facts (agents/10-meta-client.md):
//   - Base URL: https://api.meta.ai/v1
//   - Auth: Authorization: Bearer $META_API_KEY
//   - Models: muse-spark-1.3 (chat), muse-voice-transcribe-1.0 (ASR),
//     sam-3.1 (segmentation), muse-image-1.0 (generation).
//   - Endpoints: POST /v1/chat/completions and POST /v1/responses (chat),
//     POST /v1/asr/transcribe (file transcription),
//     POST /v1/responses with sam-3.1 (segmentation),
//     POST /v1/images/generations and POST /v1/images/edits (images).
//
// Exact request/response JSON field names were checked against
// https://dev.meta.ai/docs/overview and https://dev.meta.ai/docs/models:
// the docs publish endpoint paths and model IDs but not full JSON
// schemas. Where a field is unclear this client uses the OpenAI-compatible
// shape (the docs describe chat as "The OpenAI-compatible messages-array
// endpoint"); each such choice is marked OPENAI-COMPAT below.
//
// New takes the API key as a parameter; it never reads env vars — the
// caller passes os.Getenv("META_API_KEY"). No live calls are made in
// tests (httptest.Server only).
package meta

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DefaultBaseURL is the Meta Model API root all endpoint paths hang off.
const DefaultBaseURL = "https://api.meta.ai/v1"

// Model IDs AfriOracle needs (verified against dev.meta.ai/docs/models).
const (
	ModelChat       = "muse-spark-1.3"
	ModelTranscribe = "muse-voice-transcribe-1.0"
	ModelSegment    = "sam-3.1"
	ModelImage      = "muse-image-1.0"
)

// Client is a Meta Model API client. Construct with New.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New returns a Client authenticating with apiKey against DefaultBaseURL.
func New(apiKey string) *Client {
	return &Client{baseURL: DefaultBaseURL, apiKey: apiKey, http: http.DefaultClient}
}

// postJSON POSTs body as JSON to baseURL+path and returns the raw response
// body, erroring on non-2xx status.
func (c *Client) postJSON(ctx context.Context, path string, body any) ([]byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("meta: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("meta: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("meta: do request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("meta: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("meta: %s %s: %s", path, resp.Status, truncate(raw, 512))
	}
	return raw, nil
}

func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[:n]
}

// Message is one chat turn. OPENAI-COMPAT: {role, content} shape.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is a chat/structured-extraction call. An empty Model selects
// ModelChat (muse-spark-1.3). JSONObject requests json_object
// response_format for structured extraction.
type ChatRequest struct {
	Model      string
	Messages   []Message
	MaxTokens  int
	JSONObject bool
}

// ChatResponse carries the assistant text plus the raw response body.
type ChatResponse struct {
	Text string
	Raw  []byte
}

// Chat calls POST /v1/chat/completions (Muse Spark) and returns the first
// choice's message content. OPENAI-COMPAT: request fields
// {model, messages, max_tokens?, response_format?} and response field
// {choices[0].message.content}.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = ModelChat
	}
	body := map[string]any{"model": model, "messages": req.Messages}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	if req.JSONObject {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	raw, err := c.postJSON(ctx, "/chat/completions", body)
	if err != nil {
		return ChatResponse{}, err
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return ChatResponse{}, fmt.Errorf("meta: decode chat response: %w", err)
	}
	out := ChatResponse{Raw: raw}
	if len(parsed.Choices) > 0 {
		out.Text = parsed.Choices[0].Message.Content
	}
	return out, nil
}

// Transcribe POSTs raw audio bytes to /v1/asr/transcribe (Muse Voice
// Transcribe) and returns the transcript text. format is an audio subtype
// such as "wav" or "mp3" (defaults to "wav"). OPENAI-COMPAT: response
// field {text}; the docs confirm the endpoint and model but not the exact
// response keys.
func (c *Client) Transcribe(ctx context.Context, audio []byte, format string) (string, error) {
	if format == "" {
		format = "wav"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/asr/transcribe", bytes.NewReader(audio))
	if err != nil {
		return "", fmt.Errorf("meta: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "audio/"+format)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("meta: do request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("meta: read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("meta: /asr/transcribe %s: %s", resp.Status, truncate(raw, 512))
	}
	var parsed struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("meta: decode transcribe response: %w", err)
	}
	return parsed.Text, nil
}

// Segment is one detected object: a label plus a normalized [x1 y1 x2 y2]
// box. Field names follow the docs' "Boxes and masks" description; if the
// API returns masks they are ignored by this demo client.
type Segment struct {
	Label string     `json:"label"`
	Box   [4]float64 `json:"box"`
}

// Segment sends a text prompt plus a JPEG/PNG image to SAM 3.1 via
// POST /v1/responses and returns the detected objects. The request uses the
// Responses input shape ({model, input[0].content[0..1]} with
// input_text/input_image parts, image as a data URL); the docs confirm the
// endpoint and model but not the exact JSON, so this is best-effort and
// marked OPENAI-COMPAT. The response is parsed defensively: either a direct
// {segments|objects|detections:[{label,box}]} payload or the same JSON
// embedded in a Responses output_text part.
func (c *Client) Segment(ctx context.Context, prompt string, image []byte) ([]Segment, error) {
	if len(image) == 0 {
		return nil, fmt.Errorf("meta: Segment: empty image")
	}
	imgURL := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(image)
	body := map[string]any{
		"model": ModelSegment,
		"input": []any{
			map[string]any{
				"role": "user",
				"content": []any{
					map[string]any{"type": "input_text", "text": prompt},
					map[string]any{"type": "input_image", "image_url": imgURL},
				},
			},
		},
	}
	raw, err := c.postJSON(ctx, "/responses", body)
	if err != nil {
		return nil, err
	}
	if segs := parseSegments(raw); len(segs) > 0 {
		return segs, nil
	}
	// Responses wrapper: output[].content[] text parts holding JSON.
	var wrapped struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("meta: decode segment response: %w", err)
	}
	for _, item := range wrapped.Output {
		for _, part := range item.Content {
			if part.Text == "" {
				continue
			}
			if segs := parseSegments([]byte(part.Text)); len(segs) > 0 {
				return segs, nil
			}
		}
	}
	return nil, nil
}

// parseSegments extracts [{label,box}] from a direct payload, accepting
// "segments", "objects" or "detections" keys.
func parseSegments(raw []byte) []Segment {
	var parsed struct {
		Segments   []Segment `json:"segments"`
		Objects    []Segment `json:"objects"`
		Detections []Segment `json:"detections"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil
	}
	for _, list := range [][]Segment{parsed.Segments, parsed.Objects, parsed.Detections} {
		if len(list) > 0 {
			return list
		}
	}
	return nil
}

// GenerateImage sends a text prompt to Muse Image via
// POST /v1/images/generations and returns the decoded image bytes.
// OPENAI-COMPAT: request {model, prompt}, response {data[0].b64_json}; a
// URL-only response yields an error since fetching it would be a second,
// unbilled-here network call.
func (c *Client) GenerateImage(ctx context.Context, prompt string) ([]byte, error) {
	body := map[string]any{"model": ModelImage, "prompt": prompt}
	raw, err := c.postJSON(ctx, "/images/generations", body)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("meta: decode image response: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("meta: image response has no data")
	}
	if parsed.Data[0].B64JSON != "" {
		img, err := base64.StdEncoding.DecodeString(parsed.Data[0].B64JSON)
		if err != nil {
			return nil, fmt.Errorf("meta: decode image bytes: %w", err)
		}
		return img, nil
	}
	return nil, fmt.Errorf("meta: image response returned url-only data (unsupported)")
}
