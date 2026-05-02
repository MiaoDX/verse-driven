# feature-rate-limit (benchmark fixture)

Implement ``TokenBucket(rate, burst)`` and a ``rate_limit(rate, burst)``
decorator in ``limiter.py``. The bucket refills at ``rate`` tokens per
second with capacity ``burst``; ``acquire(n)`` blocks, ``try_acquire(n)``
returns immediately. The decorator wraps a function so each call
``acquire(1)`` from a per-decorator bucket.

Acceptance command: ``pytest -q``.
