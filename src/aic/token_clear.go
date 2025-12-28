package aic

import (
	"fmt"
	"os"
)

type ClearHandler struct {
	noSpecial
}

func (ClearHandler) Name() string { return "clear" }

func (ClearHandler) Validate(args []string, d *AiDir) error {
	if len(args) != 0 {
		return fmt.Errorf("$clear takes no args")
	}
	if d == nil {
		return fmt.Errorf("$clear requires AiDir")
	}
	return nil
}

func (ClearHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = r
	_ = index
	_ = literal
	_ = args

	// Reset prompt.md to the header
	if err := os.WriteFile(d.PromptPath(), []byte(promptHeader), 0o644); err != nil {
		return "", fmt.Errorf("clear prompt.md: %w", err)
	}
	return "", nil
}
