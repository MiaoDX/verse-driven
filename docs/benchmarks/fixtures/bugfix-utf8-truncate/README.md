# bugfix-utf8-truncate (benchmark fixture)

Fix ``truncate_bytes`` so it never returns bytes that split a UTF-8
codepoint mid-sequence. The output must:

  * have length ``<= n``,
  * decode cleanly as UTF-8,
  * be the longest valid prefix of the input that fits.

Acceptance command: ``pytest -q``.
