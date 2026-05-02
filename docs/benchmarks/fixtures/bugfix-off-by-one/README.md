# bugfix-off-by-one (benchmark fixture)

Reference fixture for the issue #8 coding-quality benchmark task pack.
Each fixture in this directory follows the same convention so the
runner can drive every task uniformly.

## Convention

- `<fixture>/` — the working directory the agent operates in.
- One or more source files containing the seed code (here:
  `windows.py`).
- `test_<name>.py` — acceptance tests. The runner grades a task by
  running `pytest -q` after the agent finishes editing.
- `README.md` (this file) — what the agent is told to do, in the
  same form a real PR description might take.

The agent's prompt for this task is the `description` field in
`docs/benchmarks/tasks.json`. The fixture directory is mounted (or
copied) read-write before the run; tests run against the post-edit
state.

All ten fixtures referenced by `tasks.json` follow this same
convention; this directory is the original template.
