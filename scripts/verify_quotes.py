#!/usr/bin/env python3
"""Recompute SHA-256 over each verse's text and compare to the stored checksum.

Walks every internal/packs/*/verses.jsonl.gz, recomputes the SHA-256 over
the bytes of the `t` field, and fails if any row's stored `s` doesn't match.

Run:    python3 scripts/verify_quotes.py
Exit 0  - all packs verified
Exit 1  - at least one mismatch (or missing file)

This script intentionally does not print verse text on mismatch — only the
verse `id`, expected hash, and recomputed hash. Sacred-text bodies are kept
out of CI logs and out of any Claude Code transcript that scrapes them.
See CLAUDE.md.
"""

from __future__ import annotations

import gzip
import hashlib
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
PACKS_DIR = ROOT / "internal" / "packs"


def _sha256(s: str) -> str:
    return hashlib.sha256(s.encode("utf-8")).hexdigest()


def verify_pack(pack_dir: Path) -> tuple[int, int]:
    """Return (verses_checked, mismatches_for_this_pack)."""
    jsonl_gz = pack_dir / "verses.jsonl.gz"
    meta_path = pack_dir / "metadata.json"
    if not meta_path.exists():
        print(f"  [{pack_dir.name}] MISSING metadata.json", file=sys.stderr)
        return 0, 1
    if not jsonl_gz.exists():
        print(f"  [{pack_dir.name}] MISSING verses.jsonl.gz", file=sys.stderr)
        return 0, 1
    meta = json.loads(meta_path.read_text(encoding="utf-8"))
    declared_count = meta.get("verse_count", -1)

    n = 0
    bad = 0
    with gzip.open(jsonl_gz, "rb") as f:
        for raw in f:
            line = raw.decode("utf-8").strip()
            if not line:
                continue
            row = json.loads(line)
            n += 1
            actual = _sha256(row["t"])
            if actual != row["s"]:
                # Print only ids and hashes — never row["t"].
                print(f"  [{pack_dir.name}] MISMATCH id={row['id']}", file=sys.stderr)
                print(f"    stored:     {row['s']}", file=sys.stderr)
                print(f"    recomputed: {actual}", file=sys.stderr)
                bad += 1
    if declared_count >= 0 and declared_count != n:
        print(
            f"  [{pack_dir.name}] verse_count mismatch: metadata={declared_count} jsonl={n}",
            file=sys.stderr,
        )
        bad += 1
    print(f"  [{pack_dir.name}] verses={n} mismatches={bad}")
    return n, bad


def main() -> int:
    if not PACKS_DIR.exists():
        print(f"no packs at {PACKS_DIR}", file=sys.stderr)
        return 1
    total = 0
    bad = 0
    pack_dirs = sorted(p for p in PACKS_DIR.iterdir() if p.is_dir())
    print(f"verify_quotes: scanning {len(pack_dirs)} pack(s)")
    for pd in pack_dirs:
        n, b = verify_pack(pd)
        total += n
        bad += b
    print(f"verify_quotes: total verses={total} mismatches={bad}")
    return 0 if bad == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
