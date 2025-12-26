package aic_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phillip-england/aic/src/aic"
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
	root := filepath.Join(td, "project")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}

	oldwd, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	if _, err := aic.NewAiDir(false); err != nil {
		t.Fatalf("NewAiDir: %v", err)
	}

	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir deep: %v", err)
	}
	if err := os.Chdir(deep); err != nil {
		t.Fatalf("chdir deep: %v", err)
	}

	aiDir, err := aic.OpenAiDir()
	if err != nil {
		t.Fatalf("OpenAiDir: %v", err)
	}

	wantWorking := mustEvalSymlinksT(t, root)
	gotWorking := mustEvalSymlinksT(t, aiDir.WorkingDir)

	if gotWorking != wantWorking {
		t.Fatalf("WorkingDir mismatch:\nwant: %s\ngot:  %s", wantWorking, gotWorking)
	}

	wantRoot := filepath.Join(wantWorking, "ai")
	gotRoot := mustEvalSymlinksT(t, aiDir.Root)

	if gotRoot != mustEvalSymlinksT(t, wantRoot) {
		t.Fatalf("Root mismatch:\nwant: %s\ngot:  %s", wantRoot, gotRoot)
	}
}

func TestDollarToken_At_RejectsPathsThatEscapeProjectRootViaSymlink(t *testing.T) {
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

	aiDir, err := aic.NewAiDir(false)
	if err != nil {
		t.Fatalf("NewAiDir: %v", err)
	}

	outside := filepath.Join(td, "outside.txt")
	if err := os.WriteFile(outside, []byte("nope\n"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}

	linkPath := filepath.Join(root, "leak.txt")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	// Was: NewAtToken("@leak.txt")
	// Now: NewDollarToken(`$at("leak.txt")`)
	tok := aic.NewDollarToken(`$at("leak.txt")`)
	
	if err := tok.Validate(aiDir); err == nil {
		t.Fatalf("expected Validate to fail for symlink escaping project root, but got nil")
	}
}

func TestDollarToken_At_AllowsNormalPathsUnderProjectRoot(t *testing.T) {
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

	aiDir, err := aic.NewAiDir(false)
	if err != nil {
		t.Fatalf("NewAiDir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "ok.txt"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write ok.txt: %v", err)
	}

	// Was: NewAtToken("@ok.txt")
	tok := aic.NewDollarToken(`$at("ok.txt")`)
	
	if err := tok.Validate(aiDir); err != nil {
		t.Fatalf("expected Validate ok, got: %v", err)
	}
}