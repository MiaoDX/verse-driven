#!/usr/bin/env python3
"""bench_runner.py - coding-quality benchmark runner for issue #8.

Runs the task pack in docs/benchmarks/tasks.json against Claude Code,
Codex, or both. For each adapter it exercises the four benchmark modes,
copies the task fixture into .bench-scratch, dispatches the configured
agent, runs the fixture acceptance command, and writes both Markdown and
JSON reports.

Default dispatch scripts:
  scripts/bench_dispatch_claude.sh
  scripts/bench_dispatch_codex.sh

Useful env overrides:
  BENCH_AGENT_CMD          command used for every adapter
  BENCH_CLAUDE_CMD         command used only for --adapter=claude
  BENCH_CODEX_CMD          command used only for --adapter=codex
  BENCH_TASK_FILTER        comma-separated task ids
  BENCH_MODES              comma-separated mode ids
  BENCH_REPEATS            attempts per (adapter, task, mode)
  BENCH_AGENT_TIMEOUT_SEC  per-agent timeout, default 900
  BENCH_OUT                output report path
"""

from __future__ import annotations

import argparse
import json
import os
import shlex
import shutil
import statistics
import subprocess
import sys
import time
from dataclasses import asdict, dataclass, field
from datetime import date
from pathlib import Path
from typing import Optional, Union

REPO_ROOT = Path(__file__).resolve().parent.parent
TASKS_FILE = REPO_ROOT / "docs" / "benchmarks" / "tasks.json"
DEFAULT_MODES = ("baseline", "preview-only", "inject-once", "recap-only")
DEFAULT_ADAPTERS = ("claude", "codex")


@dataclass
class TaskResult:
    task_id: str
    mode: str
    adapter: str
    repeat: int
    success: bool
    input_tokens: Optional[int]
    output_tokens: Optional[int]
    latency_ms: Optional[float]
    cost_usd: Optional[float] = None
    error: Optional[str] = None


@dataclass
class BenchSummary:
    date: str
    adapters: list[str]
    modes: list[str]
    repeats: int
    versions: dict[str, str]
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
            sys.exit(f"unknown task id in BENCH_TASK_FILTER/--tasks: {tid}")
        out.append(by_id[tid])
    return out


def normalize_acceptance_command(command: str) -> str:
    """Prefer pytest on PATH, but fall back to this Python interpreter.

    On macOS, pip --user commonly installs the pytest console script under
    ~/Library/Python/.../bin, which is not always on PATH. The module form
    remains reliable once pytest is installed.
    """
    if command.strip() == "pytest -q" and shutil.which("pytest") is None:
        return f"{shlex.quote(sys.executable)} -m pytest -q"
    return command


def acceptance(fixture_dir: Path, command: str) -> tuple[bool, str]:
    command = normalize_acceptance_command(command)
    if not fixture_dir.exists():
        return False, f"fixture dir missing: {fixture_dir}"
    try:
        proc = subprocess.run(
            command,
            shell=True,
            cwd=str(fixture_dir),
            capture_output=True,
            text=True,
            timeout=180,
        )
    except subprocess.TimeoutExpired as e:
        return False, f"timeout: {e}"
    out = (proc.stdout or "") + (proc.stderr or "")
    return proc.returncode == 0, out


def command_for_adapter(adapter: str) -> Optional[str]:
    specific = os.environ.get(f"BENCH_{adapter.upper()}_CMD")
    if specific:
        return specific
    shared = os.environ.get("BENCH_AGENT_CMD")
    if shared:
        return shared
    candidate = REPO_ROOT / "scripts" / f"bench_dispatch_{adapter}.sh"
    if candidate.exists():
        return f"bash {shlex.quote(str(candidate))}"
    return None


