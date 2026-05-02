"""Acceptance tests for refactor-config-merge.

Passes when:
  1. ``merged_config`` still preserves precedence (override > user > defaults).
  2. A ``deep_merge`` helper exists at module level.
  3. ``merged_config`` delegates to ``deep_merge`` (we monkey-patch and
     observe the substitution flows through).
"""

import config
from config import merged_config


def test_override_wins():
    out = merged_config({"a": 1}, {"a": 2}, {"a": 3})
    assert out == {"a": 3}


def test_user_beats_defaults_when_override_absent():
    out = merged_config({"a": 1}, {"a": 2}, {})
    assert out == {"a": 2}


def test_defaults_used_when_others_absent():
    out = merged_config({"a": 1}, {}, {})
    assert out == {"a": 1}


def test_nested_dict_merge():
    out = merged_config(
        {"db": {"host": "localhost", "port": 5432, "user": "x"}},
        {"db": {"port": 6000}},
        {"db": {"user": "admin"}},
    )
    assert out == {"db": {"host": "localhost", "port": 6000, "user": "admin"}}


def test_non_dict_leaf_replaces_dict():
    out = merged_config({"x": {"a": 1}}, {}, {"x": "scalar"})
    assert out == {"x": "scalar"}


def test_deep_merge_helper_exists():
    assert hasattr(config, "deep_merge"), "deep_merge helper not introduced"


def test_merged_config_delegates_to_deep_merge(monkeypatch):
    sentinel = {"sentinel": True}
    monkeypatch.setattr(config, "deep_merge", lambda *args, **kwargs: sentinel)
    out = merged_config({"a": 1}, {"b": 2}, {"c": 3})
    assert out == sentinel
