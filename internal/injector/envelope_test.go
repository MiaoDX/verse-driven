package injector

import (
	"strings"
	"testing"

	"github.com/MiaoDX/verse-driven/internal/schema"
)

func TestEnvelopeWrapsScriptureCard(t *testing.T) {
	v := schema.Verse{
		ID:        "bible.kjv.john.3.16",
		Tradition: "bible",
		Lang:      "en",
		Work:      "KJV",
		CanonicalRef: schema.CanonicalRef{
			Book: "John", Chapter: 3, VerseStart: 16,
		},
		Text:           "TXT",
		Source:         schema.Source{Provider: "PG", License: "PD", Attribution: "KJV"},
		ChecksumSHA256: strings.Repeat("0", 64),
	}
	out := Envelope(v)
	for _, want := range []string{
		"Temporary reflection context",
		"Do not preach",
		"<scripture_card>",
		"TXT",
		"John 3:16",
		"KJV",
		"checksum: sha256:" + strings.Repeat("0", 64),
		"</scripture_card>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("envelope missing %q\n--- got ---\n%s", want, out)
		}
	}
}

func TestDisplayRefByTradition(t *testing.T) {
	cases := []struct {
		v    schema.Verse
		want string
	}{
		{
			v:    schema.Verse{Tradition: "bible", CanonicalRef: schema.CanonicalRef{Book: "John", Chapter: 3, VerseStart: 16}},
			want: "John 3:16",
		},
		{
			v:    schema.Verse{Tradition: "bible", CanonicalRef: schema.CanonicalRef{Book: "John", Chapter: 3, VerseStart: 16, VerseEnd: 18}},
			want: "John 3:16-18",
		},
		{
			v:    schema.Verse{Tradition: "dao", CanonicalRef: schema.CanonicalRef{Chapter: 11, VerseStart: 1}},
			want: "道德经 第11章",
		},
		{
			v:    schema.Verse{Tradition: "sutra"},
			want: "心经",
		},
		{
			v:    schema.Verse{Tradition: "quran", CanonicalRef: schema.CanonicalRef{Chapter: 2, VerseStart: 255}},
			want: "Quran 2:255",
		},
		{
			v: schema.Verse{
				Tradition:  "bible",
				DisplayRef: map[string]string{"en": "Romans 8:28 (custom)"},
			},
			want: "Romans 8:28 (custom)",
		},
	}
	for _, c := range cases {
		if got := DisplayRef(c.v); got != c.want {
			t.Errorf("DisplayRef(%+v) = %q, want %q", c.v.Tradition, got, c.want)
		}
	}
}
