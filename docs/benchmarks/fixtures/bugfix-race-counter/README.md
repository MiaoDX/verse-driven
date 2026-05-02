# bugfix-race-counter (benchmark fixture)

Make ``Counter`` thread-safe. Concurrent ``increment`` and ``add``
calls from many threads must never lose updates. The public API
(``Counter``, ``increment``, ``add``, ``value``) must not change.

A ``threading.Lock`` around the read-modify-write is the canonical
fix; alternative approaches (``threading.RLock``, atomic primitives
from ``itertools.count`` semantics, etc.) are acceptable as long as
the tests pass.

Acceptance command: ``pytest -q``.
