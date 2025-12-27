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
			name:  "$path dot at start",
			in:    `$path(".")`,
			types: []aic.PromptTokenType{aic.PromptTokenDollar},
			lits:  []string{`$path(".")`},
		},
		{
			name:  "$path mid-line",
			in:    `hi $path("f.txt") there`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"hi ", `$path("f.txt")`, " there"},
		},
		{
			name:  "$clear() at start",
			in:    "$clear()",
			types: []aic.PromptTokenType{aic.PromptTokenDollar},
			lits:  []string{"$clear()"},
		},
		{
			name:  "$clear() mid-line",
			in:    "hi $clear() there",
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"hi ", "$clear()", " there"},
		},
		{
			name:  "not a token when not word-start",
			in:    "hello$clear()",
			types: []aic.PromptTokenType{aic.PromptTokenRaw},
			lits:  []string{"hello$clear()"},
		},
		{
			name:  "whitespace boundaries tabs/spaces",
			in:    "a\t$clear()  $path(\".\")\nend",
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"a\t", "$clear()", "  ", `$path(".")`, "\nend"},
		},
		{
			name:  "$shell simple string",
			in:    `$shell("ls")`,
			types: []aic.PromptTokenType{aic.PromptTokenDollar},
			lits:  []string{`$shell("ls")`},
		},
		{
			name:  "$shell with spaces in string",
			in:    `x $shell("ls -la") y`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"x ", `$shell("ls -la")`, " y"},
		},
		{
			name:  "$path with multiple args",
			in:    `$path("path", "to", "file")`,
			types: []aic.PromptTokenType{aic.PromptTokenDollar},
			lits:  []string{`$path("path", "to", "file")`},
		},
		{
			name:  "$http token",
			in:    `$http("google.com")`,
			types: []aic.PromptTokenType{aic.PromptTokenDollar},
			lits:  []string{`$http("google.com")`},
		},
		{
			name:  "$shell with escaped quotes",
			in:    `x $shell("echo \"a(b)c\"") y`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"x ", `$shell("echo \"a(b)c\"")`, " y"},
		},
		{
			name:  "$shell missing close paren falls back to ws token",
			in:    `x $shell("echo hi" y`,
			types: []aic.PromptTokenType{aic.PromptTokenRaw, aic.PromptTokenDollar, aic.PromptTokenRaw},
			lits:  []string{"x ", `$shell("echo`, ` hi" y`},
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
