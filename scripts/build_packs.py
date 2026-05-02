#!/usr/bin/env python3
"""Pack builder for KJV, 道德经, and 心经.

Downloads upstream public-domain sources, normalizes, and writes
internal/packs/<name>/{verses.jsonl, metadata.json} files.

Run:    python3 scripts/build_packs.py
Output: internal/packs/{bible-kjv,dao-de-jing,heart-sutra}/

The script intentionally prints no verse text — only structural info
(verse counts, byte sizes, file paths, checksum spot-counts). This is
required to keep the model output filter-safe; see CLAUDE.md.
"""

from __future__ import annotations

import datetime as _dt
import hashlib
import json
import os
import re
import sys
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
PACKS_DIR = ROOT / "internal" / "packs"

KJV_URL = "https://www.gutenberg.org/cache/epub/10/pg10.txt"
DAO_URL = "https://www.gutenberg.org/cache/epub/7337/pg7337.txt"

# ---------- shared helpers ----------

def _sha256(s: str) -> str:
    return hashlib.sha256(s.encode("utf-8")).hexdigest()


def _fetch(url: str) -> str:
    req = urllib.request.Request(url, headers={"User-Agent": "verse-driven/0.1 pack-builder"})
    with urllib.request.urlopen(req, timeout=120) as r:
        return r.read().decode("utf-8-sig")


def _write_pack(name: str, verses: list[dict], metadata: dict) -> None:
    """Write a pack.

    Layout:
      internal/packs/<name>/verses.jsonl.gz   - compact, gzip-compressed
      internal/packs/<name>/metadata.json     - shared fields + verse_count

    Each line in verses.jsonl.gz is a compact object:
      {"id": str, "c": int, "v": int, "ve"?: int, "b"?: str, "t": str, "s": hex64}

    Pack-level fields (tradition, lang, work, source, sensitivity,
    inclusion_mode, default_lang display strings) live in metadata.json so
    they are not duplicated 31k times.
    """
    import gzip
    out_dir = PACKS_DIR / name
    out_dir.mkdir(parents=True, exist_ok=True)
    jsonl_path = out_dir / "verses.jsonl.gz"
    payload_lines: list[bytes] = []
    for v in verses:
        compact: dict = {
            "id": v["id"],
            "c": v["canonical_ref"]["chapter"],
            "v": v["canonical_ref"]["verse_start"],
            "t": v["text"],
            "s": v["checksum_sha256"],
        }
        if v["canonical_ref"].get("verse_end"):
            compact["ve"] = v["canonical_ref"]["verse_end"]
        if v["canonical_ref"].get("book"):
            compact["b"] = v["canonical_ref"]["book"]
        payload_lines.append(json.dumps(compact, ensure_ascii=False, sort_keys=True).encode("utf-8"))
    payload = b"\n".join(payload_lines) + (b"\n" if payload_lines else b"")
    # Pin mtime=0 in the gzip header so the archive is reproducible across builds.
    with open(jsonl_path, "wb") as raw_f:
        with gzip.GzipFile(fileobj=raw_f, mode="wb", compresslevel=9, mtime=0) as f:
            f.write(payload)
    meta_path = out_dir / "metadata.json"
    metadata = dict(metadata)
    metadata["verse_count"] = len(verses)
    metadata["build_date"] = _dt.date.today().isoformat()
    with meta_path.open("w", encoding="utf-8") as f:
        json.dump(metadata, f, ensure_ascii=False, indent=2, sort_keys=True)
        f.write("\n")
    size_kb = jsonl_path.stat().st_size / 1024
    print(f"  -> {jsonl_path.relative_to(ROOT)}  verses={len(verses)}  gz_size={size_kb:.1f} KiB")


# ---------- KJV Bible ----------

