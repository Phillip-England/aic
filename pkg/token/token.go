package token

import (
	"strconv"
	"strings"
	"time"

	"github.com/phillip-england/aic/pkg/dir"
)

// PostActionKind defines the type of side effect
type PostActionKind string

const (
	ActionJump  PostActionKind = "jump"
	ActionClick PostActionKind = "click"
	ActionType  PostActionKind = "type"
	ActionSleep PostActionKind = "sleep"
	ActionPress PostActionKind = "press"
	ActionClear PostActionKind = "clear"
)

// PostAction holds the details for a side effect
type PostAction struct {
	Kind  PostActionKind
	After bool // If true, runs after clipboard copy

	// Jump
	X, Y int    // Literal coordinates
	XVar string // Variable name for X (e.g. "AIC_X_START")
	YVar string // Variable name for Y

	// Type / Press
	Text  string
	Mods  []string
	Delay time.Duration
	Key   string

	// Click
	Button string
}

type Token interface {
	Literal() string
	Render(d *dir.AiDir, vars map[string]string) (string, []PostAction, error)
}

// --- Tokenizer ---

func Tokenize(text string) []Token {
	var tokens []Token
	current := text

	for {
		dollarIdx := strings.Index(current, "$")
		if dollarIdx == -1 {
			if current != "" {
				tokens = append(tokens, &RawToken{lit: current})
			}
			break
		}

		// Text before $
		if dollarIdx > 0 {
			tokens = append(tokens, &RawToken{lit: current[:dollarIdx]})
		}

		// Attempt to parse $token(...)
		rest := current[dollarIdx:]
		
		// Find closing parenthesis
		parenClose := -1
		depth := 0
		for i, char := range rest {
			if char == '(' {
				depth++
			} else if char == ')' {
				depth--
				if depth == 0 {
					parenClose = i
					break
				}
			}
		}

		if parenClose != -1 {
			// Extract full token string: $name(...)
			fullToken := rest[:parenClose+1]
			// Verify it looks like a function call
			parenOpen := strings.Index(fullToken, "(")
			if parenOpen != -1 {
				name := fullToken[1:parenOpen]
				// Basic sanity check: name shouldn't have whitespace
				if !strings.ContainsAny(name, " \t\n") {
					tokens = append(tokens, &DollarToken{lit: fullToken})
					current = rest[parenClose+1:]
					continue
				}
			}
		}

		// Fallback: treat $ as literal text if parsing failed
		tokens = append(tokens, &RawToken{lit: "$"})
		current = rest[1:]
	}

	return tokens
}

// --- Token Types ---

type RawToken struct{ lit string }
func (t *RawToken) Literal() string { return t.lit }
func (t *RawToken) Render(d *dir.AiDir, v map[string]string) (string, []PostAction, error) {
	return t.lit, nil, nil
}

type DollarToken struct{ lit string }
func (t *DollarToken) Literal() string { return t.lit }
func (t *DollarToken) Render(d *dir.AiDir, v map[string]string) (string, []PostAction, error) {
	// Remove $ and parse
	clean := strings.TrimPrefix(t.lit, "$")
	name, argsRaw := parseCall(clean)
	args := splitArgs(argsRaw)

	switch name {
	case "path":
		if len(args) == 0 {
			return t.lit, nil, nil
		}
		// Attempt to read file content
		// We expect d.ReadAnyFile to be available or we simulate it here
		// Assuming we can access the file system via dir package or similar
		content, err := d.ReadAnyFile(args[0])
		if err != nil {
			return fmtError(t.lit, err), nil, nil
		}
		return "FILE: " + args[0] + "\n" + content + "\n", nil, nil

	case "jump":
		// Usage: $jump(10, 20) OR $jump(AIC_X_START, AIC_Y_START)
		if len(args) < 2 {
			return "", nil, nil
		}
		
		pa := PostAction{Kind: ActionJump, After: true}

		// Parse X
		if val, err := strconv.Atoi(args[0]); err == nil {
			pa.X = val
		} else {
			pa.XVar = args[0] // It's a variable
		}

		// Parse Y
		if val, err := strconv.Atoi(args[1]); err == nil {
			pa.Y = val
		} else {
			pa.YVar = args[1] // It's a variable
		}

		return "", []PostAction{pa}, nil

	case "click":
		btn := "left"
		if len(args) > 0 {
			btn = strings.Trim(args[0], "\"")
		}
		return "", []PostAction{{Kind: ActionClick, Button: btn, After: true}}, nil

	case "type":
		// Usage: $type("text", ["CMD"])
		if len(args) < 1 { return "", nil, nil }
		
		text := strings.Trim(args[0], "\"")
		var mods []string
		
		if len(args) > 1 {
			// Parse ["A", "B"]
			rawMods := args[1]
			rawMods = strings.TrimPrefix(rawMods, "[")
			rawMods = strings.TrimSuffix(rawMods, "]")
			if rawMods != "" {
				parts := strings.Split(rawMods, ",")
				for _, p := range parts {
					m := strings.TrimSpace(p)
					m = strings.Trim(m, "\"")
					mods = append(mods, m)
				}
			}
		}
		return "", []PostAction{{Kind: ActionType, Text: text, Mods: mods, After: true}}, nil

	case "press":
		if len(args) < 1 { return "", nil, nil }
		key := strings.Trim(args[0], "\"")
		return "", []PostAction{{Kind: ActionPress, Key: key, After: true}}, nil

	case "sleep":
		if len(args) < 1 { return "", nil, nil }
		ms, _ := strconv.Atoi(args[0])
		return "", []PostAction{{Kind: ActionSleep, Delay: time.Duration(ms) * time.Millisecond, After: true}}, nil
	}

	return t.lit, nil, nil
}

// Helpers

func parseCall(s string) (string, string) {
	idx := strings.Index(s, "(")
	if idx == -1 {
		return s, ""
	}
	// Name is before (, args are inside the last )
	name := s[:idx]
	args := s[idx+1 : len(s)-1]
	return name, args
}

func splitArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	inBracket := false

	for _, r := range s {
		switch r {
		case '"':
			inQuote = !inQuote
			current.WriteRune(r)
		case '[':
			inBracket = true
			current.WriteRune(r)
		case ']':
			inBracket = false
			current.WriteRune(r)
		case ',':
			if !inQuote && !inBracket {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, strings.TrimSpace(current.String()))
	}
	return args
}

func fmtError(lit string, err error) string {
	return lit + " [Error: " + err.Error() + "]"
}