package resolver

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type bibleAlias struct {
	alias     string
	canonical string
}

type quranAlias struct {
	alias  string
	number int
}

var (
	bibleAliasIndex []bibleAlias
	quranAliasIndex []quranAlias

	daoAliases       = []string{"tao te ching", "dao de jing", "daodejing", "道德经", "dao", "tao"}
	sutraAliases     = []string{"般若波罗蜜多心经", "heart sutra", "sutra heart", "心经"}
	quranTextAliases = []string{"古兰经", "quran", "qur'an", "koran"}
	surahKeywords    = []string{"surah", "sura"}

	reChapterVerse = regexp.MustCompile(`^\s*(\d+)\s*[:：]\s*(\d+)\s*(?:[\-–]\s*(\d+))?\s*$`)
	reVerse        = regexp.MustCompile(`^\s*(\d+)\s*(?:[\-–]\s*(\d+))?\s*$`)
)

func init() {
	for _, b := range bibleBooks {
		for _, a := range b.aliases {
			bibleAliasIndex = append(bibleAliasIndex, bibleAlias{alias: a, canonical: b.canonical})
		}
	}
	sort.SliceStable(bibleAliasIndex, func(i, j int) bool {
		return len(bibleAliasIndex[i].alias) > len(bibleAliasIndex[j].alias)
	})
	for _, q := range quranSurahs {
		for _, a := range q.aliases {
			quranAliasIndex = append(quranAliasIndex, quranAlias{alias: a, number: q.number})
		}
	}
	sort.SliceStable(quranAliasIndex, func(i, j int) bool {
		return len(quranAliasIndex[i].alias) > len(quranAliasIndex[j].alias)
	})
	sortByLenDesc(daoAliases)
	sortByLenDesc(sutraAliases)
	sortByLenDesc(quranTextAliases)
	sortByLenDesc(surahKeywords)
}

func sortByLenDesc(s []string) {
	sort.SliceStable(s, func(i, j int) bool { return len(s[i]) > len(s[j]) })
}

// Resolve parses a free-form scripture reference into a canonical Reference.
// Errors wrap one of the package sentinels (ErrEmpty, ErrUnrecognized,
// ErrInvalidNumber, ErrInvalidRange) or are *AmbiguousError when the input
// cannot be uniquely classified.
func Resolve(input string) (Reference, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return Reference{}, ErrEmpty
	}
	norm := strings.ToLower(trimmed)

	if rest, ok := matchAlias(norm, sutraAliases); ok {
		if strings.TrimSpace(rest) != "" {
			return Reference{}, fmt.Errorf("%w: heart sutra takes no chapter/verse", ErrUnrecognized)
		}
		return Reference{Tradition: TraditionSutra, Work: WorkHeartSutra}, nil
	}

	if rest, ok := matchAlias(norm, daoAliases); ok {
		return parseDao(rest)
	}

	if rest, ok := matchAlias(norm, quranTextAliases); ok {
		return parseQuranNumeric(rest)
	}
	if rest, ok := matchAlias(norm, surahKeywords); ok {
		return parseQuranSurahByName(rest)
	}

	for _, e := range bibleAliasIndex {
		if rest, ok := tryMatchPrefix(norm, e.alias); ok {
			return parseBible(e.canonical, rest)
		}
	}

	if c, v, _, ok := parseChapterVerse(norm); ok {
		return Reference{}, &AmbiguousError{
			Input: input,
			Candidates: []Reference{
				{Tradition: TraditionQuran, Work: WorkQuran, Chapter: c, VerseStart: v},
				{Tradition: TraditionBible, Work: WorkKJV, Chapter: c, VerseStart: v},
			},
		}
	}

	return Reference{}, fmt.Errorf("%w: %q", ErrUnrecognized, input)
}

func parseDao(rest string) (Reference, error) {
	rest = strings.TrimSpace(rest)
	if r, ok := tryMatchPrefix(rest, "chapter"); ok {
		rest = strings.TrimSpace(r)
	}
	rest = strings.TrimPrefix(rest, "第")
	rest = strings.TrimSuffix(rest, "章")
	rest = strings.TrimSpace(rest)
	n, ok := parseNumber(rest)
	if !ok || n < 1 {
		return Reference{}, fmt.Errorf("%w: dao chapter %q", ErrInvalidNumber, rest)
	}
	return Reference{Tradition: TraditionDao, Work: WorkDaodejing, Chapter: n}, nil
}

