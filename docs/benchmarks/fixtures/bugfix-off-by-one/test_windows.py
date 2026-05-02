"""Acceptance tests for bugfix-off-by-one. The benchmark runner grades
a task by running pytest -q against this file after the agent edits
windows.py.
"""

import pytest

from windows import rolling_sum


def test_window_1_returns_full_length():
    assert rolling_sum([1, 2, 3, 4], 1) == [1, 2, 3, 4]


def test_window_2():
    assert rolling_sum([1, 2, 3, 4], 2) == [3, 5, 7]


def test_window_equals_input_length():
    assert rolling_sum([1, 2, 3, 4], 4) == [10]


def test_window_zero_raises():
    with pytest.raises(ValueError):
        rolling_sum([1, 2, 3], 0)


def test_empty_input_returns_empty():
    assert rolling_sum([], 3) == []
