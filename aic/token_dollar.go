package aic

import (
	"fmt"
	"os"
	"strings"
)

type DollarToken struct {
	TokenCtx
	literal string // includes leading "$"
}

func NewDollarToken(lit string) PromptToken {
	return &DollarToken{literal: lit}
}

func (t *DollarToken) Type() PromptTokenType { return PromptTokenDollar }
func (t *DollarToken) Literal() string       { return t.literal }

func (t *DollarToken) Value() string {
	return strings.TrimPrefix(t.literal, "$")
}

func (t *DollarToken) Validate(d *AiDir) error {
	// No validation rules yet. (Still a real token.)
	return nil
}

func (t *DollarToken) AfterValidate(r *PromptReader, index int) error {
	t.bind(r, index)
	return nil
}

func (t *DollarToken) Render(d *AiDir) (string, error) {
	// System command: $CLEAR
	// When present in prompt.md, it resets prompt.md to just the header.
	if t.Value() == "CLEAR" {
		if d == nil {
			return "", fmt.Errorf("$CLEAR: missing ai dir")
		}
		path := d.PromptPath()
		if path == "" {
			return "", fmt.Errorf("$CLEAR: missing prompt path")
		}

		// Overwrite prompt.md with the header only.
		if err := os.WriteFile(path, []byte(promptHeader), 0o644); err != nil {
			return "", fmt.Errorf("$CLEAR: write prompt.md: %w", err)
		}

		// Do not render the token into the output.
		return "", nil
	}

	// Default: render literally as typed.
	return t.literal, nil
}

func (t *DollarToken) String() string {
	return fmt.Sprintf("<%s: %q>", t.Type().String(), t.literal)
}
