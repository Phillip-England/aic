package aic_test

import (
	"testing"

	"github.com/phillip-england/aic/src/aic"
)

func TestTokenizePrompt_Table(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		types []aic.PromptTokenType
		lits  []string
	}{
		{
			name:  "raw only",
			in:    "hello world",
			types: []aic.PromptTokenType{aic.PromptTokenRaw},
			lits:  []string{"hello world"},
		},
		{
			name:  "@. at start",
			in:    "@.",
			types: []aic.PromptTokenType{aic.PromptTokenAt},
			lits:  []string{"@."},
		},
		{
			name:  "@. mid-line",
			in:    "hi @. there",
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenAt, aic.PromptTokenRaw},
			lits:  []string{"hi ", "@.", " there"},
		},
		{
			name:  "@. after newline",
			in:    "hi\n@.\nthere",
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenAt, aic.PromptTokenRaw},
			lits:  []string{"hi\n", "@.", "\nthere"},
		},
		{
			name:  "$clr() at start",
			in:    "$clr()",
			types: []aic.PromptTokenType{aic.PromptTokenDollar},
			lits:  []string{"$clr()"},
		},
		{
			name:  "$clr() mid-line",
			in:    "hi $clr() there",
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"hi ", "$clr()", " there"},
		},
		{
			name:  "not a token when not word-start",
			in:    "hello$clr()",
			types: []aic.PromptTokenType{aic.PromptTokenRaw},
			lits:  []string{"hello$clr()"},
		},
		{
			name:  "whitespace boundaries tabs/spaces",
			in:    "a\t$clr()  @.\nend",
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw, aic.PromptTokenAt, aic.PromptTokenRaw},
			lits:  []string{"a\t", "$clr()", "  ", "@.", "\nend"},
		},

		{
			name:  "$sh simple string",
			in:    `$sh("ls")`,
			types: []aic.PromptTokenType{aic.PromptTokenDollar},
			lits:  []string{`$sh("ls")`},
		},
		{
			name:  "$sh with spaces in string",
			in:    `x $sh("ls -la") y`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"x ", `$sh("ls -la")`, " y"},
		},
		{
			name:  "$sh with escaped quotes",
			in:    `x $sh("echo \"a(b)c\"") y`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"x ", `$sh("echo \"a(b)c\"")`, " y"},
		},
		{
			name:  "$sh missing close paren falls back to ws token",
			in:    `x $sh("echo hi" y`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"x ", `$sh("echo`, ` hi" y`},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			toks := aic.TokenizePrompt(tt.in)

			if len(toks) != len(tt.types) {
				t.Fatalf("expected %d tokens, got %d", len(tt.types), len(toks))
			}

			for i := range toks {
				if toks[i].Type() != tt.types[i] {
					t.Fatalf("token[%d] type: expected %v, got %v", i, tt.types[i], toks[i].Type())
				}
				if toks[i].Literal() != tt.lits[i] {
					t.Fatalf("token[%d] literal: expected %q, got %q", i, tt.lits[i], toks[i].Literal())
				}
			}

			var out string
			for _, tok := range toks {
				out += tok.Literal()
			}
			if out != tt.in {
				t.Fatalf("concat(literals) mismatch:\nexpected: %q\ngot:      %q", tt.in, out)
			}
		})
	}
}
