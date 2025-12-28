package aic

import "fmt"

type SleepHandler struct {
	noSpecial
}

func (SleepHandler) Name() string { return "sleep" }

func (SleepHandler) Validate(args []string, d *AiDir) error {
	_ = d
	// Sleep is parsed specially in DollarToken.Validate (supports int ms or duration string),
	// so this handler is mostly here so LookupDollarHandler knows the token exists.
	// If we end up here, enforce 1 arg for safety.
	if len(args) != 1 {
		return fmt.Errorf("$sleep takes exactly 1 arg")
	}
	return nil
}

func (SleepHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = d
	_ = r
	_ = index
	_ = args
	// Actual sleep behavior is emitted as a PostAction by DollarToken when it recognizes $sleep(...)
	return "", nil
}
