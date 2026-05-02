"""A small CLI that summarizes a list of (name, count) pairs.

Today it prints a human-readable summary. The agent is asked to add an
opt-in ``--json`` flag that emits the same data as a single JSON
object on stdout, while keeping the default human-readable output
exactly as it is today.

The summary is computed by ``summarize(items)`` and rendered for
humans by ``render_text(summary)``. Add a ``render_json(summary)``
that returns a JSON string with the same fields.

Public functions to keep:
  summarize(items) -> dict
  render_text(summary) -> str
  main(argv) -> int           # writes to stdout, returns exit code
"""

from __future__ import annotations

import argparse
import sys


def summarize(items):
    total = sum(c for _, c in items)
    top = max(items, key=lambda x: x[1])[0] if items else None
    return {
        "total": total,
        "count": len(items),
        "top": top,
    }


def render_text(summary):
    return (
        f"items: {summary['count']}\n"
        f"total: {summary['total']}\n"
        f"top: {summary['top']}\n"
    )


def main(argv=None):
    parser = argparse.ArgumentParser(description="summarize (name,count) pairs")
    parser.add_argument(
        "pairs",
        nargs="*",
        help="name=count tokens, e.g. apples=3 bananas=7",
    )
    args = parser.parse_args(argv)
    items = []
    for tok in args.pairs:
        if "=" not in tok:
            print(f"bad pair: {tok}", file=sys.stderr)
            return 2
        name, count = tok.split("=", 1)
        items.append((name, int(count)))
    summary = summarize(items)
    sys.stdout.write(render_text(summary))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
