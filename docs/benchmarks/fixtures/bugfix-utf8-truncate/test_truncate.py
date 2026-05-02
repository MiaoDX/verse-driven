"""Acceptance tests for bugfix-utf8-truncate.

Passes when ``truncate_bytes`` returns valid UTF-8 of length <= n that
is the longest valid prefix of the input fitting in n bytes.
"""

import pytest

from truncate import truncate_bytes


def test_ascii_unchanged_when_budget_sufficient():
    assert truncate_bytes("hello", 10) == b"hello"


def test_ascii_clipped_at_byte_budget():
    assert truncate_bytes("hello", 3) == b"hel"


def test_multibyte_does_not_split_codepoint():
    # "你" encodes to 3 bytes in UTF-8.
    out = truncate_bytes("你好", 4)
    assert len(out) <= 4
    out.decode("utf-8")  # must not raise
    assert out == "你".encode("utf-8")


def test_emoji_does_not_split_codepoint():
    # Smiling face emoji is 4 bytes in UTF-8.
    out = truncate_bytes("ab😀cd", 4)
    out.decode("utf-8")
    assert out == b"ab"


def test_zero_budget_returns_empty():
    assert truncate_bytes("hello", 0) == b""


def test_negative_budget_raises():
    with pytest.raises(ValueError):
        truncate_bytes("hi", -1)


def test_full_string_when_budget_huge():
    assert truncate_bytes("héllo", 100) == "héllo".encode("utf-8")
