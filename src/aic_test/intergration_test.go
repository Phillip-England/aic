package aic_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/phillip-england/aic/src/aic"
)

func TestPromptReader_ValidateOrDowngrade_DowngradesInvalidDollarToken(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{
		WorkingDir: td,
		Root:       filepath.Join(td, "ai"),
	}

	pr := aic.NewPromptReader(`hello $path("missing") world`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	if len(pr.Tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(pr.Tokens))
	}

	if pr.Tokens[0].Type() != aic.PromptTokenRaw || pr.Tokens[0].Literal() != "hello " {
		t.Fatalf("token[0] unexpected: %v %q", pr.Tokens[0].Type(), pr.Tokens[0].Literal())
	}

	if pr.Tokens[1].Type() != aic.PromptTokenDollar || pr.Tokens[1].Literal() != `$path("missing")` {
		t.Fatalf("token[1] should remain Dollar, got: %v %q", pr.Tokens[1].Type(), pr.Tokens[1].Literal())
	}

	if pr.Tokens[2].Type() != aic.PromptTokenRaw || pr.Tokens[2].Literal() != " world" {
		t.Fatalf("token[2] unexpected: %v %q", pr.Tokens[2].Type(), pr.Tokens[2].Literal())
	}
}

func TestPromptReader_String_ReconstructsOriginalViaLiterals(t *testing.T) {
	in := `a $path(".") b $clear() c`
	pr := aic.NewPromptReader(in)
	if got := pr.String(); got != in {
		t.Fatalf("expected %q, got %q", in, got)
	}

	td := t.TempDir()
	_ = os.MkdirAll(filepath.Join(td, "ai"), 0o755)
	d := &aic.AiDir{WorkingDir: td, Root: filepath.Join(td, "ai")}

	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	if got := pr.String(); got != in {
		t.Fatalf("expected %q, got %q", in, got)
	}
}
