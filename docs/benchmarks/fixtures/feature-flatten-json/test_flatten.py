"""Acceptance tests for feature-flatten-json. Four nesting shapes
plus error cases.
"""

import pytest

from flatten import flatten_json


def test_flat_dict_passes_through():
    assert flatten_json({"a": 1, "b": 2}) == {"a": 1, "b": 2}


def test_nested_dicts_dotted():
    assert flatten_json({"a": {"b": {"c": 9}}}) == {"a.b.c": 9}


def test_list_uses_bracket_indices():
    out = flatten_json({"xs": [10, 20, 30]})
    assert out == {"xs[0]": 10, "xs[1]": 20, "xs[2]": 30}


def test_mixed_dict_and_list():
    out = flatten_json({"a": {"b": [{"c": 1}, {"d": 2}]}})
    assert out == {"a.b[0].c": 1, "a.b[1].d": 2}


def test_custom_separator():
    assert flatten_json({"a": {"b": 1}}, sep="/") == {"a/b": 1}


def test_empty_dict_yields_empty():
    assert flatten_json({}) == {}


def test_scalar_top_level_raises():
    with pytest.raises(TypeError):
        flatten_json(5)


def test_none_value_preserved():
    assert flatten_json({"a": None}) == {"a": None}
