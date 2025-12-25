package aic

func TokenizePrompt(prompt string) []PromptToken {
	if prompt == "" {
		return nil
	}
	return scanPrompt(prompt)
}

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
		b := prompt[i]
		if !isWordStart(i) {
			continue
		}

		if b != '@' && b != '$' {
			continue
		}

		flushRaw(i)

		start := i
		j := i
		for j < len(prompt) && !isWS(prompt[j]) {
			j++
		}

		lit := prompt[start:j]
		if b == '@' {
			tokens = append(tokens, NewAtToken(lit))
		} else {
			tokens = append(tokens, NewDollarToken(lit))
		}

		i = j - 1
		rawStart = j
	}

	flushRaw(len(prompt))
	return tokens
}
