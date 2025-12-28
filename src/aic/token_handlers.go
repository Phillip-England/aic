package aic

type DollarHandler interface {
	Name() string
	Validate(args []string, d *AiDir) error
	Render(d *AiDir, r *PromptReader, index int, literal string, args []string) (string, error)
	RenderJump(d *AiDir, r *PromptReader, index int, literal string, x, y int) (string, error)
	RenderClick(d *AiDir, r *PromptReader, index int, literal string, button string) (string, error)
}

var dollarHandlers = map[string]DollarHandler{
	"clear":      ClearHandler{},
	"clearAfter": ClearAfterHandler{},

	"shell": ShellHandler{},
	"sh":    ShellHandler{},
	"skill": SkillHandler{},
	"path":  PathHandler{},
	"http":  HttpHandler{},

	"jump":  JumpHandler{},
	"click": ClickHandler{},
	"type":  TypeHandler{},
}

func LookupDollarHandler(name string) DollarHandler {
	return dollarHandlers[name]
}

type noSpecial struct{}

func (noSpecial) RenderJump(d *AiDir, r *PromptReader, index int, literal string, x, y int) (string, error) {
	_ = d
	_ = r
	_ = index
	_ = x
	_ = y
	return literal, nil
}

func (noSpecial) RenderClick(d *AiDir, r *PromptReader, index int, literal string, button string) (string, error) {
	_ = d
	_ = r
	_ = index
	_ = button
	return literal, nil
}
