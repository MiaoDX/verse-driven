package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/MiaoDX/verse-driven/internal/schema"
)

func TestRunLookupJSONHappy(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := runLookup([]string{"--format=json", "John 3:16"}, Streams{Out: &out, Err: &errBuf})
	if code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, errBuf.String())
	}
	var v schema.Verse
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatalf("output is not JSON Verse: %v\n%s", err, out.String())
	}
	if v.ID != "bible.kjv.john.3.16" {
		t.Errorf("id %q, want bible.kjv.john.3.16", v.ID)
	}
	if v.ChecksumSHA256 == "" {
		t.Error("checksum_sha256 missing in JSON output")
	}
	if v.Source.Provider == "" {
		t.Error("source.provider missing")
	}
}

func TestRunLookupJSONIsDefault(t *testing.T) {
	var out bytes.Buffer
	if code := runLookup([]string{"道德经 11"}, Streams{Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !json.Valid(out.Bytes()) {
		t.Errorf("default format must produce valid JSON; got: %s", out.String())
	}
}

func TestRunLookupTextFormat(t *testing.T) {
	var out bytes.Buffer
	if code := runLookup([]string{"--format=text", "John 3:16"}, Streams{Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	s := out.String()
	if !strings.Contains(s, "John 3:16") {
		t.Errorf("text output missing display ref:\n%s", s)
	}
	if !strings.Contains(s, "checksum:") {
		t.Errorf("text output missing checksum line:\n%s", s)
	}
}

func TestRunLookupAmbiguous(t *testing.T) {
	var errBuf bytes.Buffer
	code := runLookup([]string{"3:16"}, Streams{Out: &bytes.Buffer{}, Err: &errBuf})
	if code != 3 {
		t.Errorf("exit code = %d, want 3 (ambiguous)\nstderr=%s", code, errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "ambiguous") {
		t.Errorf("stderr should mention ambiguity:\n%s", errBuf.String())
	}
}

func TestRunLookupUnrecognized(t *testing.T) {
	var errBuf bytes.Buffer
	code := runLookup([]string{"some gibberish"}, Streams{Out: &bytes.Buffer{}, Err: &errBuf})
	if code != 1 {
		t.Errorf("exit code = %d, want 1\nstderr=%s", code, errBuf.String())
	}
}

func TestRunLookupMissingArg(t *testing.T) {
	var errBuf bytes.Buffer
	if code := runLookup([]string{}, Streams{Out: &bytes.Buffer{}, Err: &errBuf}); code != 2 {
		t.Errorf("exit code = %d, want 2 (usage)", code)
	}
}

func TestRunLookupSutraBundled(t *testing.T) {
	var out, errBuf bytes.Buffer
	if code := runLookup([]string{"心经"}, Streams{Out: &out, Err: &errBuf}); code != 0 {
		t.Fatalf("exit %d, stderr=%q", code, errBuf.String())
	}
	var v schema.Verse
	if err := json.Unmarshal(out.Bytes(), &v); err != nil {
		t.Fatalf("output is not JSON Verse: %v\n%s", err, out.String())
	}
	if v.ID != "sutra.heart-sutra.1" {
		t.Errorf("id %q, want sutra.heart-sutra.1", v.ID)
	}
	if v.InclusionMode != "bundled" {
		t.Errorf("inclusion_mode %q, want bundled", v.InclusionMode)
	}
	if v.ChecksumSHA256 == "" {
		t.Error("checksum_sha256 missing")
	}
}

func TestRunLookupBilingualPacks(t *testing.T) {
	cases := []struct {
		name string
		ref  string
		id   string
		lang string
	}{
		{"bible_zh", "约翰福音 3:16", "bible.cuv-s.john.3.16", "zh-Hans"},
		{"dao_en", "dao 11", "dao.legge.11.1", "en"},
		{"dao_zh", "道德经 11", "dao.daodejing.11.1", "zh-Hans"},
		{"sutra_en", "Heart Sutra", "sutra.heart-sutra-en.1", "en"},
		{"quran_en", "Quran 2:255", "quran.pickthall.2.255", "en"},
		{"quran_zh", "古兰经 2:255", "quran.majian.2.255", "zh-Hans"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errBuf bytes.Buffer
			if code := runLookup([]string{tc.ref}, Streams{Out: &out, Err: &errBuf}); code != 0 {
				t.Fatalf("exit %d, stderr=%q", code, errBuf.String())
			}
			var v schema.Verse
			if err := json.Unmarshal(out.Bytes(), &v); err != nil {
				t.Fatalf("output is not JSON Verse: %v", err)
			}
			if v.ID != tc.id || v.Lang != tc.lang {
				t.Errorf("lookup %q got id=%q lang=%q; want id=%q lang=%q", tc.ref, v.ID, v.Lang, tc.id, tc.lang)
			}
		})
	}
}

func TestLookupFromPromptSlashMarker(t *testing.T) {
	in := strings.NewReader("/bible John 3:16 Refactor the cron-string scheduler.")
	var out, errBuf bytes.Buffer
	if code := runLookupFromPrompt(nil, Streams{In: in, Out: &out, Err: &errBuf}); code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errBuf.String())
	}
	var resp struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out.String())
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "<scripture_card>") {
		t.Errorf("envelope missing scripture_card tags: %s", resp.HookSpecificOutput.AdditionalContext)
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "John 3:16") {
		t.Errorf("envelope missing display ref: %s", resp.HookSpecificOutput.AdditionalContext)
	}
	if resp.HookSpecificOutput.HookEventName != "UserPromptExpansion" {
		t.Errorf("hookSpecificOutput hookEventName = %q, want UserPromptExpansion",
			resp.HookSpecificOutput.HookEventName)
	}
}

