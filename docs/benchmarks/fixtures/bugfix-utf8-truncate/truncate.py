"""Truncate a string so its UTF-8 encoding fits a byte budget.

There is a bug: the current implementation slices the encoded bytes at
``n`` directly, which can land in the middle of a multi-byte UTF-8
sequence and produce invalid UTF-8 (or, when decoded with errors set,
a "replacement-character" suffix).

The agent must fix ``truncate_bytes`` so the returned bytes:
  * have length ``<= n``,
  * decode cleanly as UTF-8,
  * are the longest valid prefix of the input that fits.
"""


def truncate_bytes(s: str, n: int) -> bytes:
    if n < 0:
        raise ValueError("n must be >= 0")
    encoded = s.encode("utf-8")
    return encoded[:n]