def dispatch_agent(task: dict, mode: str, adapter: str, repeat: int, work_dir: Path) -> dict:
    cmd = command_for_adapter(adapter)
    if not cmd:
        return {"error": f"no dispatch command configured for adapter {adapter}"}

    prompt = build_prompt(task, mode, adapter)
    env = os.environ.copy()
    env.update(
        {
            "BENCH_TASK_ID": task["id"],
            "BENCH_TASK_NAME": task.get("name", task["id"]),
            "BENCH_MODE": mode,
            "BENCH_ADAPTER": adapter,
            "BENCH_REPEAT": str(repeat),
            "BENCH_WORK_DIR": str(work_dir),
            "BENCH_REPO_ROOT": str(REPO_ROOT),
            "BENCH_PROMPT": prompt,
            "BENCH_ACCEPTANCE": normalize_acceptance_command(task["acceptance"]),
        }
    )
    timeout = int(os.environ.get("BENCH_AGENT_TIMEOUT_SEC", "900"))
    started = time.perf_counter()
    try:
        proc = subprocess.run(
            cmd,
            shell=True,
            capture_output=True,
            text=True,
            env=env,
            timeout=timeout,
        )
    except subprocess.TimeoutExpired as e:
        elapsed_ms = (time.perf_counter() - started) * 1000.0
        return {"latency_ms": elapsed_ms, "error": f"agent timeout after {timeout}s: {e}"}
    elapsed_ms = (time.perf_counter() - started) * 1000.0
    out: dict = {"latency_ms": elapsed_ms}
    if proc.returncode != 0:
        err = (proc.stderr or proc.stdout or "").strip()
        out["error"] = f"agent exited {proc.returncode}: {trim(err, 900)}"
        return out
    try:
        body = json.loads(proc.stdout.strip() or "{}")
        for key in ("input_tokens", "output_tokens"):
            if isinstance(body.get(key), int):
                out[key] = body[key]
        if isinstance(body.get("cost_usd"), (int, float)):
            out["cost_usd"] = float(body["cost_usd"])
    except json.JSONDecodeError:
        # The dispatch script did the work but did not emit metrics.
        pass
    return out


def build_prompt(task: dict, mode: str, adapter: str) -> str:
    body = task["description"]
    if mode == "inject-once" and adapter == "claude":
        return f"/bible John 3:16 {body}"
    if mode == "inject-once" and adapter == "codex":
        return f"{body}\n\n$bible.John.3.16"
    return body


def run(adapters: list[str], modes: list[str], task_filter: Optional[list[str]], repeats: int) -> BenchSummary:
    pack = load_tasks()
    tasks = filter_tasks(pack, task_filter)
    summary = BenchSummary(
        date=str(date.today()),
        adapters=adapters,
        modes=modes,
        repeats=repeats,
        versions=tool_versions(),
    )
    for adapter in adapters:
        for task in tasks:
            for mode in modes:
                for repeat in range(1, repeats + 1):
                    print(
                        f"running {adapter} {task['id']} {mode} r{repeat}/{repeats}",
                        file=sys.stderr,
                        flush=True,
                    )
                    fixture_src = TASKS_FILE.parent / task["fixture_dir"]
                    scratch = (
                        REPO_ROOT
                        / ".bench-scratch"
                        / f"{task['id']}__{mode}__{adapter}__r{repeat}"
                    )
                    if scratch.exists():
                        shutil.rmtree(scratch)
                    scratch.parent.mkdir(parents=True, exist_ok=True)
                    if fixture_src.exists():
                        shutil.copytree(fixture_src, scratch)
                    else:
                        scratch.mkdir(parents=True)
                    metrics = dispatch_agent(task, mode, adapter, repeat, scratch)
                    passed, accept_out = acceptance(scratch, task["acceptance"])
                    error = metrics.get("error")
                    if not passed and not error:
                        error = "acceptance failed: " + trim(accept_out, 900)
                    summary.results.append(
                        TaskResult(
                            task_id=task["id"],
                            mode=mode,
                            adapter=adapter,
                            repeat=repeat,
                            success=passed,
                            input_tokens=metrics.get("input_tokens"),
                            output_tokens=metrics.get("output_tokens"),
                            latency_ms=metrics.get("latency_ms"),
                            cost_usd=metrics.get("cost_usd"),
                            error=error,
                        )
                    )
    return summary


