package aic

import "fmt"

type ClearAfterHandler struct {
	noSpecial
}

func (ClearAfterHandler) Name() string { return "clearAfter" }

func (ClearAfterHandler) Validate(args []string, d *AiDir) error {
	if len(args) != 0 {
		return fmt.Errorf("$clearAfter takes no args")
	}
	if d == nil {
		return fmt.Errorf("$clearAfter requires AiDir")
	}
	return nil
}

func (ClearAfterHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = d
	_ = args

	r.AddPostAction(PostAction{
		Phase: PostActionAfter,
		Kind:  PostActionClear,
		Index: index,
		Lit:   literal,
	})

	return "", nil
}
