#!/usr/bin/env python3
"""Pack builder for bilingual Bible, Dao, Quran, and Heart Sutra packs.

Downloads upstream public-domain sources, normalizes, and writes
internal/packs/<name>/{verses.jsonl, metadata.json} files.

Run:    python3 scripts/build_packs.py
Output: internal/packs/<pack-name>/

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
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
PACKS_DIR = ROOT / "internal" / "packs"

KJV_URL = "https://www.gutenberg.org/cache/epub/10/pg10.txt"
CUV_S_URL = "https://raw.githubusercontent.com/seven1m/open-bibles/master/chi-cuv-simp.usfx.xml"
DAO_URL = "https://www.gutenberg.org/cache/epub/7337/pg7337.txt"
DAO_LEGGE_URL = "https://classics.mit.edu/Lao/taote.mb.txt"
SUTRA_XML_URL = "https://raw.githubusercontent.com/cbeta-org/xml-p5/master/T/T08/T08n0251.xml"
SUTRA_SOURCE_URL = "https://cbetaonline.dila.edu.tw/zh/T0251_001"
SUTRA_EN_RAW_URL = "https://en.wikisource.org/w/index.php?title=Translation:Shorter_Praj%C3%B1%C4%81p%C4%81ramit%C4%81_H%E1%B9%9Bdaya_S%C5%ABtra&action=raw"
QURAN_PICKTHALL_URL = "https://tanzil.net/trans/en.pickthall"
QURAN_MAJIAN_URL = "https://tanzil.net/trans/zh.jian"

# ---------- shared helpers ----------

def _sha256(s: str) -> str:
    return hashlib.sha256(s.encode("utf-8")).hexdigest()


def _fetch(url: str) -> str:
    req = urllib.request.Request(url, headers={"User-Agent": "verse-driven/0.1 pack-builder"})
    with urllib.request.urlopen(req, timeout=120) as r:
        return r.read().decode("utf-8-sig")


def _t2s(text: str) -> str:
    try:
        from opencc import OpenCC
    except ImportError:
        raise RuntimeError(
            "opencc-python-reimplemented is required to rebuild zh-Hans packs; "
            "install it in a virtualenv before running `make packs`"
        ) from None
    cc = OpenCC("t2s")
    return cc.convert(text)


def _write_pack(name: str, verses: list[dict], metadata: dict) -> None:
    """Write a pack.

    Layout:
      internal/packs/<name>/verses.jsonl.gz   - compact, gzip-compressed
      internal/packs/<name>/metadata.json     - shared fields + verse_count

    Each line in verses.jsonl.gz is a compact object:
      {"id": str, "c": int, "v": int, "ve"?: int, "b"?: str, "d"?: map, "t": str, "s": hex64}

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
        if v.get("display_ref"):
            compact["d"] = v["display_ref"]
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

USFX_BOOK_CODES = [
    "GEN", "EXO", "LEV", "NUM", "DEU", "JOS", "JDG", "RUT",
    "1SA", "2SA", "1KI", "2KI", "1CH", "2CH", "EZR", "NEH", "EST",
    "JOB", "PSA", "PRO", "ECC", "SNG", "ISA", "JER", "LAM", "EZK",
    "DAN", "HOS", "JOL", "AMO", "OBA", "JON", "MIC", "NAM", "HAB",
    "ZEP", "HAG", "ZEC", "MAL", "MAT", "MRK", "LUK", "JHN", "ACT",
    "ROM", "1CO", "2CO", "GAL", "EPH", "PHP", "COL", "1TH", "2TH",
    "1TI", "2TI", "TIT", "PHM", "HEB", "JAS", "1PE", "2PE", "1JN",
    "2JN", "3JN", "JUD", "REV",
]

