"""Three-layer config merge implemented as a nested if-cascade.

The agent is asked to replace the cascade with a single ``deep_merge``
helper while preserving precedence semantics:

  defaults < user < override

i.e. ``override`` wins, then ``user``, then ``defaults``. Nested dicts
merge recursively; non-dict leaves replace.
"""


def merged_config(defaults, user, override):
    out = {}
    for k in set(defaults) | set(user) | set(override):
        if k in override:
            if (
                isinstance(override[k], dict)
                and k in user
                and isinstance(user[k], dict)
                and k in defaults
                and isinstance(defaults[k], dict)
            ):
                inner = {}
                for ik in set(defaults[k]) | set(user[k]) | set(override[k]):
                    if ik in override[k]:
                        inner[ik] = override[k][ik]
                    elif ik in user[k]:
                        inner[ik] = user[k][ik]
                    else:
                        inner[ik] = defaults[k][ik]
                out[k] = inner
            elif (
                isinstance(override[k], dict)
                and k in user
                and isinstance(user[k], dict)
            ):
                inner = {}
                for ik in set(user[k]) | set(override[k]):
                    if ik in override[k]:
                        inner[ik] = override[k][ik]
                    else:
                        inner[ik] = user[k][ik]
                out[k] = inner
            else:
                out[k] = override[k]
        elif k in user:
            if (
                isinstance(user[k], dict)
                and k in defaults
                and isinstance(defaults[k], dict)
            ):
                inner = {}
                for ik in set(defaults[k]) | set(user[k]):
                    if ik in user[k]:
                        inner[ik] = user[k][ik]
                    else:
                        inner[ik] = defaults[k][ik]
                out[k] = inner
            else:
                out[k] = user[k]
        else:
            out[k] = defaults[k]
    return out
