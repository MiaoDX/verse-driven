"""Thread-shared counter with a concurrency bug.

The current ``Counter.add`` does an explicit read-modify-write with a
small ``time.sleep(0)`` between the read and the write — a classic
non-atomic pattern that races even under the CPython GIL because the
sleep is a guaranteed thread-yield point. Many concurrent ``add``
calls will therefore lose updates.

The agent must make the counter thread-safe (e.g. with a
``threading.Lock`` around the read-modify-write) without changing the
public API.

Public API to preserve:

  Counter()             - construct a new counter at zero
  Counter().increment() - add one
  Counter().add(n)      - add n
  Counter().value       - read the current count
"""

import time


class Counter:
    def __init__(self):
        self._n = 0

    def increment(self):
        self.add(1)

    def add(self, n):
        cur = self._n
        time.sleep(0)  # force a GIL release between read and write
        self._n = cur + n

    @property
    def value(self):
        return self._n
