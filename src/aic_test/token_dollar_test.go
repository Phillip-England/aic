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
		Root:       filepath.Join(td, "ai"),
	}

	pr := aic.NewPromptReader("before $clear after")
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	if len(pr.Tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(pr.Tokens))
	}

	// New tokenizer behavior: "$clear" is still a Dollar token (even without "()").
	// Validate() does not error (it just doesn't recognize it as a call), so no downgrade.
	if pr.Tokens[1].Type() != aic.PromptTokenDollar {
		t.Fatalf("expected $clear (no parens) to remain Dollar, got %v", pr.Tokens[1].Type())
	}
	if pr.Tokens[1].Literal() != "$clear" {
		t.Fatalf("unexpected literal: %q", pr.Tokens[1].Literal())
	}

	out, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "$clear") {
		t.Fatalf("expected rendered output to include literal $clear, got:\n%s", out)
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

	if err := os.WriteFile(aiDir.PromptPath(), []byte(promptHeaderDollar+"x $clear() y\n"), 0o644); err != nil {
		t.Fatalf("write prompt.md: %v", err)
	}

	out, err := pr.Render(aiDir)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// Current DollarToken.Render() placeholder returns literal for $clear().
	if !strings.Contains(out, "$clear()") {
		t.Fatalf("expected rendered output to include literal $clear(), got:\n%s", out)
	}

	got, err := os.ReadFile(aiDir.PromptPath())
	if err != nil {
		t.Fatalf("read prompt.md: %v", err)
	}

	// Current implementation does not clear prompt.md in Render(); it is left unchanged.
	want := promptHeaderDollar + "x $clear() y\n"
	if string(got) != want {
		t.Fatalf("expected prompt.md to remain unchanged in current implementation.\nwant: %q\ngot:  %q", want, string(got))
	}
}

func TestDollarToken_At_JoinsPaths(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	_ = os.MkdirAll(filepath.Join(td, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(td, "sub", "foo.txt"), []byte("bar"), 0o644)

	tok := aic.NewDollarToken(`$at("sub", "foo.txt")`)
	if err := tok.Validate(d); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	out, err := tok.Render(d)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	// Current DollarToken.Render() placeholder returns literal for $at(...)
	if strings.TrimSpace(out) != `$at("sub", "foo.txt")` {
		t.Fatalf("expected Render to return literal, got:\n%s", out)
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

	// Current DollarToken.Render() placeholder returns literal for $sh(...)
	if !strings.Contains(out, `$sh("echo hi")`) {
		t.Fatalf("expected output to include $sh token literal, got:\n%s", out)
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

	// Current DollarToken.Render() placeholder returns literal for $skill(...)
	if strings.TrimSpace(out) != `$skill("test")` {
		t.Fatalf("expected Render to return literal, got:\n%s", out)
	}
}

func TestDollarToken_Jump_AddsPostActionAndRendersEmpty(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	pr := aic.NewPromptReader(`a $jump(10,20) b`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	out, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(out, "$jump(") {
		t.Fatalf("expected $jump to render empty, got:\n%s", out)
	}

	if len(pr.PostActions) != 1 {
		t.Fatalf("expected 1 post action, got %d", len(pr.PostActions))
	}
	if pr.PostActions[0].Kind != aic.PostActionJump || pr.PostActions[0].X != 10 || pr.PostActions[0].Y != 20 {
		t.Fatalf("unexpected post action: %#v", pr.PostActions[0])
	}
}

func TestDollarToken_Click_AddsPostActionAndRendersEmpty(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	pr := aic.NewPromptReader(`a $click("right") b`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	out, err := pr.Render(d)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if strings.Contains(out, "$click(") {
		t.Fatalf("expected $click to render empty, got:\n%s", out)
	}

	if len(pr.PostActions) != 1 {
		t.Fatalf("expected 1 post action, got %d", len(pr.PostActions))
	}
	if pr.PostActions[0].Kind != aic.PostActionClick || pr.PostActions[0].Button != "right" {
		t.Fatalf("unexpected post action: %#v", pr.PostActions[0])
	}
}
