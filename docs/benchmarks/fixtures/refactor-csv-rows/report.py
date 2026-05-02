"""CSV report reader using positional indexing.

The agent is asked to switch from ``row[0]`` / ``row[1]`` index access
to a namedtuple (or dataclass) keyed by header name. ``read_report``
must keep its return type — a list of ``Row`` (namedtuple) — and the
two accessor functions ``total_amount`` and ``unique_users`` must
continue to work over an iterable of rows.
"""

import csv


def read_report(path):
    rows = []
    with open(path, newline="", encoding="utf-8") as f:
        reader = csv.reader(f)
        next(reader)  # skip header
        for row in reader:
            rows.append(row)
    return rows


def total_amount(rows):
    total = 0.0
    for row in rows:
        total += float(row[2])
    return total


def unique_users(rows):
    return {row[1] for row in rows}