func TestLookupFromPromptInlineMarker(t *testing.T) {
	in := strings.NewReader("Please [[dao:11]] keep going on the helper.")
	var out bytes.Buffer
	if code := runLookupFromPrompt(nil, Streams{In: in, Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var resp struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out.String())
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "Tao Te Ching") {
		t.Errorf("dao envelope missing English display ref")
	}
}

func TestLookupFromPromptInlineMarkerChineseDao(t *testing.T) {
	in := strings.NewReader("Please [[dao:道德经第十一章]] keep going on the helper.")
	var out bytes.Buffer
	if code := runLookupFromPrompt(nil, Streams{In: in, Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var resp struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out.String())
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "道德经") {
		t.Errorf("dao envelope missing Chinese display ref")
	}
}

func TestLookupFromPromptDollarDaoAlias(t *testing.T) {
	in := strings.NewReader("Please $dao:11 keep going on the helper.")
	var out bytes.Buffer
	if code := runLookupFromPrompt([]string{"--hook-event=UserPromptSubmit"}, Streams{In: in, Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var resp struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out.String())
	}
	if resp.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
		t.Errorf("hookSpecificOutput hookEventName = %q, want UserPromptSubmit",
			resp.HookSpecificOutput.HookEventName)
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "Tao Te Ching") {
		t.Errorf("dao envelope missing English display ref")
	}
}

func TestLookupFromPromptDollarDaoDotAlias(t *testing.T) {
	in := strings.NewReader("Please $dao.11 keep going on the helper.")
	var out bytes.Buffer
	if code := runLookupFromPrompt([]string{"--hook-event=UserPromptSubmit"}, Streams{In: in, Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var resp struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out.String())
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "Tao Te Ching") {
		t.Errorf("dao envelope missing English display ref")
	}
}

func TestLookupFromPromptSlashSutraChineseAlias(t *testing.T) {
	in := strings.NewReader("/sutra 心经 refactor the formatter.")
	var out, errBuf bytes.Buffer
	if code := runLookupFromPrompt([]string{"--hook-event=UserPromptExpansion"}, Streams{In: in, Out: &out, Err: &errBuf}); code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errBuf.String())
	}
	if out.Len() == 0 {
		t.Fatalf("expected hook JSON output, got none; stderr=%s", errBuf.String())
	}
	var resp struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out.String())
	}
	if resp.HookSpecificOutput.HookEventName != "UserPromptExpansion" {
		t.Errorf("hookSpecificOutput hookEventName = %q, want UserPromptExpansion",
			resp.HookSpecificOutput.HookEventName)
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "心经") {
		t.Errorf("sutra envelope missing display ref:\n%s", resp.HookSpecificOutput.AdditionalContext)
	}
}

func TestLookupFromPromptDollarBibleAliasTrimsTrailingPrompt(t *testing.T) {
	in := strings.NewReader("Please $bible:John 3:16 refactor the helper.")
	var out bytes.Buffer
	if code := runLookupFromPrompt([]string{"--hook-event=UserPromptSubmit"}, Streams{In: in, Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var resp struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out.String())
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "John 3:16") {
		t.Errorf("bible envelope missing display ref:\n%s", resp.HookSpecificOutput.AdditionalContext)
	}
}

