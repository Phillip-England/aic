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

	isIdentChar := func(b byte) bool {
		return (b >= 'a' && b <= 'z') ||
			(b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') ||
			b == '_'
	}

	parseDollarToken := func(i int) (int, bool) {
		j := i + 1
		for j < len(prompt) && isIdentChar(prompt[j]) {
			j++
		}

		// If there's no identifier after '$', treat it as a plain word token
		// (consume until whitespace).
		if j == i+1 {
			k := i
			for k < len(prompt) && !isWS(prompt[k]) {
				k++
			}
			return k, false
		}

		// Must have '(' to be a call token.
		if j >= len(prompt) || prompt[j] != '(' {
			k := i
			for k < len(prompt) && !isWS(prompt[k]) {
				k++
			}
			return k, false
		}

		depth := 0
		inSingle := false
		inDouble := false
		inBacktick := false
		escaped := false

		k := j // at '('
		for k < len(prompt) {
			ch := prompt[k]

			if escaped {
				escaped = false
				k++
				continue
			}
			if ch == '\\' {
				escaped = true
				k++
				continue
			}

			if inBacktick {
				if ch == '`' {
					inBacktick = false
				}
				k++
				continue
			}
			if !inSingle && !inDouble && ch == '`' {
				inBacktick = true
				k++
				continue
			}

			if inSingle {
				if ch == '\'' {
					inSingle = false
				}
				k++
				continue
			}
			if inDouble {
				if ch == '"' {
					inDouble = false
				}
				k++
				continue
			}

			if ch == '\'' {
				inSingle = true
				k++
				continue
			}
			if ch == '"' {
				inDouble = true
				k++
				continue
			}

			if ch == '(' {
				depth++
				k++
				continue
			}
			if ch == ')' {
				depth--
				k++
				if depth == 0 {
					return k, true
				}
				continue
			}

			k++
		}

		// Unterminated call: fallback to consuming to whitespace.
		k = i
		for k < len(prompt) && !isWS(prompt[k]) {
			k++
		}
		return k, false
	}

	for i := 0; i < len(prompt); i++ {
		b := prompt[i]

		// Escape support: if a "$token(...)" is preceded by "\" at a word boundary,
		// treat it as raw text and drop the "\".
		//
		// Example: "\$path(".")" -> literal "$path(".")" (no expansion)
		if isWordStart(i) && b == '\\' && i+1 < len(prompt) && prompt[i+1] == '$' {
			flushRaw(i)      // flush content before the backslash
			rawStart = i + 1 // skip the backslash; keep '$' in output
			continue
		}

		if !isWordStart(i) {
			continue
		}
		if b != '$' {
			continue
		}

		flushRaw(i)
		start := i
		end, _ := parseDollarToken(i)
		lit := prompt[start:end]
		tokens = append(tokens, NewDollarToken(lit))
		i = end - 1
		rawStart = end
	}

	flushRaw(len(prompt))
	return tokens
}