def tool_versions() -> dict[str, str]:
    versions: dict[str, str] = {"python": sys.version.split()[0]}
    for name in ("scripture-mcp", "claude", "codex"):
        versions[name] = version_for(name)
    pytest_version = run_text([sys.executable, "-m", "pytest", "--version"])
    versions["pytest"] = pytest_version.splitlines()[0] if pytest_version else "unavailable"
    return versions


def version_for(name: str) -> str:
    exe = shutil.which(name)
    if not exe:
        return "unavailable"
    out = run_text([exe, "--version"])
    return out.splitlines()[0] if out else exe


def run_text(argv: list[str]) -> str:
    try:
        proc = subprocess.run(argv, capture_output=True, text=True, timeout=20)
    except Exception:
        return ""
    if proc.returncode != 0:
        return ""
    return (proc.stdout or proc.stderr or "").strip()


def render_report(summary: BenchSummary) -> str:
    lines: list[str] = []
    lines.append(f"# Coding-quality benchmark - {summary.date}")
    lines.append("")
    lines.append(f"Adapters: **{', '.join(summary.adapters)}**")
    lines.append(f"Modes: {', '.join(summary.modes)}")
    lines.append(f"Repeats per cell: {summary.repeats}")
    lines.append("")
    lines.append("## Tool Versions")
    lines.append("")
    lines.append("| tool | version |")
    lines.append("|---|---|")
    for name, version in summary.versions.items():
        lines.append(f"| {md(name)} | {md(version)} |")
    lines.append("")
    lines.append("## Gate Verdict")
    lines.append("")
    verdict, reason = gate_verdict(summary)
    lines.append(f"**{verdict}** - {reason}")
    lines.append("")
    lines.append(
        "Gate rule from issue #8: no scripture-enabled mode may regress "
        "success rate by more than 5 percentage points vs that adapter's "
        "baseline."
    )
    lines.append("")
    lines.append("## Per-mode Summary")
    lines.append("")
    lines.append("| adapter | mode | success | n | delta vs baseline | p50 latency ms | median input tok | median output tok | verdict |")
    lines.append("|---|---|---:|---:|---:|---:|---:|---:|---|")
    for adapter in summary.adapters:
        baseline = success_rate(rows_for(summary, adapter, "baseline"))
        for mode in summary.modes:
            rows = rows_for(summary, adapter, mode)
            rate = success_rate(rows)
            delta = None if baseline is None or rate is None else rate - baseline
            mode_verdict = mode_gate(mode, delta, rows)
            lines.append(
                "| "
                + " | ".join(
                    [
                        md(adapter),
                        md(mode),
                        pct(rate),
                        str(len(rows)),
                        pp(delta),
                        num(median_of([r.latency_ms for r in rows])),
                        num(median_of([r.input_tokens for r in rows])),
                        num(median_of([r.output_tokens for r in rows])),
                        md(mode_verdict),
                    ]
                )
                + " |"
            )
    lines.append("")
    lines.append("## Per-task Results")
    lines.append("")
    lines.append("| adapter | task | mode | repeat | success | input tok | output tok | latency ms | cost usd | error |")
    lines.append("|---|---|---|---:|---|---:|---:|---:|---:|---|")
    for r in summary.results:
        lines.append(
            "| "
            + " | ".join(
                [
                    md(r.adapter),
                    md(r.task_id),
                    md(r.mode),
                    str(r.repeat),
                    "pass" if r.success else "fail",
                    num(r.input_tokens),
                    num(r.output_tokens),
                    num(r.latency_ms),
                    money(r.cost_usd),
                    md(r.error or ""),
                ]
            )
            + " |"
        )
    lines.append("")
    return "\n".join(lines)


def rows_for(summary: BenchSummary, adapter: str, mode: str) -> list[TaskResult]:
    return [r for r in summary.results if r.adapter == adapter and r.mode == mode]


def success_rate(rows: list[TaskResult]) -> Optional[float]:
    if not rows:
        return None
    return sum(1 for r in rows if r.success) / len(rows)


