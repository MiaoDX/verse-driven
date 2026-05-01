package schema

import (
	"errors"
	"testing"
)

func validVerse() Verse {
	return Verse{
		ID:        "bible.kjv.john.3.16",
		Tradition: "bible",
		Lang:      "en",
		Work:      "KJV",
		CanonicalRef: CanonicalRef{
			Book:       "John",
			Chapter:    3,
			VerseStart: 16,
			VerseEnd:   16,
		},
		DisplayRef: map[string]string{
			"en":    "John 3:16",
			"zh-CN": "约翰福音 3:16",
		},
		Text: "For God so loved the world, that he gave his only begotten Son...",
		Source: Source{
			Provider:    "Project Gutenberg",
			License:     "public-domain-us",
			Attribution: "KJV",
		},
		ChecksumSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		InclusionMode:  "bundled",
		Sensitivity:    "sacred_exact_quote",
	}
}

func TestValidate_HappyPath(t *testing.T) {
	if err := Validate(validVerse()); err != nil {
		t.Fatalf("expected valid verse, got: %v", err)
	}
}

func TestValidate_MissingFields(t *testing.T) {
	cases := []struct {
		name  string
		mut   func(*Verse)
		wants error
	}{
		{"missing id", func(v *Verse) { v.ID = "" }, ErrMissingID},
		{"missing tradition", func(v *Verse) { v.Tradition = "" }, ErrMissingTradition},
		{"missing lang", func(v *Verse) { v.Lang = "" }, ErrMissingLang},
		{"missing text", func(v *Verse) { v.Text = "" }, ErrMissingText},
		{"missing chapter", func(v *Verse) { v.CanonicalRef.Chapter = 0 }, ErrMissingCanonicalRef},
		{"missing verse_start", func(v *Verse) { v.CanonicalRef.VerseStart = 0 }, ErrMissingCanonicalRef},
		{"missing source.provider", func(v *Verse) { v.Source.Provider = "" }, ErrMissingSource},
		{"missing source.license", func(v *Verse) { v.Source.License = "" }, ErrMissingSource},
		{"missing source.attribution", func(v *Verse) { v.Source.Attribution = "" }, ErrMissingSource},
		{"missing checksum", func(v *Verse) { v.ChecksumSHA256 = "" }, ErrMissingChecksum},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := validVerse()
			tc.mut(&v)
			err := Validate(v)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tc.wants) {
				t.Fatalf("expected %v, got %v", tc.wants, err)
			}
		})
	}
}

func TestValidate_BadEnumsAndFormats(t *testing.T) {
	cases := []struct {
		name  string
		mut   func(*Verse)
		wants error
	}{
		{"bad id format", func(v *Verse) { v.ID = "Bible/John/3:16" }, ErrBadID},
		{"bad checksum hex", func(v *Verse) { v.ChecksumSHA256 = "not-a-hex-checksum" }, ErrBadChecksum},
		{"short checksum", func(v *Verse) { v.ChecksumSHA256 = "abc123" }, ErrBadChecksum},
		{"bad inclusion_mode", func(v *Verse) { v.InclusionMode = "remote" }, ErrBadInclusionMode},
		{"bad sensitivity", func(v *Verse) { v.Sensitivity = "casual" }, ErrBadSensitivity},
		{"verse_end before verse_start", func(v *Verse) {
			v.CanonicalRef.VerseStart = 10
			v.CanonicalRef.VerseEnd = 5
		}, ErrBadVerseRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := validVerse()
			tc.mut(&v)
			err := Validate(v)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tc.wants) {
				t.Fatalf("expected %v, got %v", tc.wants, err)
			}
		})
	}
}

func TestValidate_OptionalEnumsMayBeEmpty(t *testing.T) {
	v := validVerse()
	v.InclusionMode = ""
	v.Sensitivity = ""
	if err := Validate(v); err != nil {
		t.Fatalf("optional enums should be allowed empty; got: %v", err)
	}
}

func TestJSONSchemaEmbedded(t *testing.T) {
	if len(JSONSchema) == 0 {
		t.Fatal("expected verse.schema.json to be embedded")
	}
}
