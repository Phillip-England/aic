package aic

import "fmt"

// Deprecated: clearing is now automatic after every successful prompt copy.
// Kept as a no-op so old prompts that still include $clear() don't break.
type ClearHandler struct {
	noSpecial
}

func (ClearHandler) Name() string { return "clear" }

func (ClearHandler) Validate(args []string, d *AiDir) error {
	_ = d
	if len(args) != 0 {
		return fmt.Errorf("$clear takes no args")
	}
	return nil
}

func (ClearHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = d
	_ = r
	_ = index
	_ = literal
	_ = args
	// no-op
	return "", nil
}
