// Package resolver parses free-form scripture references into canonical
// (tradition, work, ref) tuples. See plan.md §8.1 for the supported matrix.
package resolver

import (
	"errors"
	"fmt"
	"strings"
)

// Tradition identifiers.
const (
	TraditionBible = "bible"
	TraditionDao   = "dao"
	TraditionSutra = "sutra"
	TraditionQuran = "quran"
)

// Work identifiers used in canonical references.
const (
	WorkKJV        = "KJV"
	WorkDaodejing  = "daodejing"
	WorkHeartSutra = "heart-sutra"
	WorkQuran      = "quran"
)

// Reference is a parsed scripture reference.
//
// Book is empty for traditions without per-book structure
// (Tao Te Ching, Heart Sutra, Quran-by-number).
// Chapter == 0 means "whole work" (e.g. Heart Sutra has no chapters).
// VerseStart == 0 means "no specific verse"; VerseEnd == 0 means "single
// verse" (i.e. equivalent to VerseStart).
type Reference struct {
	Tradition  string
	Work       string
	Book       string
	Chapter    int
	VerseStart int
	VerseEnd   int
}

// AmbiguousError is returned when input could plausibly resolve to more than
// one reference (e.g. a bare "3:16" with no tradition keyword).
type AmbiguousError struct {
	Input      string
	Candidates []Reference
}

func (e *AmbiguousError) Error() string {
	parts := make([]string, 0, len(e.Candidates))
	for _, c := range e.Candidates {
		parts = append(parts, fmt.Sprintf("%s/%s %d:%d", c.Tradition, c.Work, c.Chapter, c.VerseStart))
	}
	return fmt.Sprintf("resolver: ambiguous reference %q; candidates: %s", e.Input, strings.Join(parts, ", "))
}

// Sentinel errors.
var (
	ErrEmpty         = errors.New("resolver: empty input")
	ErrUnrecognized  = errors.New("resolver: unrecognized reference")
	ErrInvalidNumber = errors.New("resolver: invalid chapter or verse")
	ErrInvalidRange  = errors.New("resolver: verse_end must be >= verse_start")
)
