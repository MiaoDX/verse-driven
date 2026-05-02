# feature-flatten-json (benchmark fixture)

Implement ``flatten_json(obj, sep='.') -> dict``: nested dicts become
dotted keys; lists become bracketed indices (``xs[0]``); scalars pass
through. Non-dict top-level argument raises ``TypeError``.

Acceptance command: ``pytest -q``.
