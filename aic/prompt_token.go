package aic

import "fmt"

type PromptTokenType int

const (
	PromptTokenRaw PromptTokenType = iota
	PromptTokenAt
)

func (t PromptTokenType) String() string {
	switch t {
	case PromptTokenRaw:
		return "Raw"
	case PromptTokenAt:
		return "At"
	default:
		return "Unknown"
	}
}

// PromptToken is produced by the prompt tokenizer.
//
// Validate() determines whether the token is "real".
// If Validate() returns an error, the tokenizer pipeline will downgrade it to Raw.
type PromptToken interface {
	fmt.Stringer
	Type() PromptTokenType
	Literal() string
	Validate() error
}