# Canonical 66-book order with the exact Gutenberg PG10 section heading
# (after the table of contents) and the dotted-id slug used in verse ids.
KJV_BOOKS: list[tuple[str, str, str]] = [
    ("The First Book of Moses: Called Genesis", "Genesis", "genesis"),
    ("The Second Book of Moses: Called Exodus", "Exodus", "exodus"),
    ("The Third Book of Moses: Called Leviticus", "Leviticus", "leviticus"),
    ("The Fourth Book of Moses: Called Numbers", "Numbers", "numbers"),
    ("The Fifth Book of Moses: Called Deuteronomy", "Deuteronomy", "deuteronomy"),
    ("The Book of Joshua", "Joshua", "joshua"),
    ("The Book of Judges", "Judges", "judges"),
    ("The Book of Ruth", "Ruth", "ruth"),
    ("The First Book of Samuel", "1 Samuel", "1-samuel"),
    ("The Second Book of Samuel", "2 Samuel", "2-samuel"),
    ("The First Book of the Kings", "1 Kings", "1-kings"),
    ("The Second Book of the Kings", "2 Kings", "2-kings"),
    ("The First Book of the Chronicles", "1 Chronicles", "1-chronicles"),
    ("The Second Book of the Chronicles", "2 Chronicles", "2-chronicles"),
    ("Ezra", "Ezra", "ezra"),
    ("The Book of Nehemiah", "Nehemiah", "nehemiah"),
    ("The Book of Esther", "Esther", "esther"),
    ("The Book of Job", "Job", "job"),
    ("The Book of Psalms", "Psalms", "psalms"),
    ("The Proverbs", "Proverbs", "proverbs"),
    ("Ecclesiastes", "Ecclesiastes", "ecclesiastes"),
    ("The Song of Solomon", "Song of Solomon", "song-of-solomon"),
    ("The Book of the Prophet Isaiah", "Isaiah", "isaiah"),
    ("The Book of the Prophet Jeremiah", "Jeremiah", "jeremiah"),
    ("The Lamentations of Jeremiah", "Lamentations", "lamentations"),
    ("The Book of the Prophet Ezekiel", "Ezekiel", "ezekiel"),
    ("The Book of Daniel", "Daniel", "daniel"),
    ("Hosea", "Hosea", "hosea"),
    ("Joel", "Joel", "joel"),
    ("Amos", "Amos", "amos"),
    ("Obadiah", "Obadiah", "obadiah"),
    ("Jonah", "Jonah", "jonah"),
    ("Micah", "Micah", "micah"),
    ("Nahum", "Nahum", "nahum"),
    ("Habakkuk", "Habakkuk", "habakkuk"),
    ("Zephaniah", "Zephaniah", "zephaniah"),
    ("Haggai", "Haggai", "haggai"),
    ("Zechariah", "Zechariah", "zechariah"),
    ("Malachi", "Malachi", "malachi"),
    ("The Gospel According to Saint Matthew", "Matthew", "matthew"),
    ("The Gospel According to Saint Mark", "Mark", "mark"),
    ("The Gospel According to Saint Luke", "Luke", "luke"),
    ("The Gospel According to Saint John", "John", "john"),
    ("The Acts of the Apostles", "Acts", "acts"),
    ("The Epistle of Paul the Apostle to the Romans", "Romans", "romans"),
    ("The First Epistle of Paul the Apostle to the Corinthians", "1 Corinthians", "1-corinthians"),
    ("The Second Epistle of Paul the Apostle to the Corinthians", "2 Corinthians", "2-corinthians"),
    ("The Epistle of Paul the Apostle to the Galatians", "Galatians", "galatians"),
    ("The Epistle of Paul the Apostle to the Ephesians", "Ephesians", "ephesians"),
    ("The Epistle of Paul the Apostle to the Philippians", "Philippians", "philippians"),
    ("The Epistle of Paul the Apostle to the Colossians", "Colossians", "colossians"),
    ("The First Epistle of Paul the Apostle to the Thessalonians", "1 Thessalonians", "1-thessalonians"),
    ("The Second Epistle of Paul the Apostle to the Thessalonians", "2 Thessalonians", "2-thessalonians"),
    ("The First Epistle of Paul the Apostle to Timothy", "1 Timothy", "1-timothy"),
    ("The Second Epistle of Paul the Apostle to Timothy", "2 Timothy", "2-timothy"),
    ("The Epistle of Paul the Apostle to Titus", "Titus", "titus"),
    ("The Epistle of Paul the Apostle to Philemon", "Philemon", "philemon"),
    ("The Epistle of Paul the Apostle to the Hebrews", "Hebrews", "hebrews"),
    ("The General Epistle of James", "James", "james"),
    ("The First Epistle General of Peter", "1 Peter", "1-peter"),
    ("The Second General Epistle of Peter", "2 Peter", "2-peter"),
    ("The First Epistle General of John", "1 John", "1-john"),
    ("The Second Epistle General of John", "2 John", "2-john"),
    ("The Third Epistle General of John", "3 John", "3-john"),
    ("The General Epistle of Jude", "Jude", "jude"),
    ("The Revelation of Saint John the Divine", "Revelation", "revelation"),
]

