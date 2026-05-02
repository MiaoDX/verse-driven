"""Rolling-window aggregation. There is an off-by-one bug for window=1.

This is the seed file the agent edits during a benchmark run.
"""


def rolling_sum(xs, window):
    if window < 1:
        raise ValueError("window must be >= 1")
    out = []
    for i in range(window, len(xs)):
        out.append(sum(xs[i - window:i]))
    return out
