package aic

import (
	"fmt"
	"strings"
)

type AtToken struct {
	literal string // includes leading "@"
}

func NewAtToken(lit string) AtToken {
	return AtToken{literal: lit}
}

func (t AtToken) Type() PromptTokenType { return PromptTokenAt }
func (t AtToken) Literal() string       { return t.literal }

func (t AtToken) Value() string {
	return strings.TrimPrefix(t.literal, "@")
}

// For now, any word beginning with "@" is considered valid.
func (t AtToken) Validate() error { return nil }

func (t AtToken) String() string {
	return fmt.Sprintf("<%s: %q>", t.Type().String(), t.literal)
}
