package aic

// TokenizePrompt scans the prompt into tokens and then validates them.
// Tokens that fail validation are downgraded to RawToken (keeping the same literal).
func TokenizePrompt(prompt string) []PromptToken {
	if prompt == "" {
		return nil
	}

	toks := scanPrompt(prompt)
	return validateOrDowngrade(toks)
}

// scanPrompt does ONLY scanning/token discovery (no validation).
func scanPrompt(prompt string) []PromptToken {
	var tokens []PromptToken
	rawStart := 0

	flushRaw := func(end int) {
		if end <= rawStart {
			return
		}
		tokens = append(tokens, NewRawToken(prompt[rawStart:end]))
		rawStart = end
	}

	isWS := func(b byte) bool {
		return b == ' ' || b == '\n' || b == '\t' || b == '\r'
	}

	isWordStart := func(i int) bool {
		if i == 0 {
			return true
		}
		return isWS(prompt[i-1])
	}

	for i := 0; i < len(prompt); i++ {
		if prompt[i] != '@' || !isWordStart(i) {
			continue
		}

		// Found '@' at a word start -> token candidate.
		flushRaw(i)

		start := i
		j := i
		for j < len(prompt) && !isWS(prompt[j]) {
			j++
		}

		atLit := prompt[start:j]
		tokens = append(tokens, NewAtToken(atLit))

		i = j - 1
		rawStart = j
	}

	flushRaw(len(prompt))
	return tokens
}

// validateOrDowngrade runs Validate() on each token.
// If Validate() fails, the token becomes RawToken with the same literal text.
func validateOrDowngrade(tokens []PromptToken) []PromptToken {
	if len(tokens) == 0 {
		return tokens
	}

	out := make([]PromptToken, 0, len(tokens))
	for _, tok := range tokens {
		if err := tok.Validate(); err != nil {
			out = append(out, NewRawToken(tok.Literal()))
			continue
		}
		out = append(out, tok)
	}
	return out
}
