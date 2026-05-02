#!/usr/bin/env bash
# Dispatch one issue #8 benchmark task to Claude Code.
#
# Input contract: scripts/bench_runner.py sets BENCH_* env vars. This
# script runs Claude Code in print mode inside BENCH_WORK_DIR, then prints
# a compact JSON metrics object on stdout. Raw agent output is not relayed
# into the benchmark report.

set -euo pipefail

: "${BENCH_WORK_DIR:?BENCH_WORK_DIR is required}"
: "${BENCH_PROMPT:?BENCH_PROMPT is required}"
: "${BENCH_ACCEPTANCE:?BENCH_ACCEPTANCE is required}"

CLAUDE_BIN="${CLAUDE_BIN:-claude}"
MODEL="${BENCH_CLAUDE_MODEL:-sonnet}"
BUDGET="${BENCH_CLAUDE_MAX_BUDGET_USD:-0.50}"
OUT="$(mktemp)"
ERR="$(mktemp)"
SETTINGS="$(mktemp)"
trap 'rm -f "$OUT" "$ERR" "$SETTINGS"' EXIT

if ! command -v "$CLAUDE_BIN" >/dev/null 2>&1; then
  echo "claude executable not found: $CLAUDE_BIN" >&2
  exit 127
fi

read -r -d '' PROMPT <<EOF || true
You are running a small coding benchmark fixture.

Work only in the current directory:
$BENCH_WORK_DIR

Task:
$BENCH_PROMPT

Rules:
- Edit implementation files only. Do not edit README.md or test_*.py.
- Run the acceptance command before finishing: $BENCH_ACCEPTANCE
- Preserve public behavior not mentioned by the task.
- Do not quote or discuss scripture text in the final response.
EOF

case "${BENCH_MODE:-}" in
  inject-once)
    cat >"$SETTINGS" <<'JSON'
{
  "hooks": {
    "UserPromptExpansion": [
      {
        "matcher": "^(bible|sutra|dao|quran)$",
        "hooks": [
          {
            "type": "command",
            "command": "scripture-mcp lookup-from-prompt --hook-event=UserPromptExpansion"
          }
        ]
      }
    ]
  }
}
JSON
    ;;
  recap-only)
    cat >"$SETTINGS" <<'JSON'
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "scripture-mcp recap --terminal"
          }
        ]
      }
    ]
  }
}
JSON
    ;;
  *)
    printf '{}\n' >"$SETTINGS"
    ;;
esac

(
  cd "$BENCH_WORK_DIR"
  printf '%s' "$PROMPT" | "$CLAUDE_BIN" -p \
    --output-format json \
    --model "$MODEL" \
    --max-budget-usd "$BUDGET" \
    --setting-sources project \
    --settings "$SETTINGS" \
    --permission-mode bypassPermissions \
    --dangerously-skip-permissions \
    --no-session-persistence >"$OUT" 2>"$ERR"
)
code=$?
if [ "$code" -ne 0 ]; then
  sed -n '1,120p' "$ERR" >&2
  sed -n '1,40p' "$OUT" >&2
  exit "$code"
fi

python3 - "$OUT" <<'PY'
import json
import sys

path = sys.argv[1]
text = open(path, "r", encoding="utf-8").read()
decoder = json.JSONDecoder()
idx = text.find("{")
obj = {}
if idx >= 0:
    try:
        obj, _ = decoder.raw_decode(text[idx:])
    except json.JSONDecodeError:
        obj = {}

input_tokens = 0
output_tokens = 0
for usage in (obj.get("modelUsage") or {}).values():
    if not isinstance(usage, dict):
        continue
    input_tokens += int(usage.get("inputTokens") or 0)
    input_tokens += int(usage.get("cacheReadInputTokens") or 0)
    input_tokens += int(usage.get("cacheCreationInputTokens") or 0)
    output_tokens += int(usage.get("outputTokens") or 0)

if not input_tokens and isinstance(obj.get("usage"), dict):
    usage = obj["usage"]
    input_tokens += int(usage.get("input_tokens") or 0)
    input_tokens += int(usage.get("cache_creation_input_tokens") or 0)
    input_tokens += int(usage.get("cache_read_input_tokens") or 0)
    output_tokens += int(usage.get("output_tokens") or 0)

print(json.dumps({
    "input_tokens": input_tokens or None,
    "output_tokens": output_tokens or None,
    "cost_usd": obj.get("total_cost_usd"),
}))
PY
