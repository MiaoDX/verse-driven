# refactor-config-merge (benchmark fixture)

Replace the nested if-cascade in ``merged_config`` with a single
``deep_merge`` helper that recursively merges two dicts (the right
operand wins). ``merged_config`` should reduce to chained
``deep_merge`` calls while preserving precedence:

    defaults < user < override

Acceptance command: ``pytest -q``.
