#!/usr/bin/env bash
# scripts/test-laya.sh — shape tests for the Laya `/v1/systemone` wire contract.
#
# What it checks, without ever modifying anything server-side:
#   1. GET  /health                      → {"status":"ok"} liveness shape
#   2. POST /v1/systemone, one case per
#      card intent from the registry     → response shape, per answer type
#   3. two malformed bodies               → 400 ("no questions") and 422 (bad question type)
#
# Run it after the server is up (or let `make start` invoke it):
#   make start            # starts laya-serve, runs this script when /health is green
#   ./scripts/test-laya.sh   # or on its own, any time, against a running server
#
# Env:
#   LAYA_URL    default http://127.0.0.1:8000
#   LAYA_TOKEN  optional bearer token when the server was started with an API key
#   LAYA_WAIT   max seconds to wait for /health (default 120) — set to 0 to skip waiting
#   LAYA_QUIET  1 to print one PASS/FAIL line per case only

set -u

LAYA_URL="${LAYA_URL:-http://127.0.0.1:8000}"
LAYA_TOKEN="${LAYA_TOKEN:-}"
LAYA_WAIT="${LAYA_WAIT:-120}"
LAYA_QUIET="${LAYA_QUIET:-0}"
MAX_CASES="${MAX_CASES:-0}"   # e.g. MAX_CASES=12 to stop after 12
CPU_HEADROOM="${CPU_HEADROOM:-20}"

AUTH=(-H "content-type: application/json")
if [ -n "$LAYA_TOKEN" ]; then AUTH+=(-H "Authorization: Bearer ${LAYA_TOKEN}"); fi

failures=0; total=0
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
note() { [ "$LAYA_QUIET" = "1" ] || echo "$*"; }
crash() { echo "❌ $*" >&2; exit 1; }

# ---------------------------------------------------------------------------
# The shared questionnaire every case sends. Named after the app's signal
# vocabulary (registry "signals") and shaped so each call exercises all three
# question types laya accepts: choice, score, noul.
# ---------------------------------------------------------------------------
read -r -d '' QUESTIONNAIRE <<'EOF'
{
  "card_category": {
    "type": "choice",
    "instructions": "Which one card does this text become?",
    "criteria": {
      "event": "a time, date or plan",
      "todo": "an action or shopping items",
      "timer": "something running or counting",
      "expense": "money spent or split",
      "math": "units, currency or arithmetic",
      "other": "everything else"
    }
  },
  "urgency": {
    "type": "score",
    "instructions": "How soon does this matter",
    "criteria": ["not urgent", "soon", "blocking"]
  },
  "recurring": {
    "type": "noul",
    "instructions": "Does this repeat on a schedule"
  }
}
EOF

# state.body per card intent, copied verbatim from frontend/src/components/intents/registry.ts
CASES=(
  "event|dinner with priya friday 8pm"
  "reminder|remind me to call mom tomorrow"
  "todo|buy milk, eggs, bread and coffee"
  "timer|25 min focus"
  "habit|meditate every morning"
  "color|#ff6b35"
  "split|split 2400 between 3"
  "expense|spent 450 on uber"
  "convert|5 miles in km"
  "calc|18% of 3450"
  "travel|flight to goa next weekend"
  "poll|pizza or burgers for friday?"
  "contact|rahul 98200 12345 rahul@mail.com"
  "link|https://vercel.com/blog check later"
  "countdown|days until christmas"
  "timezone|3pm pst in ist"
  "random|roll 2d6"
  "goal|read 12 books this year, 4 done"
  "note|the city felt so quiet this morning"
)

# JSON request body for one text. python does the quoting/escaping.
body_for() {
  python3 - "$1" "${QUESTIONNAIRE}" <<'PY'
import json, sys
body = {"state": {"body": sys.argv[1]}}
body["questions"] = json.loads(sys.argv[2])
print(json.dumps(body))
PY
}

