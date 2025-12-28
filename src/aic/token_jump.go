package aic

type JumpHandler struct {
	noSpecial
}

func (JumpHandler) Name() string { return "jump" }

func (JumpHandler) Validate(args []string, d *AiDir) error {
	_ = args
	_ = d
	return nil
}

func (JumpHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = d
	_ = r
	_ = index
	_ = literal
	_ = args
	return "", nil
}

func (JumpHandler) RenderJump(d *AiDir, r *PromptReader, index int, literal string, x, y int) (string, error) {
	_ = d
	r.AddPostAction(PostAction{
		Phase: PostActionAfter,
		Kind:  PostActionJump,
		Index: index,
		Lit:   literal,
		X:     x,
		Y:     y,
	})
	return "", nil
}
