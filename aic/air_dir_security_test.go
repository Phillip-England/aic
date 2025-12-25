package aic

import (
	"os"
	"path/filepath"
	"testing"
)

func mustEvalSymlinksT(t *testing.T, p string) string {
	t.Helper()
	ep, err := filepath.EvalSymlinks(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return filepath.Clean(ep)
}

func TestOpenAiDir_FindsAiDirByWalkingUpAndSetsWorkingDirToProjectRoot(t *testing.T) {
	td := t.TempDir()

	// project root
	root := filepath.Join(td, "project")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}

	// create ai dir at project root
	oldwd, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	if _, err := NewAiDir(false); err != nil {
		t.Fatalf("NewAiDir: %v", err)
	}

	// now chdir deep into project and OpenAiDir should still find projectRoot/ai
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir deep: %v", err)
	}
	if err := os.Chdir(deep); err != nil {
		t.Fatalf("chdir deep: %v", err)
	}

	aiDir, err := OpenAiDir()
	if err != nil {
		t.Fatalf("OpenAiDir: %v", err)
	}

	// WorkingDir must be the directory that contains "ai/"
	wantWorking := mustEvalSymlinksT(t, root)
	gotWorking := mustEvalSymlinksT(t, aiDir.WorkingDir)
	if gotWorking != wantWorking {
		t.Fatalf("WorkingDir mismatch:\nwant: %s\ngot:  %s", wantWorking, gotWorking)
	}

	// Root must be WorkingDir/ai
	wantRoot := filepath.Join(wantWorking, "ai")
	gotRoot := mustEvalSymlinksT(t, aiDir.Root)
	if gotRoot != mustEvalSymlinksT(t, wantRoot) {
		t.Fatalf("Root mismatch:\nwant: %s\ngot:  %s", wantRoot, gotRoot)
	}
}

func TestAtTokenValidate_RejectsPathsThatEscapeProjectRootViaSymlink(t *testing.T) {
	td := t.TempDir()

	// project root + ai
	root := filepath.Join(td, "project")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	oldwd, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	aiDir, err := NewAiDir(false)
	if err != nil {
		t.Fatalf("NewAiDir: %v", err)
	}

	// create a file OUTSIDE the project root
	outside := filepath.Join(td, "outside.txt")
	if err := os.WriteFile(outside, []byte("nope\n"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}

	// create a symlink INSIDE the project pointing OUTSIDE
	linkPath := filepath.Join(root, "leak.txt")
	if err := os.Symlink(outside, linkPath); err != nil {
		// If symlinks are disallowed in env, skip (rare, but safe).
		t.Skipf("symlink not supported: %v", err)
	}

	// Validate should reject @leak.txt because it resolves outside WorkingDir.
	tok := NewAtToken("@leak.txt")
	if err := tok.Validate(aiDir); err == nil {
		t.Fatalf("expected Validate to fail for symlink escaping project root, but got nil")
	}
}

func TestAtTokenValidate_AllowsNormalPathsUnderProjectRoot(t *testing.T) {
	td := t.TempDir()

	root := filepath.Join(td, "project")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}
	oldwd, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	aiDir, err := NewAiDir(false)
	if err != nil {
		t.Fatalf("NewAiDir: %v", err)
	}

	// normal file under root
	if err := os.WriteFile(filepath.Join(root, "ok.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write ok.txt: %v", err)
	}

	tok := NewAtToken("@ok.txt")
	if err := tok.Validate(aiDir); err != nil {
		t.Fatalf("expected Validate ok, got: %v", err)
	}
}
