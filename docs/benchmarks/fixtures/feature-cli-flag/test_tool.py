"""Acceptance tests for feature-cli-flag.

Passes when:
  1. The default output (no --json) is unchanged.
  2. ``--json`` emits a single JSON object with ``total``, ``count``,
     ``top`` fields and nothing extra to stdout.
  3. A ``render_json(summary) -> str`` helper exists.
"""

import json

import tool
from tool import main, render_text, summarize


def test_summarize():
    summary = summarize([("a", 1), ("b", 5), ("c", 2)])
    assert summary == {"total": 8, "count": 3, "top": "b"}


def test_default_output_unchanged(capsys):
    rc = main(["apples=3", "bananas=7"])
    out = capsys.readouterr().out
    assert rc == 0
    assert out == render_text(summarize([("apples", 3), ("bananas", 7)]))


def test_json_flag_emits_single_object(capsys):
    rc = main(["--json", "apples=3", "bananas=7"])
    out = capsys.readouterr().out
    assert rc == 0
    body = json.loads(out)
    assert body == {"total": 10, "count": 2, "top": "bananas"}


def test_render_json_helper_exists():
    assert hasattr(tool, "render_json"), "render_json helper not added"
    summary = summarize([("a", 1)])
    body = json.loads(tool.render_json(summary))
    assert body == summary
