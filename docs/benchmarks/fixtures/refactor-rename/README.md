# refactor-rename (benchmark fixture)

Rename the ambiguously-named module-level function ``do`` and the
local variable ``tmp`` to descriptive names. The public API
``mean_variance(values) -> (mean, variance)`` must keep its name and
behavior.

Acceptance command: ``pytest -q``.
