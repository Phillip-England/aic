package aic

import "testing"

func TestTokenizePrompt_Table(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		types []PromptTokenType
		lits  []string
	}{
		{
			name:  "raw only",
			in:    "hello world",
			types: []PromptTokenType{PromptTokenRaw},
			lits:  []string{"hello world"},
		},
		{
			name:  "@. at start",
			in:    "@.",
			types: []PromptTokenType{PromptTokenAt},
			lits:  []string{"@."},
		},
		{
			name:  "@. mid-line",
			in:    "hi @. there",
			types: []PromptTokenType{PromptTokenRaw, PromptTokenAt, PromptTokenRaw},
			lits:  []string{"hi ", "@.", " there"},
		},
		{
			name:  "@. after newline",
			in:    "hi\n@.\nthere",
			types: []PromptTokenType{PromptTokenRaw, PromptTokenAt, PromptTokenRaw},
			lits:  []string{"hi\n", "@.", "\nthere"},
		},
		{
			name:  "$CLEAR at start",
			in:    "$CLEAR",
			types: []PromptTokenType{PromptTokenDollar},
			lits:  []string{"$CLEAR"},
		},
		{
			name:  "$CLEAR mid-line",
			in:    "hi $CLEAR there",
			types: []PromptTokenType{PromptTokenRaw, PromptTokenDollar, PromptTokenRaw},
			lits:  []string{"hi ", "$CLEAR", " there"},
		},
		{
			name:  "not a token when not word-start",
			in:    "hello$CLEAR",
			types: []PromptTokenType{PromptTokenRaw},
			lits:  []string{"hello$CLEAR"},
		},
		{
			name:  "whitespace boundaries tabs/spaces",
			in:    "a\t$CLEAR  @.\nend",
			types: []PromptTokenType{PromptTokenRaw, PromptTokenDollar, PromptTokenRaw, PromptTokenAt, PromptTokenRaw},
			lits:  []string{"a\t", "$CLEAR", "  ", "@.", "\nend"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			toks := TokenizePrompt(tt.in)

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

			// Invariant: concatenated literals reproduce input exactly
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
