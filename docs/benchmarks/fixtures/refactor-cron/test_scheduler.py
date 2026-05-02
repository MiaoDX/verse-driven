"""Acceptance tests for refactor-cron.

The fixture passes when:
  1. ``schedule_job`` still produces the same dict shape it did before.
  2. A top-level ``parse_cron(spec) -> CronSpec`` helper exists and
     returns a populated ``CronSpec``.
  3. ``schedule_job`` actually delegates to ``parse_cron`` (i.e. the
     parsing is not duplicated). We assert this by monkey-patching
     ``parse_cron`` and observing that ``schedule_job`` picks up the
     replacement.
"""

import inspect

import pytest

import scheduler
from scheduler import CronSpec, schedule_job


def test_schedule_job_shape_preserved():
    job = schedule_job("nightly", "0 3 * * *")
    assert job["name"] == "nightly"
    assert job["minute"] == 0
    assert job["hour"] == 3
    assert job["day"] == -1
    assert job["month"] == -1
    assert job["weekday"] == -1


def test_invalid_field_count_raises():
    with pytest.raises(ValueError):
        schedule_job("bad", "0 3 *")


def test_parse_cron_helper_exists():
    assert hasattr(scheduler, "parse_cron"), "parse_cron helper not extracted"
    sig = inspect.signature(scheduler.parse_cron)
    assert len(sig.parameters) == 1


def test_parse_cron_returns_cronspec():
    spec = scheduler.parse_cron("15 4 1 * *")
    assert isinstance(spec, CronSpec)
    assert spec.minute == 15
    assert spec.hour == 4
    assert spec.day == 1
    assert spec.month == -1
    assert spec.weekday == -1


def test_schedule_job_delegates_to_parse_cron(monkeypatch):
    sentinel = CronSpec(minute=99, hour=99, day=99, month=99, weekday=99)
    monkeypatch.setattr(scheduler, "parse_cron", lambda spec: sentinel)
    job = scheduler.schedule_job("x", "0 0 0 0 0")
    assert job["minute"] == 99
    assert job["hour"] == 99