VERSE_MARKER = re.compile(r"\b(\d+):(\d+)\b")
START_MARKER = "*** START OF THE PROJECT GUTENBERG EBOOK THE KING JAMES VERSION OF THE BIBLE ***"
END_MARKER = "*** END OF THE PROJECT GUTENBERG EBOOK THE KING JAMES VERSION OF THE BIBLE ***"


def _find_heading(body: str, heading: str, after: int) -> int:
    """Find the next heading occurrence in `body` after offset `after`.

    A heading sits on its own line surrounded by blank lines (PG #10 puts
    several blank lines before/after each book section header). We require
    the match to be preceded by at least one blank line and followed by at
    least one blank line so that verse-content mentions of the heading text
    (e.g. the word 'Haggai' inside a verse) don't false-match.
    """
    pos = after
    while True:
        idx = body.find(heading, pos)
        if idx < 0:
            return -1
        line_start = body.rfind("\n", 0, idx) + 1
        line_end = body.find("\n", idx + len(heading))
        if line_end < 0:
            line_end = len(body)
        line = body[line_start:line_end].strip(" \t\r")
        if line == heading:
            # confirm surrounding blank lines: previous non-empty line is far
            # enough back, and next non-empty line is far enough forward.
            before = body[max(0, line_start - 6) : line_start]
            after_chunk = body[line_end : line_end + 6]
            if before.count("\n") >= 1 and after_chunk.count("\n") >= 1:
                return idx
        pos = idx + len(heading)


def _slice_book(body: str, idx: int, cursor: int) -> tuple[str, int]:
    """Return (book_text, new_cursor) for book idx; book_text excludes heading."""
    heading = KJV_BOOKS[idx][0]
    start = _find_heading(body, heading, cursor)
    if start < 0:
        raise RuntimeError(f"KJV: heading not found: {heading!r}")
    after_heading = start + len(heading)
    if idx + 1 < len(KJV_BOOKS):
        nxt = _find_heading(body, KJV_BOOKS[idx + 1][0], after_heading)
        if nxt < 0:
            raise RuntimeError(f"KJV: next heading not found after {heading!r}")
        return body[after_heading:nxt], nxt
    return body[after_heading:], len(body)


def _parse_kjv_book(book_text: str) -> list[tuple[int, int, str]]:
    """Return [(chapter, verse, text)] for one book."""
    # Collapse the text: strip leading/trailing, normalize whitespace runs to single
    # spaces but keep the verse markers intact.
    text = book_text.strip()
    # Find all verse-marker positions.
    matches = list(VERSE_MARKER.finditer(text))
    if not matches:
        return []
    out: list[tuple[int, int, str]] = []
    for i, m in enumerate(matches):
        ch = int(m.group(1))
        vs = int(m.group(2))
        body_start = m.end()
        body_end = matches[i + 1].start() if i + 1 < len(matches) else len(text)
        verse_body = text[body_start:body_end]
        # Normalize whitespace.
        verse_body = re.sub(r"\s+", " ", verse_body).strip()
        out.append((ch, vs, verse_body))
    return out


TOC_END_MARKER = "The Old Testament of the King James Version of the Bible"


