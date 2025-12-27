package aic

import (
	"fmt"
	"strings"
)

type PostActionKind string

const (
	PostActionJump  PostActionKind = "jump"
	PostActionClick PostActionKind = "click"
)

type PostAction struct {
	Kind  PostActionKind
	Index int
	Lit   string

	X int
	Y int

	Button string
}

type PromptReader struct {
	Text        string
	Tokens      []PromptToken
	PostActions []PostAction
}

func NewPromptReader(text string) *PromptReader {
	toks := TokenizePrompt(text)
	return &PromptReader{
		Text:   text,
		Tokens: toks,
	}
}

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

func (p *PromptReader) BindTokens() {
	for i, tok := range p.Tokens {
		_ = tok.AfterValidate(p, i)
	}
}

func (p *PromptReader) AddPostAction(a PostAction) {
	p.PostActions = append(p.PostActions, a)
}

func (p *PromptReader) Render(d *AiDir) (string, error) {
	p.PostActions = nil

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
