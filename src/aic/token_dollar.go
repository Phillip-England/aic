package aic

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type DollarToken struct {
	TokenCtx

	literal string
	name    string
	args    string
	argList []string
	handler DollarHandler

	jumpX, jumpY         int
	jumpXExpr, jumpYExpr string

	clickButton string

	typeText    string
	typeMods    []string
	typeDelayMs int

	sleepDur time.Duration

	pressKey string
}

func NewDollarToken(lit string) PromptToken {
	return &DollarToken{literal: lit}
}

func (t *DollarToken) Type() PromptTokenType { return PromptTokenDollar }
func (t *DollarToken) Literal() string       { return t.literal }
func (t *DollarToken) String() string        { return fmt.Sprintf("<Dollar %q>", t.literal) }

func (t *DollarToken) Value() string {
	return strings.TrimPrefix(t.literal, "$")
}

func (t *DollarToken) AfterValidate(r *PromptReader, index int) error {
	t.bind(r, index)
	return nil
}

func (t *DollarToken) Validate(d *AiDir) error {
	lit := t.literal
	*t = DollarToken{literal: lit}

	name, args, ok := parseDollarCall(t.Value())
	if !ok {
		return nil
	}

	t.name = name
	t.args = args

	switch name {
	case "jump":
		x, y, xExpr, yExpr, err := parseIntOrIdentPair(args)
		if err != nil {
			return fmt.Errorf("$jump: %w", err)
		}
		t.jumpX, t.jumpY = x, y
		t.jumpXExpr, t.jumpYExpr = xExpr, yExpr
		t.handler = JumpHandler{}
		return nil

	case "click":
		if strings.TrimSpace(args) == "" {
			t.clickButton = "left"
			t.handler = ClickHandler{}
			return nil
		}
		btn, err := parseOneArg(args)
		if err != nil {
			return fmt.Errorf("$click: %w", err)
		}
		btn = strings.ToLower(strings.TrimSpace(btn))
		if btn != "left" && btn != "right" {
			return fmt.Errorf(`$click: expected "left" or "right"`)
		}
		t.clickButton = btn
		t.handler = ClickHandler{}
		return nil

	case "type":
		text, mods, delayMs, err := parseTypeArgs(args)
		if err != nil {
			return fmt.Errorf("$type: %w", err)
		}
		t.typeText = text
		t.typeMods = mods
		t.typeDelayMs = delayMs
		t.handler = TypeHandler{}
		return nil

	case "sleep":
		dur, err := parseSleepArgs(args)
		if err != nil {
			return fmt.Errorf("$sleep: %w", err)
		}
		t.sleepDur = dur
		t.handler = SleepHandler{}
		return nil

	case "press":
		key, err := parseOneArg(args)
		if err != nil {
			return fmt.Errorf("$press: %w", err)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("$press: key cannot be empty")
		}
		if !looksLikeHotkeyKey(key) {
			return fmt.Errorf("$press: unsupported key %q", key)
		}
		t.pressKey = key
		t.handler = PressHandler{}
		return nil
	}

	list, err := parseMultiStringArgs(args)
	if err != nil {
		return err
	}
	t.argList = list

	h := LookupDollarHandler(name)
	if h == nil {
		return nil
	}
	if err := h.Validate(list, d); err != nil {
		return err
	}
	t.handler = h
	return nil
}

func (t *DollarToken) Render(d *AiDir) (string, error) {
	if t.handler == nil {
		return t.literal, nil
	}

	switch t.name {
	case "jump":
		t.Reader().AddPostAction(PostAction{
			Phase: PostActionAfter,
			Kind:  PostActionJump,
			Index: t.Index(),
			Lit:   t.literal,
			X:     t.jumpX,
			Y:     t.jumpY,
			XExpr: t.jumpXExpr,
			YExpr: t.jumpYExpr,
		})
		return "", nil

	case "click":
		t.Reader().AddPostAction(PostAction{
			Phase:  PostActionAfter,
			Kind:   PostActionClick,
			Index:  t.Index(),
			Lit:    t.literal,
			Button: t.clickButton,
		})
		return "", nil

	case "type":
		t.Reader().AddPostAction(PostAction{
			Phase:   PostActionAfter,
			Kind:    PostActionType,
			Index:   t.Index(),
			Lit:     t.literal,
			Text:    t.typeText,
			Mods:    append([]string(nil), t.typeMods...),
			DelayMs: t.typeDelayMs,
		})
		return "", nil

	case "sleep":
		t.Reader().AddPostAction(PostAction{
			Phase: PostActionAfter,
			Kind:  PostActionSleep,
			Index: t.Index(),
			Lit:   t.literal,
			Sleep: t.sleepDur,
		})
		return "", nil

	case "press":
		t.Reader().AddPostAction(PostAction{
			Phase: PostActionAfter,
			Kind:  PostActionPress,
			Index: t.Index(),
			Lit:   t.literal,
			Key:   t.pressKey,
		})
		return "", nil

	default:
		return t.handler.Render(d, t.Reader(), t.Index(), t.literal, t.argList)
	}
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

func parseIntOrIdentPair(args string) (x int, y int, xExpr string, yExpr string, err error) {
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return 0, 0, "", "", fmt.Errorf("expected two values (x,y)")
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	if left == "" || right == "" {
		return 0, 0, "", "", fmt.Errorf("expected two values (x,y)")
	}
	parseSide := func(s string) (int, string, error) {
		if n, aerr := strconv.Atoi(s); aerr == nil {
			return n, "", nil
		}
		if isIdent(s) {
			return 0, s, nil
		}
		return 0, "", fmt.Errorf("invalid value %q (want int or IDENT)", s)
	}
	xv, xe, e1 := parseSide(left)
	if e1 != nil {
		return 0, 0, "", "", e1
	}
	yv, ye, e2 := parseSide(right)
	if e2 != nil {
		return 0, 0, "", "", e2
	}
	return xv, yv, xe, ye, nil
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '_' {
			continue
		}
		return false
	}
	if _, err := strconv.Atoi(s); err == nil {
		return false
	}
	return true
}
