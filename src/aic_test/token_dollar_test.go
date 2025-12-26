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

func TestDollarToken_Validate_ClrRequiresParens(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{
		WorkingDir: td,
		Root:       filepath.Join(td, "ai"),
	}

	pr := aic.NewPromptReader("before $clr after")
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	if len(pr.Tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(pr.Tokens))
	}
	// $clr (no parens) must be downgraded
	if pr.Tokens[1].Type() != aic.PromptTokenRaw {
		t.Fatalf("expected $clr to be downgraded to Raw when missing (), got %v", pr.Tokens[1].Type())
	}
	if pr.Tokens[1].Literal() != "$clr" {
		t.Fatalf("unexpected literal: %q", pr.Tokens[1].Literal())
	}
}

func TestDollarToken_Validate_ClrAcceptsEmptyCall(t *testing.T) {
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

	pr := aic.NewPromptReader("before $clr() after")
	pr.ValidateOrDowngrade(aiDir)
	pr.BindTokens()

	if len(pr.Tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(pr.Tokens))
	}
	if pr.Tokens[1].Type() != aic.PromptTokenDollar {
		t.Fatalf("expected $clr() to remain Dollar token, got %v", pr.Tokens[1].Type())
	}

	// Rendering should clear the prompt file and not output the token text.
	// Put something in prompt.md so we can observe it being overwritten.
	// We append content *after* the header to simulate usage.
	if err := os.WriteFile(aiDir.PromptPath(), []byte(promptHeaderDollar+"x $clr() y\n"), 0o644); err != nil {
		t.Fatalf("write prompt.md: %v", err)
	}

	out, err := pr.Render(aiDir)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(out, "$clr") {
		t.Fatalf("expected rendered output not to include $clr, got:\n%s", out)
	}

	got, err := os.ReadFile(aiDir.PromptPath())
	if err != nil {
		t.Fatalf("read prompt.md: %v", err)
	}

	// It should revert to just the header (because the separator was present in promptHeaderDollar,
	// and we cleared everything after it).
	if string(got) != promptHeaderDollar {
		t.Fatalf("expected prompt.md to be cleared to header.\nwant: %q\ngot:  %q", promptHeaderDollar, string(got))
	}
}

func TestDollarToken_Validate_ShRequiresStringArg(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{
		WorkingDir: td,
		Root:       filepath.Join(td, "ai"),
	}

	pr := aic.NewPromptReader("before $sh(ls) after")
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	if len(pr.Tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(pr.Tokens))
	}

	// $sh(ls) must be downgraded (not a string)
	if pr.Tokens[1].Type() != aic.PromptTokenRaw {
		t.Fatalf("expected $sh(ls) to be downgraded to Raw, got %v", pr.Tokens[1].Type())
	}
	if pr.Tokens[1].Literal() != "$sh(ls)" {
		t.Fatalf("unexpected literal: %q", pr.Tokens[1].Literal())
	}
}

func TestDollarToken_Render_ShExecutesCommand(t *testing.T) {
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

	pr := aic.NewPromptReader(`before $sh("echo hi") after`)
	pr.ValidateOrDowngrade(aiDir)
	pr.BindTokens()

	out, err := pr.Render(aiDir)
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

func TestDollarToken_Render_ShDecodesEscapes(t *testing.T) {
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

	pr := aic.NewPromptReader(`before $sh("echo \"hi\"") after`)
	pr.ValidateOrDowngrade(aiDir)
	pr.BindTokens()

	out, err := pr.Render(aiDir)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(out, "$sh(") {
		t.Fatalf("expected output to NOT include $sh token literal, got:\n%s", out)
	}
	// echo "hi" should produce hi
	if !strings.Contains(out, "hi") {
		t.Fatalf("expected decoded output to include hi, got:\n%s", out)
	}
}
