package resolver

import (
	"errors"
	"testing"
)

func TestResolve(t *testing.T) {
	type want struct {
		tradition  string
		work       string
		book       string
		chapter    int
		verseStart int
		verseEnd   int
	}

	cases := []struct {
		name  string
		input string
		want  want
	}{
		// Bible — English canonical
		{"bible/john_full", "John 3:16", want{TraditionBible, WorkKJV, "John", 3, 16, 0}},
		{"bible/john_lower", "john 3:16", want{TraditionBible, WorkKJV, "John", 3, 16, 0}},
		{"bible/john_upper", "JOHN 3:16", want{TraditionBible, WorkKJV, "John", 3, 16, 0}},
		{"bible/john_abbrev_jn", "Jn 3:16", want{TraditionBible, WorkKJV, "John", 3, 16, 0}},
		{"bible/john_abbrev_joh", "Joh 3:16", want{TraditionBible, WorkKJV, "John", 3, 16, 0}},
		{"bible/genesis", "Genesis 1:1", want{TraditionBible, WorkKJV, "Genesis", 1, 1, 0}},
		{"bible/genesis_abbrev", "Gen 1:1", want{TraditionBible, WorkKJV, "Genesis", 1, 1, 0}},
		{"bible/exodus", "Exodus 20:3", want{TraditionBible, WorkKJV, "Exodus", 20, 3, 0}},
		{"bible/psalms", "Psalms 23:1", want{TraditionBible, WorkKJV, "Psalms", 23, 1, 0}},
		{"bible/psalms_chapter_only", "Psalms 23", want{TraditionBible, WorkKJV, "Psalms", 23, 0, 0}},
		{"bible/matthew_range", "Matthew 5:3-12", want{TraditionBible, WorkKJV, "Matthew", 5, 3, 12}},
		{"bible/romans", "Romans 8:28", want{TraditionBible, WorkKJV, "Romans", 8, 28, 0}},
		{"bible/revelation", "Revelation 22:21", want{TraditionBible, WorkKJV, "Revelation", 22, 21, 0}},
		{"bible/jonah_vs_jn", "Jonah 1:1", want{TraditionBible, WorkKJV, "Jonah", 1, 1, 0}},
		{"bible/joel_vs_john", "Joel 2:28", want{TraditionBible, WorkKJV, "Joel", 2, 28, 0}},

		// Bible — numbered books, English variants
		{"bible/1john_spaced", "1 John 3:16", want{TraditionBible, WorkKJV, "1 John", 3, 16, 0}},
		{"bible/1john_unspaced", "1john 3:16", want{TraditionBible, WorkKJV, "1 John", 3, 16, 0}},
		{"bible/1john_abbrev", "1 Jn 3:16", want{TraditionBible, WorkKJV, "1 John", 3, 16, 0}},
		{"bible/1john_roman", "I John 3:16", want{TraditionBible, WorkKJV, "1 John", 3, 16, 0}},
		{"bible/2john", "2 John 1:1", want{TraditionBible, WorkKJV, "2 John", 1, 1, 0}},
		{"bible/3john", "3 John 1:4", want{TraditionBible, WorkKJV, "3 John", 1, 4, 0}},
		{"bible/1corinthians", "1 Corinthians 13:4", want{TraditionBible, WorkKJV, "1 Corinthians", 13, 4, 0}},
		{"bible/2samuel_abbrev", "2 Sam 7:12", want{TraditionBible, WorkKJV, "2 Samuel", 7, 12, 0}},

		// Bible — Simplified Chinese
		{"bible/john_zh", "约翰福音 3:16", want{TraditionBible, WorkKJV, "John", 3, 16, 0}},
		{"bible/john_zh_no_space", "约翰福音3:16", want{TraditionBible, WorkKJV, "John", 3, 16, 0}},
		{"bible/john_zh_fullwidth_colon", "约翰福音 3：16", want{TraditionBible, WorkKJV, "John", 3, 16, 0}},
		{"bible/1john_zh", "约翰一书 3:16", want{TraditionBible, WorkKJV, "1 John", 3, 16, 0}},
		{"bible/1john_zh_range", "约翰一书 3:16-17", want{TraditionBible, WorkKJV, "1 John", 3, 16, 17}},
		{"bible/2john_zh", "约翰二书 1:1", want{TraditionBible, WorkKJV, "2 John", 1, 1, 0}},
		{"bible/3john_zh", "约翰三书 1:4", want{TraditionBible, WorkKJV, "3 John", 1, 4, 0}},
		{"bible/genesis_zh", "创世记 1:1", want{TraditionBible, WorkKJV, "Genesis", 1, 1, 0}},
		{"bible/psalms_zh", "诗篇 23:1", want{TraditionBible, WorkKJV, "Psalms", 23, 1, 0}},
		{"bible/matthew_zh", "马太福音 5:3", want{TraditionBible, WorkKJV, "Matthew", 5, 3, 0}},
		{"bible/whitespace_padding", "   John 3:16   ", want{TraditionBible, WorkKJV, "John", 3, 16, 0}},

		// Tao Te Ching
		{"dao/zh_short", "道德经 11", want{TraditionDao, WorkDaodejing, "", 11, 0, 0}},
		{"dao/zh_di_zhang", "道德经第十一章", want{TraditionDao, WorkDaodejing, "", 11, 0, 0}},
		{"dao/zh_di_arabic", "道德经第11章", want{TraditionDao, WorkDaodejing, "", 11, 0, 0}},
		{"dao/zh_chapter_one", "道德经第一章", want{TraditionDao, WorkDaodejing, "", 1, 0, 0}},
		{"dao/zh_chapter_eighty_one", "道德经第八十一章", want{TraditionDao, WorkDaodejing, "", 81, 0, 0}},
		{"dao/en_dao", "dao 11", want{TraditionDao, WorkDaodejing, "", 11, 0, 0}},
		{"dao/en_dao_colon", "dao:11", want{TraditionDao, WorkDaodejing, "", 11, 0, 0}},
		{"dao/en_dao_upper", "DAO 11", want{TraditionDao, WorkDaodejing, "", 11, 0, 0}},
		{"dao/en_daodejing_chapter", "daodejing chapter 11", want{TraditionDao, WorkDaodejing, "", 11, 0, 0}},
		{"dao/en_tao_te_ching", "Tao Te Ching 11", want{TraditionDao, WorkDaodejing, "", 11, 0, 0}},

		// Heart Sutra
		{"sutra/zh_short", "心经", want{TraditionSutra, WorkHeartSutra, "", 0, 0, 0}},
		{"sutra/zh_full", "般若波罗蜜多心经", want{TraditionSutra, WorkHeartSutra, "", 0, 0, 0}},
		{"sutra/en_heart_sutra", "Heart Sutra", want{TraditionSutra, WorkHeartSutra, "", 0, 0, 0}},
		{"sutra/en_sutra_heart", "sutra heart", want{TraditionSutra, WorkHeartSutra, "", 0, 0, 0}},

		// Quran
		{"quran/numeric", "Quran 2:255", want{TraditionQuran, WorkQuran, "", 2, 255, 0}},
		{"quran/numeric_colon_after_alias", "quran:2:255", want{TraditionQuran, WorkQuran, "", 2, 255, 0}},
		{"quran/numeric_lower", "quran 2:255", want{TraditionQuran, WorkQuran, "", 2, 255, 0}},
		{"quran/zh", "古兰经 2:255", want{TraditionQuran, WorkQuran, "", 2, 255, 0}},
		{"quran/zh_first_surah", "古兰经 1:1", want{TraditionQuran, WorkQuran, "", 1, 1, 0}},
		{"quran/last_surah", "Quran 114:6", want{TraditionQuran, WorkQuran, "", 114, 6, 0}},
		{"quran/range", "Quran 2:1-5", want{TraditionQuran, WorkQuran, "", 2, 1, 5}},
		{"quran/surah_by_name", "Surah Al-Baqarah 255", want{TraditionQuran, WorkQuran, "", 2, 255, 0}},
		{"quran/surah_by_name_lower", "surah al-fatihah 1", want{TraditionQuran, WorkQuran, "", 1, 1, 0}},
		{"quran/surah_alt_spelling", "Surah Al Baqarah 255", want{TraditionQuran, WorkQuran, "", 2, 255, 0}},
		{"quran/sura_keyword", "Sura Yasin 1", want{TraditionQuran, WorkQuran, "", 36, 1, 0}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Resolve(tc.input)
			if err != nil {
				t.Fatalf("Resolve(%q) returned error: %v", tc.input, err)
			}
			w := tc.want
			if got.Tradition != w.tradition || got.Work != w.work || got.Book != w.book ||
				got.Chapter != w.chapter || got.VerseStart != w.verseStart || got.VerseEnd != w.verseEnd {
				t.Errorf("Resolve(%q) = %+v; want %+v", tc.input, got, w)
			}
		})
	}
}

