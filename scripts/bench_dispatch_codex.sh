#!/usr/bin/env bash
# Dispatch one issue #8 benchmark task to Codex CLI.
#
# Input contract: scripts/bench_runner.py sets BENCH_* env vars. This
# script edits BENCH_WORK_DIR via `codex exec`, then prints a compact JSON
# metrics object on stdout. Agent stdout/stderr stays out of the Markdown
# report to avoid leaking large prompts or scripture bodies.

set -euo pipefail

: "${BENCH_WORK_DIR:?BENCH_WORK_DIR is required}"
: "${BENCH_PROMPT:?BENCH_PROMPT is required}"
: "${BENCH_ACCEPTANCE:?BENCH_ACCEPTANCE is required}"

CODEX_BIN="${CODEX_BIN:-codex}"
if [ "${BENCH_MODE:-}" = "recap-only" ]; then
  if [ -n "${CODEX_WRAPPER_BIN:-}" ]; then
    CODEX_BIN="$CODEX_WRAPPER_BIN"
  elif command -v cdx >/dev/null 2>&1; then
    CODEX_BIN="cdx"
  fi
fi
MODEL="${BENCH_CODEX_MODEL:-gpt-5.4-mini}"
EFFORT="${BENCH_CODEX_EFFORT:-low}"
OUT="$(mktemp)"
ERR="$(mktemp)"
trap 'rm -f "$OUT" "$ERR"' EXIT

if ! command -v "$CODEX_BIN" >/dev/null 2>&1; then
  echo "codex executable not found: $CODEX_BIN" >&2
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

(
  cd "$BENCH_WORK_DIR"
  "$CODEX_BIN" exec \
    --json \
    --skip-git-repo-check \
    --ephemeral \
    --sandbox workspace-write \
    -m "$MODEL" \
    -c "model_reasoning_effort=\"$EFFORT\"" \
    "$PROMPT" >"$OUT" 2>"$ERR"
)
code=$?
if [ "$code" -ne 0 ]; then
  sed -n '1,120p' "$ERR" >&2
  exit "$code"
fi

python3 - "$OUT" <<'PY'
import json
import sys

path = sys.argv[1]
input_tokens = 0
output_tokens = 0
with open(path, "r", encoding="utf-8") as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        usage = event.get("usage")
        if isinstance(usage, dict):
            input_tokens += int(usage.get("input_tokens") or 0)
            input_tokens += int(usage.get("cached_input_tokens") or 0)
            output_tokens += int(usage.get("output_tokens") or 0)
            output_tokens += int(usage.get("reasoning_output_tokens") or 0)

print(json.dumps({
    "input_tokens": input_tokens or None,
    "output_tokens": output_tokens or None,
}))
PY
