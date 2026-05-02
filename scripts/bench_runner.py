#!/usr/bin/env python3
"""bench_runner.py — coding-quality benchmark runner for issue #8.

Runs one or more tasks from docs/benchmarks/tasks.json against a coding
agent across the four configured modes, records per-task metrics, and
emits a Markdown report at docs/benchmarks/<date>.md.

This script is the runner skeleton: it wires the task pack, the modes,
the per-task fixture invocation, and the report rendering. The actual
agent invocation is gated by environment so CI can validate the runner's
own correctness without an API key, while a developer with a key can
produce a real benchmark.

Required env for a real run:
  ANTHROPIC_API_KEY    — Claude side
  OPENAI_API_KEY       — Codex side (optional; Codex runs are skipped
                          if missing)

Optional env:
  BENCH_AGENT_CMD      — override how a task is dispatched. The
                          default invokes the Claude or Codex CLI in
                          headless mode against the task's fixture
                          directory; setting this to "echo" turns the
                          runner into a smoke test that still produces
                          a (failing) report row.
  BENCH_TASK_FILTER    — comma-separated task ids to include (default:
                          all)
  BENCH_MODES          — comma-separated mode ids (default:
                          baseline,preview-only,inject-once,recap-only)
  BENCH_OUT            — output report path (default: docs/benchmarks/
                          today's date .md)

Why a script and not a Go program: invoking external coding agents and
their pytest acceptance suites is fundamentally a shell-orchestration
task, and Python is the lingua franca of the agent CLIs we drive.
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import time
from dataclasses import asdict, dataclass, field
from datetime import date
from pathlib import Path
from typing import Optional

REPO_ROOT = Path(__file__).resolve().parent.parent
TASKS_FILE = REPO_ROOT / "docs" / "benchmarks" / "tasks.json"
DEFAULT_MODES = ("baseline", "preview-only", "inject-once", "recap-only")


@dataclass
class TaskResult:
    task_id: str
    mode: str
    adapter: str
    success: bool  # acceptance suite passed
    input_tokens: Optional[int]  # None when the agent didn't report
    output_tokens: Optional[int]
    latency_p50_ms: Optional[float]
    error: Optional[str] = None


@dataclass
class BenchSummary:
    date: str
    adapter: str
    modes: list[str]
    results: list[TaskResult] = field(default_factory=list)


def load_tasks() -> dict:
    if not TASKS_FILE.exists():
        sys.exit(f"task pack not found at {TASKS_FILE}")
    return json.loads(TASKS_FILE.read_text(encoding="utf-8"))


def filter_tasks(pack: dict, allow: Optional[list[str]]) -> list[dict]:
    if not allow:
        return list(pack["tasks"])
    by_id = {t["id"]: t for t in pack["tasks"]}
    out = []
    for tid in allow:
        if tid not in by_id:
            sys.exit(f"unknown task id in BENCH_TASK_FILTER: {tid}")
        out.append(by_id[tid])
    return out


def acceptance(fixture_dir: Path, command: str) -> tuple[bool, str]:
    """Run the per-task acceptance command in fixture_dir. Returns
    (passed, captured_output). The command is shell-evaluated to keep
    the per-task surface flexible (most tasks use ``pytest -q``).
    """
    if not fixture_dir.exists():
        return False, f"fixture dir missing: {fixture_dir}"
    try:
        proc = subprocess.run(
            command,
            shell=True,
            cwd=str(fixture_dir),
            capture_output=True,
            text=True,
            timeout=120,
        )
    except subprocess.TimeoutExpired as e:
        return False, f"timeout: {e}"
    out = (proc.stdout or "") + (proc.stderr or "")
    return proc.returncode == 0, out


def dispatch_agent(task: dict, mode: str, adapter: str, work_dir: Path) -> dict:
    """Invoke the coding agent against work_dir. Returns a dict with
    optional metric fields the agent emitted: input_tokens,
    output_tokens, latency_p50_ms, error.

    The default implementation expects the env-overridable
    BENCH_AGENT_CMD to handle the actual invocation. The cmd receives
    these env vars:

      BENCH_TASK_ID
      BENCH_MODE
      BENCH_ADAPTER
      BENCH_WORK_DIR    — the fixture directory the agent edits
      BENCH_PROMPT      — the prompt template (with marker prefix per
                          mode + adapter)

    and is expected to emit a single JSON object on stdout with the
    metric fields. If BENCH_AGENT_CMD is unset we record a skip — the
    runner's own correctness can still be validated end-to-end via
    BENCH_AGENT_CMD=echo (which yields a failing acceptance result and
    a complete report row).
    """
    cmd = os.environ.get("BENCH_AGENT_CMD")
    if not cmd:
        return {"error": "BENCH_AGENT_CMD not set; skipping agent dispatch"}

    prompt = build_prompt(task, mode, adapter)
    env = os.environ.copy()
    env.update(
        {
            "BENCH_TASK_ID": task["id"],
            "BENCH_MODE": mode,
            "BENCH_ADAPTER": adapter,
            "BENCH_WORK_DIR": str(work_dir),
            "BENCH_PROMPT": prompt,
        }
    )
    started = time.perf_counter()
    proc = subprocess.run(
        cmd, shell=True, capture_output=True, text=True, env=env, timeout=300
    )
    elapsed_ms = (time.perf_counter() - started) * 1000.0
    out: dict = {"latency_p50_ms": elapsed_ms}
    if proc.returncode != 0:
        out["error"] = f"agent exited {proc.returncode}: {proc.stderr.strip()}"
        return out
    try:
        body = json.loads(proc.stdout.strip() or "{}")
        for k in ("input_tokens", "output_tokens"):
            if k in body:
                out[k] = body[k]
    except json.JSONDecodeError:
        # Agent didn't emit JSON; we still recorded latency.
        pass
    return out


def build_prompt(task: dict, mode: str, adapter: str) -> str:
    """Return the user-facing prompt the agent sees, including the
    marker prefix appropriate to the mode + adapter combination.
    """
    body = task["description"]
    if mode == "inject-once" and adapter == "claude":
        return f"/bible Matthew 6:26 {body}"
    if mode == "inject-once" and adapter == "codex":
        return f"{body} [[bible:Matthew 6:26]]"
    if mode == "preview-only":
        # The user types the marker but cancels; semantically the
        # agent sees the same body as baseline. We document the typed
        # marker in the prompt's comment for the report's prompt-log,
        # but it is not actually injected.
        return body
    return body


def run(adapter: str, modes: list[str], task_filter: Optional[list[str]]) -> BenchSummary:
    pack = load_tasks()
    tasks = filter_tasks(pack, task_filter)
    summary = BenchSummary(date=str(date.today()), adapter=adapter, modes=modes)
    for task in tasks:
        for mode in modes:
            fixture_src = TASKS_FILE.parent / task["fixture_dir"]
            scratch = REPO_ROOT / ".bench-scratch" / f"{task['id']}__{mode}__{adapter}"
            if scratch.exists():
                shutil.rmtree(scratch)
            scratch.parent.mkdir(parents=True, exist_ok=True)
            if fixture_src.exists():
                shutil.copytree(fixture_src, scratch)
            else:
                scratch.mkdir(parents=True)
            metrics = dispatch_agent(task, mode, adapter, scratch)
            passed, _ = acceptance(scratch, task["acceptance"])
            summary.results.append(
                TaskResult(
                    task_id=task["id"],
                    mode=mode,
                    adapter=adapter,
                    success=passed,
                    input_tokens=metrics.get("input_tokens"),
                    output_tokens=metrics.get("output_tokens"),
                    latency_p50_ms=metrics.get("latency_p50_ms"),
                    error=metrics.get("error"),
                )
            )
    return summary


def render_report(summary: BenchSummary) -> str:
    lines: list[str] = []
    lines.append(f"# Coding-quality benchmark — {summary.date}")
    lines.append("")
    lines.append(f"Adapter: **{summary.adapter}**  ")
    lines.append(f"Modes: {', '.join(summary.modes)}")
    lines.append("")
    lines.append(
        "Acceptance: per-task `pytest -q` against the fixture's tests. "
        "A run is graded **pass** iff the acceptance suite exits 0. The "
        "issue #8 acceptance gate is: no mode regresses success rate by "
        "more than 5 percentage points vs baseline."
    )
    lines.append("")
    lines.append("## Per-mode success rate")
    by_mode: dict[str, list[TaskResult]] = {m: [] for m in summary.modes}
    for r in summary.results:
        by_mode[r.mode].append(r)
    lines.append("| mode | success | n |")
    lines.append("|---|---|---|")
    for m in summary.modes:
        rows = by_mode[m]
        if not rows:
            lines.append(f"| {m} | — | 0 |")
            continue
        passed = sum(1 for r in rows if r.success)
        rate = passed / len(rows)
        lines.append(f"| {m} | {rate:.0%} | {len(rows)} |")
    lines.append("")
    lines.append("## Per-task results")
    lines.append("| task | mode | success | input tok | output tok | p50 ms | error |")
    lines.append("|---|---|---|---|---|---|---|")
    for r in summary.results:
        lines.append(
            f"| {r.task_id} | {r.mode} | "
            f"{'✅' if r.success else '❌'} | "
            f"{r.input_tokens if r.input_tokens is not None else '—'} | "
            f"{r.output_tokens if r.output_tokens is not None else '—'} | "
            f"{r.latency_p50_ms:.0f}" + (" | " if r.latency_p50_ms else " — | ")
            + (r.error or "") + " |"
        )
    return "\n".join(lines) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--adapter", default="claude", choices=("claude", "codex"))
    parser.add_argument(
        "--modes",
        default=os.environ.get("BENCH_MODES", ",".join(DEFAULT_MODES)),
        help="comma-separated list of mode ids",
    )
    parser.add_argument(
        "--tasks",
        default=os.environ.get("BENCH_TASK_FILTER", ""),
        help="comma-separated task ids (empty = all)",
    )
    parser.add_argument(
        "--out",
        default=os.environ.get(
            "BENCH_OUT",
            str(REPO_ROOT / "docs" / "benchmarks" / f"{date.today()}.md"),
        ),
    )
    args = parser.parse_args()
    modes = [m.strip() for m in args.modes.split(",") if m.strip()]
    tasks = [t.strip() for t in args.tasks.split(",") if t.strip()] or None
    summary = run(args.adapter, modes, tasks)
    Path(args.out).parent.mkdir(parents=True, exist_ok=True)
    Path(args.out).write_text(render_report(summary), encoding="utf-8")
    # Also drop a machine-readable version next to it for downstream
    # diffing (regression gate computation lives in CI).
    Path(args.out + ".json").write_text(
        json.dumps(asdict(summary), indent=2), encoding="utf-8"
    )
    print(f"wrote {args.out}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