func TestResolveNegative(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"empty", "", ErrEmpty},
		{"whitespace_only", "   ", ErrEmpty},
		{"unrecognized_word", "asdfghjkl", ErrUnrecognized},
		{"unrecognized_phrase", "the quick brown fox", ErrUnrecognized},
		{"bible_zero_chapter", "John 0:1", ErrInvalidNumber},
		{"bible_zero_verse", "John 1:0", ErrInvalidNumber},
		{"bible_negative_garbage", "John 3:abc", ErrInvalidNumber},
		{"bible_book_only", "John", ErrUnrecognized},
		{"bible_inverted_range", "John 3:16-15", ErrInvalidRange},
		{"dao_zero", "道德经 0", ErrInvalidNumber},
		{"dao_garbage", "道德经 abc", ErrInvalidNumber},
		{"sutra_with_chapter", "心经 1:1", ErrUnrecognized},
		{"quran_no_ref", "Quran", ErrUnrecognized},
		{"quran_garbage", "Quran abc", ErrInvalidNumber},
		{"quran_zero_chapter", "Quran 0:1", ErrInvalidNumber},
		{"quran_unknown_surah", "Surah Foobar 1", ErrUnrecognized},
		{"quran_inverted_range", "Quran 2:5-1", ErrInvalidRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Resolve(tc.input)
			if err == nil {
				t.Fatalf("Resolve(%q) expected error %v, got nil", tc.input, tc.wantErr)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Resolve(%q) error = %v; want errors.Is(_, %v)", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestResolveAmbiguous(t *testing.T) {
	cases := []string{"3:16", "1:1", "2:255"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, err := Resolve(in)
			if err == nil {
				t.Fatalf("Resolve(%q) expected AmbiguousError, got nil", in)
			}
			var amb *AmbiguousError
			if !errors.As(err, &amb) {
				t.Fatalf("Resolve(%q) error = %v; want *AmbiguousError", in, err)
			}
			if amb.Input != in {
				t.Errorf("AmbiguousError.Input = %q; want %q", amb.Input, in)
			}
			if len(amb.Candidates) == 0 {
				t.Errorf("AmbiguousError.Candidates is empty")
			}
		})
	}
}

func TestParseChineseNumeral(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"一", 1, true},
		{"二", 2, true},
		{"九", 9, true},
		{"十", 10, true},
		{"十一", 11, true},
		{"十九", 19, true},
		{"二十", 20, true},
		{"二十三", 23, true},
		{"八十一", 81, true},
		{"九十九", 99, true},
		{"一百", 100, true},
		{"一百零一", 101, true},
		{"一百二十三", 123, true},
		{"", 0, false},
		{"零", 0, true}, // 零 alone parses as 0; callers reject n<1
		{"百", 0, false},
		{"abc", 0, false},
		{"十十", 0, false},
		{"二三", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := parseChineseNumeral(tc.in)
			if ok != tc.ok {
				t.Fatalf("parseChineseNumeral(%q) ok = %v; want %v", tc.in, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("parseChineseNumeral(%q) = %d; want %d", tc.in, got, tc.want)
			}
		})
	}
}
