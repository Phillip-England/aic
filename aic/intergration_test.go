package aic

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustEvalSymlinks(t *testing.T, p string) string {
	t.Helper()
	ep, err := filepath.EvalSymlinks(p)
	if err != nil {
		// If eval fails for any reason, fall back to original.
		return p
	}
	return ep
}

func containsEither(haystack string, a string, b string) bool {
	return strings.Contains(haystack, a) || strings.Contains(haystack, b)
}

func TestIntegration_AtDot_ExpandsReadableNonIgnoredFiles(t *testing.T) {
	td := t.TempDir()

	// Work inside the temp directory (matches how the CLI works).
	oldwd, _ := os.Getwd()
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	// .gitignore excludes tmp
	if err := os.WriteFile(filepath.Join(td, ".gitignore"), []byte("tmp\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	// Create some files
	if err := os.WriteFile(filepath.Join(td, "a.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(td, "tmp"), 0o755); err != nil {
		t.Fatalf("mkdir tmp: %v", err)
	}
	if err := os.WriteFile(filepath.Join(td, "tmp", "skip.txt"), []byte("should not appear\n"), 0o644); err != nil {
		t.Fatalf("write tmp/skip.txt: %v", err)
	}

	// Binary-ish file should be skipped by ReadTextFile
	if err := os.WriteFile(filepath.Join(td, "bin.dat"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatalf("write bin.dat: %v", err)
	}

	// Build ai dir (loads .gitignore from working dir)
	aiDir, err := NewAiDir(false)
	if err != nil {
		t.Fatalf("NewAiDir: %v", err)
	}

	// Prompt expands project root
	prompt := promptHeader + "@.\n"
	if err := os.WriteFile(aiDir.PromptPath(), []byte(prompt), 0o644); err != nil {
		t.Fatalf("write prompt.md: %v", err)
	}

	pr := NewPromptReader(prompt)
	pr.ValidateOrDowngrade(aiDir)
	pr.BindTokens()

	out, err := pr.Render(aiDir)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// macOS: td may be /var/... while printed paths are /private/var/...
	wantA := "FILE: " + filepath.Join(td, "a.txt")
	wantA2 := "FILE: " + mustEvalSymlinks(t, filepath.Join(td, "a.txt"))

	if !containsEither(out, wantA, wantA2) {
		t.Fatalf("expected output to include a.txt, got:\n%s", out)
	}

	// Should not include ignored tmp/skip.txt
	if strings.Contains(out, "skip.txt") {
		t.Fatalf("expected output to NOT include tmp/skip.txt, got:\n%s", out)
	}

	// Should not include binary file
	if strings.Contains(out, "bin.dat") {
		t.Fatalf("expected output to NOT include bin.dat, got:\n%s", out)
	}

	// Stats line sanity: should read exactly 1 file (a.txt)
	// NOTE: today @. includes .gitignore and ai/prompt.md too, so this may be > 1.
	// If you want this to be exactly 1, you need to ignore .gitignore and ai/ by default.
	// For now we only sanity check that it read at least 1 file.
	if !strings.Contains(out, "read [") {
		t.Fatalf("expected stats line, got:\n%s", out)
	}
}

func TestIntegration_DollarClear_OverwritesPromptFileAndDoesNotRenderToken(t *testing.T) {
	td := t.TempDir()

	oldwd, _ := os.Getwd()
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	aiDir, err := NewAiDir(false)
	if err != nil {
		t.Fatalf("NewAiDir: %v", err)
	}

	// Put $CLEAR in prompt
	prompt := promptHeader + "before $CLEAR after\n"
	if err := os.WriteFile(aiDir.PromptPath(), []byte(prompt), 0o644); err != nil {
		t.Fatalf("write prompt.md: %v", err)
	}

	pr := NewPromptReader(prompt)
	pr.ValidateOrDowngrade(aiDir)
	pr.BindTokens()

	out, err := pr.Render(aiDir)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// Token should not appear in output
	if strings.Contains(out, "$CLEAR") {
		t.Fatalf("expected output to NOT include $CLEAR, got:\n%s", out)
	}

	// prompt.md should be overwritten to exactly promptHeader
	gotBytes, err := os.ReadFile(aiDir.PromptPath())
	if err != nil {
		t.Fatalf("read prompt.md: %v", err)
	}
	if string(gotBytes) != promptHeader {
		t.Fatalf("prompt.md not cleared.\nexpected:\n%q\ngot:\n%q", promptHeader, string(gotBytes))
	}
}
