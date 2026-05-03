package packs

import (
	"errors"
	"testing"

	"github.com/MiaoDX/verse-driven/internal/resolver"
)

func TestReferenceID(t *testing.T) {
	cases := []struct {
		name string
		ref  resolver.Reference
		want string
	}{
		{
			name: "single-word book",
			ref:  resolver.Reference{Tradition: "bible", Work: "KJV", Book: "John", Chapter: 3, VerseStart: 16},
			want: "bible.kjv.john.3.16",
		},
		{
			name: "numbered book stays numbered in slug",
			ref:  resolver.Reference{Tradition: "bible", Work: "KJV", Book: "1 John", Chapter: 4, VerseStart: 8},
			want: "bible.kjv.1-john.4.8",
		},
		{
			name: "multi-word book becomes hyphen slug",
			ref:  resolver.Reference{Tradition: "bible", Work: "KJV", Book: "Song of Solomon", Chapter: 1, VerseStart: 1},
			want: "bible.kjv.song-of-solomon.1.1",
		},
		{
			name: "dao defaults verse to 1 when omitted",
			ref:  resolver.Reference{Tradition: "dao", Work: "daodejing", Chapter: 11},
			want: "dao.daodejing.11.1",
		},
		{
			name: "sutra uses single-segment id",
			ref:  resolver.Reference{Tradition: "sutra", Work: "heart-sutra"},
			want: "sutra.heart-sutra.1",
		},
		{
			name: "quran follows surah:verse",
			ref:  resolver.Reference{Tradition: "quran", Work: "quran", Chapter: 2, VerseStart: 255},
			want: "quran.quran.2.255",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ReferenceID(c.ref)
			if err != nil {
				t.Fatalf("ReferenceID(%v) err: %v", c.ref, err)
			}
			if got != c.want {
				t.Errorf("ReferenceID = %q, want %q", got, c.want)
			}
		})
	}
}

func TestReferenceIDErrors(t *testing.T) {
	cases := []resolver.Reference{
		{Tradition: "bible", Work: "KJV", Chapter: 3, VerseStart: 16},   // missing book
		{Tradition: "bible", Work: "KJV", Book: "John", VerseStart: 16}, // missing chapter
		{Tradition: "dao", Work: "daodejing"},                           // missing chapter
		{Tradition: "quran", Work: "quran", VerseStart: 1},              // missing surah
		{Tradition: "unknown"},
	}
	for i, ref := range cases {
		if _, err := ReferenceID(ref); err == nil {
			t.Errorf("case %d: ReferenceID(%+v) returned no error", i, ref)
		}
	}
}

func TestLookupReferenceBibleFound(t *testing.T) {
	v, err := LookupReference(resolver.Reference{
		Tradition:  "bible",
		Work:       "KJV",
		Book:       "John",
		Chapter:    3,
		VerseStart: 16,
	})
	if err != nil {
		t.Fatalf("LookupReference err: %v", err)
	}
	if v.ID != "bible.kjv.john.3.16" {
		t.Errorf("got id %q", v.ID)
	}
	if v.Tradition != "bible" {
		t.Errorf("got tradition %q", v.Tradition)
	}
}

func TestLookupReferenceDaoFound(t *testing.T) {
	v, err := LookupReference(resolver.Reference{
		Tradition: "dao",
		Work:      "daodejing",
		Chapter:   11,
	})
	if err != nil {
		t.Fatalf("LookupReference err: %v", err)
	}
	if v.ID != "dao.daodejing.11.1" {
		t.Errorf("got id %q", v.ID)
	}
}

func TestLookupReferenceSutraFound(t *testing.T) {
	v, err := LookupReference(resolver.Reference{Tradition: "sutra", Work: "heart-sutra"})
	if err != nil {
		t.Fatalf("LookupReference err: %v", err)
	}
	if v.ID != "sutra.heart-sutra.1" {
		t.Errorf("got id %q", v.ID)
	}
	if v.Tradition != "sutra" {
		t.Errorf("got tradition %q", v.Tradition)
	}
}

func TestLookupReferenceQuranNotBundled(t *testing.T) {
	_, err := LookupReference(resolver.Reference{Tradition: "quran", Work: "quran", Chapter: 2, VerseStart: 255})
	if !errors.Is(err, ErrNotBundled) {
		t.Errorf("got %v, want ErrNotBundled", err)
	}
}

func TestLookupReferenceMissBibleVerse(t *testing.T) {
	_, err := LookupReference(resolver.Reference{Tradition: "bible", Work: "KJV", Book: "John", Chapter: 99, VerseStart: 99})
	if err == nil {
		t.Fatal("expected error for missing verse, got nil")
	}
	if errors.Is(err, ErrNotBundled) {
		t.Errorf("missing-verse error wrongly classified as ErrNotBundled: %v", err)
	}
}
