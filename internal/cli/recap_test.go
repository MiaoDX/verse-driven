package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRecapDeterministicWithSeed(t *testing.T) {
	home := tempHome(t)
	var a, b bytes.Buffer
	for _, w := range []*bytes.Buffer{&a, &b} {
		if code := runRecap([]string{"--seed=42"}, Streams{
			Out:    w,
			Err:    &bytes.Buffer{},
			HomeFn: func() (string, error) { return home, nil },
		}); code != 0 {
			t.Fatalf("exit %d", code)
		}
	}
	if a.String() != b.String() {
		t.Errorf("same seed → different output:\nA: %s\nB: %s", a.String(), b.String())
	}
}

func TestRecapTraditionFilter(t *testing.T) {
	home := tempHome(t)
	var out bytes.Buffer
	if code := runRecap([]string{"--tradition=dao", "--seed=1"}, Streams{
		Out:    &out,
		Err:    &bytes.Buffer{},
		HomeFn: func() (string, error) { return home, nil },
	}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out.String(), "道德经") {
		t.Errorf("dao recap should mention 道德经:\n%s", out.String())
	}
}

func TestRecapBibleHasAttribution(t *testing.T) {
	home := tempHome(t)
	var out bytes.Buffer
	if code := runRecap([]string{"--tradition=bible", "--seed=7"}, Streams{
		Out:    &out,
		Err:    &bytes.Buffer{},
		HomeFn: func() (string, error) { return home, nil },
	}); code != 0 {
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
	home := tempHome(t)
	var out bytes.Buffer
	if code := runRecap([]string{"--tradition=bible", "--seed=7", "--first-letter"}, Streams{
		Out:    &out,
		Err:    &bytes.Buffer{},
		HomeFn: func() (string, error) { return home, nil },
	}); code != 0 {
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

func TestRecapLearningUsesDueCardFromState(t *testing.T) {
	home := tempHome(t)
	cfgDir := filepath.Join(home, ".config", "scripture-mcp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	state := `{
  "version": 1,
  "cards": {
    "dao.daodejing.11.1": {
      "repetitions": 0,
      "interval_days": 0,
      "ease_factor": 2.5,
      "due": "2000-01-01T00:00:00Z"
    },
    "bible.kjv.john.3.16": {
      "repetitions": 1,
      "interval_days": 6,
      "ease_factor": 2.5,
      "due": "2999-01-01T00:00:00Z"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(cfgDir, "learning.json"), []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errBuf bytes.Buffer
	code := runRecap([]string{"--learning"}, Streams{
		Out:    &out,
		Err:    &errBuf,
		HomeFn: func() (string, error) { return home, nil },
	})
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "道德经 第11章") {
		t.Errorf("learning recap did not select due card; output:\n%s", out.String())
	}
	body, err := os.ReadFile(filepath.Join(cfgDir, "learning.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"last_seen"`) {
		t.Errorf("learning state was not updated:\n%s", string(body))
	}
}

func TestRecapUsesLearningConfig(t *testing.T) {
	home := tempHome(t)
	cfgDir := filepath.Join(home, ".config", "scripture-mcp")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{"version":1,"learning_enabled":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	state := `{
  "version": 1,
  "cards": {
    "dao.daodejing.11.1": {
      "repetitions": 0,
      "interval_days": 0,
      "ease_factor": 2.5,
      "due": "2000-01-01T00:00:00Z"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(cfgDir, "learning.json"), []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errBuf bytes.Buffer
	code := runRecap(nil, Streams{
		Out:    &out,
		Err:    &errBuf,
		HomeFn: func() (string, error) { return home, nil },
	})
	if code != 0 {
		t.Fatalf("exit %d, stderr=%s", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "道德经 第11章") {
		t.Errorf("configured learning recap did not select due card; output:\n%s", out.String())
	}
}

func TestRecapUnknownTradition(t *testing.T) {
	home := tempHome(t)
	var errBuf bytes.Buffer
	code := runRecap([]string{"--tradition=nonsense", "--seed=1"}, Streams{
		Out:    &bytes.Buffer{},
		Err:    &errBuf,
		HomeFn: func() (string, error) { return home, nil },
	})
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