def gate_verdict(summary: BenchSummary) -> tuple[str, str]:
    if not summary.results:
        return "INCOMPLETE", "no benchmark rows were produced"
    skipped = [r for r in summary.results if r.error and "no dispatch command configured" in r.error]
    if skipped:
        return "INCOMPLETE", f"{len(skipped)} row(s) skipped because an adapter dispatch command was missing"
    failures: list[str] = []
    for adapter in summary.adapters:
        baseline = success_rate(rows_for(summary, adapter, "baseline"))
        if baseline is None:
            failures.append(f"{adapter}: missing baseline")
            continue
        for mode in summary.modes:
            if mode == "baseline":
                continue
            rows = rows_for(summary, adapter, mode)
            rate = success_rate(rows)
            if rate is None:
                failures.append(f"{adapter}/{mode}: missing rows")
                continue
            if rate < baseline - 0.05:
                failures.append(f"{adapter}/{mode}: {rate:.0%} vs baseline {baseline:.0%}")
    if failures:
        return "FAIL", "; ".join(failures)
    return "PASS", "all enabled modes stayed within 5pp of baseline"


def mode_gate(mode: str, delta: Optional[float], rows: list[TaskResult]) -> str:
    if not rows:
        return "missing"
    if mode == "baseline":
        return "baseline"
    if delta is None:
        return "missing baseline"
    return "pass" if delta >= -0.05 else "fail"


def median_of(values: list[Optional[Union[float, int]]]) -> Optional[float]:
    xs = [float(v) for v in values if v is not None]
    if not xs:
        return None
    return float(statistics.median(xs))


def pct(value: Optional[float]) -> str:
    return "-" if value is None else f"{value:.0%}"


def pp(value: Optional[float]) -> str:
    return "-" if value is None else f"{value * 100:+.0f}pp"


def num(value: Optional[Union[float, int]]) -> str:
    if value is None:
        return "-"
    return f"{float(value):.0f}"


def money(value: Optional[float]) -> str:
    if value is None:
        return "-"
    return f"{value:.4f}"


def md(value: str) -> str:
    return value.replace("|", "\\|").replace("\n", "<br>").strip()


def trim(value: str, limit: int) -> str:
    value = value.replace("\r", "")
    if len(value) <= limit:
        return value
    return value[:limit] + "...[truncated]"


def parse_csv(value: str) -> list[str]:
    return [part.strip() for part in value.split(",") if part.strip()]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--adapter", default="all", choices=("all", "claude", "codex"))
    parser.add_argument(
        "--adapters",
        default="",
        help="comma-separated adapter ids; overrides --adapter",
    )
    parser.add_argument(
        "--modes",
        default=os.environ.get("BENCH_MODES", ",".join(DEFAULT_MODES)),
        help="comma-separated mode ids",
    )
    parser.add_argument(
        "--tasks",
        default=os.environ.get("BENCH_TASK_FILTER", ""),
        help="comma-separated task ids (empty = all)",
    )
    parser.add_argument(
        "--repeats",
        type=int,
        default=int(os.environ.get("BENCH_REPEATS", "1")),
        help="attempts per (adapter, task, mode)",
    )
    parser.add_argument(
        "--out",
        default=os.environ.get(
            "BENCH_OUT",
            str(REPO_ROOT / "docs" / "benchmarks" / f"{date.today()}.md"),
        ),
    )
    args = parser.parse_args()

    if args.adapters:
        adapters = parse_csv(args.adapters)
    elif args.adapter == "all":
        adapters = list(DEFAULT_ADAPTERS)
    else:
        adapters = [args.adapter]
    for adapter in adapters:
        if adapter not in DEFAULT_ADAPTERS:
            sys.exit(f"unknown adapter: {adapter}")

    modes = parse_csv(args.modes)
    for mode in modes:
        if mode not in DEFAULT_MODES:
            sys.exit(f"unknown mode: {mode}")
    tasks = parse_csv(args.tasks) or None
    if args.repeats < 1:
        sys.exit("--repeats must be >= 1")

    summary = run(adapters, modes, tasks, args.repeats)
    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(render_report(summary), encoding="utf-8")
    Path(str(out) + ".json").write_text(
        json.dumps(asdict(summary), indent=2), encoding="utf-8"
    )
    print(f"wrote {out}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main())
