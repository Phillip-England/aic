package aic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPromptReader_ValidateOrDowngrade_DowngradesInvalidAtToken(t *testing.T) {
	td := t.TempDir()

	// Minimal AiDir context for validation
	d := &AiDir{
		WorkingDir: td,
		Root:       filepath.Join(td, "ai"),
	}

	// @missing should fail validation and be downgraded to Raw.
	pr := NewPromptReader("hello @missing world")
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	if len(pr.Tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(pr.Tokens))
	}

	if pr.Tokens[0].Type() != PromptTokenRaw || pr.Tokens[0].Literal() != "hello " {
		t.Fatalf("token[0] unexpected: %v %q", pr.Tokens[0].Type(), pr.Tokens[0].Literal())
	}

	if pr.Tokens[1].Type() != PromptTokenRaw || pr.Tokens[1].Literal() != "@missing" {
		t.Fatalf("token[1] should be downgraded to Raw '@missing', got: %v %q", pr.Tokens[1].Type(), pr.Tokens[1].Literal())
	}

	if pr.Tokens[2].Type() != PromptTokenRaw || pr.Tokens[2].Literal() != " world" {
		t.Fatalf("token[2] unexpected: %v %q", pr.Tokens[2].Type(), pr.Tokens[2].Literal())
	}
}

func TestPromptReader_String_ReconstructsOriginalViaLiterals(t *testing.T) {
	in := "a @. b $CLEAR c"
	pr := NewPromptReader(in)

	// String() is concatenation of token literals (no validation required)
	if got := pr.String(); got != in {
		t.Fatalf("expected %q, got %q", in, got)
	}

	// After validation/bind should still reconstruct original literals (token content not mutated)
	td := t.TempDir()
	_ = os.MkdirAll(filepath.Join(td, "ai"), 0o755)
	d := &AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	if got := pr.String(); got != in {
		t.Fatalf("expected %q, got %q", in, got)
	}
}
