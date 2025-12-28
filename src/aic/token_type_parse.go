package aic

import (
	"fmt"
	"strconv"
	"strings"
)

func parseTypeArgs(args string) (text string, mods []string, delayMs int, err error) {
	s := strings.TrimSpace(args)
	if s == "" {
		return "", nil, 0, fmt.Errorf("missing args")
	}

	// 1) text: required quoted string
	if len(s) == 0 || s[0] != '"' {
		return "", nil, 0, fmt.Errorf(`first arg must be a quoted string, e.g. $type("hello", ["SHIFT"])`)
	}
	txt, n, err := parseQuotedStringPrefix(s)
	if err != nil {
		return "", nil, 0, err
	}
	text = txt
	s = strings.TrimSpace(s[n:])
	if s == "" {
		return text, nil, 0, nil
	}

	// 2) expect comma then either [mods] OR delayMs
	if s[0] != ',' {
		return "", nil, 0, fmt.Errorf("expected ',' after first arg")
	}
	s = strings.TrimSpace(s[1:])
	if s == "" {
		return "", nil, 0, fmt.Errorf("expected modifier list or delay after ','")
	}

	// Parse either modifier list or delay
	if strings.HasPrefix(s, "[") {
		mods, n, err = parseStringListPrefix(s)
		if err != nil {
			return "", nil, 0, err
		}
		s = strings.TrimSpace(s[n:])
		if s == "" {
			return text, mods, 0, nil
		}

		// Optional third arg: delayMs
		if s[0] != ',' {
			return "", nil, 0, fmt.Errorf("unexpected trailing content after modifiers: %q", s)
		}
		s = strings.TrimSpace(s[1:])
		if s == "" {
			return "", nil, 0, fmt.Errorf("expected delayMs after ','")
		}
	}

	// Now parse delayMs (bare integer) if present
	if s != "" {
		i := 0
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		if i == 0 {
			return "", nil, 0, fmt.Errorf("expected delayMs as an integer (milliseconds), got: %q", s)
		}
		v, aerr := strconv.Atoi(s[:i])
		if aerr != nil {
			return "", nil, 0, fmt.Errorf("invalid delayMs: %w", aerr)
		}
		delayMs = v
		s = strings.TrimSpace(s[i:])
		if s != "" {
			return "", nil, 0, fmt.Errorf("unexpected trailing content after delayMs: %q", s)
		}
	}

	return text, mods, delayMs, nil
}

// parseQuotedStringPrefix parses a JSON-ish quoted string at s[0] == '"' and returns (value, consumedBytes).
// Supports common escapes: \" \\ \n \t \r
func parseQuotedStringPrefix(s string) (string, int, error) {
	if s == "" || s[0] != '"' {
		return "", 0, fmt.Errorf("expected '\"' at start of string")
	}
	var buf strings.Builder
	escaped := false
	i := 1
	for i < len(s) {
		ch := s[i]
		if escaped {
			switch ch {
			case '"':
				buf.WriteByte('"')
			case '\\':
				buf.WriteByte('\\')
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case 'r':
				buf.WriteByte('\r')
			default:
				buf.WriteByte(ch)
			}
			escaped = false
			i++
			continue
		}
		if ch == '\\' {
			escaped = true
			i++
			continue
		}
		if ch == '"' {
			// consumed includes closing quote
			return buf.String(), i + 1, nil
		}
		buf.WriteByte(ch)
		i++
	}
	return "", 0, fmt.Errorf("unterminated string")
}

// parseStringListPrefix parses:
//
//	["SHIFT", "CONTROL"]  (quotes optional for identifiers; e.g. [SHIFT, CONTROL] is allowed too)
//
// returns (items, consumedBytes).
func parseStringListPrefix(s string) ([]string, int, error) {
	if s == "" || s[0] != '[' {
		return nil, 0, fmt.Errorf("expected '[' at start of list")
	}

	i := 1
	skipWS := func() {
		for i < len(s) {
			switch s[i] {
			case ' ', '\n', '\t', '\r':
				i++
			default:
				return
			}
		}
	}

	var out []string
	for {
		skipWS()
		if i >= len(s) {
			return nil, 0, fmt.Errorf("unterminated list")
		}
		if s[i] == ']' {
			i++
			return out, i, nil
		}

		// item: quoted string or identifier
		if s[i] == '"' {
			val, n, err := parseQuotedStringPrefix(s[i:])
			if err != nil {
				return nil, 0, err
			}
			out = append(out, val)
			i += n
		} else {
			start := i
			for i < len(s) {
				ch := s[i]
				if (ch >= 'a' && ch <= 'z') ||
					(ch >= 'A' && ch <= 'Z') ||
					(ch >= '0' && ch <= '9') ||
					ch == '_' {
					i++
					continue
				}
				break
			}
			if i == start {
				return nil, 0, fmt.Errorf("expected modifier name or quoted string in list")
			}
			out = append(out, s[start:i])
		}

		skipWS()
		if i >= len(s) {
			return nil, 0, fmt.Errorf("unterminated list")
		}
		if s[i] == ',' {
			i++
			continue
		}
		if s[i] == ']' {
			i++
			return out, i, nil
		}
		return nil, 0, fmt.Errorf("expected ',' or ']' in list, got %q", s[i])
	}
}
