"""Acceptance tests for feature-rate-limit.

Passes when ``TokenBucket`` and the ``rate_limit`` decorator behave
correctly in the burst and steady-state regimes.
"""

import time

from limiter import TokenBucket, rate_limit


def test_burst_capacity_is_immediate():
    bucket = TokenBucket(rate=1, burst=5)
    started = time.perf_counter()
    for _ in range(5):
        assert bucket.try_acquire(1) is True
    elapsed = time.perf_counter() - started
    assert elapsed < 0.05


def test_try_acquire_returns_false_when_empty():
    bucket = TokenBucket(rate=0.5, burst=2)
    assert bucket.try_acquire(1) is True
    assert bucket.try_acquire(1) is True
    assert bucket.try_acquire(1) is False


def test_acquire_blocks_until_refill():
    bucket = TokenBucket(rate=20, burst=1)
    bucket.acquire(1)  # drain
    started = time.perf_counter()
    bucket.acquire(1)  # must wait ~1/20 = 50ms
    elapsed = time.perf_counter() - started
    assert 0.03 <= elapsed < 0.5


def test_rate_limit_decorator_caps_throughput():
    calls = []

    @rate_limit(rate=20, burst=1)
    def f():
        calls.append(time.perf_counter())

    started = time.perf_counter()
    for _ in range(4):
        f()
    elapsed = time.perf_counter() - started
    # 4 calls at 20/s with burst=1 must take at least ~3*(1/20) = 150ms.
    assert elapsed >= 0.12
    assert len(calls) == 4