CUV_BOOK_NAMES = [
    "创世记", "出埃及记", "利未记", "民数记", "申命记", "约书亚记", "士师记", "路得记",
    "撒母耳记上", "撒母耳记下", "列王纪上", "列王纪下", "历代志上", "历代志下", "以斯拉记", "尼希米记", "以斯帖记",
    "约伯记", "诗篇", "箴言", "传道书", "雅歌", "以赛亚书", "耶利米书", "耶利米哀歌", "以西结书",
    "但以理书", "何西阿书", "约珥书", "阿摩司书", "俄巴底亚书", "约拿书", "弥迦书", "那鸿书", "哈巴谷书",
    "西番雅书", "哈该书", "撒迦利亚书", "玛拉基书", "马太福音", "马可福音", "路加福音", "约翰福音", "使徒行传",
    "罗马书", "哥林多前书", "哥林多后书", "加拉太书", "以弗所书", "腓立比书", "歌罗西书", "帖撒罗尼迦前书", "帖撒罗尼迦后书",
    "提摩太前书", "提摩太后书", "提多书", "腓利门书", "希伯来书", "雅各书", "彼得前书", "彼得后书", "约翰一书",
    "约翰二书", "约翰三书", "犹大书", "启示录",
]

CUV_BOOKS = {
    code: {
        "display": KJV_BOOKS[i][1],
        "slug": KJV_BOOKS[i][2],
        "zh": CUV_BOOK_NAMES[i],
    }
    for i, code in enumerate(USFX_BOOK_CODES)
}

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


# ---------- Chinese Union Version Bible (Simplified) ----------

def build_cuv_s() -> None:
    print("[cuv-s] downloading...")
    raw = _fetch(CUV_S_URL)
    root = ET.fromstring(raw)
    verses: list[dict] = []
    seen_books: set[str] = set()
    for book_el in root:
        if _local_name(book_el.tag) != "book":
            continue
        code = book_el.attrib.get("id", "")
        info = CUV_BOOKS.get(code)
        if info is None:
            raise RuntimeError(f"CUV-S: unknown USFX book code {code!r}")
        seen_books.add(code)
        chapter = 0
        for child in book_el:
            tag = _local_name(child.tag)
            if tag == "c":
                chapter = int(child.attrib["id"])
                continue
            if tag != "v":
                continue
            if chapter < 1:
                raise RuntimeError(f"CUV-S: verse before chapter in {code}")
            verse_id = child.attrib.get("id", "")
            if not verse_id.isdigit():
                raise RuntimeError(f"CUV-S: unsupported verse id {verse_id!r} in {code} {chapter}")
            verse = int(verse_id)
            text = re.sub(r"\s+", "", child.tail or "")
            if not text:
                raise RuntimeError(f"CUV-S: empty verse in {code} {chapter}:{verse}")
            vid = f"bible.cuv-s.{info['slug']}.{chapter}.{verse}"
            verses.append({
                "id": vid,
                "tradition": "bible",
                "lang": "zh-Hans",
                "work": "CUV-S",
                "canonical_ref": {"book": info["display"], "chapter": chapter, "verse_start": verse},
                "display_ref": {
                    "zh-Hans": f"{info['zh']} {chapter}:{verse}",
                    "en": f"{info['display']} {chapter}:{verse}",
                },
                "text": text,
                "source": {
                    "provider": "open-bibles",
                    "license": "Public domain",
                    "attribution": "Chinese Union Version (Simplified), open-bibles USFX",
                },
                "checksum_sha256": _sha256(text),
                "inclusion_mode": "bundled",
                "sensitivity": "sacred_exact_quote",
            })
    if seen_books != set(USFX_BOOK_CODES):
        missing = sorted(set(USFX_BOOK_CODES) - seen_books)
        raise RuntimeError(f"CUV-S: missing book codes: {missing}")
    meta = {
        "tradition": "bible",
        "work": "CUV-S",
        "lang": "zh-Hans",
        "provider": "open-bibles",
        "source_url": CUV_S_URL,
        "license": "Public domain",
        "attribution": "Chinese Union Version (Simplified), open-bibles USFX",
        "edition_id": "open-bibles-cuv-simp",
        "inclusion_mode": "bundled",
        "sensitivity": "sacred_exact_quote",
        "books": {KJV_BOOKS[i][2]: KJV_BOOKS[i][1] for i in range(len(KJV_BOOKS))},
    }
    _write_pack("bible-cuv-s", verses, meta)


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
    return _t2s(text)


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


