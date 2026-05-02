"""Acceptance tests for bugfix-race-counter.

Passes when concurrent increments from many threads land the right
total. The single-threaded contract still holds.
"""

import threading

from counter import Counter


def test_single_threaded_increment():
    c = Counter()
    for _ in range(100):
        c.increment()
    assert c.value == 100


def test_single_threaded_add():
    c = Counter()
    c.add(7)
    c.add(3)
    assert c.value == 10


def test_concurrent_increment_no_lost_updates():
    c = Counter()
    n_threads = 16
    per_thread = 5_000

    def worker():
        for _ in range(per_thread):
            c.increment()

    threads = [threading.Thread(target=worker) for _ in range(n_threads)]
    for t in threads:
        t.start()
    for t in threads:
        t.join()
    assert c.value == n_threads * per_thread


def test_concurrent_add_no_lost_updates():
    c = Counter()
    n_threads = 8
    per_thread = 1_000

    def worker():
        for _ in range(per_thread):
            c.add(3)

    threads = [threading.Thread(target=worker) for _ in range(n_threads)]
    for t in threads:
        t.start()
    for t in threads:
        t.join()
    assert c.value == n_threads * per_thread * 3
