package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func runInitWithHome(t *testing.T, home string, args ...string) (string, string, int) {
	t.Helper()
	var out, errBuf bytes.Buffer
	code := runInit(args, Streams{
		Out: &out, Err: &errBuf,
		HomeFn: func() (string, error) { return home, nil },
	})
	return out.String(), errBuf.String(), code
}

func TestInitClaudeCreatesFile(t *testing.T) {
	home := tempHome(t)
	_, errOut, code := runInitWithHome(t, home, "--target=claude-code")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	body, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}
	s := string(body)
	for _, want := range []string{
		"// >>> verse-driven",
		"// <<< verse-driven",
		"\"command\": \"scripture-mcp\"",
		"\"args\": [\"serve\"]",
		"UserPromptExpansion",
		"scripture-mcp lookup-from-prompt",
		"scripture-mcp recap --terminal",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("settings.json missing %q\n--- file ---\n%s", want, s)
		}
	}
}

func TestInitClaudeRecapOff(t *testing.T) {
	home := tempHome(t)
	if _, _, code := runInitWithHome(t, home, "--target=claude-code", "--recap=off"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	body, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if strings.Contains(string(body), "recap --terminal") {
		t.Errorf("recap=off should omit Stop hook:\n%s", body)
	}
}

func TestInitIdempotent(t *testing.T) {
	home := tempHome(t)
	if _, _, code := runInitWithHome(t, home, "--target=claude-code"); code != 0 {
		t.Fatalf("first run exit %d", code)
	}
	first, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	stdout, _, code := runInitWithHome(t, home, "--target=claude-code")
	if code != 0 {
		t.Fatalf("second run exit %d", code)
	}
	if !strings.Contains(stdout, "already up to date") {
		t.Errorf("second run should report no-op: %q", stdout)
	}
	second, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if string(first) != string(second) {
		t.Errorf("idempotent run modified file:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestInitPreservesUserContent(t *testing.T) {
	home := tempHome(t)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	user := "// my custom settings\n{\"theme\":\"dark\"}\n"
	path := filepath.Join(home, ".claude", "settings.json")
	if err := os.WriteFile(path, []byte(user), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, code := runInitWithHome(t, home, "--target=claude-code"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	body, _ := os.ReadFile(path)
	s := string(body)
	if !strings.Contains(s, "my custom settings") {
		t.Errorf("user content was clobbered:\n%s", s)
	}
	if !strings.Contains(s, "// >>> verse-driven") {
		t.Errorf("snippet not appended:\n%s", s)
	}
}

func TestInitUninstall(t *testing.T) {
	home := tempHome(t)
	if _, _, code := runInitWithHome(t, home, "--target=claude-code"); code != 0 {
		t.Fatalf("install exit %d", code)
	}
	if _, _, code := runInitWithHome(t, home, "--target=claude-code", "--uninstall"); code != 0 {
		t.Fatalf("uninstall exit %d", code)
	}
	body, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if strings.Contains(string(body), "verse-driven") {
		t.Errorf("snippet not removed:\n%s", body)
	}
}

func TestInitCodexCreatesToml(t *testing.T) {
	home := tempHome(t)
	if _, _, code := runInitWithHome(t, home, "--target=codex"); code != 0 {
		t.Fatalf("exit %d", code)
	}
	body, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("config.toml not created: %v", err)
	}
	s := string(body)
	for _, want := range []string{
		"# >>> verse-driven",
		"# <<< verse-driven",
		"[mcp_servers.scripture]",
		"command = \"scripture-mcp\"",
		"[features]",
		"codex_hooks = true",
		"[[hooks.UserPromptSubmit]]",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("config.toml missing %q\n--- file ---\n%s", want, s)
		}
	}
}

func TestInitCodexRecapOnPrintsCdxAliasHint(t *testing.T) {
	home := tempHome(t)
	stdout, _, code := runInitWithHome(t, home, "--target=codex", "--recap=on")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{
		"cdx",
		"alias cdx=",
		"adapters/codex/wrapper/cdx",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("recap=on stdout should mention %q:\n%s", want, stdout)
		}
	}
}

func TestInitCodexRecapOffSkipsCdxAliasHint(t *testing.T) {
	home := tempHome(t)
	stdout, _, code := runInitWithHome(t, home, "--target=codex", "--recap=off")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(stdout, "alias cdx=") {
		t.Errorf("recap=off should not print cdx alias hint:\n%s", stdout)
	}
}

func TestInitMissingTarget(t *testing.T) {
	home := tempHome(t)
	_, errOut, code := runInitWithHome(t, home)
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(errOut, "--target") {
		t.Errorf("stderr should mention --target: %s", errOut)
	}
}

func TestInitDryRun(t *testing.T) {
	home := tempHome(t)
	stdout, _, code := runInitWithHome(t, home, "--target=claude-code", "--dry-run")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "would write") {
		t.Errorf("dry-run should report 'would write': %s", stdout)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json")); err == nil {
		t.Error("dry-run wrote a file")
	}
}
