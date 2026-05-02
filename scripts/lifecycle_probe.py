#!/usr/bin/env python3
"""External lifecycle probe for issue #8.

The Go lifecycle tests exercise the hook surface in-process. This script
does the same checks through the installed scripture-mcp binary so the
release report records the behavior of the actual CLI users wire into
Claude Code and Codex.

It intentionally records hashes and byte counts, not scripture bodies.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
import subprocess
import sys
from dataclasses import dataclass
from datetime import date
from pathlib import Path
from typing import Optional


@dataclass
class ProbeResult:
    adapter: str
    hook_event: str
    turn_n_context: bool
    turn_n_context_sha256: Optional[str]
    turn_n_plus_1_context: bool
    followup_turns: int
    followup_context_leaks: int
    recap_hashes_seen: int
    recap_leaks: int


def run_cmd(argv: list[str], stdin: str = "") -> subprocess.CompletedProcess:
    return subprocess.run(argv, input=stdin, text=True, capture_output=True, timeout=60)


def lookup_from_prompt(binary: str, hook_event: str, prompt: str) -> str:
    proc = run_cmd([binary, "lookup-from-prompt", f"--hook-event={hook_event}"], prompt)
    if proc.returncode != 0:
        raise RuntimeError(proc.stderr.strip() or f"lookup-from-prompt exited {proc.returncode}")
    if not proc.stdout.strip():
        return ""
    body = json.loads(proc.stdout)
    return body.get("hookSpecificOutput", {}).get("additionalContext", "")


def recap(binary: str, seed: int) -> str:
    proc = run_cmd([binary, "recap", "--terminal", "--tradition=dao", f"--seed={seed}"])
    if proc.returncode != 0:
        raise RuntimeError(proc.stderr.strip() or f"recap exited {proc.returncode}")
    return proc.stdout


def sha(value: str) -> Optional[str]:
    if not value:
        return None
    return hashlib.sha256(value.encode("utf-8")).hexdigest()


def probe_adapter(binary: str, adapter: str) -> ProbeResult:
    if adapter == "claude":
        hook_event = "UserPromptExpansion"
        injecting_prompt = "/bible John 3:16 Refactor the scheduler."
    elif adapter == "codex":
        hook_event = "UserPromptSubmit"
        injecting_prompt = "Refactor the scheduler. $bible.John.3.16"
    else:
        raise ValueError(adapter)

    turn_n = lookup_from_prompt(binary, hook_event, injecting_prompt)
    turn_n_plus_1 = lookup_from_prompt(binary, hook_event, "Continue with no marker.")

    followup_leaks = 0
    followups = 30
    for i in range(followups):
        ctx = lookup_from_prompt(binary, hook_event, f"follow-up turn {i}: no marker here")
        if ctx:
            followup_leaks += 1

    recap_hashes = set()
    recap_leaks = 0
    model_inputs = [injecting_prompt, "Continue with no marker."]
    for i in range(followups):
        text = recap(binary, seed=100 + i)
        h = sha(text)
        if h:
            recap_hashes.add(h)
        future_input = f"future turn {i}: still no marker"
        model_inputs.append(future_input)
        if text and any(text in mi for mi in model_inputs):
            recap_leaks += 1

    return ProbeResult(
        adapter=adapter,
        hook_event=hook_event,
        turn_n_context=bool(turn_n),
        turn_n_context_sha256=sha(turn_n),
        turn_n_plus_1_context=bool(turn_n_plus_1),
        followup_turns=followups,
        followup_context_leaks=followup_leaks,
        recap_hashes_seen=len(recap_hashes),
        recap_leaks=recap_leaks,
    )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--binary", default=shutil.which("scripture-mcp") or "scripture-mcp")
    parser.add_argument("--out", default="")
    args = parser.parse_args()

    binary = args.binary
    version = run_cmd([binary, "--version"]).stdout.strip()
    results = [probe_adapter(binary, adapter) for adapter in ("claude", "codex")]
    passed = all(
        r.turn_n_context
        and not r.turn_n_plus_1_context
        and r.followup_context_leaks == 0
        and r.recap_leaks == 0
        for r in results
    )
    body = {
        "date": str(date.today()),
        "binary": binary,
        "version": version,
        "passed": passed,
        "results": [r.__dict__ for r in results],
    }
    text = json.dumps(body, indent=2)
    if args.out:
        Path(args.out).parent.mkdir(parents=True, exist_ok=True)
        Path(args.out).write_text(text + "\n", encoding="utf-8")
    print(text)
    return 0 if passed else 1


if __name__ == "__main__":
    sys.exit(main())
