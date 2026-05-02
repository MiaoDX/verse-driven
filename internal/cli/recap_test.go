package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRecapDeterministicWithSeed(t *testing.T) {
	var a, b bytes.Buffer
	for _, w := range []*bytes.Buffer{&a, &b} {
		if code := runRecap([]string{"--seed=42"}, Streams{Out: w, Err: &bytes.Buffer{}}); code != 0 {
			t.Fatalf("exit %d", code)
		}
	}
	if a.String() != b.String() {
		t.Errorf("same seed → different output:\nA: %s\nB: %s", a.String(), b.String())
	}
}

func TestRecapTraditionFilter(t *testing.T) {
	var out bytes.Buffer
	if code := runRecap([]string{"--tradition=dao", "--seed=1"}, Streams{Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "道德经") {
		t.Errorf("dao recap should mention 道德经:\n%s", out.String())
	}
}

func TestRecapBibleHasAttribution(t *testing.T) {
	var out bytes.Buffer
	if code := runRecap([]string{"--tradition=bible", "--seed=7"}, Streams{Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	s := out.String()
	if !strings.Contains(s, "📖") {
		t.Errorf("recap output missing scripture marker: %s", s)
	}
	if !strings.Contains(s, "King James Version") {
		t.Errorf("bible recap missing KJV attribution: %s", s)
	}
}

func TestRecapFirstLetterMode(t *testing.T) {
	var out bytes.Buffer
	if code := runRecap([]string{"--tradition=bible", "--seed=7", "--first-letter"}, Streams{Out: &out, Err: &bytes.Buffer{}}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	// First-letter masking replaces letters past the first with `_`.
	// The masked body should contain underscores; without masking the
	// output never contains the standalone underscore character.
	s := out.String()
	if !strings.Contains(s, "_") {
		t.Errorf("first-letter mode should produce underscores: %s", s)
	}
}

func TestRecapUnknownTradition(t *testing.T) {
	var errBuf bytes.Buffer
	code := runRecap([]string{"--tradition=nonsense", "--seed=1"}, Streams{Out: &bytes.Buffer{}, Err: &errBuf})
	if code != 1 {
		t.Errorf("exit %d, want 1 for unknown tradition; stderr=%q", code, errBuf.String())
	}
}

func TestFirstLetterMaskASCII(t *testing.T) {
	got := firstLetterMask("For God so loved the world")
	want := "F__ G__ s_ l____ t__ w____"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestFirstLetterMaskHan(t *testing.T) {
	got := firstLetterMask("三十辐共一毂")
	if !strings.Contains(got, "三 _") {
		t.Errorf("Han masking incorrect: %q", got)
	}
}
