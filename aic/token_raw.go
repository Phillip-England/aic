package aic

import "fmt"

type RawToken struct {
	literal string
}

func NewRawToken(lit string) RawToken {
	return RawToken{literal: lit}
}

func (t RawToken) Type() PromptTokenType { return PromptTokenRaw }
func (t RawToken) Literal() string       { return t.literal }

func (t RawToken) Validate() error { return nil }

func (t RawToken) String() string {
	display := t.literal
	if len(display) > 20 {
		display = display[:20] + "..."
	}
	return fmt.Sprintf("<%s: %q>", t.Type().String(), display)
}
