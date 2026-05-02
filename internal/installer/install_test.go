// Package installer holds tests that drive the top-level install.sh
// end-to-end against a fake release tarball. We don't ship Go code for
// the installer itself — install.sh is the contract — but the test lives
// in a Go package so it runs under `go test ./...` in CI.
package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot returns the absolute path of the repository root, located
// from this test file. We hop two parents up: internal/installer → repo.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(wd, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "install.sh")); err != nil {
		t.Fatalf("expected install.sh at %s: %v", root, err)
	}
	return root
}

// buildScriptureMCP compiles the real binary into outDir. We use the
// real binary (not a stub) so init writes the canonical snippets and
// the test doubles as a smoke check that install.sh + init agree.
func buildScriptureMCP(t *testing.T, outDir string) string {
	t.Helper()
	root := repoRoot(t)
	bin := filepath.Join(outDir, "scripture-mcp")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/scripture-mcp")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// makeArchive packs binPath into a .tar.gz at archivePath, with the
// binary stored at the top level as `scripture-mcp` (the path install.sh
// expects to find after extraction).
func makeArchive(t *testing.T, binPath, archivePath string) {
	t.Helper()
	body, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read binary: %v", err)
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{
		Name: "scripture-mcp",
		Mode: 0o755,
		Size: int64(len(body)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	if err := os.WriteFile(archivePath, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write archive: %v", err)
	}
}

// writeStub writes a minimal executable script to dir/name so that
// command -v <name> finds it on PATH. We don't care what it does; the
// installer only needs to detect its presence.
func writeStub(t *testing.T, dir, name string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write stub %s: %v", name, err)
	}
}

// runInstall invokes install.sh with the supplied args, in an
// environment where HOME, PATH, and the scratch dirs are sandboxed so
// the test can never touch the user's real config.
func runInstall(t *testing.T, root, fakeHome, agentDir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("bash", append([]string{filepath.Join(root, "install.sh")}, args...)...)
	cmd.Env = append(os.Environ(),
		"HOME="+fakeHome,
		// PATH carries only the agent stubs and core utilities; we
		// deliberately omit any system-wide claude/codex so detection
		// only sees what the test put in agentDir.
		"PATH="+agentDir+":/usr/bin:/bin:/usr/local/bin",
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestInstallScriptHelp(t *testing.T) {
	root := repoRoot(t)
	out, err := exec.Command("bash", filepath.Join(root, "install.sh"), "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh --help: %v\n%s", err, out)
	}
	for _, want := range []string{"--version", "--prefix", "--from-archive", "--no-wire", "--uninstall"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("--help output missing %q\n%s", want, out)
		}
	}
}

func TestInstallScriptRejectsUnknownFlag(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("bash", filepath.Join(root, "install.sh"), "--bogus")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit for --bogus, got success\n%s", out)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2 for unknown flag, got %v\n%s", err, out)
	}
}

