"""Token-bucket rate limiter — implementation TODO.

The agent is asked to implement:

  * ``TokenBucket(rate, burst)`` — a token bucket that refills at
    ``rate`` tokens per second with capacity ``burst``. Method
    ``acquire(n=1)`` blocks until n tokens are available, then
    consumes them. Method ``try_acquire(n=1)`` returns True/False
    immediately.
  * ``rate_limit(rate, burst)`` — a decorator that throttles a
    function with a per-decorator ``TokenBucket``.

Tests cover both steady-state (rate caps long runs) and burst (the
first ``burst`` calls are immediate).
"""

import time


class TokenBucket:
    def __init__(self, rate, burst):
        raise NotImplementedError

    def acquire(self, n=1):
        raise NotImplementedError

    def try_acquire(self, n=1):
        raise NotImplementedError


def rate_limit(rate, burst):
    """Return a decorator that wraps a function with a TokenBucket
    of (rate, burst). The wrapped function blocks via ``acquire(1)``
    before invoking the underlying callable.
    """
    raise NotImplementedError
