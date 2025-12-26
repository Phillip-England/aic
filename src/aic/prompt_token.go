package aic

import "fmt"

type PromptTokenType int

const (
	PromptTokenRaw PromptTokenType = iota
	PromptTokenDollar
)

func (t PromptTokenType) String() string {
	switch t {
	case PromptTokenRaw:
		return "Raw"
	case PromptTokenDollar:
		return "Dollar"
	default:
		return "Unknown"
	}
}

type PromptToken interface {
	fmt.Stringer
	Type() PromptTokenType
	Literal() string
	Validate(d *AiDir) error
	AfterValidate(r *PromptReader, index int) error
	Render(d *AiDir) (string, error)
	Reader() *PromptReader
	Index() int
	Prev() PromptToken
	Next() PromptToken
}