// TestInstallEndToEnd is the central integration test. It builds the
// real scripture-mcp binary, packs it into a tarball, then drives
// install.sh through three phases:
//
//  1. install + wire (with stub agents on PATH) → binary in place,
//     ~/.claude/settings.json and ~/.codex/config.toml gain the
//     verse-driven block.
//  2. re-run → upgrade is idempotent on the binary; init reports
//     "already up to date" for both configs.
//  3. uninstall → init --uninstall strips the snippets, binary removed.
func TestInstallEndToEnd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is POSIX bash; Windows is out of scope for issue #7")
	}
	root := repoRoot(t)

	scratch := t.TempDir()
	binSrcDir := filepath.Join(scratch, "src")
	if err := os.MkdirAll(binSrcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binPath := buildScriptureMCP(t, binSrcDir)

	archive := filepath.Join(scratch, "scripture-mcp.tar.gz")
	makeArchive(t, binPath, archive)

	prefix := filepath.Join(scratch, "bin")
	fakeHome := filepath.Join(scratch, "home")
	agentDir := filepath.Join(scratch, "agents")
	if err := os.MkdirAll(fakeHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStub(t, agentDir, "claude")
	writeStub(t, agentDir, "codex")

	args := []string{
		"--from-archive", archive,
		"--prefix", prefix,
		"--yes",
	}

	// Phase 1: fresh install.
	out, err := runInstall(t, root, fakeHome, agentDir, args...)
	if err != nil {
		t.Fatalf("install run 1: %v\n%s", err, out)
	}
	installedBin := filepath.Join(prefix, "scripture-mcp")
	st, err := os.Stat(installedBin)
	if err != nil {
		t.Fatalf("expected binary at %s: %v\n%s", installedBin, err, out)
	}
	if st.Mode().Perm()&0o111 == 0 {
		t.Errorf("expected installed binary to be executable, got mode %v", st.Mode())
	}

	claudeCfg := filepath.Join(fakeHome, ".claude", "settings.json")
	codexCfg := filepath.Join(fakeHome, ".codex", "config.toml")
	mustContain(t, claudeCfg, ">>> verse-driven", "scripture-mcp")
	mustContain(t, codexCfg, ">>> verse-driven", "scripture-mcp")

	// Phase 2: re-run is idempotent on configs and replaces the binary.
	out2, err := runInstall(t, root, fakeHome, agentDir, args...)
	if err != nil {
		t.Fatalf("install run 2: %v\n%s", err, out2)
	}
	if !strings.Contains(out2, "already up to date") {
		t.Errorf("expected re-run to report 'already up to date', got:\n%s", out2)
	}
	// Binary should still exist after the re-run.
	if _, err := os.Stat(installedBin); err != nil {
		t.Errorf("binary missing after re-run: %v", err)
	}

	// Phase 3: uninstall strips the snippets and removes the binary.
	out3, err := runInstall(t, root, fakeHome, agentDir, "--prefix", prefix, "--uninstall")
	if err != nil {
		t.Fatalf("install --uninstall: %v\n%s", err, out3)
	}
	if _, err := os.Stat(installedBin); !os.IsNotExist(err) {
		t.Errorf("expected binary to be removed, got err=%v", err)
	}
	mustNotContain(t, claudeCfg, ">>> verse-driven")
	mustNotContain(t, codexCfg, ">>> verse-driven")
}

// TestInstallNoAgentsDetected covers the empty-PATH case: no claude or
// codex on PATH, the installer still places the binary and reports
// nothing to wire.
func TestInstallNoAgentsDetected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is POSIX bash")
	}
	root := repoRoot(t)
	scratch := t.TempDir()

	binSrcDir := filepath.Join(scratch, "src")
	if err := os.MkdirAll(binSrcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binPath := buildScriptureMCP(t, binSrcDir)
	archive := filepath.Join(scratch, "scripture-mcp.tar.gz")
	makeArchive(t, binPath, archive)

	prefix := filepath.Join(scratch, "bin")
	fakeHome := filepath.Join(scratch, "home")
	emptyAgentDir := filepath.Join(scratch, "noagents") // exists but empty
	if err := os.MkdirAll(fakeHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(emptyAgentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	out, err := runInstall(t, root, fakeHome, emptyAgentDir,
		"--from-archive", archive, "--prefix", prefix, "--yes")
	if err != nil {
		t.Fatalf("install: %v\n%s", err, out)
	}
	if !strings.Contains(out, "no supported agents detected") {
		t.Errorf("expected 'no supported agents detected' message, got:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(prefix, "scripture-mcp")); err != nil {
		t.Errorf("binary should still be installed when no agents present: %v", err)
	}
	// No config dirs should have been created.
	if _, err := os.Stat(filepath.Join(fakeHome, ".claude")); !os.IsNotExist(err) {
		t.Errorf("expected no ~/.claude dir, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(fakeHome, ".codex")); !os.IsNotExist(err) {
		t.Errorf("expected no ~/.codex dir, got err=%v", err)
	}
}

// TestInstallNoWireSkipsAgents verifies --no-wire installs the binary
// without touching any agent config, even when claude/codex are on PATH.
func TestInstallNoWireSkipsAgents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is POSIX bash")
	}
	root := repoRoot(t)
	scratch := t.TempDir()

	binSrcDir := filepath.Join(scratch, "src")
	if err := os.MkdirAll(binSrcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binPath := buildScriptureMCP(t, binSrcDir)
	archive := filepath.Join(scratch, "scripture-mcp.tar.gz")
	makeArchive(t, binPath, archive)

	prefix := filepath.Join(scratch, "bin")
	fakeHome := filepath.Join(scratch, "home")
	agentDir := filepath.Join(scratch, "agents")
	if err := os.MkdirAll(fakeHome, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeStub(t, agentDir, "claude")
	writeStub(t, agentDir, "codex")

	out, err := runInstall(t, root, fakeHome, agentDir,
		"--from-archive", archive, "--prefix", prefix, "--no-wire")
	if err != nil {
		t.Fatalf("install --no-wire: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(fakeHome, ".claude")); !os.IsNotExist(err) {
		t.Errorf("--no-wire should not create ~/.claude, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(fakeHome, ".codex")); !os.IsNotExist(err) {
		t.Errorf("--no-wire should not create ~/.codex, got err=%v", err)
	}
}

// TestInstallMissingArchive verifies the script fails fast when
// --from-archive points at a file that doesn't exist.
func TestInstallMissingArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("install.sh is POSIX bash")
	}
	root := repoRoot(t)
	scratch := t.TempDir()
	cmd := exec.Command("bash", filepath.Join(root, "install.sh"),
		"--from-archive", filepath.Join(scratch, "nope.tar.gz"),
		"--prefix", filepath.Join(scratch, "bin"),
		"--no-wire")
	cmd.Env = append(os.Environ(), "HOME="+scratch, "PATH=/usr/bin:/bin")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure for missing archive, got success\n%s", out)
	}
	if !strings.Contains(string(out), "file not found") {
		t.Errorf("expected 'file not found' diagnostic, got:\n%s", out)
	}
}

func mustContain(t *testing.T, path string, fragments ...string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, frag := range fragments {
		if !strings.Contains(string(b), frag) {
			t.Errorf("%s does not contain %q", path, frag)
		}
	}
}

func mustNotContain(t *testing.T, path, fragment string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		// File missing also satisfies "doesn't contain".
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(b), fragment) {
		t.Errorf("%s should no longer contain %q", path, fragment)
	}
}