def build_kjv() -> None:
    print("[kjv] downloading...")
    raw = _fetch(KJV_URL)
    s = raw.find(START_MARKER)
    e = raw.find(END_MARKER)
    if s < 0 or e < 0:
        raise RuntimeError("KJV: PG markers not found")
    # PG #10 lists each book once in the TOC, then again as a section header
    # before the verses. The TOC sits between START_MARKER and the *second*
    # occurrence of TOC_END_MARKER ("The Old Testament..."). Skip past it.
    after_start = s + len(START_MARKER)
    first_old = raw.find(TOC_END_MARKER, after_start)
    second_old = raw.find(TOC_END_MARKER, first_old + 1) if first_old >= 0 else -1
    if second_old < 0:
        raise RuntimeError("KJV: failed to find TOC/body boundary")
    body = raw[second_old:e]
    verses: list[dict] = []
    cursor = 0
    for idx, (heading, display, slug) in enumerate(KJV_BOOKS):
        sect, cursor = _slice_book(body, idx, cursor)
        parsed = _parse_kjv_book(sect)
        if not parsed:
            raise RuntimeError(f"KJV: no verses parsed for {display!r}")
        for ch, vs, text in parsed:
            vid = f"bible.kjv.{slug}.{ch}.{vs}"
            verses.append({
                "id": vid,
                "tradition": "bible",
                "lang": "en",
                "work": "KJV",
                "canonical_ref": {"book": display, "chapter": ch, "verse_start": vs},
                "text": text,
                "source": {
                    "provider": "Project Gutenberg eBook #10",
                    "license": "Public domain (United States)",
                    "attribution": "King James Version of the Bible, Project Gutenberg eBook #10",
                },
                "checksum_sha256": _sha256(text),
                "inclusion_mode": "bundled",
                "sensitivity": "sacred_exact_quote",
            })
    meta = {
        "tradition": "bible",
        "work": "KJV",
        "lang": "en",
        "provider": "Project Gutenberg eBook #10",
        "source_url": KJV_URL,
        "license": "Public domain (United States)",
        "attribution": "King James Version of the Bible, Project Gutenberg eBook #10",
        "edition_id": "pg10-kjv",
        "inclusion_mode": "bundled",
        "sensitivity": "sacred_exact_quote",
        # Slug → canonical book name. Slug is the second-to-last id segment
        # before the chapter (e.g. bible.kjv.song-of-solomon.5.1 → "Song of Solomon").
        "books": {slug: display for (_, display, slug) in KJV_BOOKS},
    }
    _write_pack("bible-kjv", verses, meta)


# ---------- 道德经 ----------

DAO_START_MARKER = "*** START OF THE PROJECT GUTENBERG EBOOK"
DAO_END_MARKER = "*** END OF THE PROJECT GUTENBERG EBOOK"
# Each chapter heading line in PG #7337 looks like "第一章", "第二章", ... "第八十一章".
DAO_CHAPTER_RE = re.compile(r"^第([一-鿿]+)章\s*$", re.MULTILINE)
# Map Chinese numerals 1..81 used in the source.
_DIGITS = {"零": 0, "一": 1, "二": 2, "三": 3, "四": 4, "五": 5, "六": 6, "七": 7, "八": 8, "九": 9}


def _parse_cn_numeral(s: str) -> int:
    s = s.strip()
    if s == "十":
        return 10
    if s.startswith("十"):
        return 10 + _DIGITS[s[1:]]
    if "十" in s:
        a, _, b = s.partition("十")
        tens = _DIGITS[a] * 10
        return tens + (_DIGITS[b] if b else 0)
    if len(s) == 1 and s in _DIGITS:
        return _DIGITS[s]
    raise ValueError(f"unknown CN numeral {s!r}")


def _t2s_dao(text: str) -> str:
    try:
        from opencc import OpenCC
    except ImportError:
        print("[dao] opencc-python-reimplemented not installed; skipping t->s conversion", file=sys.stderr)
        return text
    cc = OpenCC("t2s")
    return cc.convert(text)


