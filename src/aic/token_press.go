package aic

import (
	"fmt"
	"strings"
)

type PressHandler struct {
	noSpecial
}

func (PressHandler) Name() string { return "press" }

func (PressHandler) Validate(args []string, d *AiDir) error {
	_ = d
	if len(args) != 1 {
		return fmt.Errorf("$press takes exactly 1 arg")
	}
	key := strings.TrimSpace(args[0])
	if key == "" {
		return fmt.Errorf("$press key cannot be empty")
	}
	if !looksLikeHotkeyKey(key) {
		return fmt.Errorf("$press: unsupported key %q", key)
	}
	return nil
}

func (PressHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = d
	key := strings.TrimSpace(args[0])
	r.AddPostAction(PostAction{
		Phase: PostActionAfter,
		Kind:  PostActionPress,
		Index: index,
		Lit:   literal,
		Key:   key,
	})
	return "", nil
}
