package aic

import "fmt"

// TypeHandler is treated like jump/click: it renders empty and registers a PostAction.
// Validation/parsing is handled in DollarToken.Validate via parseTypeArgs.
type TypeHandler struct {
	noSpecial
}

func (TypeHandler) Name() string { return "type" }

func (TypeHandler) Validate(args []string, d *AiDir) error {
	_ = d
	// Not used for $type (custom parsing), but keep sane behavior if called.
	if len(args) == 0 {
		return fmt.Errorf("$type expects at least 1 arg")
	}
	return nil
}

func (TypeHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = d
	_ = r
	_ = index
	_ = literal
	_ = args
	// $type is rendered via DollarToken.Render special-case.
	return "", nil
}
