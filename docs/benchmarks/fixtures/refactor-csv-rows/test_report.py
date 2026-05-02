"""Acceptance tests for refactor-csv-rows.

Passes when the readers stop accessing CSV cells by integer index and
expose them by header name. We assert this both behaviorally (a Row's
fields are reachable by attribute or key) and structurally (no
``row[0]`` / ``row[1]`` / ``row[2]`` substrings remain in report.py).
"""

import re
from pathlib import Path

import report
from report import read_report, total_amount, unique_users


SAMPLE_CSV = (
    "id,user,amount\n"
    "1,alice,12.50\n"
    "2,bob,7.00\n"
    "3,alice,3.25\n"
)


def _write_sample(tmp_path):
    p = tmp_path / "sample.csv"
    p.write_text(SAMPLE_CSV, encoding="utf-8")
    return p


def test_total_amount(tmp_path):
    rows = read_report(_write_sample(tmp_path))
    assert abs(total_amount(rows) - 22.75) < 1e-9


def test_unique_users(tmp_path):
    rows = read_report(_write_sample(tmp_path))
    assert unique_users(rows) == {"alice", "bob"}


def test_rows_expose_named_fields(tmp_path):
    rows = read_report(_write_sample(tmp_path))
    assert rows, "read_report returned no rows"
    first = rows[0]
    user = getattr(first, "user", None)
    if user is None and hasattr(first, "__getitem__") and not isinstance(first, list):
        user = first["user"]
    assert user == "alice"


def test_no_positional_indexing_left_in_source():
    src = Path(report.__file__).read_text(encoding="utf-8")
    assert not re.search(r"row\[\s*[0-9]+\s*\]", src), (
        "report.py still indexes rows by integer position"
    )
