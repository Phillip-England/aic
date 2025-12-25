package aic

import "fmt"

type PromptTokenType int

const (
	PromptTokenRaw PromptTokenType = iota
	PromptTokenAt
	PromptTokenDollar
)

func (t PromptTokenType) String() string {
	switch t {
	case PromptTokenRaw:
		return "Raw"
	case PromptTokenAt:
		return "At"
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

	// Validate runs during the validation pass, using AiDir context.
	// If it returns an error, the reader downgrades this token to Raw.
	Validate(d *AiDir) error

	// AfterValidate runs after validation/downgrade and binds reader/index.
	AfterValidate(r *PromptReader, index int) error

	// Render produces this token's contribution to the final output string.
	Render(d *AiDir) (string, error)

	Reader() *PromptReader
	Index() int
	Prev() PromptToken
	Next() PromptToken
}
