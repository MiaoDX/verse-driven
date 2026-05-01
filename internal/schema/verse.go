// Package schema defines the Verse contract that every pack must satisfy.
// See plan.md §6.1 and internal/schema/verse.schema.json.
package schema

import (
	_ "embed"
	"errors"
	"fmt"
	"regexp"
)

//go:embed verse.schema.json
var JSONSchema []byte

type CanonicalRef struct {
	Book       string `json:"book,omitempty"`
	Chapter    int    `json:"chapter"`
	VerseStart int    `json:"verse_start"`
	VerseEnd   int    `json:"verse_end,omitempty"`
}

type Source struct {
	Provider    string `json:"provider"`
	License     string `json:"license"`
	Attribution string `json:"attribution"`
}

type Verse struct {
	ID             string            `json:"id"`
	Tradition      string            `json:"tradition"`
	Lang           string            `json:"lang"`
	Work           string            `json:"work,omitempty"`
	CanonicalRef   CanonicalRef      `json:"canonical_ref"`
	DisplayRef     map[string]string `json:"display_ref,omitempty"`
	Text           string            `json:"text"`
	Source         Source            `json:"source"`
	ChecksumSHA256 string            `json:"checksum_sha256"`
	InclusionMode  string            `json:"inclusion_mode,omitempty"`
	Sensitivity    string            `json:"sensitivity,omitempty"`
}

var (
	ErrMissingID             = errors.New("verse: id is required")
	ErrMissingTradition      = errors.New("verse: tradition is required")
	ErrMissingLang           = errors.New("verse: lang is required")
	ErrMissingText           = errors.New("verse: text is required")
	ErrMissingCanonicalRef   = errors.New("verse: canonical_ref.chapter and canonical_ref.verse_start are required")
	ErrMissingSource         = errors.New("verse: source.provider/license/attribution are required")
	ErrMissingChecksum       = errors.New("verse: checksum_sha256 is required")
	ErrBadChecksum           = errors.New("verse: checksum_sha256 must be 64 lowercase hex characters")
	ErrBadID                 = errors.New("verse: id must be dotted lowercase, e.g. bible.kjv.john.3.16")
	ErrBadInclusionMode      = errors.New("verse: inclusion_mode must be one of bundled, api_only")
	ErrBadSensitivity        = errors.New("verse: sensitivity must be one of sacred_exact_quote")
	ErrBadVerseRange         = errors.New("verse: verse_end must be >= verse_start when set")
)

var (
	idPattern       = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9]+)+$`)
	checksumPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

var (
	validInclusionModes = map[string]struct{}{
		"bundled":  {},
		"api_only": {},
	}
	validSensitivities = map[string]struct{}{
		"sacred_exact_quote": {},
	}
)

// Validate checks that v conforms to the verse schema.
// Returns nil on success; returns the first violation encountered otherwise.
func Validate(v Verse) error {
	if v.ID == "" {
		return ErrMissingID
	}
	if !idPattern.MatchString(v.ID) {
		return fmt.Errorf("%w: %q", ErrBadID, v.ID)
	}
	if v.Tradition == "" {
		return ErrMissingTradition
	}
	if v.Lang == "" {
		return ErrMissingLang
	}
	if v.Text == "" {
		return ErrMissingText
	}
	if v.CanonicalRef.Chapter < 1 || v.CanonicalRef.VerseStart < 1 {
		return ErrMissingCanonicalRef
	}
	if v.CanonicalRef.VerseEnd != 0 && v.CanonicalRef.VerseEnd < v.CanonicalRef.VerseStart {
		return ErrBadVerseRange
	}
	if v.Source.Provider == "" || v.Source.License == "" || v.Source.Attribution == "" {
		return ErrMissingSource
	}
	if v.ChecksumSHA256 == "" {
		return ErrMissingChecksum
	}
	if !checksumPattern.MatchString(v.ChecksumSHA256) {
		return fmt.Errorf("%w: %q", ErrBadChecksum, v.ChecksumSHA256)
	}
	if v.InclusionMode != "" {
		if _, ok := validInclusionModes[v.InclusionMode]; !ok {
			return fmt.Errorf("%w: %q", ErrBadInclusionMode, v.InclusionMode)
		}
	}
	if v.Sensitivity != "" {
		if _, ok := validSensitivities[v.Sensitivity]; !ok {
			return fmt.Errorf("%w: %q", ErrBadSensitivity, v.Sensitivity)
		}
	}
	return nil
}