# Full-body shape check: exactly the fields the Jev contract roots at.
check_body() { python3 - "$1" <<'PY'
import json, sys
r = json.loads(sys.argv[1])
assert isinstance(r.get("model"), str) and r["model"], "top-level 'model' string missing"
assert isinstance(r.get("answers"), dict) and r["answers"], "'answers' object missing"
for qid, a in r["answers"].items():
    t = a.get("type")
    if t == "choice":
        for k in ("choice", "probabilities", "confidence"): assert k in a, f"{qid}: missing {k}"
    elif t == "score":
        for k in ("score", "probabilities", "confidence", "legend"): assert k in a, f"{qid}: missing {k}"
    elif t == "noul":
        assert "noul" in a, f"{qid}: missing noul"
    else:
        raise AssertionError(f"{qid}: unknown answer type {t!r}")
u = r.get("usage", {})
assert isinstance(u.get("input_tokens", 0), int), "usage.input_tokens not an int"
assert isinstance(u.get("output_tokens", 0), int), "usage.output_tokens not an int"
PY
}

wait_for_health() {
  [ "$LAYA_WAIT" -gt 0 ] || return 0
  note "⏳ Waiting up to ${LAYA_WAIT}s for ${LAYA_URL}/health ..."
  waited=0
  while [ "$waited" -lt "$LAYA_WAIT" ]; do
    if curl -sf -m 2 "${AUTH[@]}" "$LAYA_URL/health" > /dev/null 2>&1; then return 0; fi
    sleep 2; waited=$((waited + 2))
  done
  crash "server did not answer /health within ${LAYA_WAIT}s; check /tmp/laya-server.log"
}

note "🏥 health shape"
total=$((total + 1))
health="$(curl -s -m 2 "${AUTH[@]}" "$LAYA_URL/health")" || crash "GET /health failed"
echo "$health" | python3 -c 'import json,sys; h=json.load(sys.stdin); assert h.get("status")=="ok", h; print("ok", h.get("device",""))' \
  && note "✅ health" || { failures=$((failures + 1)); note "❌ health: $health"; }

note ""
note "🧪 POST /v1/systemone — one case per card intent (${#CASES[@]} cases)"
for c in "${CASES[@]}"; do
  name="${c%%|*}";  text="${c#*|}"
  tmp="$(mktemp)"
  body_for "$text" > "$tmp"
  # one call, capture body and status together (-w separator must not collide with body)
  raw="$(curl -s -m "${CPU_HEADROOM}" "${AUTH[@]}" -d@"$tmp" -w "\n%{http_code}" "$LAYA_URL/v1/systemone")"
  rm -f "$tmp"
  status="${raw##*$'\n'}"
  resp="${raw%$'\n'*}"
  total=$((total + 1))
  if [ "$status" != "200" ]; then
    echo "❌ $name (HTTP $status): $resp"
    failures=$((failures + 1)); continue
  fi
  if ! check_body "$resp"; then
    echo "❌ $name: unexpected body shape: $resp"
    failures=$((failures + 1)); continue
  fi
  detail="$(echo "$resp" | python3 -c 'import json,sys; r=json.load(sys.stdin); print(" model=%s" % r["model"])')"
  note "✅ $name  → category: $(echo "$resp" | python3 -c 'import json,sys; a=json.load(sys.stdin)["answers"]["card_category"]; print(a.get("choice", a.get("score", a.get("noul"))))')  urgency: $(echo "$resp" | python3 -c 'import json,sys; a=json.load(sys.stdin)["answers"]["urgency"]; print(a.get("score"))')${detail}"
done

note ""
note "🧪 contract refusals (must refuse without leaking internals)"
probe() { # $1 label, $2 expected status, $3 body
  got="$(curl -s -o "$WORK.body" -w "%{http_code}" -X POST "${AUTH[@]}" -d "$3" "$LAYA_URL/v1/systemone")"
  total=$((total + 1))
  if [ "$got" = "$2" ]; then note "✅ $1 (HTTP $got)"; else
    echo "❌ $1: expected HTTP $2, got HTTP $got: $(cat "$WORK.body" | head -c 300)"
    failures=$((failures + 1))
  fi
}
probe "missing questions  → 400" 400 '{"state":{"body":"hi"}}'
probe "bad question type  → 422" 422 '{"state":{"body":"hi"},"questions":{"x":{"type":"nonsense","instructions":"y","criteria":{"a":"b"}}}}'

echo ""
if [ "$failures" -eq 0 ]; then
  echo "✅ all $total shape checks passed against $LAYA_URL"
else
  echo "⚠️  $failures of $total checks failed"
fi
exit "$([ "$failures" -eq 0 ] && echo 0 || echo 1)"
