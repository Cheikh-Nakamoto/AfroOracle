#!/usr/bin/env bash
# AfriOracle end-to-end demo run.
#
# Submits 8 honest agents (A, B, C, E, F, G, H + FAO automatic node) plus the
# scripted fraudster (D, who reuses A's photo), then runs consensus and asks
# the assistant. Expected: median 187, confidence 100, D excluded (score 12),
# assistant answers with the published value.
#
# Usage: ./scripts/demo.sh  (run from the repo root)
# Env:   BASE (default http://localhost:8000). If no server answers at BASE,
#        this script starts one (`cd backend && go run .`) and stops it on exit.
set -euo pipefail

BASE="${BASE:-http://localhost:8000}"
STARTED_BY_ME=0
SERVER_PID=""

cleanup() {
  if [ "$STARTED_BY_ME" -eq 1 ] && [ -n "$SERVER_PID" ]; then
    kill "$SERVER_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

# Start the backend if nothing answers at BASE. We build a binary (not
# `go run`, whose child outlives a kill of the wrapper) so the EXIT trap
# stops exactly what we started.
if ! curl -sf -o /dev/null "$BASE/api/state" 2>/dev/null; then
  echo "==> no server at $BASE, starting backend (mock AI path)..."
  (cd backend && go build -o /tmp/afrioracle-demo-bin .)
  PORT="${PORT:-8000}" META_API_KEY= AFRIORACLE_DEMO_MODE=1 /tmp/afrioracle-demo-bin > /tmp/afrioracle-demo.log 2>&1 &
  SERVER_PID="$!"
  STARTED_BY_ME=1
  for _ in $(seq 1 40); do
    if curl -sf -o /dev/null "$BASE/api/state" 2>/dev/null; then
      break
    fi
    sleep 0.5
  done
  curl -sf -o /dev/null "$BASE/api/state" || { echo "FAIL: backend did not start (see /tmp/afrioracle-demo.log)"; exit 1; }
  echo "==> backend up (pid $SERVER_PID)"
fi

b64() { printf '%s' "$1" | base64 -w 0; }

node_key() {
  printf 'afrioracle-demo-key-v1-%s-d7f3a0c948b16e2d' "$1"
}

# submit <node> <tier> <rep> <value> <photo_price> <photo_tag> <channel>
# photo_tag is the unique-per-photo label; the fraudster reuses another
# agent's tag to trigger the duplicate-photo path (score 12).
submit() {
  local node="$1" tier="$2" rep="$3" value="$4" pprice="$5" ptag="$6" channel="$7"
  local photo_json="null"
  if [ -n "$pprice" ]; then
    photo_json="$pprice"
  fi
  local code
  code=$(jq -n \
    --arg node "$node" --argjson tier "$tier" --argjson rep "$rep" \
    --argjson value "$value" --argjson pprice "$photo_json" \
    --arg photo "$(b64 "photo-$ptag-kaolack-millet")" --arg channel "$channel" \
    '{node_id: $node, tier: $tier, rep_score: $rep, value: $value,
      photo_price: $pprice, photo_b64: $photo,
      transcript: ("Le prix du mil à Kaolack est de \($value) francs CFA le kilo."),
      channel: $channel}' \
    | curl -s -o /tmp/afrioracle-submit.json -w '%{http_code}' \
      -X POST "$BASE/api/submit" -H 'Content-Type: application/json' \
      -H "Authorization: Bearer $(node_key "$node")" -d @-)
  if [ "$code" != "201" ]; then
    echo "FAIL: submit $node -> HTTP $code: $(cat /tmp/afrioracle-submit.json)"
    exit 1
  fi
  echo "  accepted $node (tier $tier, rep $rep, value $value, channel $channel)"
}

echo "==> submitting 8 honest agents..."
submit A 2 88 187 187 "A-slate-187" "app"
submit B 2 72 191 191 "B-slate-191" "whatsapp"
submit C 2 65 185 185 "C-slate-185" "app"
submit E 2 66 186 186 "E-slate-186" "app"
submit F 2 63 188 188 "F-slate-188" "whatsapp"
submit G 2 70 189 189 "G-slate-189" "app"
submit H 2 68 184 184 "H-slate-184" "app"
# FAO: tier-3 automatic node (source hash) — no photo fields, always accepted.
code=$(jq -n '{node_id: "FAO", tier: 3, rep_score: 0, value: 190,
  transcript: "FAO source hash: millet Kaolack 190.", channel: "api"}' \
  | curl -s -o /tmp/afrioracle-submit.json -w '%{http_code}' \
    -X POST "$BASE/api/submit" -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $(node_key FAO)" -d @-)
[ "$code" = "201" ] || { echo "FAIL: submit FAO -> HTTP $code"; exit 1; }
echo "  accepted FAO (tier 3 automatic, value 190)"

echo "==> submitting scripted fraudster D (reuses A's photo)..."
submit D 2 61 240 240 "A-slate-187" "app"

echo "==> running consensus..."
code=$(curl -s -o /tmp/afrioracle-consensus.json -w '%{http_code}' -X POST "$BASE/api/consensus" \
  -H "Authorization: Bearer $(node_key A)")
[ "$code" = "200" ] || { echo "FAIL: consensus -> HTTP $code: $(cat /tmp/afrioracle-consensus.json)"; exit 1; }

MEDIAN=$(jq -r .median /tmp/afrioracle-consensus.json)
CONF=$(jq -r .confidence /tmp/afrioracle-consensus.json)
# Note: the backend has no reset endpoint (agent 04's surface, frozen), so on a
# reused server prior runs accumulate. Assertions below normalize over that:
# every D outcome must be excluded, whatever the run count.
DSTATUSES=$(jq -r '.outcomes[] | select(.NodeID=="D") | .Status' /tmp/afrioracle-consensus.json | sort -u)
TX=$(jq -r .tx /tmp/afrioracle-consensus.json)
CHAIN=$(jq -r .chain /tmp/afrioracle-consensus.json)

[ "$MEDIAN" = "187" ] || { echo "FAIL: median=$MEDIAN, want 187"; exit 1; }
[ "$CONF" = "100" ] || { echo "FAIL: confidence=$CONF, want 100"; exit 1; }
[ "$DSTATUSES" = "excluded" ] || { echo "FAIL: fraudster D statuses: $(echo $DSTATUSES | tr '\n' ' '), want all excluded"; exit 1; }
echo "  median=$MEDIAN confidence=$CONF fraudster D excluded tx=$TX chain=$CHAIN"

echo "==> asking the assistant..."
ASK=$(curl -s "$BASE/api/ask?q=prix%20du%20mil%20%C3%A0%20Kaolack%20%3F")
echo "  answer: $(echo "$ASK" | jq -r .answer)"
echo "$ASK" | jq -r .answer | grep -q "187" || { echo "FAIL: assistant answer lacks the published value 187"; exit 1; }
if echo "$ASK" | jq -r .answer | grep -qi "estim"; then
  echo "FAIL: assistant estimated instead of quoting published data"
  exit 1
fi

echo ""
echo "DEMO OK: median 187 FCFA/kg (confidence 100, tx $TX [$CHAIN]), fraudster D excluded, assistant quotes the published value."
