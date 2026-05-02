# feature-cli-flag (benchmark fixture)

Add an opt-in ``--json`` flag to ``tool.py`` that emits the same
summary as a single JSON object on stdout. The default human-readable
output must be unchanged.

Add a ``render_json(summary) -> str`` helper that mirrors
``render_text``.

Acceptance command: ``pytest -q``.