def build_dao_legge() -> None:
    print("[dao-legge] downloading...")
    raw = _fetch(DAO_LEGGE_URL)
    markers = list(re.finditer(r"\bChapter\s+(\d{1,2})\b", raw))
    if len(markers) != 81:
        raise RuntimeError(f"DAO-LEGGE: expected 81 chapter markers, found {len(markers)}")

    verses: list[dict] = []
    for i, m in enumerate(markers):
        chapter = int(m.group(1))
        if chapter != i + 1:
            raise RuntimeError(f"DAO-LEGGE: chapter sequence broke at {chapter}, want {i + 1}")
        start = m.end()
        if i + 1 < len(markers):
            end = markers[i + 1].start()
        else:
            end = raw.find("----------------------------------------------------------------------", start)
            if end < 0:
                end = len(raw)
        text = re.sub(r"\s+", " ", raw[start:end]).strip()
        if not text:
            raise RuntimeError(f"DAO-LEGGE: empty body for chapter {chapter}")
        vid = f"dao.legge.{chapter}.1"
        verses.append({
            "id": vid,
            "tradition": "dao",
            "lang": "en",
            "work": "legge",
            "canonical_ref": {"chapter": chapter, "verse_start": 1},
            "display_ref": {"en": f"Tao Te Ching, Chapter {chapter}", "zh-Hans": f"道德经第{chapter}章"},
            "text": text,
            "source": {
                "provider": "Internet Classics Archive",
                "license": "Public domain source text",
                "attribution": "Tao Te Ching, translated by James Legge (1891), Internet Classics Archive text",
            },
            "checksum_sha256": _sha256(text),
            "inclusion_mode": "bundled",
            "sensitivity": "sacred_exact_quote",
        })
    meta = {
        "tradition": "dao",
        "work": "legge",
        "lang": "en",
        "provider": "Internet Classics Archive",
        "source_url": DAO_LEGGE_URL,
        "license": "Public domain source text",
        "attribution": "Tao Te Ching, translated by James Legge (1891), Internet Classics Archive text",
        "edition_id": "legge-1891",
        "inclusion_mode": "bundled",
        "sensitivity": "sacred_exact_quote",
    }
    _write_pack("dao-legge", verses, meta)


# ---------- 心经 ----------

def _local_name(tag: str) -> str:
    return tag.rsplit("}", 1)[-1]


def _tei_attr(el: ET.Element, name: str) -> str:
    return el.attrib.get(name, el.attrib.get(f"{{http://www.w3.org/XML/1998/namespace}}{name}", ""))


def _tei_text(el: ET.Element) -> str:
    """Extract reading text from a small CBETA TEI subtree.

    We keep lemma readings in apparatus entries and skip line/page anchors,
    notes, and metadata-only nodes. Tails are retained so inline markup does
    not accidentally concatenate adjacent readings.
    """
    tag = _local_name(el.tag)
    if tag in {"lb", "pb", "anchor", "note", "mulu"}:
        return el.tail or ""
    if tag == "app":
        parts: list[str] = []
        for child in el:
            if _local_name(child.tag) == "lem":
                parts.append(_tei_text(child))
                break
        parts.append(el.tail or "")
        return "".join(parts)

    parts = [el.text or ""]
    for child in el:
        parts.append(_tei_text(child))
    parts.append(el.tail or "")
    return "".join(parts)


def _extract_sutra_body(xml_text: str) -> str:
    root = ET.fromstring(xml_text)
    jing = None
    for div in root.iter():
        if _local_name(div.tag) == "div" and _tei_attr(div, "type") == "jing":
            jing = div
            break
    if jing is None:
        raise RuntimeError("SUTRA: TEI div type=jing not found")

    paragraphs: list[str] = []
    for p in jing.iter():
        if _local_name(p.tag) != "p":
            continue
        text = re.sub(r"\s+", "", _tei_text(p))
        if text:
            paragraphs.append(text)
    if not paragraphs:
        raise RuntimeError("SUTRA: no body paragraphs parsed")
    return "\n".join(paragraphs)

