// --- START FILE: aic/prompt_reader.go ---
package aic

import "fmt"

// PromptReader is a tiny holder for the prompt text.
// Next stage can add parsing, sections, etc.
type PromptReader struct {
	Text string
}

func NewPromptReader(text string) *PromptReader {
	return &PromptReader{Text: text}
}

func (p *PromptReader) String() string {
	return p.Text
}

func (p *PromptReader) Print() {
	fmt.Print(p.Text)
}

// --- END FILE: aic/prompt_reader.go ---
