"""Statistics module with two ambiguously-named identifiers.

The agent is asked to rename:
  * the function ``do`` (which computes a running mean+variance update)
  * the local variable ``tmp`` (which holds the current delta)

to something descriptive, without altering the public API:
``mean_variance(values) -> (mean, variance)``.
"""


def do(state, x):
    n, mean, m2 = state
    n += 1
    tmp = x - mean
    mean += tmp / n
    m2 += tmp * (x - mean)
    return n, mean, m2


def mean_variance(values):
    state = (0, 0.0, 0.0)
    for v in values:
        state = do(state, v)
    n, mean, m2 = state
    if n < 2:
        return mean, 0.0
    return mean, m2 / (n - 1)
