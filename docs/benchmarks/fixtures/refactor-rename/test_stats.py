"""Acceptance tests for refactor-rename.

Passes when:
  1. ``mean_variance`` (the public API) still returns correct values.
  2. The ambiguous identifier ``do`` is no longer a module-level name.
  3. The literal token ``tmp`` does not appear in the source.
"""

import re
from pathlib import Path

import stats
from stats import mean_variance


def test_mean_variance_basic():
    mean, variance = mean_variance([2, 4, 4, 4, 5, 5, 7, 9])
    assert abs(mean - 5.0) < 1e-9
    assert abs(variance - 32.0 / 7.0) < 1e-9


def test_mean_variance_singleton():
    mean, variance = mean_variance([42])
    assert mean == 42
    assert variance == 0.0


def test_do_renamed():
    assert not hasattr(stats, "do"), "module-level 'do' was not renamed"


def test_tmp_renamed():
    src = Path(stats.__file__).read_text(encoding="utf-8")
    assert not re.search(r"\btmp\b", src), "local 'tmp' was not renamed"