def build_sutra() -> None:
    print("[sutra] downloading...")
    raw = _fetch(SUTRA_XML_URL)
    traditional = _extract_sutra_body(raw)
    simplified = _t2s(traditional)
    verses: list[dict] = [{
        "id": "sutra.heart-sutra.1",
        "tradition": "sutra",
        "lang": "zh-Hans",
        "work": "heart-sutra",
        "canonical_ref": {"chapter": 1, "verse_start": 1},
        "text": simplified,
        "source": {
            "provider": "CBETA XML P5 T0251",
            "license": "Ancient source text; CBETA digital edition terms apply",
            "attribution": "《般若波罗蜜多心经》, translated by Xuanzang (玄奘), CBETA XML P5 T0251.",
        },
        "checksum_sha256": _sha256(simplified),
        "inclusion_mode": "bundled",
        "sensitivity": "sacred_exact_quote",
    }]
    meta = {
        "tradition": "sutra",
        "work": "heart-sutra",
        "lang": "zh-Hans",
        "provider": "CBETA XML P5 T0251",
        "source_url": SUTRA_SOURCE_URL,
        "license": "Ancient source text; CBETA digital edition terms apply",
        "attribution": "《般若波罗蜜多心经》, translated by Xuanzang (玄奘), CBETA XML P5 T0251.",
        "edition_id": "xuanzang-heart-sutra",
        "inclusion_mode": "bundled",
        "sensitivity": "sacred_exact_quote",
        "transform": "CBETA XML P5 body extraction; OpenCC t2s (Traditional → Simplified)",
        "note": "Bundled for non-commercial release use; source page links to the original CBETA edition.",
    }
    _write_pack("heart-sutra", verses, meta)


# ---------- Heart Sutra (English Wikisource translation) ----------

def _clean_wikisource_translation(raw: str) -> str:
    body: list[str] = []
    for line in raw.splitlines():
        s = line.strip()
        if not s:
            continue
        if s.startswith(("{{", "}}", "|", "[[", "<!--", "==")):
            continue
        body.append(s)
    text = "\n".join(body)
    text = re.sub(r"\{\{[^}]+\}\}", "", text)
    text = re.sub(r"\[\[(?:[^|\]]+\|)?([^\]]+)\]\]", r"\1", text)
    text = re.sub(r"[ \t]+", " ", text)
    text = re.sub(r" *\n *", "\n", text).strip()
    if not text:
        raise RuntimeError("SUTRA-EN: no translation body parsed")
    return text


def build_sutra_en() -> None:
    print("[sutra-en] downloading...")
    raw = _fetch(SUTRA_EN_RAW_URL)
    text = _clean_wikisource_translation(raw)
    verses: list[dict] = [{
        "id": "sutra.heart-sutra-en.1",
        "tradition": "sutra",
        "lang": "en",
        "work": "heart-sutra-en",
        "canonical_ref": {"chapter": 1, "verse_start": 1},
        "display_ref": {"en": "Heart Sutra", "zh-Hans": "心经"},
        "text": text,
        "source": {
            "provider": "Wikisource",
            "license": "Creative Commons Attribution-ShareAlike",
            "attribution": "Shorter Prajnaparamita Hrdaya Sutra, Wikisource translation",
        },
        "checksum_sha256": _sha256(text),
        "inclusion_mode": "bundled",
        "sensitivity": "sacred_exact_quote",
    }]
    meta = {
        "tradition": "sutra",
        "work": "heart-sutra-en",
        "lang": "en",
        "provider": "Wikisource",
        "source_url": SUTRA_EN_RAW_URL,
        "license": "Creative Commons Attribution-ShareAlike",
        "attribution": "Shorter Prajnaparamita Hrdaya Sutra, Wikisource translation",
        "edition_id": "wikisource-heart-sutra-en",
        "inclusion_mode": "bundled",
        "sensitivity": "sacred_exact_quote",
        "note": "Bundled with source attribution; this is an English translation, not the Xuanzang Chinese text.",
    }
    _write_pack("heart-sutra-en", verses, meta)