func TestLookupFromPromptDollarBibleDotAlias(t *testing.T) {
	cases := []string{
		"Please $bible.John.3.16 refactor the helper.",
		"Please $bible.1.John.3.16 refactor the helper.",
		"Please $bible.Song.of.Solomon.1.1 refactor the helper.",
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			var out bytes.Buffer
			if code := runLookupFromPrompt([]string{"--hook-event=UserPromptSubmit"}, Streams{In: strings.NewReader(input), Out: &out, Err: &bytes.Buffer{}}); code != 0 {
				t.Fatalf("exit %d", code)
			}
			var resp struct {
				HookSpecificOutput struct {
					AdditionalContext string `json:"additionalContext"`
				} `json:"hookSpecificOutput"`
			}
			if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
				t.Fatalf("output not JSON: %v\n%s", err, out.String())
			}
			if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "<scripture_card>") {
				t.Errorf("bible envelope missing scripture card:\n%s", resp.HookSpecificOutput.AdditionalContext)
			}
		})
	}
}

func TestLookupFromPromptDollarSkillNameDoesNotTrigger(t *testing.T) {
	var out bytes.Buffer
	code := runLookupFromPrompt(nil, Streams{
		In:  strings.NewReader("$setup-matt-pocock-skills"),
		Out: &out,
		Err: &bytes.Buffer{},
	})
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output for unrelated $skill syntax, got: %q", out.String())
	}
}

func TestLookupFromPromptDollarSkillDotNameDoesNotTrigger(t *testing.T) {
	var out bytes.Buffer
	code := runLookupFromPrompt(nil, Streams{
		In:  strings.NewReader("$setup.matt.pocock.skills"),
		Out: &out,
		Err: &bytes.Buffer{},
	})
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output for unrelated $skill syntax, got: %q", out.String())
	}
}

func TestLookupFromPromptJSONInputForm(t *testing.T) {
	body, _ := json.Marshal(map[string]string{
		"hook_event_name": "UserPromptSubmit",
		"prompt":          "/bible John 3:16 do the thing",
	})
	var out bytes.Buffer
	code := runLookupFromPrompt(nil, Streams{In: bytes.NewReader(body), Out: &out, Err: &bytes.Buffer{}})
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if out.Len() == 0 {
		t.Fatal("expected JSON output for valid marker, got empty")
	}
	var resp struct {
		HookSpecificOutput struct {
			HookEventName string `json:"hookEventName"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out.String())
	}
	if resp.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
		t.Errorf("hookSpecificOutput hookEventName = %q, want UserPromptSubmit",
			resp.HookSpecificOutput.HookEventName)
	}
}

func TestLookupFromPromptHookEventFlag(t *testing.T) {
	in := strings.NewReader("Please [[dao:11]] keep going on the helper.")
	var out bytes.Buffer
	if code := runLookupFromPrompt([]string{"--hook-event=UserPromptSubmit"}, Streams{In: in, Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var resp struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("output not JSON: %v\n%s", err, out.String())
	}
	if resp.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
		t.Errorf("hookSpecificOutput hookEventName = %q, want UserPromptSubmit",
			resp.HookSpecificOutput.HookEventName)
	}
	if !strings.Contains(resp.HookSpecificOutput.AdditionalContext, "Tao Te Ching") {
		t.Errorf("dao envelope missing English display ref")
	}
	var raw map[string]any
	if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
		t.Fatalf("output not JSON object: %v", err)
	}
	if _, ok := raw["additionalContext"]; ok {
		t.Errorf("Codex-compatible hook output must not include legacy top-level additionalContext")
	}
}

func TestLookupFromPromptNoMarkerSilent(t *testing.T) {
	var out bytes.Buffer
	code := runLookupFromPrompt(nil, Streams{
		In:  strings.NewReader("just refactor this no marker here"),
		Out: &out,
		Err: &bytes.Buffer{},
	})
	if code != 0 {
		t.Errorf("exit %d, want 0 (silent)", code)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output, got: %q", out.String())
	}
}

func TestLookupFromPromptUnresolvableMarkerSoftFails(t *testing.T) {
	// /bible without a ref → bare marker → no marker treated as no-op
	in := strings.NewReader("/bible")
	var out, errBuf bytes.Buffer
	code := runLookupFromPrompt(nil, Streams{In: in, Out: &out, Err: &errBuf})
	if code != 0 {
		t.Errorf("exit %d, want 0", code)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output, got: %q", out.String())
	}
}
