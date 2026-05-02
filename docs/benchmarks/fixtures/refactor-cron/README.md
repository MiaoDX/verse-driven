# refactor-cron (benchmark fixture)

Extract the cron-string parsing inlined inside ``schedule_job`` into a
top-level helper ``parse_cron(spec: str) -> CronSpec``. ``schedule_job``
must keep the same external behavior and now call ``parse_cron``.

The agent's prompt is the ``description`` field for ``refactor-cron``
in ``docs/benchmarks/tasks.json``. The acceptance command is
``pytest -q``.
