# Bundled Text Sources

This file records the upstream sources, terms, and attribution metadata for
the scripture packs bundled into `scripture-mcp`. It intentionally contains no
scripture passage bodies.

Each pack also stores the same source metadata in
`internal/packs/<pack>/metadata.json`, and every bundled row stores a
SHA-256 checksum over its text bytes.

## Source Matrix

| Pack | Tradition | Work | Language | Source | Terms | Attribution |
|---|---|---|---|---|---|---|
| `bible-kjv` | Bible | KJV | English | [Project Gutenberg eBook #10](https://www.gutenberg.org/cache/epub/10/pg10.txt) | Public domain (United States) | King James Version of the Bible, Project Gutenberg eBook #10 |
| `bible-cuv-s` | Bible | CUV-S | Simplified Chinese | [open-bibles USFX](https://raw.githubusercontent.com/seven1m/open-bibles/master/chi-cuv-simp.usfx.xml) | Public domain | Chinese Union Version (Simplified), open-bibles USFX |
| `dao-de-jing` | Dao | daodejing | Simplified Chinese | [Project Gutenberg eBook #7337](https://www.gutenberg.org/cache/epub/7337/pg7337.txt) | Public domain | `道德經`, Project Gutenberg eBook #7337, produced by Ching-yi Chen |
| `dao-legge` | Dao | legge | English | [Internet Classics Archive](https://classics.mit.edu/Lao/taote.mb.txt) | Public domain source text | Tao Te Ching, translated by James Legge (1891), Internet Classics Archive text |
| `heart-sutra` | Sutra | heart-sutra | Simplified Chinese | [CBETA XML P5 T0251](https://cbetaonline.dila.edu.tw/zh/T0251_001) | Ancient source text; CBETA digital edition terms apply | `般若波罗蜜多心经`, translated by Xuanzang, CBETA XML P5 T0251 |
| `heart-sutra-en` | Sutra | heart-sutra-en | English | [Wikisource raw page](https://en.wikisource.org/w/index.php?title=Translation:Shorter_Praj%C3%B1%C4%81p%C4%81ramit%C4%81_H%E1%B9%9Bdaya_S%C5%ABtra&action=raw) | Creative Commons Attribution-ShareAlike | Shorter Prajnaparamita Hrdaya Sutra, Wikisource translation |
| `quran-pickthall` | Quran | pickthall | English | [Tanzil `en.pickthall`](https://tanzil.net/trans/en.pickthall) | Tanzil translation terms: non-commercial use with attribution | Quran English translation by Mohammed Marmaduke William Pickthall, Tanzil |
| `quran-majian` | Quran | majian | Simplified Chinese | [Tanzil `zh.jian`](https://tanzil.net/trans/zh.jian) | Tanzil translation terms: non-commercial use with attribution | Quran Chinese translation by Ma Jian, Tanzil |

## Transform Notes

| Pack | Transform |
|---|---|
| `dao-de-jing` | Traditional Chinese source normalized to Simplified Chinese with OpenCC `t2s`. |
| `heart-sutra` | CBETA XML P5 body extraction, then OpenCC `t2s`. |
| `heart-sutra-en` | Wikisource raw wiki markup cleaned to the translation body. |
| `bible-cuv-s` | USFX verse markers parsed into compact JSONL rows. |
| `quran-pickthall` / `quran-majian` | Tanzil pipe-delimited translation rows parsed into compact JSONL rows. |

## Cautions

- Quran translation packs are not public-domain packs; retain Tanzil
  attribution and non-commercial translation terms in releases.
- CBETA-derived text should retain the CBETA attribution and terms note.
- Do not paste passage bodies into docs, logs, PR descriptions, or chat output;
  cite pack IDs, references, checksums, and source metadata instead.
