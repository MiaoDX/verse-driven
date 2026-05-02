"""flatten_json: turn nested JSON-shaped data into a flat dict.

The agent is asked to implement ``flatten_json(obj, sep='.') -> dict``:

  * Nested dicts: keys are joined with ``sep``.
  * Lists: indices appear as ``[i]`` segments (no separator before
    the bracket).
  * Scalars (str, int, float, bool, None) are leaves.
  * The top-level argument must be a dict; raise ``TypeError`` for
    other shapes.

Examples:

    flatten_json({"a": {"b": 1}})           -> {"a.b": 1}
    flatten_json({"xs": [10, 20]})          -> {"xs[0]": 10, "xs[1]": 20}
    flatten_json({"a": {"b": [{"c": 1}]}})  -> {"a.b[0].c": 1}
    flatten_json({}, sep='/')               -> {}
"""


def flatten_json(obj, sep="."):
    raise NotImplementedError
