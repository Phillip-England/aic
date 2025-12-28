package aic_test

import (
	"testing"

	"github.com/phillip-england/aic/src/aic"
)

func TestTokenizePrompt_Table(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		want  string
		types []aic.PromptTokenType
		lits  []string
	}{
		{
			name:  "$type with modifier list",
			in:    `a $type("v", ["CONTROL"]) b`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"a ", `$type("v", ["CONTROL"])`, " b"},
		},
		{
			name:  "$type with delay",
			in:    `a $type("hello", ["SHIFT"], 15) b`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"a ", `$type("hello", ["SHIFT"], 15)`, " b"},
		},
		{
			name:  "$sleep with int ms",
			in:    `a $sleep(250) b`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"a ", `$sleep(250)`, " b"},
		},
		{
			name:  `$sleep with duration string`,
			in:    `a $sleep("750ms") b`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"a ", `$sleep("750ms")`, " b"},
		},
		{
			name:  `$press with key string`,
			in:    `a $press("BACKSPACE") b`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"a ", `$press("BACKSPACE")`, " b"},
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
			want := tt.want
			if want == "" {
				want = tt.in
			}
			if out != want {
				t.Fatalf("concat(literals) mismatch:\nexpected: %q\ngot:      %q", want, out)
			}
		})
	}
}
