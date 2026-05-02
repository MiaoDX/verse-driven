"""Job scheduler with inline cron-string parsing.

The agent is asked to extract the cron-parsing logic into a top-level
helper named ``parse_cron(spec: str) -> CronSpec`` without changing the
external behavior of ``schedule_job``.
"""

from dataclasses import dataclass


@dataclass
class CronSpec:
    minute: int
    hour: int
    day: int
    month: int
    weekday: int


def schedule_job(name, cron_spec):
    parts = cron_spec.strip().split()
    if len(parts) != 5:
        raise ValueError("cron spec must have 5 fields")
    fields = []
    for p in parts:
        if p == "*":
            fields.append(-1)
        else:
            try:
                fields.append(int(p))
            except ValueError as e:
                raise ValueError(f"unsupported cron field {p!r}") from e
    minute, hour, day, month, weekday = fields
    return {
        "name": name,
        "minute": minute,
        "hour": hour,
        "day": day,
        "month": month,
        "weekday": weekday,
    }
