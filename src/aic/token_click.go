package aic

type ClickHandler struct {
	noSpecial
}

func (ClickHandler) Name() string { return "click" }

func (ClickHandler) Validate(args []string, d *AiDir) error {
	_ = args
	_ = d
	return nil
}

func (ClickHandler) Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error) {
	_ = d
	_ = r
	_ = index
	_ = literal
	_ = args
	return "", nil
}

func (ClickHandler) RenderClick(d *AiDir, r *PromptReader, index int, literal string, button string) (string, error) {
	_ = d
	r.AddPostAction(PostAction{
		Phase:  PostActionAfter,
		Kind:   PostActionClick,
		Index:  index,
		Lit:    literal,
		Button: button,
	})
	return "", nil
}
