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
