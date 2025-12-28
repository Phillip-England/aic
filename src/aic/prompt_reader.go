package aic

import (
	"fmt"
	"strings"
)

type PostActionKind string

const (
	PostActionJump  PostActionKind = "jump"
	PostActionClick PostActionKind = "click"
	PostActionType  PostActionKind = "type"
	PostActionClear PostActionKind = "clear"
)

type PostActionPhase string

const (
	PostActionBefore PostActionPhase = "before"
	PostActionAfter  PostActionPhase = "after"
)

type PostAction struct {
	Phase  PostActionPhase
	Kind   PostActionKind
	Index  int
	Lit    string
	X      int
	Y      int
	XExpr  string
	YExpr  string
	Button string

	Text    string
	Mods    []string
	DelayMs int
}

type PromptReader struct {
	Text        string
	Tokens      []PromptToken
	PostActions []PostAction
	Vars        map[string]string
}

func NewPromptReader(text string) *PromptReader {
	toks := TokenizePrompt(text)
	return &PromptReader{
		Text:   text,
		Tokens: toks,
		Vars:   make(map[string]string),
	}
}

func (p *PromptReader) SetVar(key, val string) {
	if p.Vars == nil {
		p.Vars = make(map[string]string)
	}
	p.Vars[key] = val
}

func (p *PromptReader) GetVar(key string) (string, bool) {
	if p.Vars == nil {
		return "", false
	}
	v, ok := p.Vars[key]
	return v, ok
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
