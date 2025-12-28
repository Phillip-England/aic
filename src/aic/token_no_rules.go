package aic

import "fmt"

type NoRulesHandler struct {
	noSpecial
}

func (NoRulesHandler) Name() string { return "norules" }

func (NoRulesHandler) Validate(args []string, d *AiDir) error {
	if len(args) != 0 {
		return fmt.Errorf("$norules takes no args")
	}
	return nil
}

func (NoRulesHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	// This token renders to nothing; its presence is checked in CLI.renderPromptToClipboard
	return "", nil
}