# ---------- Quran translations ----------

def _parse_tanzil_translation(raw: str, code: str) -> list[tuple[int, int, str]]:
    verses: list[tuple[int, int, str]] = []
    for line in raw.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = line.split("|", 2)
        if len(parts) != 3:
            raise RuntimeError(f"QURAN {code}: malformed translation row")
        surah = int(parts[0])
        ayah = int(parts[1])
        text = parts[2].strip()
        if not text:
            raise RuntimeError(f"QURAN {code}: empty translation at {surah}:{ayah}")
        verses.append((surah, ayah, text))
    if len(verses) != 6236:
        raise RuntimeError(f"QURAN {code}: expected 6236 ayat, found {len(verses)}")
    return verses


def build_quran_translation(
    *,
    pack_name: str,
    work: str,
    lang: str,
    url: str,
    provider_name: str,
    attribution: str,
) -> None:
    print(f"[{pack_name}] downloading...")
    raw = _fetch(url)
    verses: list[dict] = []
    for surah, ayah, text in _parse_tanzil_translation(raw, work):
        vid = f"quran.{work}.{surah}.{ayah}"
        verses.append({
            "id": vid,
            "tradition": "quran",
            "lang": lang,
            "work": work,
            "canonical_ref": {"chapter": surah, "verse_start": ayah},
            "display_ref": {"en": f"Quran {surah}:{ayah}", "zh-Hans": f"古兰经 {surah}:{ayah}"},
            "text": text,
            "source": {
                "provider": "Tanzil Quran Translations",
                "license": "Tanzil translations terms: non-commercial use with attribution",
                "attribution": attribution,
            },
            "checksum_sha256": _sha256(text),
            "inclusion_mode": "bundled",
            "sensitivity": "sacred_exact_quote",
        })
    meta = {
        "tradition": "quran",
        "work": work,
        "lang": lang,
        "provider": "Tanzil Quran Translations",
        "source_url": url,
        "license": "Tanzil translations terms: non-commercial use with attribution",
        "attribution": attribution,
        "edition_id": provider_name,
        "inclusion_mode": "bundled",
        "sensitivity": "sacred_exact_quote",
        "note": "Bundled translation data retains Tanzil attribution and non-commercial translation terms.",
    }
    _write_pack(pack_name, verses, meta)


def build_quran_pickthall() -> None:
    build_quran_translation(
        pack_name="quran-pickthall",
        work="pickthall",
        lang="en",
        url=QURAN_PICKTHALL_URL,
        provider_name="tanzil-en.pickthall",
        attribution="Quran English translation by Mohammed Marmaduke William Pickthall, Tanzil",
    )


def build_quran_majian() -> None:
    build_quran_translation(
        pack_name="quran-majian",
        work="majian",
        lang="zh-Hans",
        url=QURAN_MAJIAN_URL,
        provider_name="tanzil-zh.jian",
        attribution="古兰经中文译本（马坚），Tanzil",
    )


# ---------- entrypoint ----------

def main() -> int:
    targets = sys.argv[1:] or ["kjv", "cuv-s", "dao", "dao-en", "sutra", "sutra-en", "quran-en", "quran-zh"]
    if "kjv" in targets:
        build_kjv()
    if "cuv-s" in targets:
        build_cuv_s()
    if "dao" in targets:
        build_dao()
    if "dao-en" in targets:
        build_dao_legge()
    if "sutra" in targets:
        build_sutra()
    if "sutra-en" in targets:
        build_sutra_en()
    if "quran-en" in targets:
        build_quran_pickthall()
    if "quran-zh" in targets:
        build_quran_majian()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
