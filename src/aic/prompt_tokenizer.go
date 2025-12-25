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
		// Keep it simple: A-Z a-z 0-9 _
		return (b >= 'a' && b <= 'z') ||
			(b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') ||
			b == '_'
	}

	// parseDollarToken tries to parse either:
	//  $NAME(...)    (paren form, may include whitespace inside)
	// or falls back to:
	//  $NAME         (until whitespace)
	//
	// Returns (endIndexExclusive, okParsedParenForm)
	parseDollarToken := func(i int) (int, bool) {
		// prompt[i] must be '$'
		j := i + 1
		for j < len(prompt) && isIdentChar(prompt[j]) {
			j++
		}
		// No identifier? fall back to whitespace form
		if j == i+1 {
			k := i
			for k < len(prompt) && !isWS(prompt[k]) {
				k++
			}
			return k, false
		}

		// If next char is not '(', it's whitespace-delimited token
		if j >= len(prompt) || prompt[j] != '(' {
			k := i
			for k < len(prompt) && !isWS(prompt[k]) {
				k++
			}
			return k, false
		}

		// We have $NAME( ... ) form. Parse matching ')' with:
		// - nesting depth for parentheses
		// - quote awareness (single/double)
		// - backtick awareness (shell commands, e.g. $EXE(`ls`))
		// - backslash escapes
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

			// 1. Handle Backticks
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

			// 2. Handle Single Quotes
			if inSingle {
				if ch == '\'' {
					inSingle = false
				}
				k++
				continue
			}

			// 3. Handle Double Quotes
			if inDouble {
				if ch == '"' {
					inDouble = false
				}
				k++
				continue
			}

			// Not in any quotes/backticks, check for start of quotes
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

			// 4. Handle Parentheses
			if ch == '(' {
				depth++
				k++
				continue
			}
			if ch == ')' {
				depth--
				k++
				if depth == 0 {
					// k is exclusive end
					return k, true
				}
				continue
			}

			k++
		}

		// No closing ')' found -> fall back to whitespace-delimited
		k = i
		for k < len(prompt) && !isWS(prompt[k]) {
			k++
		}
		return k, false
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

		if b == '@' {
			// existing behavior: @ token ends at whitespace
			start := i
			j := i
			for j < len(prompt) && !isWS(prompt[j]) {
				j++
			}
			lit := prompt[start:j]
			tokens = append(tokens, NewAtToken(lit))
			i = j - 1
			rawStart = j
			continue
		}

		// Dollar token: possibly $EXE(...)
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
