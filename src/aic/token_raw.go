package aic

import "fmt"

type RawToken struct {
	TokenCtx
	literal string
}

func NewRawToken(lit string) PromptToken {
	return &RawToken{literal: lit}
}

func (t *RawToken) Type() PromptTokenType { return PromptTokenRaw }
func (t *RawToken) Literal() string       { return t.literal }

func (t *RawToken) Validate(d *AiDir) error { return nil }

func (t *RawToken) AfterValidate(r *PromptReader, index int) error {
	t.bind(r, index)
	return nil
}

func (t *RawToken) Render(d *AiDir) (string, error) {
	return t.literal, nil
}

func (t *RawToken) String() string {
	display := t.literal
	if len(display) > 20 {
		display = display[:20] + "..."
	}
	return fmt.Sprintf("<%s: %q>", t.Type().String(), display)
}
