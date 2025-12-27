package aic

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type DollarToken struct {
	TokenCtx
	literal string

	isSh      bool
	shCmd     string
	shCmdDisp string

	isClear bool

	isSkill   bool
	skillName string

	isAt     bool
	atTarget string

	isHttp  bool
	httpUrl string

	isJump bool
	jumpX  int
	jumpY  int

	isClick     bool
	clickButton string
}

func NewDollarToken(lit string) PromptToken {
	return &DollarToken{literal: lit}
}

func (t *DollarToken) Type() PromptTokenType { return PromptTokenDollar }
func (t *DollarToken) Literal() string       { return t.literal }

func (t *DollarToken) Value() string {
	return strings.TrimPrefix(t.literal, "$")
}

func parseIntPairArgs(args string) (int, int, error) {
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected two integers")
	}
	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, err
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func parseOneArg(args string) (string, error) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", fmt.Errorf("missing argument")
	}
	if strings.HasPrefix(args, `"`) {
		list, err := parseMultiStringArgs(args)
		if err != nil {
			return "", err
		}
		if len(list) != 1 {
			return "", fmt.Errorf("expected exactly one argument")
		}
		return list[0], nil
	}
	if strings.Contains(args, ",") {
		return "", fmt.Errorf("expected single argument")
	}
	return args, nil
}

func (t *DollarToken) Validate(d *AiDir) error {
	*t = DollarToken{literal: t.literal}

	name, args, ok := parseDollarCall(t.Value())
	if !ok {
		return nil
	}

	switch name {
	case "jump":
		x, y, err := parseIntPairArgs(args)
		if err != nil {
			return fmt.Errorf("$jump: %w", err)
		}
		t.isJump = true
		t.jumpX = x
		t.jumpY = y
		return nil

	case "click":
		btn, err := parseOneArg(args)
		if err != nil {
			return fmt.Errorf("$click: %w", err)
		}
		btn = strings.ToLower(btn)
		if btn != "left" && btn != "right" {
			return fmt.Errorf(`$click: expected "left" or "right"`)
		}
		t.isClick = true
		t.clickButton = btn
		return nil
	}

	return t.validateExistingTokens(d, name, args)
}

func (t *DollarToken) validateExistingTokens(d *AiDir, name, args string) error {
	argList, err := parseMultiStringArgs(args)
	if err != nil {
		return err
	}

	switch name {
	case "clear":
		if len(argList) != 0 {
			return fmt.Errorf("$clear takes no args")
		}
		t.isClear = true

	case "shell", "sh":
		if len(argList) != 1 {
			return fmt.Errorf("$sh takes 1 arg")
		}
		t.isSh = true
		t.shCmd = argList[0]
		t.shCmdDisp = fmt.Sprintf("%q", argList[0])

	case "skill":
		if len(argList) != 1 {
			return fmt.Errorf("$skill takes 1 arg")
		}
		t.isSkill = true
		t.skillName = argList[0]

	case "path", "at":
		if len(argList) == 0 {
			return fmt.Errorf("$at needs paths")
		}
		t.isAt = true
		t.atTarget = filepath.Join(append([]string{d.WorkingDir}, argList...)...)

	case "http":
		if len(argList) != 1 {
			return fmt.Errorf("$http takes 1 arg")
		}
		t.isHttp = true
		t.httpUrl = argList[0]
	}

	return nil
}

func (t *DollarToken) AfterValidate(r *PromptReader, index int) error {
	t.bind(r, index)
	return nil
}

func (t *DollarToken) Render(d *AiDir) (string, error) {
	if t.isJump {
		t.Reader().AddPostAction(PostAction{
			Kind:  PostActionJump,
			Index: t.Index(),
			Lit:   t.literal,
			X:     t.jumpX,
			Y:     t.jumpY,
		})
		return "", nil
	}

	if t.isClick {
		t.Reader().AddPostAction(PostAction{
			Kind:   PostActionClick,
			Index:  t.Index(),
			Lit:    t.literal,
			Button: t.clickButton,
		})
		return "", nil
	}

	/* existing render logic unchanged below */
	/* keep your existing $clear, $sh, $path, etc code */
	return t.literal, nil
}

func (t *DollarToken) String() string {
	return fmt.Sprintf("<Dollar %q>", t.literal)
}

func parseDollarCall(val string) (string, string, bool) {
	s := strings.TrimSpace(val)
	open := strings.IndexByte(s, '(')
	if open < 0 {
		return "", "", false
	}
	if !strings.HasSuffix(s, ")") {
		return "", "", false
	}
	name := strings.TrimSpace(s[:open])
	args := s[open+1 : len(s)-1] // inside parentheses
	return name, args, true
}

// Parses a comma-separated list of *double-quoted* string args.
// Examples:
//
//	`"a"` -> ["a"]
//	`"a","b"` -> ["a","b"]
//
// Supports escapes: \" \\ \n \t \r
func parseMultiStringArgs(args string) ([]string, error) {
	var results []string
	rest := strings.TrimSpace(args)
	if rest == "" {
		return []string{}, nil
	}

	for len(rest) > 0 {
		if rest[0] != '"' {
			return nil, fmt.Errorf("expected '\"' at start of argument, got %q", rest[0])
		}

		var buf strings.Builder
		escaped := false
		i := 1
		foundEnd := false

		for i < len(rest) {
			ch := rest[i]

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
				foundEnd = true
				i++
				break
			}

			buf.WriteByte(ch)
			i++
		}

		if !foundEnd {
			return nil, fmt.Errorf("unterminated string")
		}

		results = append(results, buf.String())

		rest = strings.TrimSpace(rest[i:])
		if len(rest) == 0 {
			break
		}
		if rest[0] != ',' {
			return nil, fmt.Errorf("expected comma between arguments")
		}
		rest = strings.TrimSpace(rest[1:])
	}

	return results, nil
}
