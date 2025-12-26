package aic

import (
	"path/filepath"
	"testing"

	"github.com/phillip-england/aic/src/aic"
)

func TestPromptReader_ValidateOrDowngrade_DowngradesInvalidDollarToken_ReaderTest(t *testing.T) {
	td := t.TempDir()
	d := &aic.AiDir{
		WorkingDir: td,
		Root:        filepath.Join(td, "ai"),
	}

	// Changed @missing to $at("missing")
	pr := aic.NewPromptReader(`hello $at("missing") world`)
	pr.ValidateOrDowngrade(d)
	pr.BindTokens()

	if len(pr.Tokens) != 3 {
		t.Fatalf("expected 3 tokens, got %d", len(pr.Tokens))
	}

	if pr.Tokens[1].Type() != aic.PromptTokenRaw {
		t.Fatalf("token[1] should be downgraded to Raw, got: %v", pr.Tokens[1].Type())
	}
	if pr.Tokens[1].Literal() != `$at("missing")` {
		t.Fatalf("token[1] literal mismatch, got: %q", pr.Tokens[1].Literal())
	}
}