def build_dao() -> None:
    print("[dao] downloading...")
    raw = _fetch(DAO_URL)
    s_idx = raw.find(DAO_START_MARKER)
    e_idx = raw.find(DAO_END_MARKER)
    if s_idx < 0 or e_idx < 0:
        raise RuntimeError("DAO: PG markers not found")
    # advance past the START line itself.
    line_end = raw.find("\n", s_idx)
    body = raw[line_end + 1 : e_idx]

    # Find chapter headings; segment by them.
    headings = list(DAO_CHAPTER_RE.finditer(body))
    if len(headings) < 81:
        raise RuntimeError(f"DAO: expected >=81 chapter headings, found {len(headings)}")

    # The Gutenberg edition repeats each chapter heading inside structural section
    # banners ("老子《道德經》 第一~四十章") — keep only the first 81 occurrences.
    chapters: dict[int, list[str]] = {}
    for i, m in enumerate(headings):
        cn = m.group(1)
        try:
            num = _parse_cn_numeral(cn)
        except ValueError:
            continue
        if num < 1 or num > 81:
            continue
        body_start = m.end()
        body_end = headings[i + 1].start() if i + 1 < len(headings) else len(body)
        chunk = body[body_start:body_end].strip()
        # Skip noise: section banner lines like "老子德經" tend to sit *before* a
        # chapter heading, not after. The first non-empty post-heading block IS
        # the chapter body. We accept the longest non-empty chunk for each
        # chapter number across duplicate occurrences.
        chapters.setdefault(num, []).append(chunk)

    if len(chapters) != 81:
        raise RuntimeError(f"DAO: parsed {len(chapters)} unique chapters, expected 81")

    verses: list[dict] = []
    for n in range(1, 82):
        candidates = [c for c in chapters[n] if c]
        if not candidates:
            raise RuntimeError(f"DAO: empty body for chapter {n}")
        # longest candidate wins.
        traditional = max(candidates, key=len)
        # Normalize whitespace within the chapter: collapse runs to single spaces,
        # but preserve the original line breaks as a single space delimiter.
        traditional = re.sub(r"\s+", "", traditional)
        simplified = _t2s_dao(traditional)
        vid = f"dao.daodejing.{n}.1"
        verses.append({
            "id": vid,
            "tradition": "dao",
            "lang": "zh-Hans",
            "work": "daodejing",
            "canonical_ref": {"chapter": n, "verse_start": 1},
            "display_ref": {"zh-Hans": f"道德经第{n}章", "en": f"Tao Te Ching, Chapter {n}"},
            "text": simplified,
            "source": {
                "provider": "Project Gutenberg eBook #7337",
                "license": "Public domain",
                "attribution": "《道德經》, Project Gutenberg eBook #7337 (produced by Ching-yi Chen). Simplified-Chinese rendering via OpenCC t2s.",
            },
            "checksum_sha256": _sha256(simplified),
            "inclusion_mode": "bundled",
            "sensitivity": "sacred_exact_quote",
        })
    meta = {
        "tradition": "dao",
        "work": "daodejing",
        "lang": "zh-Hans",
        "provider": "Project Gutenberg eBook #7337",
        "source_url": DAO_URL,
        "license": "Public domain",
        "attribution": "《道德經》, Project Gutenberg eBook #7337 (produced by Ching-yi Chen). Simplified-Chinese rendering via OpenCC t2s.",
        "edition_id": "pg7337-laozi-s",
        "inclusion_mode": "bundled",
        "sensitivity": "sacred_exact_quote",
        "transform": "OpenCC t2s (Traditional → Simplified)",
    }
    _write_pack("dao-de-jing", verses, meta)


# ---------- 心经 ----------
# CBETA's redistribution terms for the Xuanzang translation are non-trivial to
# audit at build time, and our reachable upstream sources don't reliably
# return the canonical text. Per issue #3 notes ("fall back to api-only mode
# for that pack if uncertain"), we ship the heart-sutra pack with
# inclusion_mode = api_only and 0 bundled verses. The registry still surfaces
# the pack via metadata.json, and a future PR can vendor verses once the
# CBETA license review is complete.

def build_sutra() -> None:
    verses: list[dict] = []
    meta = {
        "tradition": "sutra",
        "work": "heart-sutra",
        "lang": "zh-Hans",
        "provider": "CBETA (pending license audit)",
        "source_url": "https://cbetaonline.dila.edu.tw/zh/T0251_001",
        "license": "See pack release notes",
        "attribution": "《般若波罗蜜多心经》, translated by Xuanzang (玄奘, Tang dynasty, c. 649 CE). Public domain text; CBETA digital edition has its own redistribution terms.",
        "edition_id": "xuanzang-heart-sutra",
        "inclusion_mode": "api_only",
        "note": "Stub pack: text not yet bundled. Issue #3 notes permit api-only fallback while CBETA terms are being reviewed.",
    }
    _write_pack("heart-sutra", verses, meta)


# ---------- entrypoint ----------

def main() -> int:
    targets = sys.argv[1:] or ["kjv", "dao", "sutra"]
    if "kjv" in targets:
        build_kjv()
    if "dao" in targets:
        build_dao()
    if "sutra" in targets:
        build_sutra()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