func parseQuranNumeric(rest string) (Reference, error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return Reference{}, fmt.Errorf("%w: quran reference missing surah/verse", ErrUnrecognized)
	}
	if c, v, ve, ok := parseChapterVerse(rest); ok {
		if c < 1 || v < 1 {
			return Reference{}, fmt.Errorf("%w: surah and verse must be >= 1", ErrInvalidNumber)
		}
		if ve != 0 && ve < v {
			return Reference{}, ErrInvalidRange
		}
		return Reference{Tradition: TraditionQuran, Work: WorkQuran, Chapter: c, VerseStart: v, VerseEnd: ve}, nil
	}
	if n, ok := parseNumber(rest); ok && n >= 1 {
		return Reference{Tradition: TraditionQuran, Work: WorkQuran, Chapter: n}, nil
	}
	return Reference{}, fmt.Errorf("%w: quran reference %q", ErrInvalidNumber, rest)
}

func parseQuranSurahByName(rest string) (Reference, error) {
	rest = strings.TrimSpace(rest)
	for _, e := range quranAliasIndex {
		r, ok := tryMatchPrefix(rest, e.alias)
		if !ok {
			continue
		}
		r = strings.TrimSpace(r)
		ref := Reference{Tradition: TraditionQuran, Work: WorkQuran, Chapter: e.number}
		if r == "" {
			return ref, nil
		}
		v, ve, ok := parseVerseOnly(r)
		if !ok || v < 1 {
			return Reference{}, fmt.Errorf("%w: invalid verse for surah: %q", ErrInvalidNumber, r)
		}
		if ve != 0 && ve < v {
			return Reference{}, ErrInvalidRange
		}
		ref.VerseStart = v
		ref.VerseEnd = ve
		return ref, nil
	}
	return Reference{}, fmt.Errorf("%w: unknown surah %q", ErrUnrecognized, rest)
}

func parseBible(canonical, rest string) (Reference, error) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return Reference{}, fmt.Errorf("%w: %s reference missing chapter", ErrUnrecognized, canonical)
	}
	if c, v, ve, ok := parseChapterVerse(rest); ok {
		if c < 1 || v < 1 {
			return Reference{}, fmt.Errorf("%w: chapter and verse must be >= 1", ErrInvalidNumber)
		}
		if ve != 0 && ve < v {
			return Reference{}, ErrInvalidRange
		}
		return Reference{
			Tradition:  TraditionBible,
			Work:       WorkKJV,
			Book:       canonical,
			Chapter:    c,
			VerseStart: v,
			VerseEnd:   ve,
		}, nil
	}
	if n, ok := parseNumber(rest); ok && n >= 1 {
		return Reference{
			Tradition: TraditionBible,
			Work:      WorkKJV,
			Book:      canonical,
			Chapter:   n,
		}, nil
	}
	return Reference{}, fmt.Errorf("%w: invalid bible reference %q", ErrInvalidNumber, rest)
}

func matchAlias(input string, aliases []string) (string, bool) {
	for _, a := range aliases {
		if rest, ok := tryMatchPrefix(input, a); ok {
			return rest, true
		}
	}
	return "", false
}

// tryMatchPrefix returns input[len(alias):] if input starts with alias and
// the next rune (if any) is not an ASCII letter. This prevents "jo" from
// matching "joel" while still allowing CJK runs like "道德经第十一章" — for
// CJK, longest-alias-first sorting handles the ambiguity instead.
func tryMatchPrefix(input, alias string) (string, bool) {
	if alias == "" || len(input) < len(alias) {
		return "", false
	}
	if input[:len(alias)] != alias {
		return "", false
	}
	rest := input[len(alias):]
	if rest == "" {
		return rest, true
	}
	r, _ := utf8.DecodeRuneInString(rest)
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
		return "", false
	}
	return rest, true
}

func parseChapterVerse(s string) (chapter, verse, verseEnd int, ok bool) {
	m := reChapterVerse.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, 0, false
	}
	chapter, _ = strconv.Atoi(m[1])
	verse, _ = strconv.Atoi(m[2])
	if m[3] != "" {
		verseEnd, _ = strconv.Atoi(m[3])
	}
	return chapter, verse, verseEnd, true
}

func parseVerseOnly(s string) (verse, verseEnd int, ok bool) {
	m := reVerse.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	verse, _ = strconv.Atoi(m[1])
	if m[2] != "" {
		verseEnd, _ = strconv.Atoi(m[2])
	}
	return verse, verseEnd, true
}

func parseNumber(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n, true
	}
	return parseChineseNumeral(s)
}
