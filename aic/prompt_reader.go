package aic

import (
	"fmt"
	"strings"
)

type PromptReader struct {
	Text   string
	Tokens []PromptToken
}

func NewPromptReader(text string) *PromptReader {
	toks := TokenizePrompt(text)
	return &PromptReader{
		Text:   text,
		Tokens: toks,
	}
}

// ValidateOrDowngrade runs validation for each token using AiDir context.
// If validation fails, token becomes Raw with same literal.
func (p *PromptReader) ValidateOrDowngrade(d *AiDir) {
	if len(p.Tokens) == 0 {
		return
	}
	out := make([]PromptToken, 0, len(p.Tokens))
	for _, tok := range p.Tokens {
		if err := tok.Validate(d); err != nil {
			out = append(out, NewRawToken(tok.Literal()))
			continue
		}
		out = append(out, tok)
	}
	p.Tokens = out
}

// BindTokens attaches this PromptReader + index to each token via AfterValidate.
func (p *PromptReader) BindTokens() {
	for i, tok := range p.Tokens {
		_ = tok.AfterValidate(p, i)
	}
}

// Render iterates tokens and concatenates each token's rendered output.
func (p *PromptReader) Render(d *AiDir) (string, error) {
	var sb strings.Builder
	for _, tok := range p.Tokens {
		part, err := tok.Render(d)
		if err != nil {
			return "", err
		}
		sb.WriteString(part)
	}
	return sb.String(), nil
}

func (p *PromptReader) RemoveToken(i int) bool {
	if i < 0 || i >= len(p.Tokens) {
		return false
	}
	p.Tokens = append(p.Tokens[:i], p.Tokens[i+1:]...)
	p.BindTokens()
	return true
}

func (p *PromptReader) InsertToken(i int, tok PromptToken) {
	if i < 0 {
		i = 0
	}
	if i > len(p.Tokens) {
		i = len(p.Tokens)
	}
	p.Tokens = append(p.Tokens[:i], append([]PromptToken{tok}, p.Tokens[i:]...)...)
	p.BindTokens()
}

func (p *PromptReader) ReplaceToken(i int, tok PromptToken) bool {
	if i < 0 || i >= len(p.Tokens) {
		return false
	}
	p.Tokens[i] = tok
	p.BindTokens()
	return true
}

func (p *PromptReader) String() string {
	if len(p.Tokens) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, tok := range p.Tokens {
		sb.WriteString(tok.Literal())
	}
	return sb.String()
}

func (p *PromptReader) Print() {
	fmt.Print(p.String() + "\n")
}
