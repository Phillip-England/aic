package aic_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillip-england/aic/src/aic"
)

const promptHeaderDollar = `---
---
`

func TestDollarToken_Validate_ClearRequiresParens(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{
		WorkingDir: td,
		Root:        filepath.Join(td, "ai"),
	}

	// $clear without parens should fail strict check for keywords
	pr := aic.NewPromptReader("before $clear after")
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	if len(pr.Tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(pr.Tokens))
	}
	if pr.Tokens[1].Type() != aic.PromptTokenRaw {
		t.Fatalf("expected $clear to be downgraded to Raw when missing (), got %v", pr.Tokens[1].Type())
	}
	if pr.Tokens[1].Literal() != "$clear" {
		t.Fatalf("unexpected literal: %q", pr.Tokens[1].Literal())
	}
}

func TestDollarToken_Validate_ClearAcceptsEmptyCall(t *testing.T) {
	td := t.TempDir()
	oldwd, _ := os.Getwd()
	if err := os.Chdir(td); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	aiDir, err := aic.NewAiDir(false)
	if err != nil {
		t.Fatalf("NewAiDir: %v", err)
	}

	pr := aic.NewPromptReader("before $clear() after")
	pr.ValidateOrDowngrade(aiDir)
	pr.BindTokens()

	if len(pr.Tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(pr.Tokens))
	}
	if pr.Tokens[1].Type() != aic.PromptTokenDollar {
		t.Fatalf("expected $clear() to remain Dollar token, got %v", pr.Tokens[1].Type())
	}

	// Prepare prompt file to be cleared
	if err := os.WriteFile(aiDir.PromptPath(), []byte(promptHeaderDollar+"x $clear() y\n"), 0o644); err != nil {
		t.Fatalf("write prompt.md: %v", err)
	}

	out, err := pr.Render(aiDir)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// Output of $clear() is empty string, side effect is clearing file
	if strings.Contains(out, "$clear") {
		t.Fatalf("expected rendered output not to include $clear, got:\n%s", out)
	}

	got, err := os.ReadFile(aiDir.PromptPath())
	if err != nil {
		t.Fatalf("read prompt.md: %v", err)
	}
	if string(got) != promptHeaderDollar {
		t.Fatalf("expected prompt.md to be cleared to header.\nwant: %q\ngot:  %q", promptHeaderDollar, string(got))
	}
}

func TestDollarToken_At_JoinsPaths(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}
	
	// Create subfolder and file
	_ = os.MkdirAll(filepath.Join(td, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(td, "sub", "foo.txt"), []byte("bar"), 0o644)

	// Test joining args
	tok := aic.NewDollarToken(`$at("sub", "foo.txt")`)
	if err := tok.Validate(d); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	out, err := tok.Render(d)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(out, "FILE:") || !strings.Contains(out, "bar") {
		t.Fatalf("Expected content 'bar', got:\n%s", out)
	}
}

func TestDollarToken_Sh_ExecutesCommand(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}
	
	pr := aic.NewPromptReader(`before $sh("echo hi") after`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()
	
	out, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(out, "$sh(") {
		t.Fatalf("expected output to NOT include $sh token literal, got:\n%s", out)
	}
	if !strings.Contains(out, "hi") {
		t.Fatalf("expected output to include command output 'hi', got:\n%s", out)
	}
}

func TestDollarToken_Skill_RequiresConfig(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai"), Skills: filepath.Join(td, "ai", "skills")}
	_ = os.MkdirAll(d.Skills, 0o755)
	_ = os.WriteFile(filepath.Join(d.Skills, "test.md"), []byte("skill body"), 0o644)

	tok := aic.NewDollarToken(`$skill("test")`)
	if err := tok.Validate(d); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	out, err := tok.Render(d)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(out, "skill body") {
		t.Fatalf("Expected 'skill body', got: %s", out)
	}
}