# refactor-csv-rows (benchmark fixture)

Switch ``report.py`` from positional row indexing (``row[0]``,
``row[1]``, ``row[2]``) to a namedtuple or dataclass keyed by header
name. ``read_report`` should return rows whose fields are reachable by
name (``row.user``, ``row.amount``); ``total_amount`` and
``unique_users`` must keep working.

Acceptance command: ``pytest -q``.
